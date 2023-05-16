package modals

import (
	"strconv"

	"github.com/slack-go/slack"


)

var (
	conditionCollection      = []string{"above", "below", "equal"}
	notifFrequencyCollection = []string{"Once a day", "Once an hour"}
)

// InitialAlertModal is modal for setting up an alert
type InitialAlertModal struct {
	SelectReportModal *SelectReportModal
	BaseModal         *BaseModal
}

// AlertModal is modal for setting up an alert
type AlertModal struct {
	Visuals   []string
	Submit    *slack.TextBlockObject
	BaseModal *BaseModal
}

// NewInitialAlertModal creates the new Alert modal with report selection
func NewInitialAlertModal(title, close, submit string, rs domain.GroupedReports, channelID string) *InitialAlertModal {
	m := NewSelectReportModal(title, close, submit, rs, channelID, rs)
	return &InitialAlertModal{
		SelectReportModal: m,
		BaseModal:         m.BaseModal,
	}
}

// NewAlertModal creates the new Alert modal with visual selection
func NewAlertModal(title, close, submit string, visuals []string) *AlertModal {
	return &AlertModal{
		Visuals:   visuals,
		BaseModal: NewBaseModal(title, close),
		Submit:    slackcomponents.GetSlackPlainTextBlock(submit),
	}
}

// GetViewRequest generates a modal with 1 selection for report
func (a *InitialAlertModal) GetViewRequest() slack.ModalViewRequest {
	_, reportInput, channelInput := a.SelectReportModal.GetSelectionDialog()

	blocks := slack.Blocks{
		BlockSet: []slack.Block{
			reportInput,
			channelInput,
		},
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      a.BaseModal.Title,
		Blocks:     blocks,
		Close:      a.BaseModal.CloseText,
		Submit:     a.SelectReportModal.Submit,
		CallbackID: constants.CallbackIDSaveAlertSelectReport,
	}
}

// GetViewRequest generates a modal with all inputs for alert creation except report selection
func (a *AlertModal) GetViewRequest(v *slack.View) (slack.ModalViewRequest, error) {
	r, err := withDisabledSelectionControls(v)
	if err != nil {
		return slack.ModalViewRequest{}, err
	}

	// a modal with visual names selection, channel selection, threshold, condition and frequency selection
	visualPlaceholder := slackcomponents.GetSlackPlainTextBlock("Visual Title")
	visualSelect := buildVisualsSelect(a.Visuals, visualPlaceholder, constants.ActionIDVisual)
	visualInput := slack.NewInputBlock(constants.BlockIDVisual, visualPlaceholder, visualSelect)

	conditionText := slackcomponents.GetSlackPlainTextBlock("Condition")
	conditionSelect := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, conditionText, "condition", buildOptions(conditionCollection)...)
	conditionInput := slack.NewInputBlock("Condition", conditionText, conditionSelect)

	thresholdText := slackcomponents.GetSlackPlainTextBlock("Threshold")
	thresholdField := slack.NewPlainTextInputBlockElement(thresholdText, "threshold")
	thresholdInput := slack.NewInputBlock("Threshold", thresholdText, thresholdField)

	notificationFrequencyText := slackcomponents.GetSlackPlainTextBlock("Notification Frequency")
	notificationFrequencySelect := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, notificationFrequencyText, "notificationFrequency", buildOptions(notifFrequencyCollection)...)
	notificationFrequencyInput := slack.NewInputBlock("NotificationFrequency", notificationFrequencyText, notificationFrequencySelect)

	blocksSets := []slack.Block{
		visualInput,
		conditionInput,
		thresholdInput,
		notificationFrequencyInput,
	}

	r.Blocks.BlockSet = append(r.Blocks.BlockSet, blocksSets...)
	r.CallbackID = constants.CallbackIDSaveAlert

	return *r, nil
}

func buildOptions(collection []string) []*slack.OptionBlockObject {
	var co []*slack.OptionBlockObject
	for i, con := range collection {
		t := slackcomponents.GetSlackPlainTextBlock(con)
		o := slack.NewOptionBlockObject(strconv.Itoa(i), t, nil)
		co = append(co, o)
	}

	return co
}

func buildVisualsSelect(vs []string, placeholder *slack.TextBlockObject, actionID string) *slack.SelectBlockElement {
	os := buildVisualsOptions(vs)
	e := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		placeholder,
		actionID,
		os...,
	)

	return e
}

func buildVisualsOptions(vs []string) []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject{}
	for i, v := range vs {
		t := slackcomponents.GetSlackPlainTextBlock(v)
		o := slack.NewOptionBlockObject(strconv.Itoa(i), t, nil)
		os = append(os, o)
	}

	return os
}
