package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/slack-go/slack"
	"go.uber.org/zap"

)

const (
	breakAuthorizationMessage = "You have been signed out successfully"
	connectToPowerBIButton    = "Connect to PowerBi account"
	successTitle              = "Success"
	clientID                  = "slack"
)

// interactionCommandHandler represents the data-structure for interaction payload handler
type interactionCommandHandler struct {
	reportUsecase    usecases.ReportUsecase
	userUsecase      usecases.UserUsecase
	workspaceUsecase usecases.WorkspaceUsecase
	alertUsecase     usecases.AlertUsecase
	filterUsecase    usecases.FilterUsecase
	mq               messagequeue.MessageQueue
	oauthConfig      *oauth.Config
	featuresConfig   *config.FeatureTogglesConfig
	logger           *zap.Logger
}

// NewInteractionPayloadHandler creates an interaction payload handler & registers its route.
func NewInteractionPayloadHandler(
	router *httprouter.Router,
	r usecases.ReportUsecase,
	u usecases.UserUsecase,
	w usecases.WorkspaceUsecase,
	a usecases.AlertUsecase,
	f usecases.FilterUsecase,
	m messagequeue.MessageQueue,
	s *config.SlackConfig,
	o *oauth.Config,
	t *config.FeatureTogglesConfig,
	l *zap.Logger,
) {
	h := interactionCommandHandler{
		reportUsecase:    r,
		userUsecase:      u,
		workspaceUsecase: w,
		alertUsecase:     a,
		filterUsecase:    f,
		oauthConfig:      o,
		mq:               m,
		featuresConfig:   t,
		logger:           l,
	}
	router.POST("/interaction", middlewares.NewVerifySlackRequestMiddleware(h.handle, s, l))
}

