package utils

import (
	"fmt"


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
	ClientID          string         `json:"clientId"`
	AccessToken       string         `json:"accessToken"`
	ReportID          string         `json:"reportId"`
	ReportName        string         `json:"reportName"`
	Filter            *FilterOptions `json:"filter"`
	Pages             []*PageOptions
	ChannelID         string
	WorkspaceID       string
	UserID            string
	IsScheduled       bool
	SkipPosting       bool
	RetryAttempt      int
	PostReportMessage *messagequeue.PostReportMessage
}

// PageOptions holds page parameters.
type PageOptions struct {
	ID   string
	Name string
}

// WithAccessToken adds an access token to a report.ShareOptions.
func WithAccessToken(o ShareOptions, accessToken string) *ShareOptions {
	o.AccessToken = accessToken

	return &o
}
