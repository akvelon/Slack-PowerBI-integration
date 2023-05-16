package usecases

import (
	"context"


)

// ReportUsecase represent the Report's usecases
type ReportUsecase interface {
	GetGroupedReports(userID domain.SlackUserID) (domain.GroupedReports, error)
	GetPages(userID *domain.SlackUserID, reportID string) ([]*domain.Page, error)
	GetScheduledReports(ctx context.Context, u domain.SlackUserID, reportID string) ([]*domain.PostReportTask, error)
	GetPowerBIReportIDsByUser(ctx context.Context, u domain.SlackUserID) ([]string, error)
	GetActualScheduledReports(ctx context.Context) ([]*domain.PostReportTask, error)
	ShowSelectReportModal(ctx context.Context, o *ModalOptions)
	ShowChooseReportModal(ctx context.Context, o *ModalOptions)
	ShowSchedulePostingModal(ctx context.Context, o *ModalOptions)
	ShowManageSchedulePostingModal(ctx context.Context, o *ModalOptions)
	AddPostingTask(ctx context.Context, t *domain.PostReportTask) error
	StartPostingTask(ctx context.Context) error
	StartScheduledPosting(ctx context.Context)
	UpdateCompletionStatus(ctx context.Context, id int64) (bool, error)
	Delete(ctx context.Context, id int64) error
}
