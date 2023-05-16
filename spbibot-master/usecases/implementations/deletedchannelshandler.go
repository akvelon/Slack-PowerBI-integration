package implementations

import (
	"context"


	"go.uber.org/zap"
)

const (
	prefixDeleted = "deleted_"
)

type DeletedChannelsHandler struct {
	workspaceRepository   domain.WorkspaceRepository
	postingTaskRepository domain.PostReportTaskRepository
	l                     *zap.Logger
}

func NewDeletedChannelsHandler(p domain.PostReportTaskRepository, workspaceRepository domain.WorkspaceRepository, log *zap.Logger) *DeletedChannelsHandler {
	return &DeletedChannelsHandler{
		l:                     log,
		workspaceRepository:   workspaceRepository,
		postingTaskRepository: p,
	}
}

func (deletedChannelsHandler *DeletedChannelsHandler) Handle(ctx context.Context, tasks []*domain.PostReportTask) {
	//get list chanels by token
	for _, task := range tasks {
		workspace, err := deletedChannelsHandler.workspaceRepository.GetByID(ctx, task.WorkspaceID)
		if err != nil {
			deletedChannelsHandler.l.Error("couldn't get workspace", zap.Error(err), zap.String("workspaceID", task.WorkspaceID))

			continue
		}

		api := slack.New(workspace.BotAccessToken)
		params := slack.GetConversationsParameters{
			Cursor:          "",
			ExcludeArchived: false,
			Limit:           0,
			Types:           []string{"public_channel", "private_channel"},
		}
		channelsFromTask, _, err := api.GetConversations(&params)
		if err != nil {
			deletedChannelsHandler.l.Error("couldn't get chanels", zap.Error(err))

			continue
		}

		var channels []string
		for _, channel := range channelsFromTask {
			channels = append(channels, channel.ID)
		}

		if !utils.Contains(channels, task.ChannelID) && task.ChannelID[:len(prefixDeleted)] != prefixDeleted {
			task.IsActive = false
			task.ChannelID = prefixDeleted + task.ChannelID
			err := deletedChannelsHandler.postingTaskRepository.UpdateChannelAndStatus(ctx, task)
			if err != nil {
				deletedChannelsHandler.l.Error("couldn't update reports", zap.Error(err))

				continue
			}

			analytics.DefaultAmplitudeClient().Send(analytics.EventKindStoppedReportDueToChannelRemoval, task.WorkspaceID, task.UserID, nil)
			deletedChannelsHandler.l.Info("couldn't get report due to channel removal", zap.Error(err))
		}
	}
}
