package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"go.uber.org/zap"

)

const evHomeTab = "home"

type eventsHandler struct {
	userUsecase      usecases.UserUsecase
	workspaceUsecase usecases.WorkspaceUsecase
	oauthConfig      *oauth.Config
	featuresConfig   *config.FeatureTogglesConfig
	logger           *zap.Logger
}

// NewEventsHandler creates a event handler & registers its route.
func NewEventsHandler(
	router *httprouter.Router,
	userUsecase usecases.UserUsecase,
	workspaceUsecase usecases.WorkspaceUsecase,
	s *config.SlackConfig,
	o *oauth.Config,
	f *config.FeatureTogglesConfig,
	l *zap.Logger,
) {
	h := eventsHandler{
		userUsecase:      userUsecase,
		workspaceUsecase: workspaceUsecase,
		oauthConfig:      o,
		featuresConfig:   f,
		logger:           l,
	}
	router.POST("/events", middlewares.NewVerifySlackRequestMiddleware(h.handle, s, l))
}

func (h *eventsHandler) handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	l := utils.WithContext(r.Context(), h.logger)

	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		l.Error("couldn't read body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	body := buf.String()
	// Get event object from request and cast it to related class
	event, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		l.Error("couldn't parse body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	r = r.WithContext(utils.WithAPIEvent(r.Context(), &event))
	l = utils.WithContext(r.Context(), h.logger)
	l.Info("handling event")

	err = h.dispatchEvent(r.Context(), w, &event, body)
	if err != nil {
		l.Error("couldn't dispatch event", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *eventsHandler) dispatchEvent(ctx context.Context, w http.ResponseWriter, event *slackevents.EventsAPIEvent, body string) error {
	l := utils.WithContext(ctx, h.logger)

	// There are to event types: URLVerification and CallbackEvent
	// CallbackEvent object contain InnerEvent property which has it own type
	// For home tab InnerEvent property has type AppHomeOpenedEvent
	switch event.Type {
	case slackevents.URLVerification:
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			l.Error("couldn't unmarshal body", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)

			return nil
		}

		w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeText)
		_, err = w.Write([]byte(r.Challenge))
		if err != nil {
			l.Error("couldn't write body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)

			return nil
		}

		return nil

	case slackevents.CallbackEvent:
		switch ev := event.InnerEvent.Data.(type) {
		case *slackevents.AppHomeOpenedEvent:
			if ev.Tab == evHomeTab {
				return h.handleOpenHomeTab(ctx, event, ev)
			}

			return nil

		default:
			return nil
		}

	default:
		return domain.ErrUnknownCommand(ctx)
	}
}

func (h *eventsHandler) handleOpenHomeTab(ctx context.Context, event *slackevents.EventsAPIEvent, homeEvent *slackevents.AppHomeOpenedEvent) error {
	l := utils.WithContext(ctx, h.logger)

	userID := homeEvent.User
	teamID := event.TeamID

	workspace, err := h.workspaceUsecase.Get(ctx, teamID)
	if err != nil {
		l.Error("couldn't get workspace")

		return err
	}

	id := domain.NewSlackUserID(teamID, userID)
	user, err := h.userUsecase.GetByID(ctx, id)
	if err == domain.ErrNotFound {
		api := slack.New(workspace.BotAccessToken)
		userInfo, _ := api.GetUserInfo(userID)
		user = domain.User{WorkspaceID: id.WorkspaceID, ID: id.ID, Email: userInfo.Profile.Email}

		err = h.userUsecase.Store(ctx, &user)
		if err != nil {
			l.Error("couldn't store user", zap.Error(err))

			return err
		}
	} else if err != nil {
		l.Error("couldn't get user", zap.Error(err))

		return err
	}

	return hometab.PublishHomeTab(user, workspace.BotAccessToken, h.oauthConfig.AuthCodeURL(user.HashID), h.featuresConfig)
}
