package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/slack-go/slack"
	"go.uber.org/zap"


)

const (
	authFailed  = "An error occurred while signing in. Please try again."
	authSuccess = "You are signed in now and can close this page."
)

var formValues = struct {
	userID  string
	teamID  string
	code    string
	state   string
	payload string
}{"user_id", "team_id", "code", "state", "payload"}

// AuthHandler represent the data-struct for auth handler
type AuthHandler struct {
	userUsecase      usecases.UserUsecase
	workspaceUsecase usecases.WorkspaceUsecase
	oauthConfig      *oauth.Config
	featuresConfig   *config.FeatureTogglesConfig
	logger           *zap.Logger
}

// NewAuthHandler will create an authorization handler object
func NewAuthHandler(
	router *httprouter.Router,
	userUsecase usecases.UserUsecase,
	workspaceUsecase usecases.WorkspaceUsecase,
	o *oauth.Config,
	f *config.FeatureTogglesConfig,
	l *zap.Logger,
) AuthHandler {
	handler := AuthHandler{
		userUsecase:      userUsecase,
		workspaceUsecase: workspaceUsecase,
		oauthConfig:      o,
		featuresConfig:   f,
		logger:           l,
	}
	router.GET("/authorization_response", handler.handleAuthorizationResponse)

	return handler
}

// handleAuthorization method for sign in to microsoft account and provide access to PowerBi
func (h *AuthHandler) handleAuthorization(w http.ResponseWriter, r *http.Request) error {
	l := utils.WithContext(r.Context(), h.logger)

	userID := r.FormValue(formValues.userID)
	teamID := r.FormValue(formValues.teamID)
	id := domain.NewSlackUserID(teamID, userID)
	user, err := h.userUsecase.GetByID(r.Context(), id)
	if err == domain.ErrNotFound {
		workspace, err := h.workspaceUsecase.Get(r.Context(), teamID)
		if err != nil {
			l.Error("couldn't get workspace", zap.Error(err))

			return err
		}
		api := slack.New(workspace.BotAccessToken)
		userInfo, _ := api.GetUserInfo(userID)
		user = domain.User{WorkspaceID: id.WorkspaceID, ID: id.ID, Email: userInfo.Profile.Email}
		err = h.userUsecase.Store(r.Context(), &user)
		if err != nil {
			l.Error("couldn't store user", zap.Error(err))

			return err
		}
	} else if err != nil {
		l.Error("couldn't get user", zap.Error(err), zap.String("id", id.ID), zap.String("WorkspaceID", id.WorkspaceID))

		return err
	}

	msg, err := h.getAuthorizationMessage(user.HashID)
	if err != nil {
		l.Error("couldn't get auth message", zap.Error(err))

		return err
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(msg)
	if err != nil {
		l.Error("couldn't write auth message", zap.Error(err))

		return err
	}

	return nil
}

// handleAuthorizationResponse method processes the response from microsoft authorization and get access token
func (h *AuthHandler) handleAuthorizationResponse(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	l := utils.WithContext(r.Context(), h.logger)
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeHTML)

	token, err := h.oauthConfig.Exchange(context.TODO(), r.FormValue(formValues.code))
	if err != nil {
		l.Error("couldn't get token", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(constants.AuthHTMLResponse(authFailed, true)))
		if err != nil {
			l.Error("couldn't write auth failure message", zap.Error(err))
		}

		return
	}

	hash := r.FormValue(formValues.state)

	user, err := h.userUsecase.GetByHash(r.Context(), hash)
	if err != nil {
		l.Error("couldn't get user", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	user.AccessToken = token.AccessToken
	user.RefreshToken = token.RefreshToken
	err = h.userUsecase.Update(r.Context(), &user)
	if err != nil {
		l.Error("couldn't update user", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(constants.AuthHTMLResponse(authFailed, true)))
		if err != nil {
			l.Error("couldn't write auth failure message", zap.Error(err))
		}

		return
	}

	workspace, err := h.workspaceUsecase.Get(r.Context(), user.WorkspaceID)
	if err != nil {
		l.Error("couldn't get workspace")
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(constants.AuthHTMLResponse(authFailed, true)))
		if err != nil {
			l.Error("couldn't write auth failure message", zap.Error(err))
		}

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(constants.AuthHTMLResponse(authSuccess, false)))
	if err != nil {
		l.Error("couldn't write auth success message", zap.Error(err))

		return
	}

	analytics.DefaultAmplitudeClient().Send(analytics.EventKindConnected, workspace.ID, user.ID, nil)

	err = hometab.PublishHomeTab(user, workspace.BotAccessToken, h.oauthConfig.AuthCodeURL(user.HashID), h.featuresConfig)
	if err != nil {
		l.Error("couldn't publish home tab", zap.Error(err))
	}
}

func (h *AuthHandler) getAuthorizationMessage(hashUserID string) ([]byte, error) {
	signInButtons := []*slack.ButtonBlockElement{slackcomponents.NewSlackButtonElement(
		constants.ConnectActionID,
		connectToPowerBIButton,
		h.oauthConfig.AuthCodeURL(hashUserID),
	)}
	buttonBlock := slackcomponents.GetSlackButtonBlock(signInButtons)

	msg := slackcomponents.GetSlackMessageBlock([]slack.Block{buttonBlock})

	return json.Marshal(msg)
}
