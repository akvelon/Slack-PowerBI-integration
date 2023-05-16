package implementations

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"go.uber.org/zap"

)

// UserUsecase represent the data-struct for user usecases
type UserUsecase struct {
	userRepo       domain.UserRepository
	contextTimeout time.Duration
	HashCost       int
	oauthConfig    oauth.Config
	logger         *zap.Logger
}

// NewUserUsecase creates new an userUsecase object representation of domain.UserUsecase interface
func NewUserUsecase(userRepository domain.UserRepository, timeout time.Duration, cost int, o oauth.Config, l *zap.Logger) usecases.UserUsecase {
	return &UserUsecase{
		userRepo:       userRepository,
		contextTimeout: timeout,
		HashCost:       cost,
		oauthConfig:    o,
		logger:         l,
	}
}

// GetByID method returns a user by id via user's repo
func (userUsecase *UserUsecase) GetByID(c context.Context, id *domain.SlackUserID) (res domain.User, err error) {
	ctx, cancel := context.WithTimeout(c, userUsecase.contextTimeout)
	defer cancel()

	res, err = userUsecase.userRepo.GetByID(ctx, id)
	if err == sql.ErrNoRows {
		if strings.HasPrefix(res.ID, "W") {
			updateErr := userUsecase.MigrateEnterpriseUserToUseTeamID(ctx, &res)
			if updateErr != nil {
				return
			}
			res, err = userUsecase.GetByID(ctx, id)
		}
	}

	return
}

// UpdateEnterpriseUser
func (userUsecase *UserUsecase) MigrateEnterpriseUserToUseTeamID(ctx context.Context, user *domain.User) (err error) {
	ctx, cancel := context.WithTimeout(ctx, userUsecase.contextTimeout)
	defer cancel()

	analytics.DefaultAmplitudeClient().Send("UpdatedEnterpriseUserToUseUserId", user.WorkspaceID, user.ID, "slack", nil)

	return userUsecase.userRepo.MigrateEnterpriseUserToUseTeamID(ctx, user)
}
