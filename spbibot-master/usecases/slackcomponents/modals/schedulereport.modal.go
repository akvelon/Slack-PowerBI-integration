package modals

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"

)

// ExecutionPeriodicity denotes task execution schedule.
type ExecutionPeriodicity int

const (
	//ExecutionPeriodicityHourly denotes the hourly execution schedule.
	ExecutionPeriodicityHourly ExecutionPeriodicity = iota
	// ExecutionPeriodicityDaily denotes the daily execution schedule.
	ExecutionPeriodicityDaily
	// ExecutionPeriodicityWeekly denotes the weekly execution schedule.
	ExecutionPeriodicityWeekly
	// ExecutionPeriodicityMonthly denotes the monthly execution schedule.
	ExecutionPeriodicityMonthly
)

func (p ExecutionPeriodicity) String() string {
	return strconv.FormatInt(int64(p), 10)
}

// ParseExecutionPeriodicity creates an ExecutionPeriodicity from its textual representation.
func ParseExecutionPeriodicity(s string) (ExecutionPeriodicity, error) {
	i, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return 0, err
	}

	p := ExecutionPeriodicity(i)
	if p < ExecutionPeriodicityHourly || p > ExecutionPeriodicityMonthly {
		return 0, fmt.Errorf("unsupported periodicity")
	}

	return p, nil
}

// DayOfMonth denotes day of month.
type DayOfMonth int

const (
	dayOfMonthFirst DayOfMonth = 1
	// DayOfMonthLast denotes last day of month.
	DayOfMonthLast DayOfMonth = 32
)

const layoutTime = "15:04"

type scheduleReportModal struct {
	*SelectReportModal
}

// NewScheduleReportModal creates a "schedule a report" modal.
func NewScheduleReportModal(title, close, submit string, rs domain.GroupedReports, channelID string) ISlackModal {
	return &scheduleReportModal{
		SelectReportModal: NewSelectReportModal(title, close, submit, rs, channelID, rs),
	}
}

