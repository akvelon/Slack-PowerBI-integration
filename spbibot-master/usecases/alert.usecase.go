package usecases

import (
	"context"

	"github.com/slack-go/slack"


)

// AlertUsecase represent the alert's usecases
type AlertUsecase interface {
	GetByID(ctx context.Context, id int64) (domain.Alert, error)
	Store(ctx context.Context, a *domain.Alert) error
	DeleteByID(ctx context.Context, id int64) error
	Update(ctx context.Context, a *domain.Alert) error
	GetPowerBIReportIDsByUser(ctx context.Context, userID domain.SlackUserID) ([]string, error)
	GetByUserIDAndReportID(ctx context.Context, userID domain.SlackUserID, reportID string) ([]domain.Alert, error)
	ShowInitialCreateAlertModal(ctx context.Context, o *ModalOptions)
	ShowManageAlertsModal(ctx context.Context, o *ModalOptions)
	UpdateAlertModalWithVisuals(ctx context.Context, v *slack.View, slackUserID *domain.SlackUserID)
	ScheduleAlertsCheck(ctx context.Context)
	ScheduleAlertCheck(ctx context.Context, alert *domain.Alert) error
}
