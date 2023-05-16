package implementations

import (
	"context"
	"time"


)

type filterUsecase struct {
	filterRepository domain.FilterRepository
	timeout          time.Duration
}

// NewFilterUsecase creates a domain.FilterUsecase.
func NewFilterUsecase(filterRepository domain.FilterRepository, queryTimeout time.Duration) usecases.FilterUsecase {
	return &filterUsecase{
		filterRepository: filterRepository,
		timeout:          queryTimeout,
	}
}

func (u *filterUsecase) Get(ctx context.Context, id int64) (*domain.Filter, error) {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	f, err := u.filterRepository.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (u *filterUsecase) ListByReportID(ctx context.Context, reportID string) ([]*domain.Filter, error) {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	f, err := u.filterRepository.ListByReportID(ctx, reportID)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (u *filterUsecase) Store(ctx context.Context, f *domain.Filter) error {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	return u.filterRepository.Store(ctx, f)
}

func (u *filterUsecase) Delete(ctx context.Context, filterID string) error {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	return u.filterRepository.Delete(ctx, filterID)
}

func (u *filterUsecase) Update(ctx context.Context, f *domain.Filter, oldName string) error {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	return u.filterRepository.Update(ctx, f, oldName)
}
