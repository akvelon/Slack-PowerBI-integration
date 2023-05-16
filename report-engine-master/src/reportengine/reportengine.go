package reportengine

import (
	"context"


)

var defaultReportEngine ReportEngine

// DefaultReportEngine retrieves default ReportEngine.
func DefaultReportEngine() ReportEngine {
	return defaultReportEngine
}

// SetDefaultReportEngine assigns default ReportEngine.
func SetDefaultReportEngine(e ReportEngine) {
	defaultReportEngine = e
}

// RenderedReport holds report rendering result.
type RenderedReport struct {
	ID    string
	Name  string
	Pages []*RenderedPage
}

// RenderedPage holds page rendering result.
type RenderedPage struct {
	ID        string
	Name      string
	Filename  string
	ImageData []byte
}

// ReportEngine renders reports to images.
type ReportEngine interface {
	NewContext() (context.Context, context.CancelFunc, error)
	RenderReport(ctx context.Context, o *utils.ShareOptions) (*RenderedReport, error)
}