func (m *scheduleReportModal) GetViewRequest() slack.ModalViewRequest {
	headerSection, reportInput, channelInput := m.GetSelectionDialog()
	var blockSet []slack.Block
	var workspacesInput *slack.ActionBlock
	divider := slack.NewDividerBlock()

	if len(m.PowerBIWorkspaces) > constants.PBIWorkspacesQuantityReducer {
		workspacesPlaceHolder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderPBIWorkspaces)
		workspacesSelect := buildWorkspacesSelect(m.PowerBIWorkspaces[:constants.PBIWorkspacesQuantityReducer], workspacesPlaceHolder, constants.ActionIDWorkspacePBI)
		workspacesInput = slack.NewActionBlock(constants.BlockIDWorkspacePBI, workspacesSelect)
	} else {
		workspacesPlaceHolder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderPBIWorkspaces)
		workspacesSelect := buildWorkspacesSelect(m.PowerBIWorkspaces, workspacesPlaceHolder, constants.ActionIDWorkspacePBI)
		workspacesInput = slack.NewActionBlock(constants.BlockIDWorkspacePBI, workspacesSelect)
	}

	notAllWorkspacesPresentLabel := slackcomponents.GetSlackMarkdownTextBlock(constants.LabelNotAllWorkspacesInList)
	notAllWorkspacesPresentSection := slack.NewSectionBlock(notAllWorkspacesPresentLabel, nil, nil)
	notAllWorkspacesPresentSection.BlockID = constants.LabelNotAllWorkspacesInList

	searchWorkspaceLabel := slackcomponents.GetSlackPlainTextBlock("Find Workspace")
	searchWorkspaceInput := slack.NewPlainTextInputBlockElement(searchWorkspaceLabel, constants.ActionIDSearchWorkspaceInput)
	searchInputBlock := slack.NewInputBlock(constants.BlockIDSearchWorkspaceInput, searchWorkspaceLabel, searchWorkspaceInput)
	searchInputBlock.Optional = true

	findWorkspaceText := slackcomponents.GetSlackPlainTextBlock("Find Workspace")
	findWorkspaceField := slack.NewButtonBlockElement(constants.ActionIDSearchWorkspace, constants.ValueSearchWorkspace, findWorkspaceText)
	findWorkspaceAction := slack.NewActionBlock(constants.BlockIDSearchWorkspaceButton, findWorkspaceField)

	scheduleText := slackcomponents.GetSlackPlainTextBlock(constants.HeaderSetSchedule)
	scheduleSection := slack.NewSectionBlock(scheduleText, nil, nil)

	periodicityPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderPeriodicity)
	periodicitySelect := newPeriodicitySelect(constants.ActionIDPeriodicity, periodicityPlaceholder)
	periodicityPlaceholder2 := slackcomponents.GetSlackMarkdownTextBlock(fmt.Sprintf("*%v*", constants.PlaceholderPeriodicity))
	periodicitySection := slack.NewSectionBlock(periodicityPlaceholder2, nil, &slack.Accessory{SelectElement: periodicitySelect}, slack.SectionBlockOptionBlockID(constants.BlockIDPeriodicity))
	
	
	timePlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderTime)
	timeBlock := newTimeSelect(constants.ActionIDTime, timePlaceholder, 30*time.Minute)
	timeInput := slack.NewInputBlock(constants.BlockIDTime, timePlaceholder, timeBlock)

	if len(m.PowerBIWorkspaces) > constants.PBIWorkspacesQuantityReducer {
		blockSet = append(blockSet, headerSection, searchInputBlock, findWorkspaceAction, notAllWorkspacesPresentSection, workspacesInput, channelInput, divider, scheduleSection, periodicitySection, timeInput)
	} else if len(m.PowerBIWorkspaces) > 1 && m.getReportsQuantity() > constants.ReportQuantityReducer {
		blockSet = append(blockSet, headerSection, workspacesInput, channelInput, divider, scheduleSection, periodicitySection, timeInput)
	} else {
		blockSet = append(blockSet, headerSection, reportInput, channelInput, divider, scheduleSection, periodicitySection, timeInput)
	}

	bs := slack.Blocks{
		BlockSet: blockSet,
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      m.BaseModal.Title,
		Blocks:     bs,
		Close:      m.BaseModal.CloseText,
		Submit:     m.Submit,
		CallbackID: constants.CallbackIDScheduleReport,
	}
}

// ScheduleReportInput holds state of a "schedule a report" modal.
type ScheduleReportInput struct {
	ReportSelection *ReportSelectionInput
	Schedule        *ScheduleInput
}

// NewScheduleReportReportInput builds a ScheduleReportInput from slack.View.
func NewScheduleReportReportInput(v *slack.View) (*ScheduleReportInput, error) {
	s, err := newScheduleInput(v)
	if err != nil {
		return nil, err
	}

	r, err := NewReportSelectionInput(v)
	if err != nil {
		return nil, err
	}

	return &ScheduleReportInput{
		ReportSelection: r,
		Schedule:        s,
	}, nil
}

// ScheduleInput holds state of scheduling controls.
type ScheduleInput struct {
	Periodicity ExecutionPeriodicity
	Time        time.Time
	Weekday     time.Weekday
	DayOfMonth  DayOfMonth
}

