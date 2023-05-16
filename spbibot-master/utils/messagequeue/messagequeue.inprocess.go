package messagequeue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/ksuid"
)

const bufferCapacity = 16

type inProcessMessageQueue struct {
	messages chan json.RawMessage
}

// NewInProcessMessageQueue creates a MessageQueue for local testing.
func NewInProcessMessageQueue() MessageQueue {
	return &inProcessMessageQueue{
		messages: make(chan json.RawMessage, bufferCapacity),
	}
}

func (q *inProcessMessageQueue) Push(ctx context.Context, e *Envelope, w WaitOption) error {
	e.ID = ksuid.New().String()

	j, err := json.Marshal(e)
	if err != nil {
		return err
	}

	if w == NoWait {
		select {
		case q.messages <- j:
			return nil

		case <-ctx.Done():
			return ctx.Err()

		default:
			return fmt.Errorf("queue is busy")
		}
	} else {
		select {
		case q.messages <- j:
			return nil

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (q *inProcessMessageQueue) Peek(ctx context.Context, w WaitOption) (*Envelope, error) {
	j := json.RawMessage{}
	if w == NoWait {
		select {
		case j = <-q.messages:
			break

		case <-ctx.Done():
			return nil, ctx.Err()

		default:
			return nil, ErrNoMessages
		}
	} else {
		select {
		case j = <-q.messages:
			break

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	e := Envelope{}
	err := json.Unmarshal(j, &e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (q *inProcessMessageQueue) Delete(_ context.Context, _ string) error {
	return nil
}
