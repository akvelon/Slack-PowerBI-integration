package reportengine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/replaygaming/amplitude"
	"go.uber.org/zap"

)

type attributeName string

const (
	attributeMessageKind attributeName = "messageKind"
	attributeTraceID     attributeName = "traceID"
)

type attributeType string

const (
	typeString attributeType = "String"
)

type RetryStrategy struct {
	Logger      *zap.Logger
	URL         string
	MaxAttempts int
	AWSSession  client.ConfigProvider
}

type ReportRetryStrategy interface {
	Retry(context.Context, *utils.ShareOptions, error) (bool, error)
}

func NewRetryStrategy(l *zap.Logger, url string, attempts int, awsSession client.ConfigProvider) ReportRetryStrategy {
	return &RetryStrategy{
		Logger:      l,
		URL:         url,
		MaxAttempts: attempts,
		AWSSession:  awsSession,
	}
}

func (r *RetryStrategy) Retry(renderReportCtx context.Context, shareOptions *utils.ShareOptions, errorGenerateReport error) (skipPosting bool, err error) {
	if errorGenerateReport != nil {
		if shareOptions.RetryAttempt < r.MaxAttempts-1 {
			shareOptions.PostReportMessage.RetryAttempt++
			e := messagequeue.Envelope{
				Kind:    messagequeue.MessagePostReport,
				Body:    shareOptions.PostReportMessage,
				TraceID: utils.ActivityInfo(renderReportCtx)["activityID"],
			}
			var j []byte
			j, err = json.Marshal(e)
			if err != nil {
				return false, err
			}

			s := &sqs.SendMessageInput{}
			s = s.
				SetMessageGroupId(string(e.Kind)).
				SetMessageBody(string(j)).
				SetMessageAttributes(map[string]*sqs.MessageAttributeValue{
					string(attributeMessageKind): (&sqs.MessageAttributeValue{}).
						SetDataType(string(typeString)).
						SetStringValue(string(e.Kind)),
					string(attributeTraceID): (&sqs.MessageAttributeValue{}).
						SetDataType(string(typeString)).
						SetStringValue(e.TraceID),
				})
			s.SetMessageBody(string(j))
			s = s.SetQueueUrl(r.URL)
			q := sqs.New(r.AWSSession)
			_, err = q.SendMessageWithContext(renderReportCtx, s)
			if err != nil {
				r.Logger.Error("couldn't enqueue message", zap.Error(err))
			} else {
				r.Logger.Info("retry to send", zap.String("reportID", shareOptions.ReportID), zap.Int("retryAttempt", shareOptions.RetryAttempt), zap.Error(errorGenerateReport))
				RetryAttempt := json.RawMessage(fmt.Sprintf(`{"isScheduled": %v, "reportID": "%v","retryAttempt": "%v"}`, shareOptions.IsScheduled, shareOptions.ReportID, shareOptions.RetryAttempt))
				m := amplitude.Properties{
					"retryAttempt": &RetryAttempt,
				}
				analytics.DefaultAmplitudeClient().Send(
					analytics.EventKindReportRetried,
					shareOptions.WorkspaceID,
					shareOptions.UserID,
					shareOptions.ClientID,
					m,
				)
			}

			skipPosting = true
		} else {
			r.Logger.Error("Couldn't generate report", zap.Error(err))
			skipPosting = false
			err = errorGenerateReport
		}
	}

	return skipPosting, err
}
