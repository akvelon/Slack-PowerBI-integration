package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/slack-go/slack"
	"go.uber.org/zap"


)

type slashCommandHandler struct {
	reportUsecase    usecases.ReportUsecase
	userUsecase      usecases.UserUsecase
	workspaceUsecase usecases.WorkspaceUsecase
	alertUsecase     usecases.AlertUsecase
	oauthConfig      *oauth.Config
	featuresConfig   *config.FeatureTogglesConfig
	authHandler      AuthHandler // TODO: remove authHandler usage from here
	logger           *zap.Logger
}

// NewSlashCommandHandler creates a slash command handler & registers its route.
func NewSlashCommandHandler(
	router *httprouter.Router,
	r usecases.ReportUsecase,
	u usecases.UserUsecase,
	w usecases.WorkspaceUsecase,
	a usecases.AlertUsecase,
	s *config.SlackConfig,
	o *oauth.Config,
	f *config.FeatureTogglesConfig,
	l *zap.Logger,
) {
	h := slashCommandHandler{
		authHandler:      NewAuthHandler(router, u, w, o, f, l), // TODO: remove authHandler usage from here by moving used functionality from authHandler
		userUsecase:      u,
		workspaceUsecase: w,
		alertUsecase:     a,
		reportUsecase:    r,
		oauthConfig:      o,
		featuresConfig:   f,
		logger:           l,
	}
	router.POST("/slash", middlewares.NewVerifySlackRequestMiddleware(h.handle, s, l))
}

func (h *slashCommandHandler) handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	l := utils.WithContext(r.Context(), h.logger)

	if c := r.Header.Get(constants.HTTPHeaderContentType); c != constants.MIMETypeURLEncodedForm {
		err := domain.ErrUnexpectedContentType(c)
		l.Error("invalid request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	s, err := slack.SlashCommandParse(r)
	if err != nil {
		l.Error("invalid command", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	r = r.WithContext(utils.WithSlashCommand(r.Context(), &s))
	l = utils.WithContext(r.Context(), h.logger)
	l.Info("handling slash command")

	switch s.Command {
	case "/pbi-share-report":
		hint := constants.ShareReportCommandHelp
		err = h.handleModalCommand(r.Context(), w, &s, hint, h.reportUsecase.ShowSelectReportModal)

	case "/pbi-sign-in":
		if s.Text == "help" {
			err = h.handleSlashCommandHelpPayload(slackcomponents.GetSlackMessage(constants.SignInCommandHelp), w)
		} else {
			err = h.authHandler.handleAuthorization(w, r)
		}

	case "/pbi-sign-out":
		hint := constants.SignOutCommandHelp
		err = h.handleModalCommand(r.Context(), w, &s, hint, h.userUsecase.ShowSignOutModal)

	case "/pbi-create-alert":
		hint := constants.CreateAlertCommandHelp
		err = h.handleModalCommand(r.Context(), w, &s, hint, h.alertUsecase.ShowInitialCreateAlertModal)

	case "/pbi-manage-filters":
		hint := constants.ManageFiltersCommandHelp
		err = h.handleModalCommand(r.Context(), w, &s, hint, h.reportUsecase.ShowChooseReportModal)

	case "/pbi-schedule-report":
		if h.featuresConfig.ReportScheduling {
			hint := constants.HintScheduleReport
			err = h.handleModalCommand(r.Context(), w, &s, hint, h.reportUsecase.ShowSchedulePostingModal)
		}

	case "/pbi-manage-scheduled-reports":
		hint := constants.ManageReportsCommandHelp
		err = h.handleModalCommand(r.Context(), w, &s, hint, h.reportUsecase.ShowManageSchedulePostingModal)

	case "/pbi-manage-alerts":
		hint := constants.ManageAlertsCommandHelp
		err = h.handleModalCommand(r.Context(), w, &s, hint, h.alertUsecase.ShowManageAlertsModal)

	default:
		err = domain.ErrUnknownCommand(s.Command)
	}

	if err != nil {
		l.Error("couldn't execute command", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *slashCommandHandler) handleModalCommand(
	ctx context.Context, w http.ResponseWriter, c *slack.SlashCommand, hint string,
	modalFunc func(context.Context, *usecases.ModalOptions),
) error {
	l := utils.WithContext(ctx, h.logger)

	if c.Text == "help" {
		return h.handleSlashCommandHelpPayload(slackcomponents.GetSlackMessage(hint), w)
	}

	id := domain.SlackUserIDFromSlashCommand(c)
	// TODO: we should put getUser|storeUser operation into goroutine
	user, err := h.userUsecase.GetByID(ctx, id)
	if err == domain.ErrNotFound {
		workspace, err := h.workspaceUsecase.Get(ctx, c.TeamID)
		if err != nil {
			l.Error("couldn't get workspace", zap.Error(err))

			return err
		}
		api := slack.New(workspace.BotAccessToken)
		userInfo, _ := api.GetUserInfo(c.UserID)
		user = domain.User{WorkspaceID: id.WorkspaceID, ID: id.ID, Email: userInfo.Profile.Email}
		err = h.userUsecase.Store(ctx, &user)
		if err != nil {
			l.Error("couldn't store user", zap.Error(err))

			return err
		}
	} else if err != nil {
		l.Error("couldn't get user", zap.Error(err), zap.String("id", id.ID), zap.String("WorkspaceID", id.WorkspaceID))

		return err
	}

	if user.AccessToken == "" {
		msg := slackcomponents.PowerBiNotConnectedMessage(h.oauthConfig.AuthCodeURL(user.HashID))

		return slackclient.RespondNow(w, msg)
	}

	workspace, err := h.workspaceUsecase.Get(ctx, user.WorkspaceID)
	if err != nil {
		l.Error("couldn't get workspace", zap.Error(err))

		return err
	}

	channelID := ""
	api := slack.New(workspace.BotAccessToken)
	isMember, err := slackclient.IsInConversation(api, c.ChannelID)
	if err == nil || isMember {
		channelID = c.ChannelID
	}

	o := usecases.ModalOptions{
		User:           &user,
		BotAccessToken: workspace.BotAccessToken,
		TriggerID:      c.TriggerID,
		ChannelID:      channelID,
	}
	utils.SafeRoutine(func() {
		modalFunc(context.Background(), &o)
	})

	w.WriteHeader(http.StatusOK)

	return nil
}

func (h *slashCommandHandler) handleSlashCommandHelpPayload(helpMsg slack.Msg, w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)

	jsonEncodedHelpMsg, err := json.Marshal(helpMsg)
	if err != nil {
		return err
	}

	_, err = w.Write(jsonEncodedHelpMsg)

	return err
}
