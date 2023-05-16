package modals

import (
	"github.com/slack-go/slack"

)

// MessageModal defines structure for modal with message
type MessageModal struct {
	BaseModal  *BaseModal
	CallbackID string
	Blocks     slack.Blocks
}

// NewMessageModal creates mew ISlackModal
func NewMessageModal(title, closeText, msg string) ISlackModal {
	messageText := slackcomponents.GetSlackPlainTextBlock(msg)
	messageSection := slack.NewSectionBlock(messageText, nil, nil)

	return &MessageModal{
		BaseModal: NewBaseModal(title, closeText),
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				messageSection,
			},
		},
	}
}

// NewLogOutModal creates mew question modal ISlackModal
func NewLogOutModal(title, closeText, msg, callbackID, url string) ISlackModal {
	messageModal := NewMessageModal(title, closeText, msg)

	buttons := []*slack.ButtonBlockElement{
		slackcomponents.NewSlackButtonElement(constants.DisconnectActionID, constants.DisconnectLabel, url),
	}
	buttonBlock := slackcomponents.GetSlackButtonBlock(buttons)

	messageModal.(*MessageModal).CallbackID = callbackID
	messageModal.(*MessageModal).Blocks.BlockSet = append(messageModal.(*MessageModal).Blocks.BlockSet, buttonBlock)

	return messageModal
}

// GetViewRequest is method from ISlack interface returns slack.ModalViewRequest
func (m *MessageModal) GetViewRequest() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      m.BaseModal.Title,
		Blocks:     m.Blocks,
		Close:      m.BaseModal.CloseText,
		CallbackID: m.CallbackID,
	}
}
