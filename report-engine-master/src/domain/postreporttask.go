package domain

import (
	"context"
	"time"
)

// IsActiveStatus denotes task active status.
type IsActiveStatus bool

// PostReportTask is the report posting task.
type PostReportTask struct {
	ID           int64
	WorkspaceID  string
	UserID       string
	ReportID     string
	PageIDs      []string
	ChannelID    string
	TaskTime     string
	DayOfWeek    int
	DayOfMonth   int
	IsEveryDay   bool
	IsEveryHour  bool
	TZ           string
	CompletedAt  time.Time
	IsActive     bool
	ChannelName  string
	RetryAttempt int
}

// PostReportTaskRepository is a repository of PostReportTask entities.
type PostReportTaskRepository interface {
	Add(ctx context.Context, t *PostReportTask) error
	GetScheduledReports(ctx context.Context, u SlackUserID, reportID string) ([]*PostReportTask, error)
	GetPowerBIReportIDsByUser(ctx context.Context, u SlackUserID) ([]string, error)
	GetActualScheduledReports(ctx context.Context) ([]*PostReportTask, error)
	Update(ctx context.Context, t *PostReportTask) error
	UpdateHourlyReports(ctx context.Context, id int64) error
	UpdateCompletionStatus(ctx context.Context, id int64) (bool, error)
	Delete(ctx context.Context, id int64) error
	DeleteBySlackInfo(ctx context.Context, u *SlackUserID, channelID string) error
	CheckIfReportScheduledAlready(ctx context.Context, t *PostReportTask) (bool, error)
}
