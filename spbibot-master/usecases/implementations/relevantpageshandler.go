package implementations

import (
	"context"
	"encoding/json"

	"fmt"


	"go.uber.org/zap"
)

const (
	couldntRenderScheduledPageMessage = "Couldn't render page {%v} because the page doesn't exist in scheduled report {%v} anymore"
	noActivePagesToSendMessage        = "Report {%v} had been stopped, because there are no active pages to send"
)

//ActivePagesFilter is a class for handling irrelevant pages from DB.
type ActivePagesFilter struct {
	powerBiServiceClient       powerbi.ServiceClient
	workspaceRepository        domain.WorkspaceRepository
	schedulerErrorHandler      *SchedulerErrorHandler
	logger                     *zap.Logger
	mysqlPostingTaskRepository domain.PostReportTaskRepository
}

func NewActivePagesFilter(powerBiServiceClient powerbi.ServiceClient, s *SchedulerErrorHandler, workspaceRepository domain.WorkspaceRepository, l *zap.Logger, m domain.PostReportTaskRepository) *ActivePagesFilter {
	return &ActivePagesFilter{
		powerBiServiceClient:       powerBiServiceClient,
		schedulerErrorHandler:      s,
		workspaceRepository:        workspaceRepository,
		logger:                     l,
		mysqlPostingTaskRepository: m,
	}
}

func (activePagesFilter *ActivePagesFilter) Handle(ctx context.Context, ts []*domain.PostReportTask) {
	for _, t := range ts {
		slackUserID := domain.SlackUserID{
			WorkspaceID: t.WorkspaceID,
			ID:          t.UserID,
		}

		//Get pages from PowerBi
		ps, err := activePagesFilter.powerBiServiceClient.GetPages(slackUserID, t.ReportID)
		if err != nil {
			activePagesFilter.logger.Error("couldn't get pages", zap.Error(err))

			continue
		}

		var pages []string
		for _, p := range ps.Value {
			pages = append(pages, p.Name)
		}

		deletedPages := utils.GetArrayDifference(t.PageIDs, pages)
		activePages := utils.GetArrayDifference(t.PageIDs, deletedPages)

		//delete not relevant pages
		if len(deletedPages) != 0 {
			activePagesFilter.logger.Info("The pages are no longer relevant", zap.Int("irrelevantPages", len(deletedPages)))
			var err error
			var isUpdate bool
			if len(activePages) == 0 {
				t.IsActive = false
				isUpdate = false
			} else {
				isUpdate = true
			}
			t.PageIDs = activePages
			err = activePagesFilter.mysqlPostingTaskRepository.UpdatePageIDs(ctx, t)

			if err != nil {
				activePagesFilter.logger.Error("couldn't update user", zap.Error(err))

				continue
			}

			//Get report from PowerBi for reportName
			report, err := activePagesFilter.powerBiServiceClient.GetReport(slackUserID, t.ReportID)
			if err != nil {
				activePagesFilter.logger.Error("couldn't get report", zap.Error(err))
				reportProperty := json.RawMessage(fmt.Sprintf(`"%v"`, err.Error()))
				m := amplitude.Properties{
					"error": &reportProperty,
				}
				analytics.DefaultAmplitudeClient().Send(analytics.EventKindReportReceivedFailed, t.WorkspaceID, t.UserID, m)
				continue
			}

			//send to messenger
			workspace, err := activePagesFilter.workspaceRepository.GetByID(ctx, t.WorkspaceID)
			if err != nil {
				activePagesFilter.logger.Error("couldn't get workspace", zap.Error(err), zap.String("workspaceID", t.WorkspaceID))

				continue
			}

			reportName := report.GetName()
			var textMessage string
			if isUpdate {
				for _, page := range deletedPages {
					textMessage = fmt.Sprintf(couldntRenderScheduledPageMessage, page, reportName)
					activePagesFilter.logger.Info("The page is no longer relevant", zap.String("ReportName", reportName), zap.String("PageID", page))
					analytics.DefaultAmplitudeClient().Send(analytics.EventKindPageRemovedFromSchedule, t.WorkspaceID, t.UserID, nil)
				}
			} else {
				textMessage = fmt.Sprintf(noActivePagesToSendMessage, reportName)
				activePagesFilter.logger.Info("The report no longer has pages", zap.String("ReportName", reportName))
				analytics.DefaultAmplitudeClient().Send(analytics.EventKindScheduledReportStoppedDueToNoActivePagesAvailable, t.WorkspaceID, t.UserID, nil)
			}

			api := slack.New(workspace.BotAccessToken)
			_, _, err = api.PostMessage(
				t.ChannelID,
				slack.MsgOptionText(textMessage, false),
				slack.MsgOptionAsUser(true))
			if err != nil {
				activePagesFilter.logger.Error("couldn't post error message", zap.Error(err))

				continue
			}
		}
	}
}
