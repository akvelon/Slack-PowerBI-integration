package messagequeue

import "fmt"

const (
	// MessagePostReport is for PostReportMessage.
	MessagePostReport MessageKind = "postReport"
)

// ErrNoMessages will be returned by MessageQueue.Peek for an empty MessageQueue.
var ErrNoMessages = fmt.Errorf("no messages to read")

// PageMessage keeps page info.
type PageMessage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FilterMessage keeps filter info.
type FilterMessage struct {
	Table                   string `json:"table"`
	Column                  string `json:"column"`
	Value                   string `json:"value"`
	LogicalOperator         string `json:"logicalOperator,omitempty"`
	ConditionOperator       string `json:"conditionOperator"`
	SecondValue             string `json:"secondValue,omitempty"`
	SecondConditionOperator string `json:"secondConditionOperator,omitempty"`
}

type Tokens struct {
	BotAccessToken string
	PowerBIToken   string
}

// RenderReportMessage is a command to perform report rendering.
type RenderReportMessage struct {
	ClientID     string         `json:"clientID"`
	ReportID     string         `json:"reportID"`
	ReportName   string         `json:"reportName"`
	Filter       *FilterMessage `json:"filter,omitempty"`
	Pages        []*PageMessage `json:"pages"`
	UserID       string         `json:"userID"`
	ChannelID    string         `json:"channelID"`
	WorkspaceID  string         `json:"workspaceID"`
	UniqueID     string         `json:"uniqueID"`
	Token        Tokens         `json:"tokens"`
	RetryAttempt int            `json:"retryAttempt"`
}

// PostReportMessage is a command to perform report rendering & posting.
type PostReportMessage struct {
	*RenderReportMessage
	IsScheduled bool `json:"isScheduled,omitempty"`
	SkipPosting bool `json:"skipPosting,omitempty"`
}
