package usecases

import (
	"context"


)

// ReportUsecase represent the Report's usecases
type ReportUsecase interface {
	ShareReport(ctx context.Context, u *domain.User, token string, options *utils.ShareOptions) error
}
