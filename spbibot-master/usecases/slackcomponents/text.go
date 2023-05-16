package slackcomponents

import (
	"github.com/slack-go/slack"
)

// GetSlackMarkdownTextBlock method creates text block
func GetSlackMarkdownTextBlock(text string) *slack.TextBlockObject {
	return slack.NewTextBlockObject(slack.MarkdownType, text, false, false)
}

// GetSlackPlainTextBlock method creates text block
func GetSlackPlainTextBlock(text string) *slack.TextBlockObject {
	return slack.NewTextBlockObject(slack.PlainTextType, text, false, false)
}
