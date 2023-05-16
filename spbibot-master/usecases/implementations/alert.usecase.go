package implementations

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

)

const checkAlertIntervalOnServerStart = time.Second * 30

// AlertUsecase represent the data-struct for alert usecases
type AlertUsecase struct {
	alertRepo            domain.AlertRepository
	powerBiServiceClient powerbi.ServiceClient
	userTokenRepo        domain.UserTokenRepository
	workspaceTokenRepo   domain.WorkspaceRepository
	contextTimeout       time.Duration
	logger               *zap.Logger
	botErrorHandler      *BotErrorHandler
}

// NewAlertUsecase creates new an alertUsecase object representation of domain.AlertUsecase interface
func NewAlertUsecase(a domain.AlertRepository, p powerbi.ServiceClient, u domain.UserTokenRepository, w domain.WorkspaceRepository, timeout time.Duration, l *zap.Logger, errorHandler *BotErrorHandler) usecases.AlertUsecase {
	return &AlertUsecase{
		alertRepo:            a,
		powerBiServiceClient: p,
		userTokenRepo:        u,
		workspaceTokenRepo:   w,
		contextTimeout:       timeout,
		logger:               l,
		botErrorHandler:      errorHandler,
	}
}

// GetByID method returns a alert by id via alert's repo
func (alertUsecase *AlertUsecase) GetByID(ctx context.Context, id int64) (res domain.Alert, err error) {
	ctx, cancel := context.WithTimeout(ctx, alertUsecase.contextTimeout)
	defer cancel()

	return alertUsecase.alertRepo.GetByID(ctx, id)
}

// GetPowerBIReportIDsByUser method returns a powerBI report IDs by slack user ID
func (alertUsecase *AlertUsecase) GetPowerBIReportIDsByUser(ctx context.Context, slackUserID domain.SlackUserID) (res []string, err error) {
	ctx, cancel := context.WithTimeout(ctx, alertUsecase.contextTimeout)
	defer cancel()

	return alertUsecase.alertRepo.GetPowerBIReportIDsByUser(ctx, slackUserID)
}

// GetByUserIDAndReportID method returns an alert by slack user id and report id
func (alertUsecase *AlertUsecase) GetByUserIDAndReportID(ctx context.Context, slackUserID domain.SlackUserID, reportID string) (res []domain.Alert, err error) {
	ctx, cancel := context.WithTimeout(ctx, alertUsecase.contextTimeout)
	defer cancel()

	return alertUsecase.alertRepo.GetByUserIDAndReportID(ctx, slackUserID, reportID)
}

// Store method saves the alert in storage
func (alertUsecase *AlertUsecase) Store(ctx context.Context, alert *domain.Alert) (err error) {
	ctx, cancel := context.WithTimeout(ctx, alertUsecase.contextTimeout)
	defer cancel()

	return alertUsecase.alertRepo.Store(ctx, alert)
}

// Update method updates the alert in storage
func (alertUsecase *AlertUsecase) Update(ctx context.Context, alert *domain.Alert) (err error) {
	ctx, cancel := context.WithTimeout(ctx, alertUsecase.contextTimeout)
	defer cancel()

	return alertUsecase.alertRepo.Update(ctx, alert)
}

// DeleteByID method removes the alert in storage
func (alertUsecase *AlertUsecase) DeleteByID(ctx context.Context, id int64) (err error) {
	ctx, cancel := context.WithTimeout(ctx, alertUsecase.contextTimeout)
	defer cancel()

	scheduler := utils.GetInstance()

	isDeleted := scheduler.KillTask(id)
	if !isDeleted {
		return domain.ErrTaskNotKilled
	}

	return alertUsecase.alertRepo.DeleteByID(ctx, id)
}

