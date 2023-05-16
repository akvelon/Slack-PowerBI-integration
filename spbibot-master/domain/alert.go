package domain

import (
	"context"
	"time"
)

// NotificationFrequency represents a alert interval
type NotificationFrequency string

const (
	// OnceAHour is time.Hour interval
	OnceAHour NotificationFrequency = "Once an hour"
	// OnceADay is time.Hour*24 interval
	OnceADay NotificationFrequency = "Once a day"
)

// ToHours returns hours
func (f NotificationFrequency) ToHours() time.Duration {
	switch f {
	case OnceAHour:
		return time.Hour
	case OnceADay:
		return 23 * time.Hour
	default:
		return 23 * time.Hour
	}
}

// AlertStatus defines status of the task for checking alert
type AlertStatus string

const (
	// Active is a status when a task is running
	Active AlertStatus = "Active"
	// Inactive is a status when a task is not run
	Inactive AlertStatus = "Inactive"
)

// Alert model
type Alert struct {
	ID                    int64
	UserID                string
	WorkspaceID           string
	ReportID              string
	VisualName            string
	Condition             string
	Threshold             float64
	NotificationFrequency NotificationFrequency
	ChannelID             string
	Status                AlertStatus
}

// AlertRepository represent the alert's repository contract
type AlertRepository interface {
	GetByID(ctx context.Context, id int64) (Alert, error)
	Store(ctx context.Context, a *Alert) error
	DeleteByID(ctx context.Context, id int64) error
	Update(ctx context.Context, a *Alert) error
	GetPowerBIReportIDsByUser(ctx context.Context, userID SlackUserID) ([]string, error)
	GetByUserIDAndReportID(ctx context.Context, userID SlackUserID, reportID string) ([]Alert, error)
	ListAll(ctx context.Context, status AlertStatus) ([]Alert, error)
}
