package analytics

import (
	"fmt"

	"github.com/replaygaming/amplitude"
	"go.uber.org/zap"
)

// EventKind is a special type for amplitudes metrics
type EventKind string

const (
	// EventKindConnected is a value to account users who sign in
	EventKindConnected EventKind = "accountConnected"
	// EventKindDisconnected is a value to account users who sign out
	EventKindDisconnected EventKind = "accountDisconnected"
	// EventKindReportsReceivedFailed is an event when Active Pages Filter fails to receive a report
	EventKindReportReceivedFailed EventKind = "reportReceivedFailed"
	// EvetKindReportsReduced is a value to users with reduced report count
	EventKindReportsReduced EventKind = "ReducedReportsCount"
	// EventKindAlertGenerated is a value to account alert generation
	EventKindAlertGenerated EventKind = "alertedReportGenerated"
	// EventKindAlertStopped is a value to stop alert
	EventKindAlertStopped EventKind = "alertStopped"
	// EventKindAlertResumed is a value to resume alert
	EventKindAlertResumed EventKind = "alertResumed"
	// EventKindScheduledReportStopped is a value to stop scheduled report
	EventKindScheduledReportStopped EventKind = "scheduledReportStopped"
	// EventKindScheduledReportStoppedDueToNoActivePagesAvailable is the value for stopping the scheduled report if there are no active pages.
	EventKindScheduledReportStoppedDueToNoActivePagesAvailable EventKind = "scheduledReportStoppedDueToNoActivePagesAvailable"
	// EventKindScheduledReportResumed is a value to resume scheduled report
	EventKindScheduledReportResumed EventKind = "scheduledReportResumed"
	// EventKindFilterWindowOpened is a value to open window 'Manage filters'
	EventKindFilterWindowOpened EventKind = "filterWindowOpened"
	// EventKindFilterStored is a value to save filter to database
	EventKindFilterStored EventKind = "filterStored"
	// EventKindStoppedReportDueToChannelRemoval is the value of the deleted channel
	EventKindStoppedReportDueToChannelRemoval EventKind = "stoppedReportDueToChannelRemoval"
	// EventKindFilterReused is a value to share report with previously used filter
	EventKindFilterReused EventKind = "storedFilterReused"
	// EventKindFilterDeleted is a value to delete filter
	EventKindFilterDeleted EventKind = "storedFilterDeleted"
	// EventKindAlertMainWindowOpened is a value to 'Create alert' opened
	EventKindAlertMainWindowOpened EventKind = "alertMainWindowOpened"
	// EventKindAlertVisualConfigurationWindowOpened is a value to configuration window opened
	EventKindAlertVisualConfigurationWindowOpened EventKind = "alertVisualConfigurationWindowOpened"
	// EventKindAlertNoVisualsWindowOpened is a value to 'No suitable visuals' window opened
	EventKindAlertNoVisualsWindowOpened EventKind = "alertNoVisualsWindowOpened"
	// EventKindPageRemovedFromSchedule is the value for the page Removed from the Schedule
	EventKindPageRemovedFromSchedule EventKind = "pageRemovedFromSchedule"
	// EventKindReportsScheduleFailed is a value to scheduler failed
	EventKindReportsScheduleFailed EventKind = "reportsScheduleFailed"
	// EventQueuedReportGenerationEvent is a value to send message to sqs
	EventQueuedReportGenerationEvent EventKind = "queuedReportGenerationEvent"
	// EventApplicationInstallationFailed is a value when application wasn't installed
	EventApplicationInstallationFailed EventKind = "applicationInstallationFailed"
	// EventApplicationInstallationSuccess is a value when application installed successfully
	EventApplicationInstallationSuccess EventKind = "applicationInstallationSuccess"
	//  EventUserPowerBITokenDeactivatedExternally is a value when user PowerBI Token Deactivated Externally
	EventUserPowerBITokenDeactivatedExternally EventKind = "userPowerBITokenDeactivatedExternally"
	//  EventPowerBIApiErrorOccured is a value when power BI Api Error Occurred
	EventPowerBIApiErrorOccured EventKind = "powerBIApiErrorOccurred"
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
	Send(eventName EventKind, workspaceID string, userID string, properties amplitude.Properties)
}

func (a *amplitudeReport) Send(e EventKind, workspaceID string, userID string, p amplitude.Properties) {
	slackID := UserID(fmt.Sprintf("%v-%v", workspaceID, userID))
	event := amplitude.Event{EventType: string(e), UserID: string(slackID), EventProperties: p}
	if _, err := a.client.Send(event); err != nil {
		a.logger.Error("couldn't send analytics data", zap.Error(err))
	}
}
