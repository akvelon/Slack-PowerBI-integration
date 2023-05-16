package reportengine

import (
	"fmt"


)

// NOTE: See `IReportLoadConfiguration' definition here `https://github.com/microsoft/powerbi-models/blob/master/src/models.ts'.
type reportLoadConfiguration struct {
	AccessToken string        `json:"accessToken"`
	ID          string        `json:"id"`
	Filters     []interface{} `json:"filters,omitempty"`
}

func newReportLoadConfiguration(o *utils.ShareOptions) *reportLoadConfiguration {
	conf := reportLoadConfiguration{
		AccessToken: o.AccessToken,
		ID:          o.ReportID,
	}

	if o.Filter != nil {
		conf.Filters = []interface{}{
			newAdvancedFilter(o.Filter),
		}
	}

	return &conf
}

type filterType int

const (
	filterTypeAdvanced filterType = 0
	filterTypeBasic    filterType = 1
)

// NOTE: See `IBaseTarget' definition.
type baseTarget struct {
	Table string `json:"table"`
}

// NOTE: See `IColumnTarget' definition.
type columnTarget struct {
	*baseTarget
	Column string `json:"column"`
}

type filterSchema string

const (
	filterSchemaBasic    filterSchema = "http://powerbi.com/product/schema#basic"
	filterSchemaAdvanced filterSchema = "http://powerbi.com/product/schema#advanced"
)

// NOTE: See `IFilter' definition.
type filter struct {
	Schema     filterSchema `json:"$schema"`
	Target     interface{}  `json:"target"`
	FilterType filterType   `json:"filterType"`
}

type basicFilterOperator string

const (
	basicFilterOperatorIn basicFilterOperator = "In"
)

// NOTE: See `IBasicFilter' definition.
type basicFilter struct {
	*filter
	Operator basicFilterOperator `json:"operator"`
	Values   []interface{}       `json:"values"`
}

func newBasicFilter(f *utils.FilterOptions) *basicFilter {
	return &basicFilter{
		filter: &filter{
			Schema: filterSchemaBasic,
			Target: &columnTarget{
				baseTarget: &baseTarget{
					Table: f.Table,
				},
				Column: f.Column,
			},
			FilterType: filterTypeBasic,
		},
		Operator: basicFilterOperatorIn,
		Values: []interface{}{
			f.Value,
		},
	}
}

type logicalOperator string

type conditionOperator string

type condition struct {
	Value    interface{}       `json:"value,omitempty"`
	Operator conditionOperator `json:"operator"`
}

type advancedFilter struct {
	*filter
	LogicalOperator logicalOperator `json:"logicalOperator"`
	Conditions      []*condition    `json:"conditions"`
}

func newAdvancedFilter(f *utils.FilterOptions) *advancedFilter {
	cs := []*condition(nil)
	cs = append(cs, &condition{
		Value:    f.Value,
		Operator: conditionOperator(f.ConditionOperator),
	})
	if f.LogicalOperator != "" {
		cs = append(cs, &condition{
			Value:    f.SecondValue,
			Operator: conditionOperator(f.SecondConditionOperator),
		})
	}

	return &advancedFilter{
		filter: &filter{
			Schema: filterSchemaAdvanced,
			Target: &columnTarget{
				baseTarget: &baseTarget{
					Table: f.Table,
				},
				Column: f.Column,
			},
			FilterType: filterTypeAdvanced,
		},
		LogicalOperator: logicalOperator(f.LogicalOperator),
		Conditions:      cs,
	}
}

// NOTE: See `ICustomPageSize' definition.
type customPageSize struct {
	Height int64 `json:"height,omitempty"`
	Width  int64 `json:"width,omitempty"`
}

// NOTE: See `TraceType' definition.
type traceType int

// NOTE: See `KeyValuePair' definition.
type keyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NOTE: See `ITechnicalDetails' definition.
type technicalDetails struct {
	RequestID string          `json:"requestId,omitempty"`
	ErrorInfo []*keyValuePair `json:"errorInfo,omitempty"`
}

// NOTE: See `IError' definition.
type pbiError struct {
	Message          string            `json:"message"`
	DetailedMessage  string            `json:"detailedMessage,omitempty"`
	ErrorCode        string            `json:"errorCode,omitempty"`
	Level            traceType         `json:"level,omitempty"`
	TechnicalDetails *technicalDetails `json:"technicalDetails,omitempty"`
}

func (p *pbiError) Error() string {
	s := p.Message
	if p.ErrorCode != "" {
		s = fmt.Sprintf("%v: %v", p.ErrorCode, s)
	}

	if p.DetailedMessage != "" {
		s = fmt.Sprintf("%v: %v", s, p.DetailedMessage)
	}

	return s
}

// NOTE: See `ICustomEvent' definition.
type customEvent struct {
	Detail *pbiError `json:"detail"`
}
