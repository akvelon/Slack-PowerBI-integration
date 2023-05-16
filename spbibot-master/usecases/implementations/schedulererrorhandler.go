package implementations

import (
	"context"


)

const (
	notificationMessage = "Scheduled report had been stopped. We can't obtain data from Power BI account because session had been expired. Please disconnect your Power BI account and connect again."
)

type SchedulerErrorHandler struct {
	postingTaskRepository domain.PostReportTaskRepository
	l                     *zap.Logger
	powerBiClient         *powerbi.ServiceClient
}

func NewSchedulerErrorHandler(p domain.PostReportTaskRepository, log *zap.Logger, c *powerbi.ServiceClient) *SchedulerErrorHandler {
	return &SchedulerErrorHandler{
		l:                     log,
		postingTaskRepository: p,
		powerBiClient:         c,
	}
}

func (schedulerErrorHandler *SchedulerErrorHandler) CheckingPowerBIConnection(ctx context.Context, ts []*domain.PostReportTask, workspaceRepository domain.WorkspaceRepository) {
	for _, t := range ts {
		slackUserID := domain.SlackUserID{
			WorkspaceID: t.WorkspaceID,
			ID:          t.UserID,
		}

		_, err := schedulerErrorHandler.powerBiClient.GetPages(slackUserID, t.ReportID)
		if err != nil {
			schedulerErrorHandler.l.Error("couldn't get pages", zap.Error(err))
			schedulerErrorHandler.Handle(ctx, err, slackUserID, workspaceRepository, t.ChannelID, t.ID)

			continue
		}
	}
}

func (schedulerErrorHandler *SchedulerErrorHandler) Handle(ctx context.Context, err error, slackUserID domain.SlackUserID, workspaceRepository domain.WorkspaceRepository, channelID string, ID int64) {
	//send messages to workspace
	if utils.AuthorizationError(err.Error()) {
		_, err := schedulerErrorHandler.postingTaskRepository.UpdateCompletionStatus(ctx, ID)
		if err != nil {
			schedulerErrorHandler.l.Error("couldn't update reports", zap.Error(err))
			return
		}

		analytics.DefaultAmplitudeClient().Send(analytics.EventUserPowerBITokenDeactivatedExternally, slackUserID.WorkspaceID, slackUserID.ID, nil)
		workspace, err := workspaceRepository.GetByID(ctx, slackUserID.WorkspaceID)
		if err != nil {
			schedulerErrorHandler.l.Error("couldn't get workspace", zap.Error(err), zap.String("workspaceID", slackUserID.WorkspaceID))
			return
		}

		api := slack.New(workspace.BotAccessToken)
		_, _, err = api.PostMessage(
			channelID,
			slack.MsgOptionText(notificationMessage, false),
			slack.MsgOptionAsUser(true))
		if err != nil {
			schedulerErrorHandler.l.Error("couldn't post error message", zap.Error(err))
		}
	}
}
