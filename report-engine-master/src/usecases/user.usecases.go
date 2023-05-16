package usecases

import (
	"context"

)

// UserUsecase represent the user's usecases
type UserUsecase interface {
	GetByID(ctx context.Context, id *domain.SlackUserID) (domain.User, error)
	MigrateEnterpriseUserToUseTeamID(ctx context.Context, user *domain.User) error
}
