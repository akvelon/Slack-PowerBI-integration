package implementations

import (
	"context"
	"fmt"


)

const (
	authorizationErrorMessage = "We can't obtain data from Power BI account because session had been expired.\n Please disconnect your Power BI account and connect again. "
	otherErrorMessage         = "Something went wrong. Please try again later or contact us with slack.powerb.com for support. Request ID: {%s}"
	failureTitle              = "Failure"
)

type BotErrorHandler struct {
	l *zap.Logger
}

func NewBotErrorHandler(log *zap.Logger) *BotErrorHandler {
	return &BotErrorHandler{
		l: log,
	}
}

func (botErrorHandler *BotErrorHandler) Handle(ctx context.Context, err error, response *slack.ViewResponse, api *slack.Client, o *usecases.ModalOptions) {
	if utils.AuthorizationError(err.Error()) {
		analytics.DefaultAmplitudeClient().Send(analytics.EventUserPowerBITokenDeactivatedExternally, o.User.WorkspaceID, o.User.ID, nil)
		_, err = api.UpdateView(
			modals.NewMessageModal(failureTitle, constants.CancelLabel, authorizationErrorMessage).GetViewRequest(),
			response.View.ExternalID,
			response.View.Hash,
			response.View.ID,
		)
		if err != nil {
			botErrorHandler.l.Error("couldn't update auth view", zap.Error(err))
		}
	} else {
		analytics.DefaultAmplitudeClient().Send(analytics.EventPowerBIApiErrorOccured, o.User.WorkspaceID, o.User.ID, nil)
		var requestIDOrStatus string
		if utils.RequestID(ctx) != "" {
			requestIDOrStatus = utils.RequestID(ctx)
		} else {
			requestIDOrStatus = err.Error()[len(err.Error())-3:]
		}

		_, err = api.UpdateView(
			modals.NewMessageModal(failureTitle, constants.CancelLabel, fmt.Sprintf(otherErrorMessage, requestIDOrStatus)).GetViewRequest(),
			response.View.ExternalID,
			response.View.Hash,
			response.View.ID,
		)
		if err != nil {
			botErrorHandler.l.Error("couldn't update auth view", zap.Error(err))
		}
	}
}
