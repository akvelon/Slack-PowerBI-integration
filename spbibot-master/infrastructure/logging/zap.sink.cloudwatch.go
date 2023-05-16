package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"go.uber.org/zap"


)

type cloudwatchSink struct {
	cloudWatchLogs *cloudwatchlogs.CloudWatchLogs
	config         *config.LoggerConfig
	fallbackLogger *log.Logger
	groupName      string
	streamName     string
	sequenceToken  string
	batch          []*cloudwatchlogs.InputLogEvent
	batchLocker    sync.Locker
}

func newCloudwatchSink(w *cloudwatchlogs.CloudWatchLogs, c *config.LoggerConfig, fallback *log.Logger) *cloudwatchSink {
	return &cloudwatchSink{
		cloudWatchLogs: w,
		config:         c,
		fallbackLogger: fallback,
		batchLocker:    &sync.Mutex{},
	}
}

func (s *cloudwatchSink) Write(bs []byte) (int, error) {
	s.batchLocker.Lock()
	defer s.batchLocker.Unlock()

	if len(s.batch) == int(s.config.BatchSize) {
		err := s.writeBatch()
		if err != nil {
			return 0, err
		}
	}

	err := s.fillBatch(bs)
	if err != nil {
		return 0, err
	}

	return len(bs), nil
}

func (s *cloudwatchSink) Close() error {
	return nil
}

func (s *cloudwatchSink) Sync() error {
	s.batchLocker.Lock()
	defer s.batchLocker.Unlock()

	return s.writeBatch()
}

func (s *cloudwatchSink) fillBatch(bs []byte) error {
	m := string(bs)
	e := cloudwatchlogs.InputLogEvent{
		Message: &m,
	}

	timestamp, err := extractTimestamp(bs)
	if err != nil {
		return err
	}

	unixMilliseconds := timestamp.UnixNano() / 1000000
	e.Timestamp = &unixMilliseconds

	s.batch = append(s.batch, &e)

	return nil
}

func extractTimestamp(bs []byte) (time.Time, error) {
	raw := map[string]interface{}{}
	err := json.Unmarshal(bs, &raw)
	if err != nil {
		return time.Now().UTC(), nil
	}

	timeField := raw[string(keyTime)]
	if timeField != nil {
		timeString, ok := timeField.(string)
		if !ok {
			return time.Time{}, fmt.Errorf("invalid time field value type")
		}

		timestamp, err := time.Parse(string(layoutISO8601), timeString)
		if err != nil {
			return time.Time{}, err
		}

		return timestamp, nil
	}

	return time.Time{}, fmt.Errorf("time field not found")
}

func (s *cloudwatchSink) writeBatch() error {
	if len(s.batch) == 0 {
		return nil
	}

	defer func() {
		s.batch = nil
	}()

	i := &cloudwatchlogs.PutLogEventsInput{}
	i = i.
		SetLogGroupName(s.groupName).
		SetLogStreamName(s.streamName).
		SetLogEvents(s.batch)
	if s.sequenceToken != "" {
		i = i.SetSequenceToken(s.sequenceToken)
	}

	o, err := s.cloudWatchLogs.PutLogEvents(i)
	if err != nil {
		return err
	}

	s.sequenceToken = *o.NextSequenceToken
	if o.RejectedLogEventsInfo != nil {
		expiredLogEventEndIndex := int64(0)
		if o.RejectedLogEventsInfo.ExpiredLogEventEndIndex != nil {
			expiredLogEventEndIndex = *o.RejectedLogEventsInfo.ExpiredLogEventEndIndex
		}

		tooNewLogEventStartIndex := int64(0)
		if o.RejectedLogEventsInfo.TooNewLogEventStartIndex != nil {
			tooNewLogEventStartIndex = *o.RejectedLogEventsInfo.TooNewLogEventStartIndex
		}

		tooOldLogEventEndIndex := int64(0)
		if o.RejectedLogEventsInfo.TooOldLogEventEndIndex != nil {
			tooOldLogEventEndIndex = *o.RejectedLogEventsInfo.TooOldLogEventEndIndex
		}

		return fmt.Errorf("events were rejected: expiredLogEventEndIndex=%v tooNewLogEventStartIndex=%v tooOldLogEventEndIndex=%v", expiredLogEventEndIndex, tooNewLogEventStartIndex, tooOldLogEventEndIndex)
	}

	return nil
}

type pathOptions struct {
	Host string
}

func (o *pathOptions) useWith(u *url.URL) (*url.URL, error) {
	source := u.String()
	source, err := url.PathUnescape(source)
	if err != nil {
		return nil, err
	}

	sourceTemplate, err := template.New("").Option("missingkey=error").Parse(source)
	if err != nil {
		return nil, err
	}

	targetBuffer := bytes.Buffer{}
	err = sourceTemplate.Execute(&targetBuffer, *o)
	if err != nil {
		return nil, err
	}

	target := targetBuffer.String()

	return url.Parse(target)
}

func cloudwatchSinkFactory(w *cloudwatchlogs.CloudWatchLogs, c *config.LoggerConfig, o *pathOptions, fallback *log.Logger) func(u *url.URL) (zap.Sink, error) {
	return func(u *url.URL) (zap.Sink, error) {
		u, err := o.useWith(u)
		if err != nil {
			return nil, err
		}

		group := u.Host
		cg := &cloudwatchlogs.CreateLogGroupInput{}
		cg = cg.SetLogGroupName(group)
		_, err = w.CreateLogGroup(cg)
		if err != nil {
			_, ok := err.(*cloudwatchlogs.ResourceAlreadyExistsException)
			if !ok {
				return nil, err
			}
		}

		sink := newCloudwatchSink(w, c, fallback)
		sink.groupName = group
		stream := strings.TrimPrefix(u.Path, "/")
		sink.streamName = stream

		cs := &cloudwatchlogs.CreateLogStreamInput{}
		cs = cs.
			SetLogGroupName(group).
			SetLogStreamName(stream)
		_, err = w.CreateLogStream(cs)
		if err != nil {
			_, ok := err.(*cloudwatchlogs.ResourceAlreadyExistsException)
			if !ok {
				return nil, err
			}

			d := &cloudwatchlogs.DescribeLogStreamsInput{}
			d = d.
				SetLimit(50).
				SetLogGroupName(group)
			ss, err := w.DescribeLogStreams(d)
			if err != nil {
				return nil, err
			}

			found := false
			for _, s := range ss.LogStreams {
				if s.LogStreamName != nil && *s.LogStreamName == stream {
					if s.UploadSequenceToken != nil {
						sink.sequenceToken = *s.UploadSequenceToken
					}

					found = true

					break
				}
			}

			if !found {
				return nil, fmt.Errorf("log stream not found")
			}
		}

		return sink, nil
	}
}