func (h *interactionCommandHandler) handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	l := utils.WithContext(r.Context(), h.logger)

	if c := r.Header.Get(constants.HTTPHeaderContentType); c != constants.MIMETypeURLEncodedForm {
		err := domain.ErrUnexpectedContentType(c)
		l.Error("invalid request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	err := h.handleInteractionPayloads(w, r)
	if err != nil {
		l.Error("couldn't handle interaction payload", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *interactionCommandHandler) handleInteractionPayloads(w http.ResponseWriter, r *http.Request) error {
	l := utils.WithContext(r.Context(), h.logger)

	var payload slack.InteractionCallback
	err := json.Unmarshal([]byte(r.FormValue(formValues.payload)), &payload)
	if err != nil {
		l.Error("couldn't unmarshal body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return nil
	}

	r = r.WithContext(utils.WithInteractionPayload(r.Context(), &payload))
	l = utils.WithContext(r.Context(), h.logger)
	l.Info("handling interaction payload")
	switch payload.Type {
	case slack.InteractionTypeViewSubmission:
		return h.handleViewSubmission(r.Context(), w, &payload)

	case slack.InteractionTypeBlockActions:
		return h.handleBlockActions(r.Context(), w, &payload)

	default:
		return domain.ErrUnknownPayloadType(payload.Type)
	}
}

func (h *interactionCommandHandler) handleBlockActions(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	a := c.ActionCallback.BlockActions[0]
	if strings.HasPrefix(a.BlockID, constants.BlockIDScheduledReport) {
		return h.handleEditReportsControls(ctx, w, c, a.ActionID)
	}
	if strings.HasPrefix(a.BlockID, constants.BlockIDChooseScheduledReport) {
		return h.handleEditReportsControls(ctx, w, c, a.ActionID)
	}
	if strings.HasPrefix(a.BlockID, constants.BlockIDScheduledReportForAlerts) {
		return h.handleEditAlertsControls(ctx, w, c, a.ActionID)
	}
	if strings.HasPrefix(a.BlockID, constants.BlockIDChooseAlert) {
		return h.handleEditAlertsControls(ctx, w, c, a.ActionID)
	}
	if strings.HasPrefix(a.BlockID, constants.BlockIDReport) {
		return h.handleShareReportBlockActions(ctx, w, c)
	}
	switch a.BlockID {
	case constants.BlockIDReuseFilter, constants.BlockIDSaveFilter, constants.BlockIDAddSecondFilter, constants.BlockIDRemoveSecondFilter, constants.BlockIDSearchReportButton, constants.BlockIDSearchWorkspaceButton, constants.BlockIDWorkspacePBI:
		return h.handleShareReportBlockActions(ctx, w, c)
	case constants.BlockIDEditFilter:
		return h.showManageFilterControls(ctx, w, c, a.ActionID)
	case constants.BlockIDFilterToUpdate:
		return h.showManageFilterControls(ctx, w, c, a.ActionID)
	case constants.BlockIDAddFilterForManagement:
		return h.showManageFilterControls(ctx, w, c, a.ActionID)
	case constants.BlockIDPeriodicity:
		if h.featuresConfig.ReportScheduling {
			return h.handleScheduleReportBlockActions(ctx, w, c)
		}

		return nil
	case constants.BlockIDEditScheduledReport:
		return h.handleEditReportsControls(ctx, w, c, a.ActionID)
	case constants.BlockIDEditAlert:
		return h.handleEditAlertsControls(ctx, w, c, a.ActionID)
	default:
		return h.handleAuthBlockActions(ctx, w, c)
	}
}

func (h *interactionCommandHandler) handleViewSubmission(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	switch c.View.CallbackID {
	case constants.CallbackIDManageFilters, constants.CallbackIDCreateFilter, constants.CallbackIDDeleteFilter, constants.CallbackIDUpdateCurrentFilter:
		return h.handleShareReportViewSubmission(ctx, w, c)
	}

	s, err := modals.NewReportSelectionInput(&c.View)
	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	channelID := s.ChannelID
	api := slack.New(workspace.BotAccessToken)
	isMember, err := slackClient.IsInConversation(api, channelID)
	if err != nil || !isMember {
		if err != nil {
			l.Error("couldn't get conversation", zap.Error(err))
		}

		err := slackClient.SendValidationError(w, constants.BlockIDChannel, constants.WarningBotIsNotInChan)
		if err != nil {
			l.Error("couldn't send validation error", zap.Error(err))

			return err
		}

		bot, err := api.GetBotInfo(c.View.BotID)
		if err != nil {
			l.Error("couldn't get bot info", zap.Error(err))

			return err
		}

		_, err = api.UpdateView(*modals.ShowBotIsNotInChannelWarning(&c.View, channelID, bot.UserID), c.View.ExternalID, c.View.Hash, c.View.ID)

		return err
	}

	c.View = *modals.HideBotIsNotInChannelWarning(&c.View)

	if strings.HasPrefix(c.View.CallbackID, constants.CallbackIDShareReport) {
		return h.handleShareReportViewSubmission(ctx, w, c)
	} else if strings.HasPrefix(c.View.CallbackID, constants.CallbackIDSaveAlert) {
		return h.handleSaveAlertViewSubmission(ctx, w, c)
	} else if strings.HasPrefix(c.View.CallbackID, constants.CallbackIDScheduleReport) {
		if h.featuresConfig.ReportScheduling {
			return h.handleScheduleReportViewSubmission(ctx, w, c, &workspace)
		}

		return nil
	} else {
		return domain.ErrUnknownModal
	}
}

func (h *interactionCommandHandler) handleAuthBlockActions(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	a := c.ActionCallback.BlockActions[0]
	slackUserID := domain.SlackUserIDFromInteractionCallback(c)
	user, err := h.userUsecase.GetByID(ctx, slackUserID)

	if err != nil {
		l.Error("couldn't get user", zap.Error(err), zap.String("id", slackUserID.ID), zap.String("WorkspaceID", slackUserID.WorkspaceID))
		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))
		return err
	}

	o := usecases.ModalOptions{
		User:           &user,
		BotAccessToken: workspace.BotAccessToken,
		TriggerID:      c.TriggerID,
	}
	switch a.ActionID {
	case constants.DisconnectActionID:
		user.AccessToken = ""
		user.RefreshToken = ""
		err = h.userUsecase.Update(ctx, &user)
		if err != nil {
			l.Error("couldn't update user", zap.Error(err))

			return err
		}

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(
			modals.NewMessageModal(successTitle, constants.CancelLabel, breakAuthorizationMessage).GetViewRequest(),
			c.View.ExternalID,
			c.View.Hash,
			c.View.ID,
		)
		if err != nil {
			l.Error("couldn't update auth view", zap.Error(err))

			return err
		}

		analytics.DefaultAmplitudeClient().Send(analytics.EventKindDisconnected, workspace.ID, user.ID, nil)

		err = hometab.PublishHomeTab(user, workspace.BotAccessToken, h.oauthConfig.AuthCodeURL(user.HashID), h.featuresConfig)
		if err != nil {
			l.Error("couldn't publish home tab", zap.Error(err))

			return err
		}

	case constants.HomeSignOutID:
		h.userUsecase.ShowSignOutModal(ctx, &o)

	case constants.CallbackIDShareReport:
		h.reportUsecase.ShowSelectReportModal(ctx, &o)

	case constants.CallbackIDSaveAlert:
		h.alertUsecase.ShowInitialCreateAlertModal(ctx, &o)

	case constants.CallbackIDScheduleReport:
		h.reportUsecase.ShowSchedulePostingModal(ctx, &o)

	case constants.CallbackIDManageFilters:
		h.reportUsecase.ShowChooseReportModal(ctx, &o)

	case constants.CallbackIDManageScheduledReports:
		h.reportUsecase.ShowManageSchedulePostingModal(ctx, &o)

	case constants.CallbackIDManageAlerts:
		h.alertUsecase.ShowManageAlertsModal(ctx, &o)
	}

	w.WriteHeader(http.StatusOK)

	return nil
}

func (h *interactionCommandHandler) handleShareReportBlockActions(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	a := c.ActionCallback.BlockActions[0]

	if a.BlockID == constants.BlockIDReuseFilter && a.ActionID == constants.ActionIDReuseFilter {
		if len(a.SelectedOptions) == 1 && a.SelectedOptions[0].Value == constants.ValueReuseFilter {
			return h.showReuseFilterControls(ctx, w, c)
		}

		return h.showEditFilterControls(ctx, w, c)
	} else if a.BlockID == constants.BlockIDSaveFilter && a.ActionID == constants.ActionIDSaveFilter {
		if len(a.SelectedOptions) == 1 && a.SelectedOptions[0].Value == constants.ValueSaveFilter {
			return h.showSaveFilterControls(ctx, w, c)
		}

		return h.hideSaveFilterControls(ctx, w, c)
	} else if a.BlockID == constants.BlockIDAddSecondFilter && a.ActionID == constants.ActionIDAddSecondFilter {
		return h.showAddFilterControls(ctx, w, c)
	} else if a.BlockID == constants.BlockIDRemoveSecondFilter && a.ActionID == constants.ActionIDRemoveSecondFilter {
		return h.hideAddFilterControls(ctx, w, c)
	} else if strings.HasPrefix(a.BlockID, constants.BlockIDReport) && a.ActionID == constants.ActionIDReport {
		return h.showOrUpdateChoosePagesControls(ctx, c)
	} else if a.BlockID == constants.BlockIDWorkspacePBI && a.ActionID == constants.ActionIDWorkspacePBI {
		return h.UpdateChooseReportControls(ctx, c)
	} else if a.BlockID == constants.BlockIDSearchReportButton && a.ActionID == constants.ActionIDSearchReport {
		return h.searchAndShowReports(ctx, c)
	} else if a.BlockID == constants.BlockIDSearchWorkspaceButton && a.ActionID == constants.ActionIDSearchWorkspace {
		return h.searchAndShowWorkspaces(ctx, c)
	}

	return nil
}

func (h *interactionCommandHandler) handleScheduleReportBlockActions(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	a := c.ActionCallback.BlockActions[0]
	if a.BlockID == constants.BlockIDPeriodicity && a.ActionID == constants.ActionIDPeriodicity {
		p, err := modals.ParseExecutionPeriodicity(a.SelectedOption.Value)
		if err != nil {
			l.Error("invalid input", zap.Error(err))

			return err
		}

		switch p {
		case modals.ExecutionPeriodicityDaily:
			return h.hideDayInput(ctx, w, c)

		case modals.ExecutionPeriodicityWeekly:
			return h.showWeekdayInput(ctx, w, c)

		case modals.ExecutionPeriodicityMonthly:
			return h.showDayOfMonthInput(ctx, w, c)
		case modals.ExecutionPeriodicityHourly:
			return h.hideTimeInput(ctx, w, c)
		}
	}

	return fmt.Errorf("unknown action")
}

func (h *interactionCommandHandler) handleSaveAlertViewSubmission(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	switch c.View.CallbackID {
	case constants.CallbackIDSaveAlertSelectReport:
		ctx = utils.WithInteractionPayload(ctx, c)

		loadingModal := modals.NewMessageModal(constants.CreateAlertLabel, constants.CloseLabel, constants.LoadingLabel)
		modalRequest := loadingModal.GetViewRequest()

		if err := slackClient.UpdateView(w, &modalRequest); err != nil {
			return err
		}

		view := c.View
		slackUserID := domain.SlackUserIDFromInteractionCallback(c)
		utils.SafeRoutine(func() {
			h.alertUsecase.UpdateAlertModalWithVisuals(context.Background(), &view, slackUserID)
		})

		return nil

	case constants.CallbackIDSaveAlert: // handle alert creation
		err := slackClient.ClearView(w)
		if err != nil {
			l.Error("couldn't clear view", zap.Error(err))

			return err
		}
		s := modals.ReportSelectionInput{}
		_ = json.Unmarshal([]byte(c.View.PrivateMetadata), &s)

		thr, err := strconv.ParseFloat(c.View.State.Values["Threshold"]["threshold"].Value, 64)
		if err != nil {
			l.Error("couldn't parse threshold value", zap.Error(err))

			return domain.ErrThresholdShouldBeNumber
		}

		alert := &domain.Alert{
			UserID:                c.User.ID,
			WorkspaceID:           c.Team.ID,
			ReportID:              s.ReportID,
			VisualName:            c.View.State.Values[constants.BlockIDVisual][constants.ActionIDVisual].SelectedOption.Text.Text,
			Condition:             c.View.State.Values["Condition"]["condition"].SelectedOption.Text.Text,
			Threshold:             thr,
			NotificationFrequency: domain.NotificationFrequency(c.View.State.Values["NotificationFrequency"]["notificationFrequency"].SelectedOption.Text.Text),
			ChannelID:             s.ChannelID,
			Status:                domain.Inactive,
		}
		err = h.alertUsecase.Store(ctx, alert)
		if err != nil {
			l.Error("couldn't store alert", zap.Error(err))

			return err
		}

		return h.alertUsecase.ScheduleAlertCheck(context.Background(), alert)
	}

	return nil
}

func (h *interactionCommandHandler) handleScheduleReportViewSubmission(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback, workspace *domain.Workspace) error {
	l := utils.WithContext(ctx, h.logger)

	i, err := modals.NewScheduleReportReportInput(&c.View)
	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	s := slack.New(workspace.BotAccessToken)
	u, err := s.GetUserInfo(c.User.ID)
	if err != nil {
		l.Error("couldn't get user", zap.Error(err))

		return err
	}

	pageIDs := []string(nil)
	for _, p := range i.ReportSelection.Pages {
		pageIDs = append(pageIDs, p.ID)
	}

	var dayOfWeek, dayOfMonth int
	isEveryDay := false
	isEveryHour := false

	location, err := time.LoadLocation(u.TZ)
	if err != nil {
		panic(err.Error())
	}
	now := time.Now()
	nowUTC := time.Date(now.Year(), now.Month(), int(i.Schedule.DayOfMonth), i.Schedule.Time.Hour(), i.Schedule.Time.Minute()-5, 0, 0, location).UTC()

	switch i.Schedule.Periodicity {
	case modals.ExecutionPeriodicityHourly:
		isEveryHour = true
		nowUTC = time.Date(now.Year(), now.Month(), int(now.Weekday()), i.Schedule.Time.Hour(), i.Schedule.Time.Minute(), 0, 0, location)

	case modals.ExecutionPeriodicityDaily:
		isEveryDay = true

	case modals.ExecutionPeriodicityWeekly:
		nowUTC = time.Date(now.Year(), now.Month(), 7+int(i.Schedule.Weekday), i.Schedule.Time.Hour(), i.Schedule.Time.Minute()-5, 0, 0, location).UTC()
		dayOfWeek = nowUTC.Day()%7 + 1

	case modals.ExecutionPeriodicityMonthly:
		dayOfMonth = nowUTC.Day()
		if i.Schedule.DayOfMonth == 32 {
			dayOfMonth = 32
		} else if i.Schedule.DayOfMonth == 1 && dayOfMonth != 1 {
			dayOfMonth = -1
		}
	}

	t := domain.PostReportTask{
		WorkspaceID: c.Team.ID,
		UserID:      c.User.ID,
		ReportID:    i.ReportSelection.ReportID,
		PageIDs:     pageIDs,
		ChannelID:   i.ReportSelection.ChannelID,
		TaskTime:    fmt.Sprintf("%v:%v", nowUTC.Hour(), nowUTC.Minute()),
		DayOfWeek:   dayOfWeek,
		DayOfMonth:  dayOfMonth,
		IsEveryDay:  isEveryDay,
		IsEveryHour: isEveryHour,
		TZ:          u.TZ,
		IsActive:    true,
	}
	err = h.reportUsecase.AddPostingTask(context.Background(), &t)
	if err == domain.ErrConflict {
		pagesBlockModifiedID := modals.FindBlock(c.View.Blocks.BlockSet, constants.BlockIDPages)
		var validationError error
		if t.IsEveryHour {
			validationError = slackClient.SendValidationError(w, pagesBlockModifiedID, constants.WarningScheduleExists)
		} else {
			validationError = slackClient.SendValidationError(w, constants.BlockIDTime, constants.WarningScheduleExists)
		}
		if validationError != nil {
			l.Error("couldn't send validation error", zap.Error(validationError))

			return validationError
		}
	} else if err != nil {
		l.Error("couldn't add report posting task", zap.Error(err))

		return err
	}

	err = slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	return nil
}

func (h *interactionCommandHandler) handleShareReportViewSubmission(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	switch c.View.CallbackID {
	case constants.CallbackIDShareReportSelectReport:
		i, err := modals.NewReportSelectionInput(&c.View)
		if err != nil {
			l.Error("invalid input", zap.Error(err))

			return err
		}

		if i.ApplyFilter {
			err := h.showFilterControls(ctx, w, c)
			if err != nil {
				l.Error("couldn't show filter controls", zap.Error(err))

				return err
			}
		} else {
			err := slackClient.Ack(w)
			if err != nil {
				l.Error("couldn't acknowledge", zap.Error(err))

				return err
			}

			s, err := modals.NewShareReportInput(&c.View)
			if err != nil {
				l.Error("invalid input", zap.Error(err))

				return err
			}

			o := utils.NewShareOptions(s)

			utils.SafeRoutine(func() {
				h.shareReport(utils.WithActivityInfo(context.Background(), map[string]string{
					"activityID": utils.RequestID(ctx),
				}), c, o)
			})
		}

	case constants.CallbackIDShareReportReuseFilter:
		err := slackClient.ClearView(w)
		if err != nil {
			l.Error("couldn't clear report view", zap.Error(err))

			return err
		}

		s, err := modals.NewShareReportInput(&c.View)
		if err != nil {
			l.Error("invalid input", zap.Error(err))

			return err
		}

		id, err := strconv.ParseInt(s.ReuseFilter.FilterID, 10, 64)
		if err != nil {
			l.Error("couldn't parse filter id", zap.Error(err))

			return err
		}

		f, err := h.filterUsecase.Get(ctx, id)
		if err != nil {
			l.Error("couldn't get filter", zap.Error(err), zap.Int64("filterID", id))

			return err
		}

		o := utils.NewShareOptions(s)
		o = utils.WithSavedFilter(*o, f)

		analytics.DefaultAmplitudeClient().Send(analytics.EventKindFilterReused, c.User.TeamID, c.User.ID, nil)

		utils.SafeRoutine(func() {
			h.shareReport(utils.WithActivityInfo(context.Background(), map[string]string{
				"activityID": utils.RequestID(ctx),
			}), c, o)
		})

	case constants.CallbackIDShareReportEditFilter:
		err := slackClient.ClearView(w)
		if err != nil {
			l.Error("couldn't clear report view", zap.Error(err))

			return err
		}

		s, err := modals.NewShareReportInput(&c.View)
		if err != nil {
			l.Error("invalid input", zap.Error(err))

			return err
		}

		o := utils.NewShareOptions(s)
		o = utils.WithEditedFilter(*o, s)

		utils.SafeRoutine(func() {
			h.shareReport(utils.WithActivityInfo(context.Background(), map[string]string{
				"activityID": utils.RequestID(ctx),
			}), c, o)
		})

	case constants.CallbackIDShareReportSaveFilter:
		s, err := modals.NewShareReportInput(&c.View)
		if err != nil {
			l.Error("invalid input", zap.Error(err))

			return err
		}

		f := domain.Filter{
			WorkspaceID: c.User.TeamID,
			UserID:      c.User.ID,
			ReportID:    s.ReportSelection.ReportID,
			Name:        s.SaveFilter.Name,
			Kind:        domain.FilterKindIn,
			Definition: &utils.FilterOptions{
				Table:                   s.SaveFilter.EditInFilterInput.Table,
				Column:                  s.SaveFilter.EditInFilterInput.Column,
				Value:                   s.SaveFilter.EditInFilterInput.Value,
				LogicalOperator:         s.SaveFilter.EditInFilterInput.LogicalOperator,
				ConditionOperator:       s.SaveFilter.EditInFilterInput.ConditionOperator,
				SecondValue:             s.SaveFilter.EditInFilterInput.SecondValue,
				SecondConditionOperator: s.SaveFilter.EditInFilterInput.SecondConditionOperator,
			},
		}
		err = h.filterUsecase.Store(ctx, &f)
		if err != nil {
			l.Error("couldn't store filter", zap.Error(err))
			if err == domain.ErrConflict {
				err = slackClient.SendValidationError(w, constants.BlockIDName, constants.WarningFilterExists)
				if err != nil {
					l.Error("couldn't send validation error", zap.Error(err))
				}
			}

			return err
		}

		analytics.DefaultAmplitudeClient().Send(analytics.EventKindFilterStored, c.User.TeamID, c.User.ID, nil)

		err = slackClient.ClearView(w)
		if err != nil {
			l.Error("couldn't clear report view", zap.Error(err))

			return err
		}

		o := utils.NewShareOptions(s)
		o = utils.WithEditedFilter(*o, s)

		utils.SafeRoutine(func() {
			h.shareReport(utils.WithActivityInfo(context.Background(), map[string]string{
				"activityID": utils.RequestID(ctx),
			}), c, o)
		})

	case constants.CallbackIDManageFilters:
		err := h.showManageFilterControls(ctx, w, c, constants.ActionIDCreateFilter)
		if err != nil {
			l.Error("couldn't show filter controls", zap.Error(err))

			return err
		}
	case constants.CallbackIDCreateFilter, constants.CallbackIDUpdateCurrentFilter:
		conditionOperator := c.View.State.Values[constants.BlockIDConditionOperator][constants.ActionIDConditionOperator].SelectedOption.Value
		if conditionOperator == "" {
			conditionOperator = "Is"
		}
		reportID := c.View.PrivateMetadata
		f := domain.Filter{
			WorkspaceID: c.User.TeamID,
			UserID:      c.User.ID,
			ReportID:    reportID,
			Name:        c.View.State.Values[constants.BlockIDName][constants.ActionIDName].Value,
			Kind:        domain.FilterKindIn,
			Definition: &utils.FilterOptions{
				Table:                   c.View.State.Values[constants.BlockIDTable][constants.ActionIDTable].Value,
				Column:                  c.View.State.Values[constants.BlockIDColumn][constants.ActionIDColumn].Value,
				Value:                   c.View.State.Values[constants.BlockIDValue][constants.ActionIDValue].Value,
				ConditionOperator:       conditionOperator,
				LogicalOperator:         c.View.State.Values[constants.BlockIDLogicalOperator][constants.ActionIDLogicalOperator].SelectedOption.Value,
				SecondValue:             c.View.State.Values[constants.BlockIDSecondValue][constants.ActionIDSecondValue].Value,
				SecondConditionOperator: c.View.State.Values[constants.BlockIDSecondConditionOperator][constants.ActionIDSecondConditionOperator].SelectedOption.Value,
			},
		}
		if c.View.CallbackID == constants.CallbackIDCreateFilter {
			err := h.filterUsecase.Store(ctx, &f)
			if err != nil {
				l.Error("couldn't store filter", zap.Error(err))
				if err == domain.ErrConflict {
					err = slackClient.SendValidationError(w, constants.BlockIDName, constants.WarningFilterExists)
					if err != nil {
						l.Error("couldn't send validation error", zap.Error(err))
					}
				}

				return err
			}

			analytics.DefaultAmplitudeClient().Send(analytics.EventKindFilterStored, c.User.TeamID, c.User.ID, nil)
		} else {
			err := h.filterUsecase.Update(ctx, &f, c.View.Title.Text)
			if err != nil {
				l.Error("couldn't update filter", zap.Error(err))
				if err == domain.ErrConflict {
					err = slackClient.SendValidationError(w, constants.BlockIDName, constants.WarningFilterExists)
					if err != nil {
						l.Error("couldn't send validation error", zap.Error(err))
					}
				}

				return err
			}
		}
		err := slackClient.ClearView(w)
		if err != nil {
			l.Error("couldn't clear view", zap.Error(err))

			return err
		}
	case constants.CallbackIDDeleteFilter:
		filterID := c.View.State.Values[constants.BlockIDFilterToDelete][constants.ActionIDFilterToDelete].SelectedOption.Value
		err := h.filterUsecase.Delete(ctx, filterID)
		if err != nil {
			l.Error("couldn't delete filter", zap.Error(err))
			if err == domain.ErrConflict {
				err = slackClient.SendValidationError(w, constants.BlockIDName, constants.WarningFilterExists)
				if err != nil {
					l.Error("couldn't send validation error", zap.Error(err))
				}
			}

			return err
		}

		analytics.DefaultAmplitudeClient().Send(analytics.EventKindFilterDeleted, c.User.TeamID, c.User.ID, nil)

		err = slackClient.ClearView(w)
		if err != nil {
			l.Error("couldn't clear view", zap.Error(err))

			return err
		}
	default:
		err := fmt.Errorf("unknown callback id: %v", c.View.CallbackID)
		l.Error("couldn't handle view submission", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
	}

	return nil
}

func (h *interactionCommandHandler) handleEditReportsControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback, a string) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge")

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	slackUserID := domain.SlackUserIDFromInteractionCallback(c)
	user, err := h.userUsecase.GetByID(ctx, slackUserID)
	if err != nil {
		l.Error("couldn't get user", zap.Error(err), zap.String("id", slackUserID.ID), zap.String("WorkspaceID", slackUserID.WorkspaceID))
	}

	var modal *slack.ModalViewRequest
	reportID := c.View.State.Values[constants.BlockIDScheduledReport][constants.ActionIDScheduledReport].SelectedOption.Value
	if reportID == "" {
		reportID = c.View.State.Values[constants.BlockIDScheduledReport+c.View.ID][constants.ActionIDScheduledReport].SelectedOption.Value
	}

	switch a {
	case constants.ActionIDScheduledReport:
		reports, err := h.reportUsecase.GetScheduledReports(ctx, *user.GetSlackUserID(), reportID)
		if err != nil {
			l.Error("couldn't get reports", zap.Error(err))
		}

		api := slack.New(workspace.BotAccessToken)

		for i, report := range reports {
			channel, err := api.GetConversationInfo(report.ChannelID, false)
			if err != nil {
				l.Error("couldn't get channel name", zap.Error(err))
			}
			if channel == nil {
				reports[i].ChannelName = ""
			} else {
				if channel.IsPrivate {
					reports[i].ChannelName = fmt.Sprintf("🔒%v", channel.Name)
				} else {
					reports[i].ChannelName = fmt.Sprintf("#%v", channel.Name)
				}
			}
		}

		modal = modals.ShowManageReportsControls(&c.View, reports, reportID)

		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDChooseScheduledReport:
		reports, err := h.reportUsecase.GetScheduledReports(ctx, *user.GetSlackUserID(), reportID)
		if err != nil {
			l.Error("couldn't get reports", zap.Error(err))
		}
		pages, err := h.reportUsecase.GetPages(user.GetSlackUserID(), reportID)
		if err != nil {
			l.Error("couldn't get pages", zap.Error(err), zap.String("reportID", reportID))
		}
		modal = modals.ShowScheduledReportPageNames(&c.View, reports, pages, reportID)

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDUpdateScheduledReport:
		updateID := c.View.State.Values[constants.BlockIDChooseScheduledReport+reportID][constants.ActionIDChooseScheduledReport].SelectedOption.Value
		id, err := strconv.ParseInt(updateID, 10, 64)
		if err != nil {
			l.Error("couldn't convert report id", zap.Error(err))
		}
		updatedStatus, err := h.reportUsecase.UpdateCompletionStatus(ctx, id)

		if updatedStatus {
			analytics.DefaultAmplitudeClient().Send(analytics.EventKindScheduledReportResumed, workspace.ID, user.ID, nil)
		} else {
			analytics.DefaultAmplitudeClient().Send(analytics.EventKindScheduledReportStopped, workspace.ID, user.ID, nil)
		}

		if err != nil {
			l.Error("couldn't update report", zap.Error(err))
		}

		reportsBI, err := h.reportUsecase.GetGroupedReports(*user.GetSlackUserID())
		if err != nil {
			l.Error("couldn't get grouped reports", zap.Error(err))
		}
		reportsBI, err = implementations.RemoveEmptyReports(ctx, reportsBI, h.reportUsecase, user)
		if err != nil {
			modal = modals.MessageModalView(&c.View, constants.GetReportsWithError)
		} else {
			modal = modals.ShowManageReportDeleteControls(&c.View, reportsBI)
		}

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDDeleteScheduledReport:
		deleteID := c.View.State.Values[constants.BlockIDChooseScheduledReport+reportID][constants.ActionIDChooseScheduledReport].SelectedOption.Value
		id, err := strconv.ParseInt(deleteID, 10, 64)
		if err != nil {
			l.Error("couldn't convert report id", zap.Error(err))
		}
		err = h.reportUsecase.Delete(ctx, id)
		if err != nil {
			l.Error("couldn't delete report", zap.Error(err))
		}

		reportsBI, err := h.reportUsecase.GetGroupedReports(*user.GetSlackUserID())
		if err != nil {
			l.Error("couldn't get grouped reports", zap.Error(err))
		}

		reportsBI, err = implementations.RemoveEmptyReports(ctx, reportsBI, h.reportUsecase, user)
		if err != nil {
			modal = modals.MessageModalView(&c.View, constants.GetReportsWithError)
		} else if len(reportsBI) > 0 {
			modal = modals.ShowManageReportDeleteControls(&c.View, reportsBI)
		} else {
			modal = modals.ShowEmptyScheduledReportsModal(&c.View)
		}

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	}
	return err
}

func (h *interactionCommandHandler) handleEditAlertsControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback, a string) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge")

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	slackUserID := domain.SlackUserIDFromInteractionCallback(c)
	user, err := h.userUsecase.GetByID(ctx, slackUserID)
	if err != nil {
		l.Error("couldn't get user", zap.Error(err), zap.String("id", slackUserID.ID), zap.String("WorkspaceID", slackUserID.WorkspaceID))
	}

	var modal *slack.ModalViewRequest
	reportID := c.View.State.Values[constants.BlockIDScheduledReportForAlerts][constants.ActionIDScheduledReportForAlerts].SelectedOption.Value
	if reportID == "" {
		reportID = c.View.State.Values[constants.BlockIDScheduledReportForAlerts+c.View.ID][constants.ActionIDScheduledReportForAlerts].SelectedOption.Value
	}

	switch a {
	case constants.ActionIDScheduledReportForAlerts:
		alerts, err := h.alertUsecase.GetByUserIDAndReportID(ctx, *user.GetSlackUserID(), reportID)
		if err != nil {
			l.Error("couldn't get reports", zap.Error(err))
		}

		modal = modals.ShowManageAlertsControls(&c.View, alerts, reportID)

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDChooseAlert:
		alertID := c.View.State.Values[constants.BlockIDChooseAlert+reportID][constants.ActionIDChooseAlert].SelectedOption.Value
		id, err := strconv.ParseInt(alertID, 10, 64)
		if err != nil {
			l.Error("couldn't convert alert id", zap.Error(err))
		}

		alert, err := h.alertUsecase.GetByID(ctx, id)
		if err != nil {
			l.Error("couldn't get alert", zap.Error(err))
		}

		modal = modals.ShowEditAlertControls(&c.View, alert, reportID)

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDUpdateAlert:
		alertID := c.View.State.Values[constants.BlockIDChooseAlert+reportID][constants.ActionIDChooseAlert].SelectedOption.Value
		id, err := strconv.ParseInt(alertID, 10, 64)
		if err != nil {
			l.Error("couldn't convert alert id", zap.Error(err))
		}
		alert, err := h.alertUsecase.GetByID(ctx, id)
		if err != nil {
			l.Error("couldn't get alert", zap.Error(err))
		}

		if alert.Status == domain.Inactive {
			alert.Status = domain.Active
			analytics.DefaultAmplitudeClient().Send(analytics.EventKindAlertResumed, workspace.ID, user.ID, nil)
		} else if alert.Status == domain.Active {
			alert.Status = domain.Inactive
			analytics.DefaultAmplitudeClient().Send(analytics.EventKindAlertStopped, workspace.ID, user.ID, nil)
		}

		err = h.alertUsecase.Update(ctx, &alert)
		if err != nil {
			l.Error("couldn't update alert", zap.Error(err))
		}

		reportsBI, err := h.reportUsecase.GetGroupedReports(*user.GetSlackUserID())
		if err != nil {
			l.Error("couldn't get grouped reports", zap.Error(err))
		}
		reportsBI, err = implementations.RemoveEmptyAlerts(ctx, reportsBI, h.alertUsecase, user)
		if err != nil {
			l.Error("couldn't remove powerBI reports without alerts", zap.Error(err), zap.String("slackID", user.ID))
		}

		modal = modals.ShowManageAlertDeleteControls(&c.View, reportsBI)
		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDDeleteAlert:
		deleteID := c.View.State.Values[constants.BlockIDChooseAlert+reportID][constants.ActionIDChooseAlert].SelectedOption.Value
		id, err := strconv.ParseInt(deleteID, 10, 64)
		if err != nil {
			l.Error("couldn't convert report id", zap.Error(err))
		}
		err = h.alertUsecase.DeleteByID(ctx, id)
		if err != nil {
			l.Error("couldn't delete report", zap.Error(err))
		}

		reportsBI, err := h.reportUsecase.GetGroupedReports(*user.GetSlackUserID())
		if err != nil {
			l.Error("couldn't get grouped reports", zap.Error(err))
		}
		reportsBI, err = implementations.RemoveEmptyAlerts(ctx, reportsBI, h.alertUsecase, user)
		if err != nil {
			l.Error("couldn't remove powerBI reports without alerts", zap.Error(err), zap.String("slackID", user.ID))
		}
		if len(reportsBI) > 0 {
			modal = modals.ShowManageAlertDeleteControls(&c.View, reportsBI)
		} else {
			modal = modals.ShowEmptyAlertsModal(&c.View)
		}

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	}
	return err
}

