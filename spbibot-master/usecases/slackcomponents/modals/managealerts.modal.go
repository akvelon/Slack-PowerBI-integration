package modals

import (
	"fmt"
	"strconv"

	"github.com/slack-go/slack"

)

type manageAlertsModal struct {
	*ManageReportsModal
}

// NewManageAlertsModal creates a "manage alerts" modal.
func NewManageAlertsModal(title, close string, rs domain.GroupedReports, channelID string) ISlackModal {
	return &manageAlertsModal{
		ManageReportsModal: NewManageReportModal(title, close, rs, channelID),
	}
}

func (m *manageAlertsModal) GetViewRequest() slack.ModalViewRequest {
	reportsText := slackcomponents.GetSlackMarkdownTextBlock(constants.HeaderChooseReportManagement)
	reportsSection := slack.NewSectionBlock(reportsText, nil, nil)

	reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
	reportSelect := buildReportsSelect(m.GroupedReports, reportPlaceholder, constants.ActionIDScheduledReportForAlerts)
	reportInput := slack.NewActionBlock(constants.BlockIDScheduledReportForAlerts, reportSelect)

	bs := slack.Blocks{
		BlockSet: []slack.Block{reportsSection, reportInput},
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      m.BaseModal.Title,
		Blocks:     bs,
		Close:      m.BaseModal.CloseText,
		CallbackID: constants.CallbackIDManageAlerts,
	}
}

// ShowManageAlertsControls shows choose alert dropdown
func ShowManageAlertsControls(v *slack.View, alerts []domain.Alert, rid string) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	alertsText := slackcomponents.GetSlackMarkdownTextBlock(constants.HeaderChooseAlert)
	alertsSection := slack.NewSectionBlock(alertsText, nil, nil)

	alertPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderAlert)
	alertSelect := buildAlertsSelect(alerts, alertPlaceholder, constants.ActionIDChooseAlert)
	alertAction := slack.NewActionBlock(constants.BlockIDChooseAlert+rid, alertSelect)

	r.Blocks.BlockSet = append(r.Blocks.BlockSet[:2], alertsSection, alertAction)

	r.Title = slackcomponents.GetSlackPlainTextBlock(constants.TitleManageAlerts)

	return r
}

// ShowEditAlertControls shows update/delete buttons for chosen alert.
func ShowEditAlertControls(v *slack.View, alert domain.Alert, rid string) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	var editReportAction *slack.ActionBlock
	var updateAlertText *slack.TextBlockObject
	if alert.Status == domain.Active {
		updateAlertText = slackcomponents.GetSlackPlainTextBlock(constants.ValueStopButton)
	} else {
		updateAlertText = slackcomponents.GetSlackPlainTextBlock(constants.ValueResumeButton)
	}
	deleteReportText := slackcomponents.GetSlackPlainTextBlock(constants.ValueDeleteButton)
	deleteReportButton := slack.NewButtonBlockElement(constants.ActionIDDeleteAlert, constants.ValueDeleteAlert, deleteReportText)
	updateAlertButton := slack.NewButtonBlockElement(constants.ActionIDUpdateAlert, constants.ValueUpdateAlert, updateAlertText)
	editReportAction = slack.NewActionBlock(constants.BlockIDEditAlert, updateAlertButton, deleteReportButton)

	additionalInfo := fmt.Sprintf("%v %v", alert.Condition, alert.Threshold)
	additionalInfoText := slackcomponents.GetSlackMarkdownTextBlock(additionalInfo)
	additionalInfoSection := slack.NewSectionBlock(additionalInfoText, nil, nil)
	if len(r.Blocks.BlockSet) == 4 {
		r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDChooseAlert+rid, additionalInfoSection)

		r.Blocks.BlockSet = append(r.Blocks.BlockSet, editReportAction)
	} else {
		r.Blocks.BlockSet = removeBlockAfter(r.Blocks.BlockSet, constants.BlockIDChooseAlert+rid)
		r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDChooseAlert+rid, additionalInfoSection)

		r.Blocks.BlockSet = append(r.Blocks.BlockSet[:len(r.Blocks.BlockSet)-1], editReportAction)
	}

	return r
}

// ShowManageAlertDeleteControls shows modal view after update/delete alert
func ShowManageAlertDeleteControls(v *slack.View, rs domain.GroupedReports) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	r.Blocks.BlockSet = []slack.Block{}

	alertsText := slackcomponents.GetSlackMarkdownTextBlock(constants.HeaderChooseAlert)
	alertsSection := slack.NewSectionBlock(alertsText, nil, nil)

	alertPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
	alertSelect := buildReportsSelect(rs, alertPlaceholder, constants.ActionIDScheduledReportForAlerts)
	var reportInput *slack.ActionBlock
	item := v.State.Values[constants.BlockIDScheduledReportForAlerts][constants.ActionIDScheduledReportForAlerts].SelectedOption.Value
	if item == "" {
		reportInput = slack.NewActionBlock(constants.BlockIDScheduledReportForAlerts, alertSelect)
	} else {
		reportInput = slack.NewActionBlock(constants.BlockIDScheduledReportForAlerts+v.ID, alertSelect)
	}

	r.Blocks.BlockSet = append(r.Blocks.BlockSet, alertsSection, reportInput)
	return r
}

func buildAlertsSelect(fs []domain.Alert, placeholder *slack.TextBlockObject, actionID string) *slack.SelectBlockElement {
	os := []*slack.OptionBlockObject(nil)
	for _, f := range fs {
		t := slackcomponents.GetSlackPlainTextBlock(fmt.Sprintf("%v (%v)", f.VisualName, f.Status))
		desc := slackcomponents.GetSlackPlainTextBlock(string(f.NotificationFrequency))
		o := slack.NewOptionBlockObject(strconv.FormatInt(f.ID, 10), t, desc)
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
