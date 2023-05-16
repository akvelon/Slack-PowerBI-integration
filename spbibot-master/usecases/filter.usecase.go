package usecases

import (
	"context"

)

// FilterUsecase is a filters usecase.
type FilterUsecase interface {
	Delete(ctx context.Context, filterID string) error
	Get(ctx context.Context, id int64) (*domain.Filter, error)
	ListByReportID(ctx context.Context, reportID string) ([]*domain.Filter, error)
	Store(ctx context.Context, f *domain.Filter) error
	Update(ctx context.Context, f *domain.Filter, oldName string) error
}
