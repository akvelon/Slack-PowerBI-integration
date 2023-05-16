package constants

import (
	"fmt"
)

const (
	// AppName is slack app name
	AppName = "Power BI integration"
	// ActionIDVisual is the action id of the visual selection dropdown.
	ActionIDVisual = "visual"
	// ActionIDReport is the action id of the report selection dropdown.
	ActionIDReport = "report"
	// ActionIDScheduledReport is the action id of the report selection dropdown.
	ActionIDScheduledReport = "scheduledReport"
	//ActionIDSearchReport is action id of the report search button
	ActionIDSearchReport = "searchReportAction"
	//ActionIDSearchWorkspace is action id of the workspace search button
	ActionIDSearchWorkspace = "searchWorkspaceAction"
	//ActionIDSearchReportInput is action id of the report search input field
	ActionIDSearchReportInput = "searchReportInputAction"
	//ActionIDSearchWorkspaceInput is action id of the workspace search input field
	ActionIDSearchWorkspaceInput = "searchWorkspaceInputAction"
	// ActionIDScheduledReportForAlerts is the action id of the report selection dropdown.
	ActionIDScheduledReportForAlerts = "scheduledReportForAlerts"
	// ActionIDChannel is the action id of the channel selection dropdown.
	ActionIDChannel = "channel"
	// ActionIDChooseScheduledReport is the action id of the scheduled reports selection dropdown.
	ActionIDChooseScheduledReport = "ChooseScheduledReport"
	// ActionIDChooseAlert is the action id of the alerts selection dropdown.
	ActionIDChooseAlert = "ChooseAlert"
	// ActionIDAddSecondFilter is the action id of the "Add filter" button.
	ActionIDAddSecondFilter = "addSecondFilter"
	// ActionIDAddFilterForManagement is the action id of the "Add filter" button in filter creating.
	ActionIDAddFilterForManagement = "addFilterForManagement"
	// ActionIDRemoveSecondFilter is the action id of the "Remove second filter" button.
	ActionIDRemoveSecondFilter = "removeSecondFilter"
	// ActionIDApplyFilter is the action id of the "apply a filter" checkbox.
	ActionIDApplyFilter = "applyFilter"
	// ActionIDReuseFilter is the action id of the "use a saved filter" checkbox.
	ActionIDReuseFilter = "reuseFilter"
	// ActionIDFilter is the action id of the filter selection dropdown.
	ActionIDFilter = "filter"
	// ActionIDTable is the action id of the table name input.
	ActionIDTable = "table"
	// ActionIDColumn is the action id of the column name input.
	ActionIDColumn = "column"
	// ActionIDValue is the action id of the filter value input.
	ActionIDValue = "value"
	// ActionIDLogicalOperator is the action id of the radio buttons group Or/And.
	ActionIDLogicalOperator = "logicalOperator"
	// ActionIDConditionOperator is the action id of the choose advanced option.
	ActionIDConditionOperator = "conditionOperator"
	// ActionIDSecondConditionOperator is the action id of the choose second advanced option.
	ActionIDSecondConditionOperator = "secondConditionOperator"
	// ActionIDSecondValue is the action id of the filter second value input.
	ActionIDSecondValue = "secondValue"
	// ActionIDSaveFilter is the action id of the "save edited filter" checkbox.
	ActionIDSaveFilter = "saveFilter"
	// ActionIDCreateFilter is the action id of the create filter button.
	ActionIDCreateFilter = "createFilterButtonAction"
	// ActionIDUpdateFilter is the action id of the update filter button.
	ActionIDUpdateFilter = "updateFilterButtonAction"
	// ActionIDDeleteFilter is the action id of the delete filter button.
	ActionIDDeleteFilter = "deleteFilterButtonAction"
	// ActionIDUpdateScheduledReport is the action id of the update scheduled report button.
	ActionIDUpdateScheduledReport = "updateScheduledReport"
	// ActionIDUpdateAlert is the action id of the update alert button.
	ActionIDUpdateAlert = "updateAlert"
	// ActionIDDeleteScheduledReport is the action id of the delete scheduled report button.
	ActionIDDeleteScheduledReport = "deleteScheduledReport"
	// ActionIDDeleteAlert is the action id of the delete alert button.
	ActionIDDeleteAlert = "deleteAlert"
	// ActionIDFilterToUpdate is the action id of the choose update filter dropdown.
	ActionIDFilterToUpdate = "filterToUpdate"
	// ActionIDFilterToDelete is the action id of the choose delete filter dropdown.
	ActionIDFilterToDelete = "filterToDelete"
	// ActionIDName is the action id of the filter name input.
	ActionIDName = "name"
	// ActionIDPeriodicity is the action id of the periodicity input.
	ActionIDPeriodicity = "periodicity"
	// ActionIDTime is the action id of the time input.
	ActionIDTime = "time"
	// ActionIDWeekday is the action id of the weekday input.
	ActionIDWeekday = "weekday"
	// ActionIDDayOfMonth is the action id of the day of month input.
	ActionIDDayOfMonth = "dayOfMonth"
	// ActionIDPages is the action id of the pages input.
	ActionIDPages = "pages"
	// ActionIDWorkspacePBI is the action id of the PBI workspace input
	ActionIDWorkspacePBI = "workspacesPBI"
	// BlockIDSaveFilter is the block id of the "save edited filter" checkbox.
	BlockIDSaveFilter = "SaveFilter"
	// BlockIDVisual is the block id of the visual selection dropdown.
	BlockIDVisual = "Visual"
	// BlockIDReportHeader is the block id of the report selection modal header.
	BlockIDReportHeader = "ReportHeader"
	// BlockIDReport is the block id of the report selection dropdown.
	BlockIDReport = "ReportSelect"
	// BlockIDChannel is the block id of the channel selection dropdown.
	BlockIDChannel = "Channel"
	// BlockIDEditFilter is the block id of the update/delete action.
	BlockIDEditFilter = "EditFilter"
	// BlockIDEditScheduledReport is the block id of the update/delete scheduled report action.
	BlockIDEditScheduledReport = "EditScheduledReport"
	// BlockIDEditAlert is the block id of the update/delete alert action.
	BlockIDEditAlert = "EditAlert"
	// BlockIDFilterToUpdate is the block id of the choose update filter dropdown.
	BlockIDFilterToUpdate = "FilterToUpdate"
	// BlockIDFilterToDelete is the block id of the choose delete filter dropdown.
	BlockIDFilterToDelete = "FilterToDelete"
	// BlockIDChannelWarning is the block id of the channel dropdown validation warning.
	BlockIDChannelWarning = "ChannelWarning"
	// BlockIDScheduledReport is the block id of the choose report dropdown.
	BlockIDScheduledReport = "ScheduledReport"
	// BlockIDScheduledReportForAlerts is the block id of the choose report dropdown.
	BlockIDScheduledReportForAlerts = "forAlertsScheduledReport"
	// BlockIDChooseScheduledReport is the block id of the choose scheduled report dropdown.
	BlockIDChooseScheduledReport = "ChooseScheduledReport"
	// BlockIDChooseAlert is the block id of the choose alert dropdown.
	BlockIDChooseAlert = "ChooseAlert"
	// BlockIDAddSecondFilter is the block id of the "Add filter" button.
	BlockIDAddSecondFilter = "AddSecondFilter"
	// BlockIDAddFilterForManagement is the block id of the "Add filter" button in filter creating.
	BlockIDAddFilterForManagement = "AddFilterForManagement"
	// BlockIDSearchReportInput is the block id of the "find report" input block.
	BlockIDSearchReportInput = "SearchReportInputBlock"
	// BlockIDSearchWorkspaceInput is the block id of the "find workspace" input block.
	BlockIDSearchWorkspaceInput = "SearchWorkspaceInputBlock"
	// BlockIDSearchReportButton is the block id of the "find report" button.
	BlockIDSearchReportButton = "SearchReportButtonBlock"
	// BlockIDSearchWorkspaceButton is the block id of the "find report" button.
	BlockIDSearchWorkspaceButton = "SearchWorkspaceButtonBlock"
	// BlockIDRemoveSecondFilter is the block id of the "Remove second filter" button.
	BlockIDRemoveSecondFilter = "RemoveSecondFilter"
	// BlockIDApplyFilter is the block id of the "apply a filter" checkbox.
	BlockIDApplyFilter = "ApplyFilter"
	// BlockIDReuseFilter is the block id of the "use a saved filter" checkbox.
	BlockIDReuseFilter = "ReuseFilter"
	// BlockIDFilter is the block id of the filter selection dropdown.
	BlockIDFilter = "Filter"
	// BlockIDFilterHeader is block id of the filter editing modal header.
	BlockIDFilterHeader = "FilterHeader"
	// BlockIDTable is the block id of the table name input.
	BlockIDTable = "Table"
	// BlockIDColumn is the block id of the column name input.
	BlockIDColumn = "Column"
	// BlockIDValue is the block id of the filter value input.
	BlockIDValue = "Value"
	// BlockIDLogicalOperator is the block id of the radio buttons group Or/And.
	BlockIDLogicalOperator = "LogicalOperator"
	// BlockIDConditionOperator is the block id of the choose advanced option.
	BlockIDConditionOperator = "ConditionOperator"
	// BlockIDSecondConditionOperator is the block id of the choose second advanced option.
	BlockIDSecondConditionOperator = "SecondConditionOperator"
	// BlockIDSecondValue is the block id of the filter second value input.
	BlockIDSecondValue = "SecondValue"
	// BlockIDName is the block id of the filter name input.
	BlockIDName = "Name"
	// BlockIDPeriodicity is the block id of the periodicity input.
	BlockIDPeriodicity = "Periodicity"
	// BlockIDTime is the block id of the time input.
	BlockIDTime = "Time"
	// BlockIDWeekday is the block id of the weekday input.
	BlockIDWeekday = "Weekday"
	// BlockIDDayOfMonth is the block id of the day of month input.
	BlockIDDayOfMonth = "DayOfMonth"
	// HintScheduleReport is the hint for the "schedule report" slash command.
	HintScheduleReport = "Schedule automatic report posting."
	// BlockIDPages is the block id of the pages input.
	BlockIDPages = "Pages"
	// BlockIDWorkspacePBI is the block id of the PBI workspaces input
	BlockIDWorkspacePBI = "WorkspacesPBI"
	// HeaderChooseReport is the header text for report sharing modal.
	HeaderChooseReport = "Please choose a Power BI workspace, report and a channel to share it in:"
	// HeaderChooseReportManagement is the header text for manage scheduled reports modal.
	HeaderChooseReportManagement = "Choose a report:"
	// HeaderChooseScheduledReport is the header text for choosing scheduled reports.
	HeaderChooseScheduledReport = "Choose a scheduled report:"
	// HeaderChooseAlert is the header text for choosing alert.
	HeaderChooseAlert = "Choose an alert:"
	// HeaderComposeFilter is the header text for filter editing modal.
	HeaderComposeFilter = "Compose a filter for a chosen report:"
	// HeaderSetSchedule is the header text for the posting schedule section.
	HeaderSetSchedule = "Set a posting schedule:"
	// CloseLabel composes text for close button.
	CloseLabel = "Close"
	// CreateAlertLabel composes text for create alert title.
	CreateAlertLabel = "Create an alert"
	// LoadingLabel composes text for loading message.
	LoadingLabel = "⏳ Loading..."
	// NoReportsWarning composes text empty set error.
	NoReportsWarning = "⚠️ No Power BI reports were found."
	// EmptyAlerts is a message if user don't have alerts.
	EmptyAlerts = "You don't have any alerts."
	// EmptyScheduledReports is a message if user don't have scheduled reports.
	EmptyScheduledReports = "You don't have any scheduled reports."
	// GetReportsWithError is a message if error has occurred when we try to get reports.
	GetReportsWithError = "Sorry, we couldn’t process your request. Please try again later."
	// OkLabel composes text for ok button.
	OkLabel = "OK"
	// PowerBiNotConnectedWarning composes warning for non-authenticated user.
	PowerBiNotConnectedWarning = "You don't have linked Power BI account, want to sign in?"
	// TitleShareReport is the title of report sharing modal.
	TitleShareReport = "Share a report"
	// TitleManageFilters is the title of filter managing modal.
	TitleManageFilters = "Manage filters"
	// TitleChooseFilter is the title of filter selection modal.
	TitleChooseFilter = "Choose a filter"
	// TitleComposeFilter is the title of report editing modal.
	TitleComposeFilter = "Compose a filter"
	// TitleScheduleReport is the title of report scheduling modal.
	TitleScheduleReport = "Schedule a report"
	// TitleManageScheduledReports is the title of manage reports modal.
	TitleManageScheduledReports = "Manage scheduled reports"
	// TitleManageAlerts is the title of manage alerts modal.
	TitleManageAlerts = "Manage alerts"
	// PlaceholderReport is the placeholder for report selection dropdown.
	PlaceholderReport = "Report"
	// PlaceholderScheduledReport is the placeholder  for scheduled report selection dropdown.
	PlaceholderScheduledReport = "Scheduled report"
	// PlaceholderAlert is the placeholder  for alert selection dropdown.
	PlaceholderAlert = "Alert"
	// PlaceholderChannel is the placeholder for channel selection dropdown.
	PlaceholderChannel = "Channel"
	// PlaceholderFilter is the placeholder for filter selection dropdown.
	PlaceholderFilter = "Filter"
	// PlaceholderOperation is the placeholder for radio buttons group Or/And.
	PlaceholderOperation = "Logical operation"
	// PlaceholderConditionOperator is the placeholder for choose advanced option.
	PlaceholderConditionOperator = "Show items when the value:"
	// PlaceholderPeriodicity is the placeholder for the periodicity input.
	PlaceholderPeriodicity = "Periodicity"
	// PlaceholderTime is the placeholder for the time input.
	PlaceholderTime = "Time"
	// PlaceholderWeekday is the placeholder for the weekday input.
	PlaceholderWeekday = "Weekday"
	// PlaceholderDayOfMonth is the placeholder for the day of month input.
	PlaceholderDayOfMonth = "Day of month"
	// PlaceholderPages is the placeholder of the pages input.
	PlaceholderPages = "Pages"
	// PlaceholderPBIWorkspaces is the placeholder of the PBI workspaces input
	PlaceholderPBIWorkspaces = "Workspaces"
	// SignInLabel composes text for sign-in button.
	SignInLabel = "Sign-in"
	// WarningLabel composes text for Warning title.
	WarningLabel = "Warning"
	// ShareReportCommandHelp is a description for /pbi-share-report slash command
	ShareReportCommandHelp = "Share your PowerBi reports to specific channel. Just type /pbi-share-report"
	// SignInCommandHelp is a description for /pbi-share-report slash command
	SignInCommandHelp = "Connect your PowerBi account to this application. Just type /pbi-sign-in"
	// SignOutCommandHelp is a description for /pbi-share-report slash command
	SignOutCommandHelp = "Remove your PowerBi account from this application. Just type /pbi-sign-out"
	// CreateAlertCommandHelp is a description for /pbi-create-alert slash command
	CreateAlertCommandHelp = "Create alert for your visual from some PowerBI report and set channel to send it to. Just type /pbi-create-alert"
	// ManageFiltersCommandHelp is a description for /pbi-manage-filters slash command
	ManageFiltersCommandHelp = "Manage your filters. Just type /pbi-manage-filters"
	// ManageReportsCommandHelp is a description for /pbi-manage-schedule-report slash command
	ManageReportsCommandHelp = "Manage your scheduled reports. Just type /pbi-manage-schedule-report"
	// ManageAlertsCommandHelp is a description for /pbi-manage-alerts slash command
	ManageAlertsCommandHelp = "Manage your alerts. Just type /pbi-manage-alerts"
	// NoteLabel composes text for the note.
	NoteLabel = "NOTE"
	// SignOut is used like CallbackID in select report modal
	SignOut = "SignOut"
	// CallbackIDSaveAlert corresponds to the "save alert" modal.
	CallbackIDSaveAlert = "SaveAlert"
	// CallbackIDSaveAlertSelectReport corresponds to the initial "save alert" modal with selection for report.
	CallbackIDSaveAlertSelectReport = CallbackIDSaveAlert + "Initial"
	// CallbackIDShareReport corresponds to states of the "share a report" modal.
	CallbackIDShareReport = "ShareReport"
	// CallbackIDShareReportSelectReport corresponds to the initial report selection state of the "share a report" modal.
	CallbackIDShareReportSelectReport = CallbackIDShareReport + "Initial"
	// CallbackIDShareReportEditFilter corresponds to the filter editing state of the "share a report" modal.
	CallbackIDShareReportEditFilter = CallbackIDShareReport + "EditFilter"
	// CallbackIDShareReportSaveFilter corresponds to the filter saving state of the "share a report" modal.
	CallbackIDShareReportSaveFilter = CallbackIDShareReport + "SaveFilter"
	// CallbackIDShareReportReuseFilter corresponds to the filter reusing state of the "share a report" modal.
	CallbackIDShareReportReuseFilter = CallbackIDShareReport + "ReuseFilter"
	// CallbackIDManageFilters corresponds to states of the "manage filters" modal.
	CallbackIDManageFilters = "ManageFilters"
	// CallbackIDCreateFilter corresponds to states of the "create filter" modal.
	CallbackIDCreateFilter = "CreateFilter"
	// CallbackIDUpdateFilter corresponds to states of the "update filter" modal.
	CallbackIDUpdateFilter = "UpdateFilter"
	// CallbackIDDeleteFilter corresponds to states of the "delete filter" modal.
	CallbackIDDeleteFilter = "DeleteFilter"
	// CallbackIDUpdateCurrentFilter corresponds to states of the "update current filter" modal.
	CallbackIDUpdateCurrentFilter = "UpdateCurrentFilter"
	// CallbackIDScheduleReport corresponds to the "schedule a report" modal.
	CallbackIDScheduleReport = "ScheduleReport"
	// CallbackIDManageScheduledReports corresponds to the "manage scheduled reports" modal.
	CallbackIDManageScheduledReports = "ManageScheduledReports"
	// CallbackIDManageAlerts corresponds to the "manage alerts" modal.
	CallbackIDManageAlerts = "ManageAlerts"
	// CallbackIDScheduleReportHourly corresponds to the "schedule a report" modal in the hour selection state.
	CallbackIDScheduleReportHourly = CallbackIDScheduleReport + "Hourly"
	// CallbackIDScheduleReportWeekly corresponds to the "schedule a report" modal in the weekday selection state.
	CallbackIDScheduleReportWeekly = CallbackIDScheduleReport + "Weekly"
	// CallbackIDScheduleReportMonthly corresponds to the "schedule a report" modal in the day of month selection state.
	CallbackIDScheduleReportMonthly = CallbackIDScheduleReport + "Monthly"
	// WarningBotIsNotInChan is the validation error shown when the bot isn't on channel.
	WarningBotIsNotInChan = "Application is not added to the channel."
	// WarningFilterExists is the validation error shown when user is trying to save a new filter under an existing name.
	WarningFilterExists = "A saved filter named like this already exists. Choose another name."
	// ValueAddSecondFilter is the value of the "Add filter" button.
	ValueAddSecondFilter = "AddFilterButtonValue"
	// ValueSearchReport is the value of the "search report" button
	ValueSearchReport = "searchReportValue"
	// ValueSearchWorkspace is the value of the "search report" button
	ValueSearchWorkspace = "searchWorkspaceValue"
	// WarningScheduleExists is the error shown when a user is adding a posting schedule w/ same parameters.
	WarningScheduleExists = "A posting schedule for this report, channel, & periodicity already exists."
	// ValueApplyFilter is the value of the "apply a filter" button.
	ValueApplyFilter = "applyFilter"
	// ValueOperationAnd is the value "And" of the radio buttons group.
	ValueOperationAnd = "And"
	// ValueOperationOr is the value "Or" of the radio buttons group.
	ValueOperationOr = "Or"
	// ValueRemoveSecondFilter is the value of the "Remove second filter" button.
	ValueRemoveSecondFilter = "removeSecondFilterButtonValue"
	// ValueReuseFilter is the value of the "use a saved filter" checkbox.
	ValueReuseFilter = "reuseFilter"
	// ValueSaveFilter is the value of the "save edited filter" checkbox.
	ValueSaveFilter = "saveFilter"
	// ValueUpdateFilter is the value of the "update filter" button.
	ValueUpdateFilter = "valueUpdateFilterButton"
	// ValueDeleteFilter is the value of the "delete filter" button.
	ValueDeleteFilter = "valueDeleteFilterButton"
	// ValueUpdateScheduledReport is the value of the "update" button.
	ValueUpdateScheduledReport = "valueUpdateScheduledReportButton"
	// ValueUpdateAlert is the value of the "update" alert button.
	ValueUpdateAlert = "valueUpdateAlert"
	// ValueDeleteScheduledReport is the value of the "delete" button.
	ValueDeleteScheduledReport = "valueDeleteScheduledReportButton"
	// ValueDeleteAlert is the value of the "delete" alert button.
	ValueDeleteAlert = "valueDeleteAlert"
	// ValueStopButton is a text for stop button
	ValueStopButton = "Stop"
	// ValueResumeButton is a text for resume button
	ValueResumeButton = "Resume"
	// ValueDeleteButton is a text for delete button
	ValueDeleteButton = "Delete"
	// ConnectActionID is action ID for connect to Power BI button
	ConnectActionID = "ConnectID"
	// DisconnectActionID is action ID for Disconnect button
	DisconnectActionID = "DisconnectID"
	// HomeSignOutID is action ID for Sign out button in the Home tab
	HomeSignOutID = "HomeSignOutID"
	// CancelLabel is label for modal button
	CancelLabel = "Cancel"
	// DisconnectLabel is label for button
	DisconnectLabel = "Disconnect Power BI account"
	// LabelViewReport is the title of the external report link in a message w/ a report image.
	LabelViewReport = "View the report in Power BI"
	// LabelApplyFilter is the label of the "apply a filter" checkbox.
	LabelApplyFilter = "Apply a filter"
	// LabelDayOfMonthLast is the label for the "last day of month" option.
	LabelDayOfMonthLast            = "Last"
	LabelPBIWorkspacesList         = "Power BI Workspaces"
	LabelNotAllReportsInList       = "Some reports are not available in the list. Use search to narrow it down."
	LabelNotAllWorkspacesInList    = "Some workspaces are not available in the list. Use search to narrow it down."
	ReportQuantityReducer          = 30
	PBIWorkspacesQuantityReducer   = 30
	MaxLenghthOfFilterName         = 75
	MaxLenghthOfFilterNameForTitle = 24
)

