package implementations

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

)

const logOutMsg = "Do you really want to disconnect?"

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

// GetByHash method returns a user by hash via user's repo
func (userUsecase *UserUsecase) GetByHash(c context.Context, hash string) (res domain.User, err error) {
	ctx, cancel := context.WithTimeout(c, userUsecase.contextTimeout)
	defer cancel()

	res, err = userUsecase.userRepo.GetByHash(ctx, hash)
	if err != nil {
		return
	}

	return
}

// Store method saves the user in storage
func (userUsecase *UserUsecase) Store(c context.Context, user *domain.User) (err error) {
	ctx, cancel := context.WithTimeout(c, userUsecase.contextTimeout)
	defer cancel()

	user.HashID, err = utils.HashString(user.WorkspaceID+user.ID, userUsecase.HashCost)
	if err != nil {
		return
	}

	err = userUsecase.userRepo.Store(ctx, user)

	return
}

// Update method updates the user in storage
func (userUsecase *UserUsecase) Update(c context.Context, user *domain.User) (err error) {
	ctx, cancel := context.WithTimeout(c, userUsecase.contextTimeout)
	defer cancel()

	return userUsecase.userRepo.Update(ctx, user)
}

// UpdateEnterpriseUser
func (userUsecase *UserUsecase) MigrateEnterpriseUserToUseTeamID(ctx context.Context, user *domain.User) (err error) {
	ctx, cancel := context.WithTimeout(ctx, userUsecase.contextTimeout)
	defer cancel()

	analytics.DefaultAmplitudeClient().Send("UpdatedEnterpriseUserToUseUserId", user.WorkspaceID, user.ID, nil)

	return userUsecase.userRepo.MigrateEnterpriseUserToUseTeamID(ctx, user)
}

// ShowSignOutModal opens sign-out modal
func (userUsecase *UserUsecase) ShowSignOutModal(ctx context.Context, o *usecases.ModalOptions) {
	l := utils.WithContext(ctx, userUsecase.logger)

	questModal := modals.NewLogOutModal(
		constants.WarningLabel,
		constants.CancelLabel,
		logOutMsg,
		constants.SignOut,
		userUsecase.oauthConfig.LogoutCodeURL(),
	)
	api := slack.New(o.BotAccessToken)
	_, err := api.OpenView(o.TriggerID, questModal.GetViewRequest())
	if err != nil {
		l.Error("couldn't open sign out view", zap.Error(err))

		return
	}
}
