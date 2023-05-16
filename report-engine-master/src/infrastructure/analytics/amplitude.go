package analytics

import (
	"encoding/json"
	"fmt"

	"github.com/replaygaming/amplitude"
	"go.uber.org/zap"
)

// EventKind is a special type for amplitudes metrics
type EventKind string

const (
	// EventKindReportGenerated is a value to account report generation
	EventKindReportGenerated EventKind = "reportGenerated"
	// EventKindUserDeactivated is a value to deactivating user
	EventKindUserDeactivated EventKind = "userDeletedFromWorkspace"
	// EventKindUserReactivated is a value to reactivating user
	EventKindUserReactivated EventKind = "userReturnedToWorkspace"
	// EventKindWorkspaceDeleted is a value to delete workspace
	EventKindWorkspaceDeleted EventKind = "workspaceDeleted"
	// EventKindChannelDeleted is a value to delete channel
	EventKindChannelDeleted EventKind = "channelDeleted"
	// EventKindReportFailedToGenerate is a value to report couldn't generate
	EventKindReportFailedToGenerate EventKind = "reportFailedToGenerate"
	// EventKindReportFailedToSend is a value to report couldn't send
	EventKindReportFailedToSend EventKind = "reportFailedToSend"
	// EventKindReportsScheduleFailed is a value to scheduler failed
	EventKindReportsScheduleFailed EventKind = "reportsScheduleFailed"
	// EventKindSendReportMessageFailed is a value to send report failed
	EventKindSendReportMessageFailed EventKind = "sendReportMessageFailed"
	// EventQueuedReportGenerationEvent is a value to send message to sqs
	EventQueuedReportGenerationEvent EventKind = "queuedReportGenerationEvent"
	// EventKindReportRetried is a value to report retried
	EventKindReportRetried EventKind = "reportRetried"
)

// UserID is a special type for slack users ids
type UserID string

var defaultAmplitudeClient Amplitude

// DefaultAmplitudeClient get amplitude client
func DefaultAmplitudeClient() Amplitude {
	return defaultAmplitudeClient
}

// SetDefaultAmplitudeClient set the defaultAmplitudeClient value
func SetDefaultAmplitudeClient(c *amplitude.DefaultClient, l *zap.Logger) {
	defaultAmplitudeClient = &amplitudeReport{client: c, logger: l}
}

type amplitudeReport struct {
	client *amplitude.DefaultClient
	logger *zap.Logger
}

// Amplitude is a interface for setting up defaultAmplitudeClient
type Amplitude interface {
	Send(eventName EventKind, workspaceID string, userID string, clientID string, properties amplitude.Properties)
}

func (a *amplitudeReport) Send(e EventKind, workspaceID string, userID string, clientID string, p amplitude.Properties) {
	slackID := UserID(fmt.Sprintf("%v: %v-%v", clientID, workspaceID, userID))
	clientProperty := json.RawMessage(fmt.Sprintf(`{"clientName": "%v"}`, clientID))
	if p == nil {
		p = amplitude.Properties{
			"client": &clientProperty,
		}
	} else {
		p["client"] = &clientProperty
	}
	event := amplitude.Event{EventType: string(e), UserID: string(slackID), EventProperties: p}
	if _, err := a.client.Send(event); err != nil {
		a.logger.Error("couldn't send analytics data", zap.Error(err))
	}
}