func (h *interactionCommandHandler) shareReport(ctx context.Context, c *slack.InteractionCallback, o *utils.ShareOptions) {
	ctx = utils.WithInteractionPayload(ctx, c)
	l := utils.WithContext(ctx, h.logger)

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return
	}

	pms := []*messagequeue.PageMessage(nil)
	for _, p := range o.Pages {
		pm := messagequeue.PageMessage{
			ID:   p.ID,
			Name: p.Name,
		}
		pms = append(pms, &pm)
	}

	for _, page := range pms {
		m := messagequeue.PostReportMessage{
			RenderReportMessage: &messagequeue.RenderReportMessage{
				ClientID:    clientID,
				ReportID:    o.ReportID,
				ReportName:  o.ReportName,
				Pages:       []*messagequeue.PageMessage{page},
				UserID:      c.User.ID,
				ChannelID:   o.ChannelID,
				WorkspaceID: workspace.ID,
				UniqueID:    uuid.New().String(),
				Token:       messagequeue.Tokens{},
			},
		}
		if o.Filter != nil {
			m.Filter = &messagequeue.FilterMessage{
				Table:                   o.Filter.Table,
				Column:                  o.Filter.Column,
				Value:                   o.Filter.Value,
				LogicalOperator:         o.Filter.LogicalOperator,
				ConditionOperator:       o.Filter.ConditionOperator,
				SecondValue:             o.Filter.SecondValue,
				SecondConditionOperator: o.Filter.SecondConditionOperator,
			}
		}

		e := messagequeue.Envelope{
			Kind:    messagequeue.MessagePostReport,
			Body:    m,
			TraceID: utils.ActivityInfo(ctx)["activityID"],
		}
		err = h.mq.Push(ctx, &e, messagequeue.Wait)
		if err != nil {
			l.Error("couldn't enqueue message", zap.Error(err))
		}
	}
}

