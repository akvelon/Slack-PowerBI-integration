package constants

import (
	"fmt"
)

const (
	// ErrorAccountInactive is error from slack when workspace is removed.
	ErrorAccountInactive = "account_inactive"
	// ErrorNotInChannel is error from slack when channel is removed.
	ErrorNotInChannel = "not_in_channel"
	// LabelViewReport is the title of the external report link in a message w/ a report image.
	LabelViewReport = "View the report in Power BI"
)

// FormatMessageTitle formats message title.
func FormatMessageTitle(reportName, pageName string) string {
	return fmt.Sprintf("Report: %v; Page: %v", reportName, pageName)
}

// FormatMessageTitleWithFilter formats message title.
func FormatMessageTitleWithFilter(reportName, filterDescription, pageName string) string {
	return fmt.Sprintf("Report: %v; Filter: %v; Page: %v", reportName, filterDescription, pageName)
}

// FormatPageURL formats page URL.
func FormatPageURL(reportURL, pageID string) string {
	pageURL := fmt.Sprintf("%v/%v", reportURL, pageID)

	return fmt.Sprintf("<%v|%v>", pageURL, LabelViewReport)
}