// ShowInitialCreateAlertModal shows initial modal with report selection
func (alertUsecase *AlertUsecase) ShowInitialCreateAlertModal(ctx context.Context, o *usecases.ModalOptions) {
	l := utils.WithContext(ctx, alertUsecase.logger)

	analytics.DefaultAmplitudeClient().Send(analytics.EventKindAlertMainWindowOpened, o.User.WorkspaceID, o.User.ID, nil)

	// Create a splash modal
	initModal := modals.NewMessageModal(constants.CreateAlertLabel, constants.CloseLabel, constants.LoadingLabel)
	// Open a splash modal view
	api := slack.New(o.BotAccessToken)
	response, err := api.OpenView(o.TriggerID, initModal.GetViewRequest())
	if err != nil {
		l.Error("couldn't open splash view", zap.Error(err))

		return
	}

	// Get reports
	reports, err := alertUsecase.powerBiServiceClient.GetGroupedReports(*o.User.GetSlackUserID())
	if err != nil {
		l.Error("couldn't get grouped reports", zap.Error(err))
		alertUsecase.botErrorHandler.Handle(ctx, err, response, api, o)

		return
	}

	var modalView slack.ModalViewRequest
	if len(reports) > 0 {
		modal := modals.NewInitialAlertModal(constants.CreateAlertLabel, constants.CloseLabel, constants.OkLabel, reports, o.ChannelID)
		modalView = modal.GetViewRequest()
	} else {
		modal := modals.NewMessageModal(constants.WarningLabel, constants.CloseLabel, constants.NoReportsWarning)
		modalView = modal.GetViewRequest()
	}

	_, err = api.UpdateView(modalView, response.View.ExternalID, response.View.Hash, response.View.ID)
	if err != nil {
		l.Error("couldn't update alert view", zap.Error(err))
	}
}

// ShowManageAlertsModal shows alert management modal
func (alertUsecase *AlertUsecase) ShowManageAlertsModal(ctx context.Context, o *usecases.ModalOptions) {
	l := utils.WithContext(ctx, alertUsecase.logger)

	initModal := modals.NewMessageModal(constants.TitleManageAlerts, constants.CloseLabel, constants.LoadingLabel)
	api := slack.New(o.BotAccessToken)
	response, err := api.OpenView(o.TriggerID, initModal.GetViewRequest())
	if err != nil {
		l.Error("couldn't open splash view", zap.Error(err))

		return
	}

	reportsBI, err := alertUsecase.powerBiServiceClient.GetGroupedReports(*o.User.GetSlackUserID())
	if err != nil {
		l.Error("couldn't get grouped reports", zap.Error(err))
		alertUsecase.botErrorHandler.Handle(ctx, err, response, api, o)

		return
	}

	reportsBI, err = RemoveEmptyAlerts(ctx, reportsBI, alertUsecase, *o.User)
	if err != nil {
		l.Error("couldn't remove powerBI reports without alerts", zap.Error(err), zap.String("slackID", o.User.ID))
	}

	var modalView slack.ModalViewRequest
	if len(reportsBI) > 0 {
		modal := modals.NewManageAlertsModal(constants.TitleManageAlerts, constants.CloseLabel, reportsBI, o.ChannelID)
		modalView = modal.GetViewRequest()
	} else {
		modal := modals.NewMessageModal(constants.WarningLabel, constants.CloseLabel, constants.EmptyAlerts)
		modalView = modal.GetViewRequest()
	}

	_, err = api.UpdateView(modalView, response.View.ExternalID, response.View.Hash, response.View.ID)
	if err != nil {
		l.Error("couldn't update alert view", zap.Error(err))
	}
}

