package usecases

import (
	"context"

)

// UserUsecase represent the user's usecases
type UserUsecase interface {
	GetByID(ctx context.Context, id *domain.SlackUserID) (domain.User, error)
	MigrateEnterpriseUserToUseTeamID(ctx context.Context, user *domain.User) error
	GetByHash(ctx context.Context, hash string) (domain.User, error)
	Store(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	ShowSignOutModal(ctx context.Context, o *ModalOptions)
}
