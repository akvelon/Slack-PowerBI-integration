package usecases

import (
	"context"


)

// WorkspaceUsecase represent the workspace's usecases
type WorkspaceUsecase interface {
	Get(ctx context.Context, workspaceID string) (domain.Workspace, error)
	Store(ctx context.Context, workspace *domain.Workspace) error
}