// UpdateAlertModalWithVisuals shows updated create alert modal dialog with visual selection
func (alertUsecase *AlertUsecase) UpdateAlertModalWithVisuals(ctx context.Context, v *slack.View, slackUserID *domain.SlackUserID) {
	l := utils.WithContext(ctx, alertUsecase.logger)

	accessData, err := alertUsecase.userTokenRepo.Get(ctx, *slackUserID)
	if err != nil {
		return // TODO implement a mechanism to show internal error message in modal
	}

	w, err := alertUsecase.workspaceTokenRepo.GetByID(ctx, slackUserID.WorkspaceID)
	if err != nil {
		return
	}

	api := slack.New(w.BotAccessToken)

	selectedReportID := v.State.Values[constants.BlockIDReport][constants.ActionIDReport].SelectedOption.Value
	vs, err := alertUtil.GetVisuals(accessData.GetAccessToken(), selectedReportID)
	if err != nil {
		l.Error("couldn't obtain visuals", zap.Error(err))
		return
	}

	var modalView slack.ModalViewRequest
	if len(vs) > 0 {
		analytics.DefaultAmplitudeClient().Send(analytics.EventKindAlertVisualConfigurationWindowOpened, slackUserID.WorkspaceID, slackUserID.ID, nil)

		modal := modals.NewAlertModal(constants.CreateAlertLabel, constants.CloseLabel, constants.OkLabel, vs)
		modalView, err = modal.GetViewRequest(v)
	} else {
		analytics.DefaultAmplitudeClient().Send(analytics.EventKindAlertNoVisualsWindowOpened, slackUserID.WorkspaceID, slackUserID.ID, nil)

		modal := modals.NewMessageModal(constants.WarningLabel, constants.CloseLabel, "Alert(-s) can be created only on Card visual type")
		modalView = modal.GetViewRequest()
	}

	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return
	}

	// Update modal view with reports
	_, err = api.UpdateView(modalView, v.ExternalID, "", v.ID)
	if err != nil {
		l.Error("couldn't update alert view", zap.Error(err))
	}
}

// ScheduleAlertsCheck starts tasks for checking active alerts when server is started
func (alertUsecase *AlertUsecase) ScheduleAlertsCheck(ctx context.Context) {
	go func() {
		l := utils.WithContext(ctx, alertUsecase.logger)

		activeAlerts, err := alertUsecase.alertRepo.ListAll(ctx, domain.Active)
		if err != nil {
			l.Error("couldn't list alerts", zap.Error(err))

			return
		}

		alertIndex := 0
		ticker := time.NewTicker(checkAlertIntervalOnServerStart)

		go func() {
			for ; alertIndex < len(activeAlerts); <-ticker.C {
				err = alertUsecase.ScheduleAlertCheck(ctx, &activeAlerts[alertIndex])
				if err != nil {
					l.Error("couldn't schedule alert check", zap.Error(err), zap.Int64("alertID", activeAlerts[alertIndex].ID))
				}

				alertIndex++
			}

			ticker.Stop()
		}()
	}()
}

// ScheduleAlertCheck starts a periodical job to analyze data in report and publishes a report to channel if condition is reached
func (alertUsecase *AlertUsecase) ScheduleAlertCheck(ctx context.Context, alert *domain.Alert) error {
	l := utils.
		WithContext(ctx, alertUsecase.logger).
		With(zap.Int64("alertID", alert.ID))

	acopy := *alert // do a copy of original struct to pass it to closure
	task := func() error {
		return alertUsecase.checkAndShowAlert(ctx, &acopy)
	}

	scheduler := utils.GetInstance()
	interval := alert.NotificationFrequency.ToHours()

	err := alertUsecase.updateAlertStatus(ctx, acopy, domain.Active)
	if err != nil {
		l.Error("couldn't update alert status")

		return err
	}

	scheduler.AddTask(ctx, acopy.ID, task, interval, alertUsecase.onAlertCheckException)

	return nil
}