func (h *interactionCommandHandler) searchAndShowReports(ctx context.Context, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	slackUserID := domain.SlackUserIDFromInteractionCallback(c)
	groupedReports, err := h.reportUsecase.GetGroupedReports(*slackUserID)
	if err != nil {
		l.Error("couldn't get grouped reports", zap.Error(err))
	}

	modal := modals.FindReportsByInput(&c.View, groupedReports)

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	api := slack.New(workspace.BotAccessToken)
	viewFromUpdate, err := api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update report view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}
	if c.View.Title.Text == constants.TitleManageFilters {
		return nil
	}
	i, err := modals.NewReportSelectionInput(&viewFromUpdate.View)
	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	if i.ReportID == "" {
		return nil
	}

	ps, err := h.reportUsecase.GetPages(slackUserID, i.ReportID)
	if err != nil {
		l.Error("couldn't get pages", zap.Error(err), zap.String("reportID", i.ReportID))

		return err
	}

	modal, err = modals.ShowOrUpdateChoosePagesControls(&c.View, ps, i.ReportID)
	if err != nil {
		return err
	}
	return nil
}

func (h *interactionCommandHandler) searchAndShowWorkspaces(ctx context.Context, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	slackUserID := domain.SlackUserIDFromInteractionCallback(c)
	groupedReports, err := h.reportUsecase.GetGroupedReports(*slackUserID)
	if err != nil {
		l.Error("couldn't get power bi workspaces", zap.Error(err))
	}

	var PowerBIWorkspaces []*domain.Group
	for gr := range groupedReports {
		PowerBIWorkspaces = append(PowerBIWorkspaces, gr)
	}
	modal := modals.FindWorkspaceByInput(&c.View, PowerBIWorkspaces)

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update report view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}
	return nil
}

