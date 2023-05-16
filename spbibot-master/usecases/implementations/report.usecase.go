package implementations

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/replaygaming/amplitude"
	"github.com/slack-go/slack"
	"go.uber.org/zap"

)

// ReportUsecase represent the data-struct for view usecases
type ReportUsecase struct {
	powerBiServiceClient   powerbi.ServiceClient
	workspaceRepository    domain.WorkspaceRepository
	postingTaskRepository  domain.PostReportTaskRepository
	userRepository         domain.UserRepository
	mq                     messagequeue.MessageQueue
	dbTimeout              time.Duration
	logger                 *zap.Logger
	featureToggles         *config.FeatureTogglesConfig
	botErrorHandler        *BotErrorHandler
	schedulerErrorHandler  *SchedulerErrorHandler
	activePagesFilter      *ActivePagesFilter
	deletedChannelsHandler *DeletedChannelsHandler
}

// NewReportUsecase creates new an ReportUsecase object representation of domain.ReportUsecase interface
func NewReportUsecase(
	powerBiServiceClient powerbi.ServiceClient,
	workspaceRepository domain.WorkspaceRepository,
	postingTaskRepository domain.PostReportTaskRepository,
	userRepository domain.UserRepository,
	m messagequeue.MessageQueue,
	dbTimeout time.Duration,
	l *zap.Logger,
	f *config.FeatureTogglesConfig,
	b *BotErrorHandler,
	s *SchedulerErrorHandler,
	a *ActivePagesFilter,
	c *DeletedChannelsHandler,
) usecases.ReportUsecase {
	return &ReportUsecase{
		powerBiServiceClient:   powerBiServiceClient,
		workspaceRepository:    workspaceRepository,
		postingTaskRepository:  postingTaskRepository,
		userRepository:         userRepository,
		mq:                     m,
		dbTimeout:              dbTimeout,
		logger:                 l,
		featureToggles:         f,
		botErrorHandler:        b,
		schedulerErrorHandler:  s,
		activePagesFilter:      a,
		deletedChannelsHandler: c,
	}
}

// GetPages retrieves report pages.
func (reportUsecase *ReportUsecase) GetPages(userID *domain.SlackUserID, reportID string) ([]*domain.Page, error) {
	ps, err := reportUsecase.powerBiServiceClient.GetPages(*userID, reportID)
	if err != nil {
		return nil, err
	}

	return ps.Value, nil
}

// ShowSelectReportModal shows select report modal dialog
func (reportUsecase *ReportUsecase) ShowSelectReportModal(ctx context.Context, o *usecases.ModalOptions) {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("userID", o.User.ID), zap.String("workspaceID", o.User.WorkspaceID))

	// Create a splash modal
	initModal := modals.NewMessageModal(constants.TitleShareReport, constants.CloseLabel, constants.LoadingLabel)
	// Open a splash modal view
	api := slack.New(o.BotAccessToken)
	response, err := api.OpenView(o.TriggerID, initModal.GetViewRequest())
	if err != nil {
		l.Error("couldn't open splash view", zap.Error(err))

		return
	}

	// Get reports
	reports, err := reportUsecase.GetGroupedReports(*o.User.GetSlackUserID())
	if err != nil {
		l.Error("couldn't get grouped reports", zap.Error(err))
		reportUsecase.botErrorHandler.Handle(ctx, err, response, api, o)
		return
	}

	var modal modals.ISlackModal
	if len(reports) > 0 {
		reducedReports := ReduceReportQuantity(reports, *o.User.GetSlackUserID())
		// Create a select reports modal
		modal = modals.NewSelectReportModal(constants.TitleShareReport, constants.CloseLabel, constants.OkLabel, reports, o.ChannelID, reducedReports)
	} else {
		modal = modals.NewMessageModal(constants.WarningLabel, constants.CloseLabel, constants.NoReportsWarning)
	}

	updateViewRequest := modal.GetViewRequest()

	// Update modal view with reports
	_, err = api.UpdateView(updateViewRequest, response.View.ExternalID, response.View.Hash, response.View.ID)
	if err != nil {
		l.Error("couldn't update report view", zap.Error(err))
	}
}

