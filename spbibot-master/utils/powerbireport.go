package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"go.uber.org/zap"


)

const (
	htmlDirectoryPath = "./generatedReports"
	clientID          = "slack"
)

// FilterOptions contains all the filter options
type FilterOptions struct {
	Table                   string `json:"table"`
	Column                  string `json:"column"`
	Value                   string `json:"value"`
	LogicalOperator         string `json:"logicalOperator"`
	ConditionOperator       string `json:"conditionOperator"`
	SecondValue             string `json:"secondValue"`
	SecondConditionOperator string `json:"secondConditionOperator"`
}

func (f *FilterOptions) String() string {
	if f.LogicalOperator != "" {
		return fmt.Sprintf("%v.%v %v %v %v %v %v", f.Table, f.Column, f.ConditionOperator, f.Value, f.LogicalOperator, f.SecondConditionOperator, f.SecondValue)
	}
	return fmt.Sprintf("%v.%v is %v", f.Table, f.Column, f.Value)
}

// ShareOptions holds parameters for the template.
// TODO: JSON tags are kept for compatibility w/ old renderer where ShareOptions is inserted directly into template.
type ShareOptions struct {
	ClientID    string         `json:"clientId"`
	AccessToken string         `json:"accessToken"`
	ReportID    string         `json:"reportId"`
	ReportName  string         `json:"reportName"`
	Filter      *FilterOptions `json:"filter"`
	Pages       []*PageOptions
	ChannelID   string
	UserID      string
	IsScheduled bool
	SkipPosting bool
}

// PageOptions holds page parameters.
type PageOptions struct {
	ID   string
	Name string
}

// NewShareOptions makes a report.ShareOptions from a modals.ShareReportInput.
func NewShareOptions(s *modals.ShareReportInput) *ShareOptions {
	ps := []*PageOptions(nil)
	for _, p := range s.ReportSelection.Pages {
		p := PageOptions{
			ID:   p.ID,
			Name: p.Name,
		}
		ps = append(ps, &p)
	}

	o := ShareOptions{
		ClientID:   clientID,
		ReportID:   s.ReportSelection.ReportID,
		ReportName: s.ReportSelection.ReportName,
		ChannelID:  s.ReportSelection.ChannelID,
		Pages:      ps,
	}

	return &o
}

// WithSavedFilter adds a saved filter to a report.ShareOptions.
func WithSavedFilter(o ShareOptions, f *domain.Filter) *ShareOptions {
	o.Filter = inFilterFromDomain(f)

	return &o
}

// Set up filter from slack user input
func inFilterFromInput(f *modals.EditInFilterInput) *FilterOptions {
	return &FilterOptions{
		Column:                  f.Column,
		Table:                   f.Table,
		Value:                   f.Value,
		LogicalOperator:         f.LogicalOperator,
		ConditionOperator:       f.ConditionOperator,
		SecondValue:             f.SecondValue,
		SecondConditionOperator: f.SecondConditionOperator,
	}
}

// Set up filter from saved filter
func inFilterFromDomain(f *domain.Filter) *FilterOptions {
	if f.Kind == domain.FilterKindIn {
		return &FilterOptions{
			Column:                  f.Definition.(*FilterOptions).Column,
			Table:                   f.Definition.(*FilterOptions).Table,
			Value:                   f.Definition.(*FilterOptions).Value,
			ConditionOperator:       f.Definition.(*FilterOptions).ConditionOperator,
			LogicalOperator:         f.Definition.(*FilterOptions).LogicalOperator,
			SecondConditionOperator: f.Definition.(*FilterOptions).SecondConditionOperator,
			SecondValue:             f.Definition.(*FilterOptions).SecondValue,
		}
	}

	return nil
}

// WithEditedFilter adds an edited filter to a report.ShareOptions.
func WithEditedFilter(o ShareOptions, s *modals.ShareReportInput) *ShareOptions {
	if s.EditFilter != nil {
		o.Filter = inFilterFromInput(s.EditFilter)
	} else if s.SaveFilter != nil {
		o.Filter = inFilterFromInput(&s.SaveFilter.EditInFilterInput)
	}

	return &o
}

// GetEmbeddedReport generates html with embedded power bi report
func GetEmbeddedReport(fileName string, reportTemplatePath string, replaceMarker string, replaceObject interface{}) (string, error) {
	l := zap.L()

	err := CheckAndCreateDir(htmlDirectoryPath)
	if err != nil {
		l.Error("couldn't create html files directory", zap.Error(err))

		return "", err
	}

	optionsJSON, err := json.Marshal(replaceObject)
	if err != nil {
		l.Error("couldn't marshal options", zap.Error(err))

		return "", err
	}

	templateBody, err := ReadFile(reportTemplatePath)
	if err != nil {
		zap.L().Error("couldn't read template", zap.Error(err))

		return "", err
	}

	templateBody = strings.Replace(templateBody, replaceMarker, string(optionsJSON), 1)
	newHTMLPath := path.Join(htmlDirectoryPath, GetUniqueFileName(fileName, "html"))
	err = os.WriteFile(newHTMLPath, []byte(templateBody), 0644)
	if err != nil {
		l.Error("couldn't write html", zap.Error(err), zap.String("newHTMLPath", newHTMLPath))

		return "", err
	}

	return newHTMLPath, nil
}