func newScheduleInput(v *slack.View) (*ScheduleInput, error) {
	i := ScheduleInput{}

	if strings.HasPrefix(v.CallbackID, constants.CallbackIDScheduleReport) {
		o := v.State.Values[constants.BlockIDTime][constants.ActionIDTime].SelectedOption.Value
		if o == "" {
			if time.Now().UTC().Minute() > 55 {
				o = strconv.Itoa(time.Now().UTC().Hour()+1) + ":55"
			} else {
				o = strconv.Itoa(time.Now().UTC().Hour()) + ":55"
			}
		}
		t, err := time.ParseInLocation(layoutTime, o, time.UTC)
		if err != nil {
			return nil, err
		}

		t = time.Date(1, time.January, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
		i.Time = t

		p, err := ParseExecutionPeriodicity(v.State.Values[constants.BlockIDPeriodicity][constants.ActionIDPeriodicity].SelectedOption.Value)
		if err != nil {
			return nil, err
		}

		i.Periodicity = p
	} else {
		return nil, fmt.Errorf("unknown callback id")
	}

	switch v.CallbackID {
	case constants.CallbackIDScheduleReportWeekly:
		o := v.State.Values[constants.BlockIDWeekday][constants.ActionIDWeekday].SelectedOption.Value
		w, err := strconv.ParseInt(o, 10, 0)
		if err != nil {
			return nil, err
		}

		i.Weekday = time.Weekday(w)

	case constants.CallbackIDScheduleReportMonthly:
		o := v.State.Values[constants.BlockIDDayOfMonth][constants.ActionIDDayOfMonth].SelectedOption.Value
		v, err := strconv.ParseInt(o, 10, 0)
		if err != nil {
			return nil, err
		}

		d := DayOfMonth(v)
		if d < dayOfMonthFirst || d > DayOfMonthLast {
			return nil, fmt.Errorf("invalid day of month")
		}

		i.DayOfMonth = d

	case constants.CallbackIDScheduleReport, constants.CallbackIDScheduleReportHourly:
		break

	default:
		return nil, fmt.Errorf("unknown callback id")
	}

	return &i, nil
}

func newPeriodicitySelect(actionID string, placeholder *slack.TextBlockObject) *slack.SelectBlockElement {
	os := newPeriodicityOptions()
	s := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, placeholder, actionID, os...)

	return s
}

func newPeriodicityOptions() []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject(nil)
	for p := ExecutionPeriodicityHourly; p <= ExecutionPeriodicityMonthly; p++ {
		value, text := formatPeriodicity(p)
		o := slack.NewOptionBlockObject(value, slackcomponents.GetSlackPlainTextBlock(text), nil)
		os = append(os, o)
	}

	return os
}

func formatPeriodicity(p ExecutionPeriodicity) (value string, text string) {
	value = p.String()
	text = constants.LabelPeriodicity[p]

	return
}

func newTimeSelect(actionID string, placeholder *slack.TextBlockObject, granularity time.Duration) *slack.SelectBlockElement {
	os := newTimeOptions(granularity)
	s := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, placeholder, actionID, os...)

	return s
}

func newTimeOptions(granularity time.Duration) []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject(nil)
	t := time.Time{}.UTC()
	day := t.Add(24 * time.Hour)
	for {
		value, text := formatTime24Hour(t)
		o := slack.NewOptionBlockObject(value, slackcomponents.GetSlackPlainTextBlock(text), nil)
		os = append(os, o)

		t = t.Add(granularity)
		if t.Equal(day) || t.After(day) {
			break
		}
	}

	return os
}

func formatTime24Hour(t time.Time) (value string, text string) {
	s := t.Format(layoutTime)
	value = s
	text = s

	return
}

// HideDayInput hides the weekday input & the day of month input.
func HideDayInput(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDWeekday)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDDayOfMonth)

	timePlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderTime)
	timeBlock := newTimeSelect(constants.ActionIDTime, timePlaceholder, 30*time.Minute)
	timeInput := slack.NewInputBlock(constants.BlockIDTime, timePlaceholder, timeBlock)
	timeInput.Optional = false

	timeBlockToUpdate := FindBlock(r.Blocks.BlockSet, constants.BlockIDTime)
	if timeBlockToUpdate != "" {
		r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, timeInput, constants.BlockIDTime)
	} else {
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, timeInput)
	}

	r.CallbackID = constants.CallbackIDScheduleReport

	return r
}