// ShowChooseReportModal shows choose report modal dialog
func (reportUsecase *ReportUsecase) ShowChooseReportModal(ctx context.Context, o *usecases.ModalOptions) {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("userID", o.User.ID), zap.String("workspaceID", o.User.WorkspaceID))

	analytics.DefaultAmplitudeClient().Send(analytics.EventKindFilterWindowOpened, o.User.WorkspaceID, o.User.ID, nil)

	// Create a splash modal
	initModal := modals.NewMessageModal(constants.TitleManageFilters, constants.CloseLabel, constants.LoadingLabel)
	// Open a splash modal view
	api := slack.New(o.BotAccessToken)
	response, err := api.OpenView(o.TriggerID, initModal.GetViewRequest())
	if err != nil {
		l.Error("couldn't update reports view", zap.Error(err))

		return
	}

	// Get reports
	reports, err := reportUsecase.GetGroupedReports(*o.User.GetSlackUserID())
	if err != nil {
		l.Error("couldn't get grouped reports", zap.Error(err))
		reportUsecase.botErrorHandler.Handle(ctx, err, response, api, o)

		return
	}

	var modal modals.ISlackModal
	if len(reports) > 0 {
		reducedReports := ReduceReportQuantity(reports, *o.User.GetSlackUserID())
		// Create a select reports modal
		modal = modals.NewSelectReportModal(constants.TitleManageFilters, constants.CloseLabel, constants.OkLabel, reports, o.ChannelID, reducedReports)
	} else {
		modal = modals.NewMessageModal(constants.WarningLabel, constants.CloseLabel, constants.NoReportsWarning)
	}

	updateViewRequest := modal.GetViewRequest()

	// Update modal view with reports
	_, err = api.UpdateView(updateViewRequest, response.View.ExternalID, response.View.Hash, response.View.ID)
	if err != nil {
		l.Error("couldn't update reports view", zap.Error(err))
	}
}

// GetGroupedReports returns reports grouped by Power BI groups
func (reportUsecase *ReportUsecase) GetGroupedReports(userID domain.SlackUserID) (domain.GroupedReports, error) {
	return reportUsecase.powerBiServiceClient.GetGroupedReports(userID)
}

// GetScheduledReports function return scheduled reports from database
func (reportUsecase *ReportUsecase) GetScheduledReports(ctx context.Context, u domain.SlackUserID, reportID string) ([]*domain.PostReportTask, error) {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("userID", u.ID), zap.String("workspaceID", u.WorkspaceID))

	reports, err := reportUsecase.postingTaskRepository.GetScheduledReports(ctx, u, reportID)
	if err != nil {
		l.Error("couldn't get reports", zap.Error(err))
		return nil, err
	}
	return reports, nil
}

// GetPowerBIReportIDsByUser function return powerBI report IDs from database
func (reportUsecase *ReportUsecase) GetPowerBIReportIDsByUser(ctx context.Context, u domain.SlackUserID) ([]string, error) {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("userID", u.ID), zap.String("workspaceID", u.WorkspaceID))

	reports, err := reportUsecase.postingTaskRepository.GetPowerBIReportIDsByUser(ctx, u)
	if err != nil {
		l.Error("couldn't get reports", zap.Error(err))
		return nil, err
	}
	return reports, nil
}

// GetActualScheduledReports function get actual scheduled report from database for sending
func (reportUsecase *ReportUsecase) GetActualScheduledReports(ctx context.Context) ([]*domain.PostReportTask, error) {
	l := utils.WithContext(ctx, reportUsecase.logger)

	reports, err := reportUsecase.postingTaskRepository.GetActualScheduledReports(ctx)
	if err != nil {
		l.Error("couldn't get reports", zap.Error(err))
		return nil, err
	}
	return reports, nil
}

// Delete function remove scheduled report from database
func (reportUsecase *ReportUsecase) Delete(ctx context.Context, id int64) error {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("id", strconv.FormatInt(id, 10)))

	err := reportUsecase.postingTaskRepository.Delete(ctx, id)
	if err != nil {
		l.Error("couldn't get reports", zap.Error(err))
		return err
	}
	return nil
}

// UpdateCompletionStatus function change reports completion status
func (reportUsecase *ReportUsecase) UpdateCompletionStatus(ctx context.Context, id int64) (bool, error) {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("id", strconv.FormatInt(id, 10)))

	updatedStatus, err := reportUsecase.postingTaskRepository.UpdateCompletionStatus(ctx, id)
	if err != nil {
		l.Error("couldn't get reports", zap.Error(err))
		return false, err
	}
	return updatedStatus, nil
}

// ShowSchedulePostingModal shows the "schedule a report" modal.
func (reportUsecase *ReportUsecase) ShowSchedulePostingModal(ctx context.Context, o *usecases.ModalOptions) {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("userID", o.User.ID), zap.String("workspaceID", o.User.WorkspaceID))

	loading := modals.NewMessageModal(constants.TitleScheduleReport, constants.CloseLabel, constants.LoadingLabel)
	api := slack.New(o.BotAccessToken)
	response, err := api.OpenView(o.TriggerID, loading.GetViewRequest())
	if err != nil {
		l.Error("couldn't open splash view", zap.Error(err))

		return
	}

	reports, err := reportUsecase.GetGroupedReports(*o.User.GetSlackUserID())
	if err != nil {
		l.Error("couldn't get grouped reports", zap.Error(err))
		reportUsecase.botErrorHandler.Handle(ctx, err, response, api, o)

		return
	}

	var modal modals.ISlackModal
	if len(reports) > 0 {
		reducedReports := ReduceReportQuantity(reports, *o.User.GetSlackUserID())
		modal = modals.NewScheduleReportModal(constants.TitleScheduleReport, constants.CloseLabel, constants.OkLabel, reducedReports, o.ChannelID)
	} else {
		modal = modals.NewMessageModal(constants.WarningLabel, constants.CloseLabel, constants.NoReportsWarning)
	}

	_, err = api.UpdateView(modal.GetViewRequest(), response.View.ExternalID, response.View.Hash, response.View.ID)
	if err != nil {
		l.Error("couldn't update splash view", zap.Error(err))
	}
}

