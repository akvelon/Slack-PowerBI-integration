package domain

import (
	"context"
)

// Workspace model
type Workspace struct {
	ID             string
	IsActive       string
	BotAccessToken string
}

// WorkspaceRepository represent the workspace's repository contract
type WorkspaceRepository interface {
	GetByID(ctx context.Context, id string) (Workspace, error)
	Upsert(ctx context.Context, workspace *Workspace) error
	DeleteSoft(ctx context.Context, id string) error
}