func HideTimeInput(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDTime)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDDayOfMonth)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDWeekday)

	r.CallbackID = constants.CallbackIDScheduleReportHourly

	return r
}

// ShowWeekdayInput shows the weekday input & hides the day of month input.
func ShowWeekdayInput(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDDayOfMonth)

	weekdayPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderWeekday)
	weekdaySelect := newWeekdayPicker(constants.ActionIDWeekday, weekdayPlaceholder)
	weekdayInput := slack.NewInputBlock(constants.BlockIDWeekday, weekdayPlaceholder, weekdaySelect)
	r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDPeriodicity, weekdayInput)

	timePlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderTime)
	timeBlock := newTimeSelect(constants.ActionIDTime, timePlaceholder, 30*time.Minute)
	timeInput := slack.NewInputBlock(constants.BlockIDTime, timePlaceholder, timeBlock)
	timeInput.Optional = false

	timeBlockToUpdate := FindBlock(r.Blocks.BlockSet, constants.BlockIDTime)
	if timeBlockToUpdate != "" {
		r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, timeInput, constants.BlockIDTime)
	} else {
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, timeInput)
	}

	r.CallbackID = constants.CallbackIDScheduleReportWeekly

	return r
}

func newWeekdayPicker(actionID string, placeholder *slack.TextBlockObject) *slack.SelectBlockElement {
	os := newWeekdayOptions()
	s := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, placeholder, actionID, os...)

	return s
}

func newWeekdayOptions() []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject(nil)
	for t := time.Sunday; t <= time.Saturday; t++ {
		value, text := formatWeekday(t)
		o := slack.NewOptionBlockObject(value, slackcomponents.GetSlackPlainTextBlock(text), nil)
		os = append(os, o)
	}

	return os
}

func formatWeekday(d time.Weekday) (value string, text string) {
	value = strconv.FormatInt(int64(d), 10)
	text = d.String()

	return
}

// ShowDayOfMonthInput shows the day of month input & hides the weekday input.
func ShowDayOfMonthInput(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDWeekday)

	timePlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderTime)
	timeBlock := newTimeSelect(constants.ActionIDTime, timePlaceholder, 30*time.Minute)
	timeInput := slack.NewInputBlock(constants.BlockIDTime, timePlaceholder, timeBlock)
	timeInput.Optional = false

	timeBlockToUpdate := FindBlock(r.Blocks.BlockSet, constants.BlockIDTime)
	if timeBlockToUpdate != "" {
		r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, timeInput, constants.BlockIDTime)
	} else {
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, timeInput)
	}

	dayOfMonthPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderDayOfMonth)
	dayOfMonthSelect := newDayOfMonthPicker(constants.ActionIDDayOfMonth, dayOfMonthPlaceholder)
	dayOfMonthInput := slack.NewInputBlock(constants.BlockIDDayOfMonth, dayOfMonthPlaceholder, dayOfMonthSelect)

	r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDPeriodicity, dayOfMonthInput)

	r.CallbackID = constants.CallbackIDScheduleReportMonthly

	return r
}

func newDayOfMonthPicker(actionID string, placeholder *slack.TextBlockObject) *slack.SelectBlockElement {
	os := newDayOfMonthOptions()
	s := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, placeholder, actionID, os...)

	return s
}

func newDayOfMonthOptions() []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject(nil)
	for t := dayOfMonthFirst; t <= DayOfMonthLast; t++ {
		value, text := formatDayOfMonth(t)
		o := slack.NewOptionBlockObject(value, slackcomponents.GetSlackPlainTextBlock(text), nil)
		os = append(os, o)
	}

	return os
}

func formatDayOfMonth(d DayOfMonth) (value string, text string) {
	s := strconv.FormatInt(int64(d), 10)
	value = s
	text = s

	if d == DayOfMonthLast {
		text = constants.LabelDayOfMonthLast
	}

	return
}
