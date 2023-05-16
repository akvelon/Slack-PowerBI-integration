package modals

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"


)

// ManageReportsModal defines slack modal for manage reports
type ManageReportsModal struct {
	BaseModal      *BaseModal
	GroupedReports domain.GroupedReports
	ChannelID      string
}

// NewManageReportModal returns new ManageReportsModal
func NewManageReportModal(title, close string, rs domain.GroupedReports, channelID string) *ManageReportsModal {
	return &ManageReportsModal{
		BaseModal:      NewBaseModal(title, close),
		GroupedReports: rs,
		ChannelID:      channelID,
	}
}

type manageScheduledReportsModal struct {
	*ManageReportsModal
}

// NewManageScheduledReportsModal creates a "manage scheduled report" modal.
func NewManageScheduledReportsModal(title, close string, rs domain.GroupedReports, channelID string) ISlackModal {
	return &manageScheduledReportsModal{
		ManageReportsModal: NewManageReportModal(title, close, rs, channelID),
	}
}

func (m *manageScheduledReportsModal) GetViewRequest() slack.ModalViewRequest {
	reportsText := slackcomponents.GetSlackMarkdownTextBlock(constants.HeaderChooseReportManagement)
	reportsSection := slack.NewSectionBlock(reportsText, nil, nil)

	reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
	reportSelect := buildReportsSelect(m.GroupedReports, reportPlaceholder, constants.ActionIDScheduledReport)
	reportInput := slack.NewActionBlock(constants.BlockIDScheduledReport, reportSelect)

	bs := slack.Blocks{
		BlockSet: []slack.Block{reportsSection, reportInput},
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      m.BaseModal.Title,
		Blocks:     bs,
		Close:      m.BaseModal.CloseText,
		CallbackID: constants.CallbackIDManageScheduledReports,
	}
}

// ShowManageReportsControls displays controls for create reports management dialog.
func ShowManageReportsControls(v *slack.View, reports []*domain.PostReportTask, rid string) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	scheduledReportsText := slackcomponents.GetSlackMarkdownTextBlock(constants.HeaderChooseScheduledReport)
	scheduledReportsSection := slack.NewSectionBlock(scheduledReportsText, nil, nil)

	reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderScheduledReport)
	reportSelect := buildScheduledReportsSelect(reports, reportPlaceholder, constants.ActionIDChooseScheduledReport)
	reportAction := slack.NewActionBlock(constants.BlockIDChooseScheduledReport+rid, reportSelect)

	r.Blocks.BlockSet = append(r.Blocks.BlockSet[:2], scheduledReportsSection, reportAction)

	r.Title = slackcomponents.GetSlackPlainTextBlock(constants.TitleManageScheduledReports)

	return r
}

// ShowManageReportDeleteControls is modal view after deleting report.
func ShowManageReportDeleteControls(v *slack.View, rs domain.GroupedReports) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	r.Blocks.BlockSet = []slack.Block{}

	reportsText := slackcomponents.GetSlackMarkdownTextBlock(constants.HeaderChooseReportManagement)
	reportsSection := slack.NewSectionBlock(reportsText, nil, nil)

	reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
	reportSelect := buildReportsSelect(rs, reportPlaceholder, constants.ActionIDScheduledReport)
	var reportInput *slack.ActionBlock
	item := v.State.Values[constants.BlockIDScheduledReport][constants.ActionIDScheduledReport].SelectedOption.Value
	if item == "" {
		reportInput = slack.NewActionBlock(constants.BlockIDScheduledReport, reportSelect)
	} else {
		reportInput = slack.NewActionBlock(constants.BlockIDScheduledReport+v.ID, reportSelect)
	}

	r.Blocks.BlockSet = append(r.Blocks.BlockSet, reportsSection, reportInput)
	return r
}

// ShowEmptyScheduledReportsModal is modal view if scheduled reports doesn't contain.
func ShowEmptyScheduledReportsModal(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	r.Blocks.BlockSet = []slack.Block{}
	messageText := slackcomponents.GetSlackPlainTextBlock(constants.EmptyScheduledReports)
	messageSection := slack.NewSectionBlock(messageText, nil, nil)

	r.Blocks.BlockSet = append(r.Blocks.BlockSet, messageSection)
	return r
}

// ShowEmptyAlertsModal is modal view if alerts doesn't contain.
func ShowEmptyAlertsModal(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	r.Blocks.BlockSet = []slack.Block{}
	messageText := slackcomponents.GetSlackPlainTextBlock(constants.EmptyAlerts)
	messageSection := slack.NewSectionBlock(messageText, nil, nil)

	r.Blocks.BlockSet = append(r.Blocks.BlockSet, messageSection)
	return r
}