func (h *interactionCommandHandler) showFilterControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	reportID := c.View.State.Values[constants.BlockIDReport][constants.ActionIDReport].SelectedOption.Value
	filters, err := h.filterUsecase.ListByReportID(ctx, reportID)
	if err != nil {
		l.Error("couldn't list filters", zap.Error(err), zap.String("reportID", reportID))

		return err
	}

	var modal *slack.ModalViewRequest
	if len(filters) == 0 {
		modal, err = modals.ShowEditFilterControls(&c.View)
	} else {
		modal, err = modals.ShowReuseFilterControls(&c.View, filters)
	}

	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	err = slackClient.PushView(w, modal)
	if err != nil {
		l.Error("couldn't push filter view", zap.Error(err))
	}

	return err
}

func (h *interactionCommandHandler) showManageFilterControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback, actionID string) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge")

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	var modal *slack.ModalViewRequest
	switch actionID {
	case constants.ActionIDUpdateFilter:
		reportID := c.View.PrivateMetadata
		filters, err := h.filterUsecase.ListByReportID(ctx, reportID)
		if err != nil {
			l.Error("couldn't list filters", zap.Error(err), zap.String("reportID", reportID))

			return err
		}
		modal = modals.ShowManageFilterUpdateControls(&c.View, filters)

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update filter view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDDeleteFilter:
		reportID := c.View.PrivateMetadata
		filters, err := h.filterUsecase.ListByReportID(ctx, reportID)
		if err != nil {
			l.Error("couldn't list filters", zap.Error(err), zap.String("reportID", reportID))

			return err
		}
		modal = modals.ShowManageFilterDeleteControls(&c.View, filters)

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update filter view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDFilterToUpdate:
		id, err := strconv.ParseInt(c.View.State.Values[constants.BlockIDFilterToUpdate][constants.ActionIDFilterToUpdate].SelectedOption.Value, 10, 64)
		if err != nil {
			l.Error("couldn't parse filter id", zap.Error(err))

			return err
		}

		f, err := h.filterUsecase.Get(ctx, id)
		if err != nil {
			l.Error("couldn't get filter", zap.Error(err), zap.Int64("filterID", id))

			return err
		}

		filter := map[string]string{}
		filter["Table"] = f.Definition.(*utils.FilterOptions).Table
		filter["Column"] = f.Definition.(*utils.FilterOptions).Column
		filter["Value"] = f.Definition.(*utils.FilterOptions).Value
		filter["Name"] = f.Name
		filter["ConditionOperator"] = f.Definition.(*utils.FilterOptions).ConditionOperator
		filter["SecondValue"] = f.Definition.(*utils.FilterOptions).SecondValue
		filter["SecondConditionOperator"] = f.Definition.(*utils.FilterOptions).SecondConditionOperator
		filter["LogicalOperator"] = f.Definition.(*utils.FilterOptions).LogicalOperator

		modal := modals.ShowManageFilterCurrentUpdateControls(&c.View, filter)

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update filter view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	case constants.ActionIDCreateFilter:
		reportID := c.View.State.Values[constants.BlockIDReport][constants.ActionIDReport].SelectedOption.Value
		c.View.PrivateMetadata = reportID
		modal = modals.ShowManageFilterCreateControls(&c.View)
		err = slackClient.PushView(w, modal)
		if err != nil {
			l.Error("couldn't push view", zap.Error(err))
		}
	case constants.ActionIDAddFilterForManagement:
		modal = modals.ShowAddFilterControls(&c.View)

		api := slack.New(workspace.BotAccessToken)
		_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
		if err != nil {
			l.Error("couldn't update filter view", zap.Error(err))

			return domain.ErrUpdatingView(err)
		}
	default:
		l.Error("couldn't update filter view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return err
}

func (h *interactionCommandHandler) showUpdateFilterControls(ctx context.Context, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	reportID := c.View.State.Values[constants.BlockIDReport][constants.ActionIDReport].SelectedOption.Value
	filters, err := h.filterUsecase.ListByReportID(ctx, reportID)
	if err != nil {
		l.Error("couldn't list filters", zap.Error(err), zap.String("reportID", reportID))

		return err
	}

	modal := modals.ShowManageFilterUpdateControls(&c.View, filters)

	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update filter view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return err
}

func (h *interactionCommandHandler) showReuseFilterControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge")

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	s, err := modals.NewShareReportInput(&c.View)
	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	reportID := s.ReportSelection.ReportID
	filters, err := h.filterUsecase.ListByReportID(ctx, reportID)
	if err != nil {
		l.Error("couldn't list filters", zap.Error(err), zap.String("reportID", reportID))

		return err
	}

	modal, err := modals.ShowReuseFilterControls(&c.View, filters)
	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update filter view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) showEditFilterControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace")

		return err
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal, err := modals.ShowEditFilterControls(&c.View)
	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update filter view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) showSaveFilterControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal := modals.ShowSaveFilterControls(&c.View)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update filter view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) showAddFilterControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal := modals.ShowAddFilterControls(&c.View)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update filter view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) hideAddFilterControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal := modals.HideAddFilterControls(&c.View)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update filter view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) hideSaveFilterControls(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		return domain.ErrUpdatingView(err)
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal := modals.HideSaveFilterControls(&c.View)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update filter view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) hideDayInput(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		return domain.ErrUpdatingView(err)
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal := modals.HideDayInput(&c.View)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update scheduling view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) hideTimeInput(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		return domain.ErrUpdatingView(err)
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal := modals.HideTimeInput(&c.View)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update scheduling view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) showWeekdayInput(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		return domain.ErrUpdatingView(err)
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal := modals.ShowWeekdayInput(&c.View)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update scheduling view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) showDayOfMonthInput(ctx context.Context, w http.ResponseWriter, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	err := slackClient.Ack(w)
	if err != nil {
		l.Error("couldn't acknowledge", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		return domain.ErrUpdatingView(err)
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	modal := modals.ShowDayOfMonthInput(&c.View)
	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update scheduling view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) UpdateChooseReportControls(ctx context.Context, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	chosenPBIWorkspace := c.View.State.Values[constants.BlockIDWorkspacePBI][constants.ActionIDWorkspacePBI].SelectedOption.Value
	if chosenPBIWorkspace == "" {
		return nil
	}
	slackUserID := domain.SlackUserIDFromInteractionCallback(c)
	rs, err := h.reportUsecase.GetGroupedReports(*slackUserID)
	if err != nil {
		l.Error("couldn't get slack user ID", zap.Error(err))
	}
	workspaceReports := modals.FindReportsInChosenPBIWorkspace(&c.View, rs, chosenPBIWorkspace)

	modal := modals.UpdateChooseReportControls(&c.View, *workspaceReports, chosenPBIWorkspace)

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		return domain.ErrUpdatingView(err)
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update pages view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}

func (h *interactionCommandHandler) showOrUpdateChoosePagesControls(ctx context.Context, c *slack.InteractionCallback) error {
	l := utils.WithContext(ctx, h.logger)

	i, err := modals.NewReportSelectionInput(&c.View)
	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	if i.ReportID == "" {
		return nil
	}

	u := domain.SlackUserIDFromInteractionCallback(c)
	ps, err := h.reportUsecase.GetPages(u, i.ReportID)
	if err != nil {
		l.Error("couldn't get pages", zap.Error(err), zap.String("reportID", i.ReportID))

		return err
	}

	modal, err := modals.ShowOrUpdateChoosePagesControls(&c.View, ps, i.ReportID)
	if err != nil {
		l.Error("invalid input", zap.Error(err))

		return err
	}

	workspace, err := h.workspaceUsecase.Get(ctx, c.User.TeamID)
	if err != nil {
		return domain.ErrUpdatingView(err)
	}

	if workspace.BotAccessToken == "" {
		return domain.ErrEmptyBotToken
	}

	api := slack.New(workspace.BotAccessToken)
	_, err = api.UpdateView(*modal, c.View.ExternalID, c.View.Hash, c.View.ID)
	if err != nil {
		l.Error("couldn't update pages view", zap.Error(err))

		return domain.ErrUpdatingView(err)
	}

	return nil
}
