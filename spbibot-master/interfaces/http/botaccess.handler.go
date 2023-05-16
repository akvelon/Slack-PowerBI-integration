package http

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/slack-go/slack"
	"go.uber.org/zap"


)

const (
	botAuthFailed  = "An error occurred while installing the application. Please try again"
	botAuthSuccess = "The application is installed. You can close this page"
)

// BotTokenAuthHandler represent the data-struct for auth handler
type BotTokenAuthHandler struct {
	workspaceUsecase    usecases.WorkspaceUsecase
	slackBotAccessToken oauth.Config
	logger              *zap.Logger
}

// NewBotAuthHandler will create an authorization handler object
func NewBotAuthHandler(
	router *httprouter.Router,
	workspaceUsecase usecases.WorkspaceUsecase,
	slackBotAccessToken oauth.Config,
	l *zap.Logger,
) {
	handler := &BotTokenAuthHandler{
		workspaceUsecase:    workspaceUsecase,
		slackBotAccessToken: slackBotAccessToken,
		logger:              l,
	}

	router.GET("/bot_authorization_response", handler.handleBotAuthorizationResponse)
	router.GET("/add_to_slack", handler.handleAddToSlackRequest)
}

func (authHandler *BotTokenAuthHandler) handleAddToSlackRequest(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	authURL := authHandler.slackBotAccessToken.AuthCodeURL("")
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleBotAuthorizationResponse method processes the response from slack authorization and get access token
func (authHandler *BotTokenAuthHandler) handleBotAuthorizationResponse(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	l := utils.WithContext(r.Context(), authHandler.logger)
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeHTML)

	reg := regexp.MustCompile("(code=)(.*?)(&)")
	code := reg.FindString(r.RequestURI)
	code = strings.TrimPrefix(code, "code=")
	code = strings.TrimSuffix(code, "&")
	slackResponse, err := slack.GetOAuthV2Response(&http.Client{}, authHandler.slackBotAccessToken.ClientID, authHandler.slackBotAccessToken.ClientSecret, code, "")
	if err != nil {
		l.Error("couldn't get oauth response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(botAuthFailed))
		if err != nil {
			l.Error("couldn't write auth failure message", zap.Error(err))
		}
		analytics.DefaultAmplitudeClient().Send(analytics.EventApplicationInstallationFailed, "", "", nil)

		return
	}

	workspace := domain.Workspace{ID: slackResponse.Team.ID, BotAccessToken: slackResponse.AccessToken}
	err = authHandler.workspaceUsecase.Store(r.Context(), &workspace)
	if err != nil {
		l.Error("couldn't store workspace", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(constants.AuthHTMLResponse(botAuthFailed, true)))
		if err != nil {
			l.Error("couldn't write auth failure message", zap.Error(err))
		}

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(constants.AuthHTMLResponse(botAuthSuccess, false)))
	if err != nil {
		l.Error("couldn't write auth success message", zap.Error(err))
	}

	analytics.DefaultAmplitudeClient().Send(analytics.EventApplicationInstallationSuccess, workspace.ID, "", nil)
}
