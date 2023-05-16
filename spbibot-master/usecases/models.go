package usecases

import (
	
)

// ModalOptions holds parameters required to show a modal.
type ModalOptions struct {
	User           *domain.User
	BotAccessToken string
	TriggerID      string
	ChannelID      string
}
