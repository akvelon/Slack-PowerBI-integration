package messagequeue

import (
	"context"
	"encoding/json"
)

// MessageKind allows to discern messages from each other.
type MessageKind string

// Envelope holds a message & its metadata.
type Envelope struct {
	ID      string      `json:"id,omitempty"`
	Kind    MessageKind `json:"kind,omitempty"`
	Body    interface{} `json:"body"`
	TraceID string      `json:"traceID,omitempty"`
	Handle  string      `json:"-"`
}

// UnmarshalJSON extends json.UnmarshalJSON behavior to work w/ Envelope.Body.
func (e *Envelope) UnmarshalJSON(bs []byte) error {
	type rawEnvelope Envelope
	b := json.RawMessage(nil)
	r := rawEnvelope{
		Body: &b,
	}
	err := json.Unmarshal(bs, &r)
	if err != nil {
		return err
	}

	*e = Envelope(r)

	return nil
}

// Unpack allows extracting Envelope.Body to arbitrary type determined by caller.
func (e *Envelope) Unpack(v func(j json.RawMessage) (interface{}, error)) (interface{}, error) {
	m, err := json.Marshal(e.Body)
	if err != nil {
		return nil, err
	}
	return v(m)
}

// WaitOption controls MessageQueue behavior when storing & retrieving Envelope.
type WaitOption int

const (
	// NoWait makes MessageQueue to skip any polling if there are no Envelope available.
	NoWait WaitOption = iota
	// Wait makes MessageQueue to wait for incoming Envelope if there are none available.
	Wait
)

// MessageQueue represents a message queue.
type MessageQueue interface {
	Peek(ctx context.Context, w WaitOption) (*Envelope, error)
	Delete(ctx context.Context, h string) error
}