// ShowScheduledReportPageNames is modal view after choosing current report.
func ShowScheduledReportPageNames(v *slack.View, rs []*domain.PostReportTask, ps []*domain.Page, rid string) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	id := v.State.Values[constants.BlockIDChooseScheduledReport+rid][constants.ActionIDChooseScheduledReport].SelectedOption.Value
	actualPages := findActualPages(id, rs, ps)

	actualPagesText := slackcomponents.GetSlackMarkdownTextBlock(actualPages)
	actualPagesSection := slack.NewSectionBlock(actualPagesText, nil, nil)

	var editReportAction *slack.ActionBlock
	deleteReportText := slackcomponents.GetSlackPlainTextBlock(constants.ValueDeleteButton)
	deleteReportButton := slack.NewButtonBlockElement(constants.ActionIDDeleteScheduledReport, constants.ValueDeleteScheduledReport, deleteReportText)

	actionText := getReportAction(id, rs)
	if actionText != "" {
		updateReportText := slackcomponents.GetSlackPlainTextBlock(actionText)
		updateReportButton := slack.NewButtonBlockElement(constants.ActionIDUpdateScheduledReport, constants.ValueUpdateScheduledReport, updateReportText)

		editReportAction = slack.NewActionBlock(constants.BlockIDEditScheduledReport, updateReportButton, deleteReportButton)
	} else {
		editReportAction = slack.NewActionBlock(constants.BlockIDEditScheduledReport, deleteReportButton)
	}

	if len(r.Blocks.BlockSet) == 4 {
		r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDChooseScheduledReport+rid, actualPagesSection)

		r.Blocks.BlockSet = append(r.Blocks.BlockSet, editReportAction)
	} else {
		r.Blocks.BlockSet = removeBlockAfter(r.Blocks.BlockSet, constants.BlockIDChooseScheduledReport+rid)
		r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDChooseScheduledReport+rid, actualPagesSection)

		r.Blocks.BlockSet = append(r.Blocks.BlockSet[:len(r.Blocks.BlockSet)-1], editReportAction)
	}

	return r
}

func findActualPages(r string, rs []*domain.PostReportTask, ps []*domain.Page) string {
	id, _ := strconv.ParseInt(r, 10, 64)
	var reportPages []string
	for _, e := range rs {
		if e.ID == id {
			reportPages = e.PageIDs
		}
	}
	result := "Selected pages: "
	for i, rp := range reportPages {
		for _, p := range ps {
			if p.Name == rp {
				if i == len(reportPages)-1 {
					result += p.DisplayName
				} else {
					result += p.DisplayName + ", "
				}
			}
		}
	}
	return result
}

func getReportAction(r string, rs []*domain.PostReportTask) string {
	id, _ := strconv.ParseInt(r, 10, 64)
	for _, r := range rs {
		if r.ID == id {
			if r.IsActive {
				return constants.ValueStopButton
			} else if r.ChannelID[:7] != "deleted" {
				return constants.ValueResumeButton
			}
		}
	}
	return ""
}

func buildScheduledReportsSelect(fs []*domain.PostReportTask, placeholder *slack.TextBlockObject, actionID string) *slack.SelectBlockElement {
	os := []*slack.OptionBlockObject(nil)
	for _, f := range fs {
		t := slackcomponents.GetSlackPlainTextBlock(getPeriodicityFromCronRule(f))
		o := slack.NewOptionBlockObject(strconv.FormatInt(f.ID, 10), t, nil)
		os = append(os, o)
	}

	e := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		placeholder,
		actionID,
		os...,
	)

	return e
}

func getPeriodicityFromCronRule(r *domain.PostReportTask) string {
	taskTime := strings.Split(r.TaskTime, ":")
	hours, _ := strconv.Atoi(taskTime[0])
	minutes, _ := strconv.Atoi(taskTime[1])
	utcNow := time.Now().UTC()
	location, _ := time.LoadLocation(r.TZ)

	localNow := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), hours, minutes+5, 0, 0, time.UTC).In(location)
	var periodicity string
	if r.IsEveryDay {
		periodicity = "Every day"
	} else if r.IsEveryHour {
		periodicity = "Every hour"
	} else if r.DayOfMonth == 0 {
		localNow = time.Date(utcNow.Year(), utcNow.Month(), r.DayOfWeek+7, hours, minutes+5, 0, 0, time.UTC).In(location)
		switch (localNow.Day() - 1) % 7 {
		case 0:
			periodicity = fmt.Sprintf("Every %v", "Sun")
		case 1:
			periodicity = fmt.Sprintf("Every %v", "Mon")
		case 2:
			periodicity = fmt.Sprintf("Every %v", "Tue")
		case 3:
			periodicity = fmt.Sprintf("Every %v", "Wed")
		case 4:
			periodicity = fmt.Sprintf("Every %v", "Thu")
		case 5:
			periodicity = fmt.Sprintf("Every %v", "Fri")
		case 6:
			periodicity = fmt.Sprintf("Every %v", "Sat")
		default:
			periodicity = "Invalid day of week"
		}
	} else {
		localNow = time.Date(utcNow.Year(), utcNow.Month(), r.DayOfMonth, hours, minutes+5, 0, 0, time.UTC).In(location)
		periodicity = fmt.Sprintf("Every month (day %v)", localNow.Day())
		if r.DayOfMonth == 32 {
			periodicity = "Every last day of month"
		} else if r.DayOfMonth == -1 {
			periodicity = "Every month (day 1)"
		}
	}

	utcMinutes := strconv.Itoa(localNow.Minute())
	if localNow.Minute() == 0 {
		utcMinutes = "00"
	}
	utcHours := strconv.Itoa(localNow.Hour())
	if localNow.Hour() < 10 {
		utcHours = fmt.Sprintf("0%v", utcHours)
	}
	status := "stopped"
	if r.IsActive {
		status = "active"
	} else if r.ChannelID[:7] == "deleted" {
		status = "channel not found"
	}

	if r.ChannelName == "" {
		return fmt.Sprintf("(%v) %v %v:%v", status, periodicity, utcHours, utcMinutes)
	}
	return fmt.Sprintf("(%v) %v %v:%v, %v", status, periodicity, utcHours, utcMinutes, r.ChannelName)
}
