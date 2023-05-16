package domain

import (
	"encoding/json"
	"io"
)

// ReportType is enum for report types like a report or a dashboard
type ReportType int

const (
	report ReportType = iota
)

// IReport describes all report kinds
type IReport interface {
	GetName() string
	GetID() string
	GetWebURL() string
}

// Report represents a single report
type Report struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	WebURL string `json:"webUrl"`
}

// Groups contains multiple groups (workspaces).
type Groups struct {
	Value []*Group `json:"value"`
}

// Group represents a single group (workspace).
type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GroupedReports represents a map of reports grouped by groups (workspaces)
type GroupedReports map[*Group]*ReportsContainer

// NewGroupedReports returns map[*Group]IReports
func NewGroupedReports() map[*Group]*ReportsContainer {
	return make(map[*Group]*ReportsContainer)
}

// GetID method of IReport interface returns Report ID
func (r *Report) GetID() string {
	return r.ID
}

// GetName method of IReport interface returns Report Name
func (r *Report) GetName() string {
	return r.Name
}

// GetWebURL returns report URL.
func (r *Report) GetWebURL() string {
	return r.WebURL
}

// DeserializeGroups unmarhals json
func DeserializeGroups(b io.ReadCloser) (*Groups, error) {
	out := Groups{}
	d := json.NewDecoder(b)
	if err := d.Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// DeserializeReports unmarhals json
func DeserializeReports(b io.ReadCloser) (*ReportsContainer, error) {
	out := newReportsContainer(report)
	d := json.NewDecoder(b)

	if err := d.Decode(out); err != nil {
		return nil, err
	}

	return out, nil
}

// DeserializeReport unmarhals json to Report type
func DeserializeReport(b io.ReadCloser) (*Report, error) {
	out := Report{}
	d := json.NewDecoder(b)

	if err := d.Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
