package slackcomponents

import (
	"github.com/google/uuid"
	"github.com/slack-go/slack"
)

// NewSlackButtonElement method creates new slack button element
func NewSlackButtonElement(buttonID, buttonText, url string) *slack.ButtonBlockElement {
	textBlock := GetSlackPlainTextBlock(buttonText)

	buttonElement := slack.NewButtonBlockElement(buttonID, "", textBlock)
	buttonElement.URL = url

	return buttonElement
}

// GetSlackButtonBlock method creates button block
func GetSlackButtonBlock(elements []*slack.ButtonBlockElement) *slack.ActionBlock {
	var buttonElements []slack.BlockElement

	for _, el := range elements {
		buttonElements = append(buttonElements, NewSlackButtonElement(el.ActionID, el.Text.Text, el.URL))
	}

	return getSlackActionBlock(buttonElements...)
}

func getSlackActionBlock(elements ...slack.BlockElement) *slack.ActionBlock {
	return slack.NewActionBlock(uuid.New().String(), elements...)
}
