package mq

import (
	"context"

	
)

// Manager receives & distributes incoming work items across multiple Worker instances.
type Manager interface {
	RegisterWorker(w Worker) error
	Start(ctx context.Context)
	Stop(ctx context.Context) error
}

// Worker executes arbitrary commands received in a messagequeue.Envelope.
type Worker interface {
	SupportedMessages() []messagequeue.MessageKind
	Handle(ctx context.Context, e *messagequeue.Envelope) error
}
