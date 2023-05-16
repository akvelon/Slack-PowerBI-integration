package messagequeue

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"

)

type attributeName string

const (
	attributeMessageKind attributeName = "messageKind"
	attributeTraceID     attributeName = "traceID"
)

// NOTE: This is a workaround for excessive usage of pointers to basic types in AWS SDK.
func getAttribute(n attributeName) *string {
	s := string(n)

	return &s
}

type attributeType string

const (
	typeString attributeType = "String"
)

// NOTE: This is a workaround for excessive usage of pointers to basic types in AWS SDK.
func getType(t attributeType) *string {
	s := string(t)

	return &s
}

type sqsMessageQueue struct {
	buffer       []*sqs.Message
	bufferLocker sync.Locker
	sqs          *sqs.SQS
	config       *config.MessageQueueConfig
	logger       *zap.Logger
}

// NewSQSMessageQueue creates an SQS-backed MessageQueue.
func NewSQSMessageQueue(s *sqs.SQS, c *config.MessageQueueConfig, l *zap.Logger) MessageQueue {
	return &sqsMessageQueue{
		buffer:       []*sqs.Message(nil),
		bufferLocker: &sync.Mutex{},
		sqs:          s,
		config:       c,
		logger:       l,
	}
}

func (q *sqsMessageQueue) Peek(ctx context.Context, w WaitOption) (*Envelope, error) {
	q.bufferLocker.Lock()
	defer q.bufferLocker.Unlock()

	if len(q.buffer) > 0 {
		return q.getBuffered()
	}

	i := &sqs.ReceiveMessageInput{}
	i = i.
		SetMessageAttributeNames([]*string{
			getAttribute(attributeMessageKind),
			getAttribute(attributeTraceID),
		}).
		SetQueueUrl(q.config.URL).
		SetMaxNumberOfMessages(int64(q.config.BatchSize))
	if w == Wait {
		i = i.SetWaitTimeSeconds(int64(q.config.PollingInterval.Seconds()))
	}

	o, err := q.sqs.ReceiveMessageWithContext(ctx, i)
	if err != nil {
		return nil, err
	}

	if len(o.Messages) == 0 {
		return nil, ErrNoMessages
	}

	q.fillBuffer(o.Messages)

	return q.getBuffered()
}

func (q *sqsMessageQueue) Delete(ctx context.Context, h string) error {
	d := &sqs.DeleteMessageInput{}
	d = d.
		SetQueueUrl(q.config.URL).
		SetReceiptHandle(h)
	_, err := q.sqs.DeleteMessageWithContext(ctx, d)

	return err
}

func packEnvelope(e *Envelope) (*sqs.SendMessageInput, error) {
	j, err := json.Marshal(e)
	if err != nil {
		return nil, err
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

	return s, nil
}

func unpackMessage(m *sqs.Message) (*Envelope, error) {
	e := Envelope{}
	err := json.Unmarshal([]byte(*m.Body), &e)
	if err != nil {
		return nil, err
	}

	if m.MessageId != nil {
		e.ID = *m.MessageId
	}

	if m.ReceiptHandle != nil {
		e.Handle = *m.ReceiptHandle
	}

	k := m.MessageAttributes[string(attributeMessageKind)]
	if k != nil && k.StringValue != nil {
		e.Kind = MessageKind(*k.StringValue)
	}

	t := m.MessageAttributes[string(attributeTraceID)]
	if t != nil && t.StringValue != nil {
		e.TraceID = *t.StringValue
	}

	return &e, nil
}

func (q *sqsMessageQueue) fillBuffer(ms []*sqs.Message) {
	q.buffer = ms
}

func (q *sqsMessageQueue) getBuffered() (*Envelope, error) {
	m := q.buffer[0]
	q.buffer = q.buffer[1:]
	e, err := unpackMessage(m)
	if err != nil {
		return nil, err
	}

	return e, nil
}