func (alertUsecase *AlertUsecase) onAlertCheckException(ctx context.Context, alertID int64, alertErr error) {
	l := utils.
		WithContext(ctx, alertUsecase.logger).
		With(zap.Int64("alertID", alertID))

	alert, err := alertUsecase.alertRepo.GetByID(ctx, alertID)
	if err != nil {
		l.Error("couldn't get alert", zap.Error(err), zap.Int64("alertID", alertID))

		return
	}

	err = alertUsecase.updateAlertStatus(ctx, alert, domain.Inactive)
	if err != nil {
		l.Error("couldn't update alert status", zap.Error(err))

		return
	}

	if alertErr == domain.ErrReportNotLoaded { // System error
		l.Error("report not loaded", zap.Error(err))

		return
	}

	workspace, err := alertUsecase.workspaceTokenRepo.GetByID(ctx, alert.WorkspaceID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err), zap.String("workspaceID", alert.WorkspaceID))

		return
	}

	m := fmt.Sprintf("Error occurred while analyzing report id=\"%s\" for alert (visualName: %s, threshold: %v, condition: %s)",
		alert.ReportID,
		alert.VisualName,
		alert.Threshold,
		alert.Condition)
	api := slack.New(workspace.BotAccessToken)
	_, _, err = api.PostMessage(
		alert.UserID,
		slack.MsgOptionText(m, false),
		slack.MsgOptionAsUser(true))
	if err != nil {
		l.Error("couldn't post error message", zap.Error(err))
	}
}

func (alertUsecase *AlertUsecase) checkAndShowAlert(ctx context.Context, alert *domain.Alert) error {
	slackUserID := domain.SlackUserID{ID: alert.UserID, WorkspaceID: alert.WorkspaceID}
	l := utils.
		WithContext(ctx, alertUsecase.logger).
		With(zap.Int64("alertID", alert.ID))

	report, err := alertUsecase.powerBiServiceClient.GetReport(slackUserID, alert.ReportID)
	if err != nil {
		l.Error("couldn't get report", zap.Error(err), zap.String("reportID", alert.ReportID))

		return domain.ErrReportDoesntExist
	}

	accessData, _ := alertUsecase.userTokenRepo.Get(ctx, slackUserID)
	accessToken := accessData.GetAccessToken()
	workspace, _ := alertUsecase.workspaceTokenRepo.GetByID(ctx, slackUserID.WorkspaceID)

	screenShotPath, err := alertUtil.CheckAlertAndTakeScreenShot(accessToken, alert.ReportID, alert.VisualName, alert.Threshold, alert.Condition)
	if err != nil {
		l.Error("couldn't check alert", zap.Error(err))

		return err
	}

	if screenShotPath == "" {
		l.Info("threshold isn't exceeded", zap.String("visualName", alert.VisualName), zap.String("reportID", alert.ReportID))

		return nil
	}

	api := slack.New(workspace.BotAccessToken)
	params := slack.FileUploadParameters{
		Title:          report.Name,
		File:           screenShotPath,
		Channels:       []string{alert.ChannelID},
		InitialComment: fmt.Sprintf("Alert! The value of %s is %s %v!", alert.VisualName, alert.Condition, alert.Threshold),
	}

	_, err = api.UploadFile(params)
	if err != nil {
		l.Error("couldn't upload alert screenshot", zap.Error(err))

		return err
	}

	err = os.Remove(screenShotPath)
	if err != nil {
		l.Error("couldn't remove screenshot", zap.Error(err))

		return err
	}

	analytics.DefaultAmplitudeClient().Send(analytics.EventKindAlertGenerated, workspace.ID, slackUserID.ID, nil)

	return nil
}

func (alertUsecase *AlertUsecase) updateAlertStatus(ctx context.Context, alert domain.Alert, status domain.AlertStatus) error {
	if alert.Status != status {
		alert.Status = status

		return alertUsecase.alertRepo.Update(ctx, &alert)
	}

	return nil
}

// RemoveEmptyAlerts remove powerBI reports which doesn't have alerts
func RemoveEmptyAlerts(ctx context.Context, reportsBI domain.GroupedReports, alertUsecase usecases.AlertUsecase, u domain.User) (domain.GroupedReports, error) {
	dbReportIDs, err := alertUsecase.GetPowerBIReportIDsByUser(ctx, *u.GetSlackUserID())
	if err != nil {
		return reportsBI, nil
	}

	newReportsBI := removeUnused(reportsBI, dbReportIDs)
	return newReportsBI, nil
}
