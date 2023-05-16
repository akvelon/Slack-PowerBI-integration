package slackcomponents

import (
	"github.com/slack-go/slack"

)

// GetSlackMessage create slack message
func GetSlackMessage(message string) slack.Msg {
	return slack.Msg{
		Text: message,
	}
}

// GetSlackMessageBlock method creates slack message containing blocks
func GetSlackMessageBlock(blocks []slack.Block) slack.Msg {
	return slack.Msg{
		Blocks: slack.Blocks{
			BlockSet: blocks,
		},
	}
}

// PowerBiNotConnectedMessage composes a message for a user without powerbi account connected.
func PowerBiNotConnectedMessage(authCodeURL string) *slack.Msg {
	return &slack.Msg{
		Text:         constants.PowerBiNotConnectedWarning,
		ResponseType: slack.ResponseTypeEphemeral,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				slack.NewSectionBlock(
					GetSlackPlainTextBlock(constants.PowerBiNotConnectedWarning),
					nil,
					&slack.Accessory{
						ButtonElement: NewSlackButtonElement(constants.ConnectActionID, constants.SignInLabel, authCodeURL),
					},
				),
			},
		},
	}
}