// ShowManageSchedulePostingModal shows the "Manage Scheduled reports" modal.
func (reportUsecase *ReportUsecase) ShowManageSchedulePostingModal(ctx context.Context, o *usecases.ModalOptions) {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("userID", o.User.ID), zap.String("workspaceID", o.User.WorkspaceID))

	loading := modals.NewMessageModal(constants.TitleManageScheduledReports, constants.CloseLabel, constants.LoadingLabel)
	api := slack.New(o.BotAccessToken)
	response, err := api.OpenView(o.TriggerID, loading.GetViewRequest())
	if err != nil {
		l.Error("couldn't open splash view", zap.Error(err))

		return
	}

	reportsBI, err := reportUsecase.GetGroupedReports(*o.User.GetSlackUserID())
	if err != nil {
		l.Error("couldn't get grouped reports", zap.Error(err))
		reportUsecase.botErrorHandler.Handle(ctx, err, response, api, o)

		return
	}

	reportsBI, err = RemoveEmptyReports(ctx, reportsBI, reportUsecase, *o.User)
	if err != nil {
		l.Error("couldn't remove powerBI reports without scheduled tasks", zap.Error(err), zap.String("slackID", o.User.ID))
	}

	var modal modals.ISlackModal
	if err != nil {
		modal = modals.NewMessageModal(constants.WarningLabel, constants.CloseLabel, constants.GetReportsWithError)
	} else if len(reportsBI) > 0 {
		modal = modals.NewManageScheduledReportsModal(constants.TitleManageScheduledReports, constants.CloseLabel, reportsBI, o.ChannelID)
	} else {
		modal = modals.NewMessageModal(constants.WarningLabel, constants.CloseLabel, constants.EmptyScheduledReports)
	}

	_, err = api.UpdateView(modal.GetViewRequest(), response.View.ExternalID, response.View.Hash, response.View.ID)
	if err != nil {
		l.Error("couldn't update splash view", zap.Error(err))
	}
}

// AddPostingTask saves a domain.PostReportTask.
func (reportUsecase *ReportUsecase) AddPostingTask(ctx context.Context, t *domain.PostReportTask) error {
	l := utils.
		WithContext(ctx, reportUsecase.logger).
		With(zap.String("userID", t.UserID), zap.String("workspaceID", t.WorkspaceID))

	ctx, cancel := context.WithTimeout(ctx, reportUsecase.dbTimeout)
	defer cancel()

	isScheduled, err := reportUsecase.postingTaskRepository.CheckIfReportScheduledAlready(ctx, t)
	if err != nil {
		l.Error("couldn't check report already scheduled", zap.Error(err))

		return err
	}
	if isScheduled {
		return domain.ErrConflict
	}

	return reportUsecase.postingTaskRepository.Add(ctx, t)
}

// StartPostingTask initiates execution of a domain.PostReportTask.
func (reportUsecase *ReportUsecase) StartPostingTask(ctx context.Context) error {
	l := utils.WithContext(ctx, reportUsecase.logger)

	s, err := utils.NewSchedule(time.UTC.String(), "0,30 * * * *")
	if err != nil {
		l.Error("invalid schedule", zap.Error(err))

		return err
	}

	task := utils.Task{
		ID:       1,
		Schedule: s,
		Action: func(ctx2 context.Context) error {
			return reportUsecase.postScheduledReports(ctx2)
		},
	}
	ctx = utils.WithTaskID(ctx, &task)

	return utils.DefaultTaskScheduler.Schedule(ctx, &task)
}

// StartScheduledPosting initiates execution of all runnable domain.PostReportTask.
func (reportUsecase *ReportUsecase) StartScheduledPosting(ctx context.Context) {
	l := utils.WithContext(ctx, reportUsecase.logger)

	utils.SafeRoutine(func() {
		err := reportUsecase.StartPostingTask(ctx)
		if err != nil {
			l.Error("couldn't add task", zap.Error(err))
		}
	})
}

