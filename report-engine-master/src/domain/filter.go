package domain

import (
	"context"
)

type filterKind string

// FilterKindIn corresponds to the basic "in" filter.
const FilterKindIn filterKind = "in"

// Filter is a report filter entity.
type Filter struct {
	ID          int64
	WorkspaceID string
	UserID      string
	ReportID    string
	Name        string
	Kind        filterKind
	Definition  interface{}
}

// FilterRepository is a filters repository.
type FilterRepository interface {
	Delete(ctx context.Context, filterID string) error
	Get(ctx context.Context, id int64) (*Filter, error)
	ListByReportID(ctx context.Context, reportID string) ([]*Filter, error)
	Store(ctx context.Context, f *Filter) error
	Update(ctx context.Context, f *Filter, oldName string) error
}