var (
	// LabelPeriodicity is a label of the periodicity input.
	LabelPeriodicity = [...]string{
		"Every hour",
		"Every day",
		"Every week",
		"Every month",
	}
	// AuthHTMLResponse composses html with some message
	AuthHTMLResponse = func(message string, isErrorMessage bool) string {
		messageClass := ""
		if isErrorMessage {
			messageClass = "error-text"
		}

		return `<!DOCTYPE html>
		<html>
			<head>
				<style>
					html, body {
						height: 100%;
					}

					body {
						background-color: #f3ca00;
						font-family: "Segoe UI", "Helvetica Neue", "Helvetica", Arial, sans-serif;
						font-weight: 300;
						font-size: 1.3rem;
						color: black;
					}

					.branding {
						padding-left: 10px;
						font-size: 20px;
						letter-spacing: -0.04rem;
						font-weight: 400;
						color: black;
						text-decoration: none;
					}

					.flex-container {
						display: flex;
						flex-grow: 1;
						height: 100%;
						justify-content: center;
						align-items: center;
					}

					.error-text {
						color: #720000;
					}
				</style>
			</head>
			<body>
				<div class="branding">` + AppName + `</div>
				<div class="flex-container">
					<div class="` + messageClass + `">` + message + `</div>
				</div>
			</body>
		</html>`
	}
	// BotIsNotInChannel is message when the bot is not added to channel
	BotIsNotInChannel = func(channel, bot string) string {
		return fmt.Sprintf(
			`Please invite application to the <#%s> channel by yourself. Click on <@%s> and choose *Add this app to a channel ...* or choose another channel for publication`,
			channel,
			bot,
		)
	}
)