func (reportUsecase *ReportUsecase) postScheduledReports(ctx context.Context) error {
	l := utils.WithContext(ctx, reportUsecase.logger)

	tasks, _ := reportUsecase.GetActualScheduledReports(ctx) //here we change reports
	if reportUsecase.featureToggles.DeletedChannelsHandler {
		reportUsecase.deletedChannelsHandler.Handle(ctx, tasks)
		tasks, _ = reportUsecase.GetActualScheduledReports(ctx) //here we change reports
	}
	reportUsecase.schedulerErrorHandler.CheckingPowerBIConnection(ctx, tasks, reportUsecase.workspaceRepository)
	tasks, _ = reportUsecase.GetActualScheduledReports(ctx) //here we change reports
	reportUsecase.activePagesFilter.Handle(ctx, tasks)

	ts, _ := reportUsecase.GetActualScheduledReports(ctx) //here we get checked reports

	for _, t := range ts {
		slackUserID := domain.SlackUserID{
			WorkspaceID: t.WorkspaceID,
			ID:          t.UserID,
		}

		report, err := reportUsecase.powerBiServiceClient.GetReport(slackUserID, t.ReportID)
		if err != nil {
			l.Error("couldn't get report", zap.Error(err))

			continue
		}

		ps, err := reportUsecase.powerBiServiceClient.GetPages(slackUserID, t.ReportID)
		if err != nil {
			l.Error("couldn't get pages", zap.Error(err))

			continue
		}

		pageIDsToNames := map[string]string{}
		for _, p := range ps.Value {
			pageIDsToNames[p.Name] = p.DisplayName
		}

		pms := []*messagequeue.PageMessage(nil)
		for _, i := range t.PageIDs {
			pm := messagequeue.PageMessage{
				ID:   i,
				Name: pageIDsToNames[i],
			}
			pms = append(pms, &pm)
		}

		for _, page := range pms {
			m := messagequeue.PostReportMessage{
				RenderReportMessage: &messagequeue.RenderReportMessage{
					ClientID:    "slack",
					ReportID:    t.ReportID,
					ReportName:  report.GetName(),
					Pages:       []*messagequeue.PageMessage{page},
					UserID:      t.UserID,
					ChannelID:   t.ChannelID,
					WorkspaceID: t.WorkspaceID,
					UniqueID:    uuid.New().String(),
				},
				IsScheduled: true,
			}
			e := messagequeue.Envelope{
				Kind:    messagequeue.MessagePostReport,
				Body:    m,
				TraceID: strconv.FormatInt(t.ID, 10),
			}
			err = reportUsecase.mq.Push(ctx, &e, messagequeue.Wait)
			if err != nil {
				l.Error("couldn't enqueue message", zap.Error(err))

				reportProperty := json.RawMessage(fmt.Sprintf(`{"reportID": "%v"}`, t.ReportID))
				p := amplitude.Properties{
					"report": &reportProperty,
				}
				analytics.DefaultAmplitudeClient().Send(analytics.EventKindReportsScheduleFailed, slackUserID.WorkspaceID, slackUserID.ID, p)
			}
		}
		if t.IsEveryHour {
			err := reportUsecase.postingTaskRepository.UpdateHourlyReports(ctx, t.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveEmptyReports delete reports which doesn't have scheduled reports.
func RemoveEmptyReports(ctx context.Context, reportsBI domain.GroupedReports, reportUsecase usecases.ReportUsecase, u domain.User) (domain.GroupedReports, error) {
	dbReportIDs, err := reportUsecase.GetPowerBIReportIDsByUser(ctx, *u.GetSlackUserID())
	if err != nil {
		return reportsBI, nil
	}

	newReportsBI := removeUnused(reportsBI, dbReportIDs)

	return newReportsBI, nil
}

func ReduceReportQuantity(gr domain.GroupedReports, id domain.SlackUserID) domain.GroupedReports {
	reducedReports := domain.GroupedReports{}
	reducer := constants.ReportQuantityReducer
	ReducedReportsCount := 0
	for k, v := range gr {
		if len(v.Value) > reducer {
			valueSlice := v.Value[0:reducer]
			reducedReports[k] = &domain.ReportsContainer{
				Type:     v.Type,
				Value:    valueSlice,
				RawValue: v.RawValue,
			}
			ReducedReportsCount++
		} else {
			reducedReports[k] = &domain.ReportsContainer{
				Type:     v.Type,
				Value:    v.Value,
				RawValue: v.RawValue,
			}
		}
	}
	if ReducedReportsCount > 0 {
		reportProperty := json.RawMessage(fmt.Sprintf(`{"Reduced grouped reports count": %v}`, ReducedReportsCount))
		m := amplitude.Properties{
			"report": &reportProperty,
		}
		analytics.DefaultAmplitudeClient().Send(analytics.EventKindReportsReduced, id.WorkspaceID, id.ID, m)
	}
	return reducedReports
}
