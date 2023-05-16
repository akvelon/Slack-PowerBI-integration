package implementations

import (
	"context"
	"time"


)

// WorkspaceUsecase represent the data-struct for workspace usecases
type WorkspaceUsecase struct {
	workspaceRepository domain.WorkspaceRepository
	contextTimeout      time.Duration
}

// NewWorkspaceUsecase creates new an WorkspaceUsecase object representation of domain.WorkspaceUsecase interface
func NewWorkspaceUsecase(workspaceRepository domain.WorkspaceRepository, timeout time.Duration) usecases.WorkspaceUsecase {
	return &WorkspaceUsecase{
		workspaceRepository: workspaceRepository,
		contextTimeout:      timeout,
	}
}

// Get returns workspace
func (workspaceUsecase *WorkspaceUsecase) Get(c context.Context, workspaceID string) (res domain.Workspace, err error) {
	ctx, cancel := context.WithTimeout(c, workspaceUsecase.contextTimeout)
	defer cancel()

	res, err = workspaceUsecase.workspaceRepository.GetByID(ctx, workspaceID)
	if err != nil {
		return
	}

	if res.BotAccessToken == "" {
		err = domain.ErrEmptyBotToken
	}

	return
}

// Store method saves the workspace in storage
func (workspaceUsecase *WorkspaceUsecase) Store(c context.Context, workspace *domain.Workspace) (err error) {
	ctx, cancel := context.WithTimeout(c, workspaceUsecase.contextTimeout)
	defer cancel()

	err = workspaceUsecase.workspaceRepository.Upsert(ctx, workspace)

	return
}
