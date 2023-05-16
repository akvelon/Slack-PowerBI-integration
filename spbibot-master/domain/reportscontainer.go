package domain

import (
	"encoding/json"
)

// ReportsContainer contains multiple reports
type ReportsContainer struct {
	Type     ReportType
	Value    []IReport         `json:"-"`
	RawValue []json.RawMessage `json:"value"`
}

func newReportsContainer(t ReportType) *ReportsContainer {
	return &ReportsContainer{
		Type: t,
	}
}

// UnmarshalJSON from encoding/json Unmarshaler interface
func (reportContainer *ReportsContainer) UnmarshalJSON(b []byte) (err error) {
	type rc ReportsContainer

	if err = json.Unmarshal(b, (*rc)(reportContainer)); err != nil {
		return err
	}

	for _, raw := range reportContainer.RawValue {
		var i IReport
		switch reportContainer.Type {
		case report:
			i = &Report{}
		default:
			return ErrUnknownReportType
		}

		if err = json.Unmarshal(raw, i); err != nil {
			return err
		}

		reportContainer.Value = append(reportContainer.Value, i)
	}

	return nil
}
