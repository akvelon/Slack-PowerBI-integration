package modals

import (
	"github.com/slack-go/slack"


)

// ISlackModal interface for modals
type ISlackModal interface {
	GetViewRequest() slack.ModalViewRequest
}

// BaseModal is the base modal structure
type BaseModal struct {
	Title     *slack.TextBlockObject
	CloseText *slack.TextBlockObject
}

// NewBaseModal creates new base modal
func NewBaseModal(title, close string) *BaseModal {
	return &BaseModal{
		Title:     slackcomponents.GetSlackPlainTextBlock(title),
		CloseText: slackcomponents.GetSlackPlainTextBlock(close),
	}
}

// ShowBotIsNotInChannelWarning shows a "bot isn't on channel" warning message in a modal.
func ShowBotIsNotInChannelWarning(v *slack.View, channel, bot string) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	warningText := slackcomponents.GetSlackMarkdownTextBlock(constants.BotIsNotInChannel(channel, bot))
	warningBlockID := slack.SectionBlockOptionBlockID(constants.BlockIDChannelWarning)
	warningSection := slack.NewSectionBlock(warningText, nil, nil, warningBlockID)
	r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, warningSection, constants.BlockIDChannel)

	return r
}

// HideBotIsNotInChannelWarning hides the "bot isn't on channel" warning.
func HideBotIsNotInChannelWarning(v *slack.View) *slack.View {
	r := CopyView(*v)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDChannelWarning)

	return r
}

// MessageModalView is modal view if error has occurred.
func MessageModalView(v *slack.View, message string) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	r.Blocks.BlockSet = []slack.Block{}

	messageText := slackcomponents.GetSlackPlainTextBlock(message)
	messageSection := slack.NewSectionBlock(messageText, nil, nil)

	r.Blocks.BlockSet = append(r.Blocks.BlockSet, messageSection)
	return r
}

// CopyModalRequest returns ModalViewRequest is the same as view
func CopyModalRequest(v *slack.View) *slack.ModalViewRequest {
	return &slack.ModalViewRequest{
		Type:            v.Type,
		Title:           v.Title,
		Blocks:          v.Blocks,
		Close:           v.Close,
		Submit:          v.Submit,
		PrivateMetadata: v.PrivateMetadata,
		CallbackID:      v.CallbackID,
		ClearOnClose:    v.ClearOnClose,
		NotifyOnClose:   v.NotifyOnClose,
		ExternalID:      v.ExternalID,
	}
}

// CopyView clones a slack.View.
func CopyView(v slack.View) *slack.View {
	return &v
}

func updateBlockOrAddAfter(bs []slack.Block, updateWith slack.Block, afterBlockID string) []slack.Block {
	s := []slack.Block{}
	done := false
	for _, b := range bs {
		if !done && blockID(b) == blockID(updateWith) {
			s = append(s, updateWith)
			done = true
		} else if !done && blockID(b) == afterBlockID {
			s = append(s, b, updateWith)
			done = true
		} else {
			s = append(s, b)
		}
	}

	return s
}
