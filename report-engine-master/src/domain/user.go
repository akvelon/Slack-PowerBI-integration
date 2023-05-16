package domain

import (
	"context"
)

// AccessData describes interface which provide Access and Refresh tokens
type AccessData interface {
	GetAccessToken() string
	GetRefreshToken() string
}

// User model
type User struct {
	AccessData
	WorkspaceID  string
	ID           string
	IsActive     bool
	Email        string
	HashID       string
	AccessToken  string
	RefreshToken string
}

// GetAccessToken returns AccessToken
func (u User) GetAccessToken() string {
	return u.AccessToken
}

// GetRefreshToken returns RefreshToken
func (u User) GetRefreshToken() string {
	return u.RefreshToken
}

// SlackUserID represents a unique Slack user id using a pair of local or global user id & workspace (team) or Enterprise Grid id.
type SlackUserID struct {
	WorkspaceID string
	ID          string
}

// GetSlackUserID returns slack user id from user
func (u *User) GetSlackUserID() *SlackUserID {
	return &SlackUserID{
		WorkspaceID: u.WorkspaceID,
		ID:          u.ID,
	}
}

// UserRepository represent the user's repository contract
type UserRepository interface {
	GetByID(ctx context.Context, id *SlackUserID) (User, error)
	MigrateEnterpriseUserToUseTeamID(ctx context.Context, user *User) error
	GetByHash(ctx context.Context, hash string) (User, error)
	Store(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Deactivate(ctx context.Context, id *SlackUserID) error
	Reactivate(ctx context.Context, id *SlackUserID) error
}

// UserTokenRepository represent the user's repository contract
// TODO: merge user and userToken
type UserTokenRepository interface {
	Get(ctx context.Context, id interface{}) (AccessData, error)
	Update(ctx context.Context, id interface{}, accessData AccessData) error
}
