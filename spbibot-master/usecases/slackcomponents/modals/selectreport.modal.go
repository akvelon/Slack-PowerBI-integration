package modals

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/slack-go/slack"


)

const (
	conversationTypePublic  = "public"
	conversationTypePrivate = "private"
)

// SelectReportModal defines slack modal for report selection
type SelectReportModal struct {
	BaseModal              *BaseModal
	Submit                 *slack.TextBlockObject
	GroupedReports         domain.GroupedReports
	ChannelID              string
	GroupedReportsFiltered domain.GroupedReports
	PowerBIWorkspaces      []*domain.Group
}

// NewSelectReportModal returns new SelectReportModal
func NewSelectReportModal(title, close, submit string, rs domain.GroupedReports, channelID string, grf domain.GroupedReports) *SelectReportModal {
	var PowerBIWorkspaces []*domain.Group
	for gr := range rs {
		PowerBIWorkspaces = append(PowerBIWorkspaces, gr)
	}
	return &SelectReportModal{
		BaseModal:              NewBaseModal(title, close),
		Submit:                 slackcomponents.GetSlackPlainTextBlock(submit),
		GroupedReports:         rs,
		ChannelID:              channelID,
		GroupedReportsFiltered: grf,
		PowerBIWorkspaces:      PowerBIWorkspaces,
	}
}

// GetViewRequest method is ISlackModal interface method realization
func (m *SelectReportModal) GetViewRequest() slack.ModalViewRequest {
	headerSection, reportInput, channelInput := m.GetSelectionDialog()

	var blockSet []slack.Block
	var callbackID string
	var workspacesInput *slack.ActionBlock

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
	if m.BaseModal.Title.Text == constants.TitleManageFilters {
		chooseReportHeaderText := slackcomponents.GetSlackMarkdownTextBlock("Please choose a workspace and a report to manage filters:")
		chooseReportHeaderSection := slack.NewSectionBlock(chooseReportHeaderText, nil, nil)

		if len(m.PowerBIWorkspaces) > constants.PBIWorkspacesQuantityReducer {
			blockSet = append(blockSet, chooseReportHeaderSection, searchInputBlock, findWorkspaceAction, notAllWorkspacesPresentSection, workspacesInput)
		} else if len(m.PowerBIWorkspaces) > 1 && m.getReportsQuantity() > constants.ReportQuantityReducer {
			blockSet = append(blockSet, chooseReportHeaderSection, workspacesInput)
		} else {
			reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
			reportSelect := buildReportsSelect(m.GroupedReportsFiltered, reportPlaceholder, constants.ActionIDReport)
			reportInput := slack.NewInputBlock(constants.BlockIDReport, reportPlaceholder, reportSelect)

			blockSet = append(blockSet, chooseReportHeaderSection, reportInput)
		}
		callbackID = constants.CallbackIDManageFilters
	} else {
		divider := slack.NewDividerBlock()

		applyFilterLabel := slackcomponents.GetSlackPlainTextBlock(constants.LabelApplyFilter)
		applyFilterOption := slack.NewOptionBlockObject(constants.ValueApplyFilter, applyFilterLabel, nil)
		applyFilterCheckbox := slack.NewCheckboxGroupsBlockElement(constants.ActionIDApplyFilter, applyFilterOption)
		applyFilterInput := slack.NewInputBlock(constants.BlockIDApplyFilter, applyFilterLabel, applyFilterCheckbox)
		applyFilterInput.Optional = true

		if len(m.PowerBIWorkspaces) > constants.PBIWorkspacesQuantityReducer {
			blockSet = append(blockSet, headerSection, searchInputBlock, findWorkspaceAction, notAllWorkspacesPresentSection, workspacesInput, channelInput, divider, applyFilterInput)
		} else if len(m.PowerBIWorkspaces) > 1 && m.getReportsQuantity() > constants.ReportQuantityReducer {
			blockSet = append(blockSet, headerSection, workspacesInput, channelInput, divider, applyFilterInput)
		} else {
			blockSet = append(blockSet, headerSection, reportInput, channelInput, divider, applyFilterInput)
		}
		callbackID = constants.CallbackIDShareReportSelectReport
	}

	blocks := slack.Blocks{
		BlockSet: blockSet,
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      m.BaseModal.Title,
		Blocks:     blocks,
		Close:      m.BaseModal.CloseText,
		Submit:     m.Submit,
		CallbackID: callbackID,
	}
}

// GetSelectionDialog returns base blocks in select report modal
func (m *SelectReportModal) GetSelectionDialog() (header *slack.SectionBlock /*, report *slack.InputBlock,*/, report *slack.ActionBlock, channel *slack.InputBlock) {
	headerText := slackcomponents.GetSlackPlainTextBlock(constants.HeaderChooseReport)
	headerSection := slack.NewSectionBlock(headerText, nil, nil, slack.SectionBlockOptionBlockID(constants.BlockIDReportHeader))

	// TODO: Wait for `https://github.com/slack-go/slack/pull/835' to be merged.
	//reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
	//reportSelect := buildReportsSelect(m.GroupedReports, reportPlaceholder, constants.ActionIDReport)
	//reportInput := slack.NewInputBlock(constants.BlockIDReport, reportPlaceholder, reportSelect)
	//reportInput.DispatchAction = true

	reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
	reportSelect := buildReportsSelect(m.GroupedReportsFiltered, reportPlaceholder, constants.ActionIDReport)
	reportAction := slack.NewActionBlock(constants.BlockIDReport, reportSelect)

	channelPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderChannel)
	channelSelect := slack.NewOptionsSelectBlockElement(slack.OptTypeConversations, channelPlaceholder, constants.ActionIDChannel)
	channelSelect.DefaultToCurrentConversation = true
	channelSelect.InitialConversation = m.ChannelID
	channelSelect.Filter = &slack.SelectBlockElementFilter{
		Include:                       []string{conversationTypePublic, conversationTypePrivate},
		ExcludeExternalSharedChannels: true,
		ExcludeBotUsers:               true,
	}
	channelInput := slack.NewInputBlock(constants.BlockIDChannel, channelPlaceholder, channelSelect)

	return headerSection /*, reportInput*/, reportAction, channelInput
}

// ShareReportInput holds user-entered values for various states of the "share a report" modal.
type ShareReportInput struct {
	ReportSelection *ReportSelectionInput
	ReuseFilter     *ReuseFilterInput
	EditFilter      *EditInFilterInput
	SaveFilter      *SaveInFilterInput
}

// NewShareReportInput builds a ShareReportInput from a slack.View.
func NewShareReportInput(v *slack.View) (*ShareReportInput, error) {
	r, err := NewReportSelectionInput(v)
	if err != nil {
		return nil, err
	}

	return &ShareReportInput{
		ReportSelection: r,
		ReuseFilter:     newReuseFilterInput(v),
		EditFilter:      newEditInFilterInput(v),
		SaveFilter:      newSaveInFilterInput(v),
	}, nil
}

// PageInput represents a page item.
type PageInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReportSelectionInput holds user-entered values of the "choose a report" modal.
type ReportSelectionInput struct {
	ChannelID   string       `json:"channelID"`
	ReportID    string       `json:"reportID"`
	ReportName  string       `json:"reportName"`
	ApplyFilter bool         `json:"applyFilter"`
	Pages       []*PageInput `json:"pages"`
}

// NewReportSelectionInput builds a ReportSelectionInput from a slack.View.
func NewReportSelectionInput(v *slack.View) (*ReportSelectionInput, error) {
	if strings.HasPrefix(v.CallbackID, constants.CallbackIDShareReport) {
		if v.CallbackID == constants.CallbackIDShareReportSelectReport || v.CallbackID == constants.CallbackIDManageFilters {
			return newReportSelectionInputFromState(v.State, v.Blocks.BlockSet), nil
		}

		return newReportSelectionInputFromPrivateMetadata(v.PrivateMetadata)
	}

	if strings.HasPrefix(v.CallbackID, constants.CallbackIDSaveAlert) {
		if v.CallbackID == constants.CallbackIDSaveAlertSelectReport {
			return newReportSelectionInputFromState(v.State, v.Blocks.BlockSet), nil
		}

		return newReportSelectionInputFromPrivateMetadata(v.PrivateMetadata)
	}

	if strings.HasPrefix(v.CallbackID, constants.CallbackIDScheduleReport) {
		return newReportSelectionInputFromState(v.State, v.Blocks.BlockSet), nil
	}

	return nil, fmt.Errorf("unknown callback id")
}

func newReportSelectionInputFromPrivateMetadata(m string) (*ReportSelectionInput, error) {
	i := ReportSelectionInput{}
	err := json.Unmarshal([]byte(m), &i)

	return &i, err
}

func newReportSelectionInputFromState(s *slack.ViewState, bs []slack.Block) *ReportSelectionInput {
	i := ReportSelectionInput{}

	reportsBlock := FindBlock(bs, constants.BlockIDReport)
	reportOption := s.Values[reportsBlock][constants.ActionIDReport].SelectedOption
	if reportOption.Text != nil {
		i.ReportName = reportOption.Text.Text
	}

	i.ReportID = reportOption.Value

	i.ChannelID = s.Values[constants.BlockIDChannel][constants.ActionIDChannel].SelectedConversation

	applyFilterOptions := s.Values[constants.BlockIDApplyFilter][constants.ActionIDApplyFilter].SelectedOptions
	i.ApplyFilter = len(applyFilterOptions) == 1 && applyFilterOptions[0].Value == constants.ValueApplyFilter

	pages := []*PageInput(nil)
	pagesBlock := findBlockState(s, constants.BlockIDPages)
	if pagesBlock != nil {
		pagesOptions := pagesBlock[constants.ActionIDPages].SelectedOptions
		for _, o := range pagesOptions {
			p := PageInput{
				ID:   o.Value,
				Name: o.Text.Text,
			}
			pages = append(pages, &p)
		}

		i.Pages = pages
	}

	return &i
}

func findBlockState(s *slack.ViewState, idPrefix string) map[string]slack.BlockAction {
	for k, v := range s.Values {
		if strings.HasPrefix(k, idPrefix) {
			return v
		}
	}

	return nil
}

// ReuseFilterInput holds user-entered values of the "use a saved filter" modal.
type ReuseFilterInput struct {
	FilterID   string
	FilterName string
}

func newReuseFilterInput(v *slack.View) *ReuseFilterInput {
	if v.CallbackID == constants.CallbackIDShareReportReuseFilter {
		return &ReuseFilterInput{
			FilterID:   v.State.Values[constants.BlockIDFilter][constants.ActionIDFilter].SelectedOption.Value,
			FilterName: v.State.Values[constants.BlockIDFilter][constants.ActionIDFilter].SelectedOption.Text.Text,
		}
	}

	return nil
}

// EditInFilterInput holds user-entered values of the "edit a filter" modal.
type EditInFilterInput struct {
	Column                  string
	Table                   string
	Value                   string
	ConditionOperator       string
	LogicalOperator         string
	SecondConditionOperator string
	SecondValue             string
}

func newEditInFilterInput(v *slack.View) *EditInFilterInput {
	if v.CallbackID == constants.CallbackIDShareReportEditFilter || v.CallbackID == constants.CallbackIDShareReportSaveFilter || v.CallbackID == constants.CallbackIDCreateFilter {
		logicalOperation := v.State.Values[constants.BlockIDLogicalOperator][constants.ActionIDLogicalOperator].SelectedOption.Value
		var conditionOperator string
		if logicalOperation == "" {
			conditionOperator = "Is"
		} else {
			conditionOperator = v.State.Values[constants.BlockIDConditionOperator][constants.ActionIDConditionOperator].SelectedOption.Value
		}

		return &EditInFilterInput{
			Column:                  v.State.Values[constants.BlockIDColumn][constants.ActionIDColumn].Value,
			Table:                   v.State.Values[constants.BlockIDTable][constants.ActionIDTable].Value,
			Value:                   v.State.Values[constants.BlockIDValue][constants.ActionIDValue].Value,
			ConditionOperator:       conditionOperator,
			LogicalOperator:         logicalOperation,
			SecondValue:             v.State.Values[constants.BlockIDSecondValue][constants.ActionIDSecondValue].Value,
			SecondConditionOperator: v.State.Values[constants.BlockIDSecondConditionOperator][constants.ActionIDSecondConditionOperator].SelectedOption.Value,
		}
	}

	return nil
}

// SaveInFilterInput holds user-entered values of the "save a filter" modal.
type SaveInFilterInput struct {
	EditInFilterInput
	Name string
}

func newSaveInFilterInput(v *slack.View) *SaveInFilterInput {
	if v.CallbackID == constants.CallbackIDShareReportSaveFilter || v.CallbackID == constants.CallbackIDCreateFilter {
		return &SaveInFilterInput{
			EditInFilterInput: *newEditInFilterInput(v),
			Name:              v.State.Values[constants.BlockIDName][constants.ActionIDName].Value,
		}
	}

	return nil
}

func withDisabledSelectionControls(v *slack.View) (*slack.ModalViewRequest, error) {
	r := CopyModalRequest(v)

	if r.CallbackID == constants.CallbackIDShareReportSelectReport || r.CallbackID == constants.CallbackIDSaveAlertSelectReport {
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDReportHeader)
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDApplyFilter)

		report := v.State.Values[constants.BlockIDReport][constants.ActionIDReport].SelectedOption.Text.Text
		reportText := slackcomponents.GetSlackMarkdownTextBlock(fmt.Sprintf("*%v*: %v", constants.PlaceholderReport, report))
		reportSection := slack.NewSectionBlock(reportText, nil, nil)
		r.Blocks.BlockSet, _ = replaceBlock(r.Blocks.BlockSet, constants.BlockIDReport, reportSection)

		channel := v.State.Values[constants.BlockIDChannel][constants.ActionIDChannel].SelectedConversation
		channelText := slackcomponents.GetSlackMarkdownTextBlock(fmt.Sprintf("*%v*: <#%v>", constants.PlaceholderChannel, channel))
		channelSection := slack.NewSectionBlock(channelText, nil, nil, slack.SectionBlockOptionBlockID(constants.BlockIDChannel))
		r.Blocks.BlockSet, _ = replaceBlock(r.Blocks.BlockSet, constants.BlockIDChannel, channelSection)

		pages := []string(nil)
		pagesBlock := findBlockState(v.State, constants.BlockIDPages)
		if pagesBlock != nil {
			pagesOptions := pagesBlock[constants.ActionIDPages].SelectedOptions
			for _, o := range pagesOptions {
				pages = append(pages, o.Text.Text)
			}
		}

		pagesText := slackcomponents.GetSlackMarkdownTextBlock(fmt.Sprintf("*%v*: %v", constants.PlaceholderPages, strings.Join(pages, ", ")))
		pagesSection := slack.NewSectionBlock(pagesText, nil, nil)
		pagesBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDPages)
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, pagesBlockID)
		r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDChannel, pagesSection)

		state, err := NewReportSelectionInput(v)
		if err != nil {
			return nil, err
		}

		stateJSON, err := json.Marshal(state)
		if err != nil {
			return nil, err
		}

		r.PrivateMetadata = string(stateJSON)
	}

	return r, nil
}

// ShowReuseFilterControls displays controls for choosing a saved filter.
func ShowReuseFilterControls(v *slack.View, f []*domain.Filter) (*slack.ModalViewRequest, error) {
	r, err := withDisabledSelectionControls(v)
	if err != nil {
		return nil, err
	}

	if r.CallbackID == constants.CallbackIDShareReportSaveFilter || r.CallbackID == constants.CallbackIDShareReportEditFilter {
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDFilterHeader)
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDTable)
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDColumn)
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDValue)
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDAddSecondFilter)
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDSaveFilter)
	}

	if r.CallbackID == constants.CallbackIDShareReportSaveFilter {
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDName)
	}

	if r.CallbackID == constants.CallbackIDShareReportSelectReport {
		reuseFilterLabel := slackcomponents.GetSlackPlainTextBlock("Previously used")
		reuseFilterOption := slack.NewOptionBlockObject(constants.ValueReuseFilter, reuseFilterLabel, nil)
		reuseFilterCheckbox := slack.NewCheckboxGroupsBlockElement(constants.ActionIDReuseFilter, reuseFilterOption)
		reuseFilterCheckbox.InitialOptions = []*slack.OptionBlockObject{reuseFilterOption}
		reuseFilterAction := slack.NewActionBlock(constants.BlockIDReuseFilter, reuseFilterCheckbox)
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, reuseFilterAction)
	}

	if len(f) != 0 {
		filterPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderFilter)
		filterSelect := buildFiltersSelect(f, filterPlaceholder, constants.ActionIDFilter)
		filterInput := slack.NewInputBlock(constants.BlockIDFilter, filterPlaceholder, filterSelect)
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, filterInput)
	}

	r.Title = slackcomponents.GetSlackPlainTextBlock(constants.TitleChooseFilter)

	r.CallbackID = constants.CallbackIDShareReportReuseFilter

	return r, nil
}

// ShowEditFilterControls shows controls allowing to edit a filter.
func ShowEditFilterControls(v *slack.View) (*slack.ModalViewRequest, error) {
	//fmt.Println("TITLE", v.Title.Text)
	r, err := withDisabledSelectionControls(v)
	if err != nil {
		return nil, err
	}

	if v.CallbackID == constants.CallbackIDShareReportReuseFilter {
		r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDFilter)
	}

	filterText := slackcomponents.GetSlackPlainTextBlock(constants.HeaderComposeFilter)
	filterSection := slack.NewSectionBlock(filterText, nil, nil, slack.SectionBlockOptionBlockID(constants.BlockIDFilterHeader))

	tableText := slackcomponents.GetSlackPlainTextBlock("Table name")
	tableField := slack.NewPlainTextInputBlockElement(tableText, constants.ActionIDTable)
	tableInput := slack.NewInputBlock(constants.BlockIDTable, tableText, tableField)

	columnText := slackcomponents.GetSlackPlainTextBlock("Column name")
	columnField := slack.NewPlainTextInputBlockElement(columnText, constants.ActionIDColumn)
	columnInput := slack.NewInputBlock(constants.BlockIDColumn, columnText, columnField)

	valueText := slackcomponents.GetSlackPlainTextBlock("Value")
	valueField := slack.NewPlainTextInputBlockElement(valueText, constants.ActionIDValue)
	valueInput := slack.NewInputBlock(constants.BlockIDValue, valueText, valueField)

	addFilterText := slackcomponents.GetSlackPlainTextBlock("Add advanced filter")
	addFilterField := slack.NewButtonBlockElement(constants.ActionIDAddSecondFilter, constants.ValueAddSecondFilter, addFilterText)
	addFilterAction := slack.NewActionBlock(constants.BlockIDAddSecondFilter, addFilterField)

	saveFilterLabel := slackcomponents.GetSlackPlainTextBlock("Save filter")
	saveFilterOption := slack.NewOptionBlockObject(constants.ValueSaveFilter, saveFilterLabel, nil)
	saveFilterCheckbox := slack.NewCheckboxGroupsBlockElement(constants.ActionIDSaveFilter, saveFilterOption)
	saveFilterAction := slack.NewActionBlock(constants.BlockIDSaveFilter, saveFilterCheckbox)

	r.Blocks.BlockSet = append(r.Blocks.BlockSet, filterSection, tableInput, columnInput, valueInput, addFilterAction, saveFilterAction)

	r.CallbackID = constants.CallbackIDShareReportEditFilter

	r.Title = slackcomponents.GetSlackPlainTextBlock(constants.TitleComposeFilter)

	return r, nil
}

// ShowSaveFilterControls shows controls allowing to save a filter.
func ShowSaveFilterControls(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	nameText := slackcomponents.GetSlackPlainTextBlock("Name")
	nameField := slack.NewPlainTextInputBlockElement(nameText, constants.ActionIDName)
	nameField.MaxLength = constants.MaxLenghthOfFilterName
	nameInput := slack.NewInputBlock(constants.BlockIDName, nameText, nameField)
	r.Blocks.BlockSet = append(r.Blocks.BlockSet, nameInput)

	r.CallbackID = constants.CallbackIDShareReportSaveFilter

	return r
}

func getConditionOperators() []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject{}

	containsText := slackcomponents.GetSlackPlainTextBlock("Contains")
	containsObject := slack.NewOptionBlockObject("Contains", containsText, nil)
	doesNotContainText := slackcomponents.GetSlackPlainTextBlock("Does not contain")
	doesNotContainObject := slack.NewOptionBlockObject("DoesNotContain", doesNotContainText, nil)
	startWithText := slackcomponents.GetSlackPlainTextBlock("Starts with")
	startWithObject := slack.NewOptionBlockObject("StartsWith", startWithText, nil)
	doesNotStartWithText := slackcomponents.GetSlackPlainTextBlock("Does not start with")
	doesNotStartWithObject := slack.NewOptionBlockObject("DoesNotStartWith", doesNotStartWithText, nil)
	isText := slackcomponents.GetSlackPlainTextBlock("Is")
	isObject := slack.NewOptionBlockObject("Is", isText, nil)
	isNotText := slackcomponents.GetSlackPlainTextBlock("Is not")
	isNotObject := slack.NewOptionBlockObject("IsNot", isNotText, nil)

	os = append(os, containsObject, doesNotContainObject, startWithObject, doesNotStartWithObject, isObject, isNotObject)
	return os
}

func getLogicalOperators() []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject{}
	orText := slackcomponents.GetSlackPlainTextBlock("Or")
	orObject := slack.NewOptionBlockObject(constants.ValueOperationOr, orText, nil)
	andText := slackcomponents.GetSlackPlainTextBlock("And")
	andObject := slack.NewOptionBlockObject(constants.ValueOperationAnd, andText, nil)
	os = append(os, orObject, andObject)

	return os
}

func getConditionOperatorByValue(value string) *slack.OptionBlockObject {
	for _, elem := range getConditionOperators() {
		if elem.Value == value {
			return elem
		}
	}
	return nil
}

func getLogicalOperatorByValue(value string) *slack.OptionBlockObject {
	for _, elem := range getLogicalOperators() {
		if elem.Value == value {
			return elem
		}
	}
	return nil
}

func FindReportsInChosenPBIWorkspace(v *slack.View, gr domain.GroupedReports, pbiWorkspace string) *domain.GroupedReports {
	r := CopyModalRequest(v)
	foundReports := domain.GroupedReports{}
	for key, val := range gr {
		if key.Name == pbiWorkspace {
			if len(val.Value) > constants.ReportQuantityReducer {
				foundReports[key] = &domain.ReportsContainer{
					Type:     val.Type,
					Value:    val.Value[:constants.ReportQuantityReducer],
					RawValue: val.RawValue,
				}
			} else {
				foundReports[key] = &domain.ReportsContainer{
					Type:     val.Type,
					Value:    val.Value[:len(val.Value)],
					RawValue: val.RawValue,
				}
			}
		}
	}

	switch v.Title.Text {
	case constants.TitleManageFilters:
		reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
		reportSelect := buildReportsSelect(foundReports, reportPlaceholder, constants.ActionIDReport)
		reportInput := slack.NewInputBlock(constants.BlockIDReport, reportPlaceholder, reportSelect)

		r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, reportInput, constants.BlockIDReport)

	case constants.TitleShareReport, constants.TitleScheduleReport, constants.CreateAlertLabel:
		reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
		reportSelect := buildReportsSelect(foundReports, reportPlaceholder, constants.ActionIDReport)
		reportAction := slack.NewActionBlock(constants.BlockIDReport, reportSelect)

		r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, reportAction, constants.BlockIDReport)
		r.CallbackID = constants.CallbackIDShareReportSelectReport
		if v.Title.Text == constants.CreateAlertLabel {
			r.CallbackID = constants.CallbackIDSaveAlertSelectReport
		}
	}

	return &foundReports
}

//FindReportsByInput finds reports by user input
func FindReportsByInput(v *slack.View, gr domain.GroupedReports) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	reportSearchInputBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportInput)
	userInputValue := v.State.Values[reportSearchInputBlockID][constants.ActionIDSearchReportInput].Value
	if userInputValue == "" {
		return r
	}

	workspacesBlock := FindBlock(v.Blocks.BlockSet, constants.BlockIDWorkspacePBI)

	foundReports := domain.GroupedReports{}
	var chosenPBIWorkspace string
	if workspacesBlock != "" {
		chosenPBIWorkspace = v.State.Values[constants.BlockIDWorkspacePBI][constants.ActionIDWorkspacePBI].SelectedOption.Value
		if chosenPBIWorkspace == "" {
			return r
		}
		for k, v := range gr {
			reportContainer := domain.ReportsContainer{}

			if k.Name == chosenPBIWorkspace {
				for i := 0; i < len(v.Value); i++ {
					if strings.Contains(strings.ToLower(v.Value[i].GetName()), strings.ToLower(userInputValue)) {
						reportContainer.Value = append(reportContainer.Value, v.Value[i])
					}
				}
				if len(reportContainer.Value) > 0 {
					foundReports[k] = &domain.ReportsContainer{
						Type:     v.Type,
						Value:    reportContainer.Value,
						RawValue: v.RawValue,
					}
				}
			}
		}
	} else {
		for k, v := range gr {
			reportContainer := domain.ReportsContainer{}
			for i := 0; i < len(v.Value); i++ {
				if strings.Contains(strings.ToLower(v.Value[i].GetName()), strings.ToLower(userInputValue)) {
					reportContainer.Value = append(reportContainer.Value, v.Value[i])
				}
			}
			if len(reportContainer.Value) > 0 {
				foundReports[k] = &domain.ReportsContainer{
					Type:     v.Type,
					Value:    reportContainer.Value,
					RawValue: v.RawValue,
				}
			}
		}
	}

	pagesBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDPages)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, pagesBlockID)
	reportBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDReport)

	switch v.Title.Text {
	case constants.TitleManageFilters:
		reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
		reportSelect := buildReportsSelect(foundReports, reportPlaceholder, constants.ActionIDReport)
		reportInput := slack.NewInputBlock(constants.BlockIDReport+chosenPBIWorkspace, reportPlaceholder, reportSelect)

		r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, reportInput, constants.BlockIDReport)

	case constants.TitleShareReport, constants.TitleScheduleReport, constants.CreateAlertLabel:
		reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
		reportSelect := buildReportsSelect(foundReports, reportPlaceholder, constants.ActionIDReport)
		reportAction := slack.NewActionBlock(constants.BlockIDReport, reportSelect)

		r.Blocks.BlockSet, _ = replaceBlockOrAddAfter(r.Blocks.BlockSet, reportAction, reportBlockID, constants.BlockIDSearchReportButton)
		r.CallbackID = constants.CallbackIDShareReportSelectReport
		if v.Title.Text == constants.CreateAlertLabel {
			r.CallbackID = constants.CallbackIDSaveAlertSelectReport
		}
	}

	return r
}

func FindWorkspaceByInput(v *slack.View, ws []*domain.Group) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	reportSearchInput := FindBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportInput)
	reportSearchButton := FindBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportButton)
	notAllreportsPresentBlock := FindBlock(r.Blocks.BlockSet, constants.LabelNotAllReportsInList)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, reportSearchInput)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, reportSearchButton)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, notAllreportsPresentBlock)

	userInputValue := v.State.Values[constants.BlockIDSearchWorkspaceInput][constants.ActionIDSearchWorkspaceInput].Value
	if userInputValue == "" {
		return r
	}
	var foundWorkspaces []*domain.Group
	for i := range ws {
		if strings.Contains(strings.ToLower(ws[i].Name), strings.ToLower(userInputValue)) {
			foundWorkspaces = append(foundWorkspaces, ws[i])
		}
	}
	workspacesPlaceHolder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderPBIWorkspaces)
	workspacesSelect := buildWorkspacesSelect(foundWorkspaces, workspacesPlaceHolder, constants.ActionIDWorkspacePBI)
	workspacesInput := slack.NewActionBlock(constants.BlockIDWorkspacePBI, workspacesSelect)

	r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, workspacesInput, constants.BlockIDWorkspacePBI)

	return r
}

// ShowAddFilterControls shows controls allowing to fill in second filter.
func ShowAddFilterControls(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDAddSecondFilter)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDAddFilterForManagement)

	conditionOperatorPlaceHolder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderConditionOperator)

	optionSelect := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		slackcomponents.GetSlackPlainTextBlock("Value to select"),
		constants.ActionIDConditionOperator,
		getConditionOperators()...,
	)

	conditionOperatorInput := slack.NewInputBlock(constants.BlockIDConditionOperator, conditionOperatorPlaceHolder, optionSelect)
	optionSecondSelect := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		slackcomponents.GetSlackPlainTextBlock("Value to select"),
		constants.ActionIDSecondConditionOperator,
		getConditionOperators()...,
	)
	selectSecondOptionInput := slack.NewInputBlock(constants.BlockIDSecondConditionOperator, conditionOperatorPlaceHolder, optionSecondSelect)

	operationPlaceHolder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderOperation)

	operationSelect := slack.NewRadioButtonsBlockElement(
		constants.ActionIDLogicalOperator,
		getLogicalOperators()...,
	)
	operationInput := slack.NewInputBlock(constants.BlockIDLogicalOperator, operationPlaceHolder, operationSelect)

	valueText := slackcomponents.GetSlackPlainTextBlock("Value")
	valueField := slack.NewPlainTextInputBlockElement(valueText, constants.ActionIDSecondValue)
	valueInput := slack.NewInputBlock(constants.BlockIDSecondValue, valueText, valueField)

	removeFilterText := slackcomponents.GetSlackPlainTextBlock("Remove advanced filter")
	removeFilterField := slack.NewButtonBlockElement(constants.ActionIDRemoveSecondFilter, constants.ValueRemoveSecondFilter, removeFilterText)
	removeFilterAction := slack.NewActionBlock(constants.BlockIDRemoveSecondFilter, removeFilterField)

	r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDColumn, conditionOperatorInput)
	r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDValue, operationInput, selectSecondOptionInput, valueInput, removeFilterAction)

	r.CallbackID = constants.CallbackIDCreateFilter

	return r
}

// HideAddFilterControls hides controls allowing to fill in second filter.
func HideAddFilterControls(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDRemoveSecondFilter)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDLogicalOperator)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDSecondValue)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDConditionOperator)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDSecondConditionOperator)

	addFilterText := slackcomponents.GetSlackPlainTextBlock("Add advanced filter")
	addFilterField := slack.NewButtonBlockElement(constants.ActionIDAddSecondFilter, constants.ValueAddSecondFilter, addFilterText)
	addFilterAction := slack.NewActionBlock(constants.BlockIDAddSecondFilter, addFilterField)

	r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDValue, addFilterAction)

	r.CallbackID = constants.CallbackIDShareReportEditFilter

	return r
}

// HideSaveFilterControls hides controls allowing to save a filter.
func HideSaveFilterControls(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDName)

	r.CallbackID = constants.CallbackIDShareReportEditFilter

	return r
}

// ShowManageFilterCreateControls displays controls for create filter dialog.
func ShowManageFilterCreateControls(v *slack.View) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	reportBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDReport)
	reportSearchBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportInput)
	notAllReportsPresentLabel := FindBlock(r.Blocks.BlockSet, constants.LabelNotAllReportsInList)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, reportBlockID)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDWorkspacePBI)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, reportSearchBlockID)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, notAllReportsPresentLabel)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportButton)

	tableText := slackcomponents.GetSlackPlainTextBlock("Table name")
	tableField := slack.NewPlainTextInputBlockElement(tableText, constants.ActionIDTable)
	tableInput := slack.NewInputBlock(constants.BlockIDTable, tableText, tableField)

	columnText := slackcomponents.GetSlackPlainTextBlock("Column name")
	columnField := slack.NewPlainTextInputBlockElement(columnText, constants.ActionIDColumn)
	columnInput := slack.NewInputBlock(constants.BlockIDColumn, columnText, columnField)

	valueText := slackcomponents.GetSlackPlainTextBlock("Value")
	valueField := slack.NewPlainTextInputBlockElement(valueText, constants.ActionIDValue)
	valueInput := slack.NewInputBlock(constants.BlockIDValue, valueText, valueField)

	nameText := slackcomponents.GetSlackPlainTextBlock("Filter name")
	nameField := slack.NewPlainTextInputBlockElement(nameText, constants.ActionIDName)
	nameField.MaxLength = constants.MaxLenghthOfFilterName
	nameInput := slack.NewInputBlock(constants.BlockIDName, nameText, nameField)

	addFilterText := slackcomponents.GetSlackPlainTextBlock("Add advanced filter")
	addFilterField := slack.NewButtonBlockElement(constants.ActionIDAddFilterForManagement, constants.ValueAddSecondFilter, addFilterText)
	addFilterAction := slack.NewActionBlock(constants.BlockIDAddFilterForManagement, addFilterField)

	updateFilterText := slackcomponents.GetSlackPlainTextBlock("Update")
	updateFilterButton := slack.NewButtonBlockElement(constants.ActionIDUpdateFilter, constants.ValueUpdateFilter, updateFilterText)

	deleteFilterText := slackcomponents.GetSlackPlainTextBlock("Delete")
	deleteFilterButton := slack.NewButtonBlockElement(constants.ActionIDDeleteFilter, constants.ValueDeleteFilter, deleteFilterText)

	editFilterAction := slack.NewActionBlock(constants.BlockIDEditFilter, updateFilterButton, deleteFilterButton)

	r.Blocks.BlockSet = append(r.Blocks.BlockSet[:len(r.Blocks.BlockSet)-1], editFilterAction, tableInput, columnInput, valueInput, addFilterAction, nameInput)

	r.Title = slackcomponents.GetSlackPlainTextBlock("Create filter")
	r.CallbackID = constants.CallbackIDCreateFilter

	return r
}

// ShowManageFilterUpdateControls displays controls for choosing filter for update.
func ShowManageFilterUpdateControls(v *slack.View, f []*domain.Filter) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = []slack.Block{}
	if len(f) > 0 {
		headerFiltersText := slackcomponents.GetSlackMarkdownTextBlock("Choose a filter:")
		headerSection := slack.NewSectionBlock(headerFiltersText, nil, nil)
		filterPlaceholder := slackcomponents.GetSlackPlainTextBlock("Filter")
		filterSelect := buildFiltersSelect(f, filterPlaceholder, constants.ActionIDFilterToUpdate)
		filterAction := slack.NewActionBlock(constants.BlockIDFilterToUpdate, filterSelect)
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, headerSection, filterAction)
	} else {
		emptyFiltersText := slackcomponents.GetSlackMarkdownTextBlock("*Sorry you don't have any filters*")
		emptySection := slack.NewSectionBlock(emptyFiltersText, nil, nil)
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, emptySection)
	}

	r.Title = slackcomponents.GetSlackPlainTextBlock("Update filter")
	r.CallbackID = constants.CallbackIDUpdateFilter

	return r
}

// ShowManageFilterDeleteControls displays controls for delete filter dialog.
func ShowManageFilterDeleteControls(v *slack.View, f []*domain.Filter) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	r.Blocks.BlockSet = []slack.Block{}
	if len(f) > 0 {
		filterHeading := slackcomponents.GetSlackPlainTextBlock("Choose a filter:")
		filterPlaceholder := slackcomponents.GetSlackPlainTextBlock("Filter")
		filterSelect := buildFiltersSelect(f, filterPlaceholder, constants.ActionIDFilterToDelete)
		filterAction := slack.NewInputBlock(constants.BlockIDFilterToDelete, filterHeading, filterSelect)
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, filterAction)
	} else {
		emptyFiltersText := slackcomponents.GetSlackMarkdownTextBlock("*Sorry you don't have any filters*")
		emptySection := slack.NewSectionBlock(emptyFiltersText, nil, nil)
		r.Blocks.BlockSet = append(r.Blocks.BlockSet, emptySection)
	}

	r.Title = slackcomponents.GetSlackPlainTextBlock("Delete filter")
	r.CallbackID = constants.CallbackIDDeleteFilter

	return r
}

// ShowManageFilterCurrentUpdateControls displays controls for update filter dialog.
func ShowManageFilterCurrentUpdateControls(v *slack.View, filter map[string]string) *slack.ModalViewRequest {
	r := CopyModalRequest(v)

	tableHeading := slackcomponents.GetSlackPlainTextBlock("Table name")
	initialTableText := slack.PlainTextInputBlockElement{
		Type:         slack.METPlainTextInput,
		ActionID:     constants.ActionIDTable,
		InitialValue: filter["Table"],
	}
	tableInput := slack.NewInputBlock(constants.BlockIDTable, tableHeading, initialTableText)

	columnHeading := slackcomponents.GetSlackPlainTextBlock("Column name")
	initialColumnText := slack.PlainTextInputBlockElement{
		Type:         slack.METPlainTextInput,
		ActionID:     constants.ActionIDColumn,
		InitialValue: filter["Column"],
	}
	columnInput := slack.NewInputBlock(constants.BlockIDColumn, columnHeading, initialColumnText)

	valueHeading := slackcomponents.GetSlackPlainTextBlock("Value")
	initialValueText := slack.PlainTextInputBlockElement{
		Type:         slack.METPlainTextInput,
		ActionID:     constants.ActionIDValue,
		InitialValue: filter["Value"],
	}
	valueInput := slack.NewInputBlock(constants.BlockIDValue, valueHeading, initialValueText)

	nameHeading := slackcomponents.GetSlackPlainTextBlock("Filter name")
	if len(filter["Name"]) > constants.MaxLenghthOfFilterName {
		filter["Name"] = filter["Name"][:constants.MaxLenghthOfFilterName]
	}
	initialNameText := slack.PlainTextInputBlockElement{
		Type:         slack.METPlainTextInput,
		ActionID:     constants.ActionIDName,
		InitialValue: filter["Name"],
		MaxLength:    constants.MaxLenghthOfFilterName,
	}
	nameInput := slack.NewInputBlock(constants.BlockIDName, nameHeading, initialNameText)

	r.Blocks.BlockSet = append(r.Blocks.BlockSet[:len(r.Blocks.BlockSet)-2], tableInput, columnInput, valueInput, nameInput)

	if filter["LogicalOperator"] != "" {
		valueHeading := slackcomponents.GetSlackPlainTextBlock("Value")
		initialValueText := slack.PlainTextInputBlockElement{
			Type:         slack.METPlainTextInput,
			ActionID:     constants.ActionIDSecondValue,
			InitialValue: filter["SecondValue"],
		}
		secondValueInput := slack.NewInputBlock(constants.BlockIDSecondValue, valueHeading, initialValueText)

		selectHeading := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderConditionOperator)
		conditionOperators := slack.SelectBlockElement{
			Type:          slack.OptTypeStatic,
			Placeholder:   selectHeading,
			ActionID:      constants.ActionIDConditionOperator,
			Options:       getConditionOperators(),
			InitialOption: getConditionOperatorByValue(filter["ConditionOperator"]),
		}
		selectInput := slack.NewInputBlock(constants.BlockIDConditionOperator, selectHeading, conditionOperators)

		secondConditionOperator := slack.SelectBlockElement{
			Type:          slack.OptTypeStatic,
			Placeholder:   selectHeading,
			ActionID:      constants.ActionIDSecondConditionOperator,
			Options:       getConditionOperators(),
			InitialOption: getConditionOperatorByValue(filter["SecondConditionOperator"]),
		}
		secondSelectInput := slack.NewInputBlock(constants.BlockIDSecondConditionOperator, selectHeading, secondConditionOperator)

		operationPlaceHolder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderOperation)

		logicalOperatiorSelect := slack.RadioButtonsBlockElement{
			Type:          slack.METRadioButtons,
			ActionID:      constants.ActionIDLogicalOperator,
			Options:       getLogicalOperators(),
			InitialOption: getLogicalOperatorByValue(filter["LogicalOperator"]),
		}

		operationInput := slack.NewInputBlock(constants.BlockIDLogicalOperator, operationPlaceHolder, logicalOperatiorSelect)

		r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDColumn, selectInput)
		r.Blocks.BlockSet, _ = addBlockAfter(r.Blocks.BlockSet, constants.BlockIDValue, operationInput, secondSelectInput, secondValueInput)
	}

	title := filter["Name"]
	if len(title) > constants.MaxLenghthOfFilterNameForTitle-1 {
		title = title[:constants.MaxLenghthOfFilterNameForTitle-1] + "…"
	}
	r.Title = slackcomponents.GetSlackPlainTextBlock(title)
	r.CallbackID = constants.CallbackIDUpdateCurrentFilter

	return r
}

// ShowOrUpdateChoosePagesControls shows or updates page selection controls.
func ShowOrUpdateChoosePagesControls(v *slack.View, ps []*domain.Page, stateTag string) (*slack.ModalViewRequest, error) {
	r := CopyModalRequest(v)

	pagesPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderPages)
	pagesSelect := buildPagesSelect(constants.ActionIDPages, pagesPlaceholder, ps)
	// NOTE: We have to mutate block id on every update, otherwise Slack UI won't show updates.
	pagesInput := slack.NewInputBlock(constants.BlockIDPages+stateTag, pagesPlaceholder, pagesSelect)
	i := FindBlock(r.Blocks.BlockSet, constants.BlockIDPages)
	afterBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDReport)

	r.Blocks.BlockSet, _ = replaceBlockOrAddAfter(r.Blocks.BlockSet, pagesInput, i, afterBlockID)

	return r, nil
}

//UpdateChooseReportControls updates report selection controls
func UpdateChooseReportControls(v *slack.View, rs domain.GroupedReports, stateTag string) *slack.ModalViewRequest {
	r := CopyModalRequest(v)
	pagesBlock := FindBlock(r.Blocks.BlockSet, constants.BlockIDPages)
	r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, pagesBlock)

	notAllReportsPresentLabel := slackcomponents.GetSlackMarkdownTextBlock(constants.LabelNotAllReportsInList)
	notAllReportsPresentSection := slack.NewSectionBlock(notAllReportsPresentLabel, nil, nil)
	notAllReportsPresentSection.BlockID = constants.LabelNotAllReportsInList

	searchReportLabel := slackcomponents.GetSlackPlainTextBlock("Find report")
	searchReportInput := slack.NewPlainTextInputBlockElement(searchReportLabel, constants.ActionIDSearchReportInput)
	searchInput := slack.NewInputBlock(constants.BlockIDSearchReportInput+stateTag, searchReportLabel, searchReportInput)
	searchInput.Optional = true

	findReportText := slackcomponents.GetSlackPlainTextBlock("Find Report")
	findReportField := slack.NewButtonBlockElement(constants.ActionIDSearchReport, constants.ValueSearchReport, findReportText)
	findReportAction := slack.NewActionBlock(constants.BlockIDSearchReportButton, findReportField)

	reportSum := 0
	for _, v := range rs {
		reportSum += len(v.Value)
	}

	reportBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDReport)
	reportSearchBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportInput)

	switch v.Title.Text {
	case constants.TitleManageFilters:
		reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
		reportSelect := buildReportsSelect(rs, reportPlaceholder, constants.ActionIDReport)
		reportInput := slack.NewInputBlock(constants.BlockIDReport+stateTag, reportPlaceholder, reportSelect)

		reportInputBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDReport)

		if reportSum >= constants.ReportQuantityReducer {
			r.Blocks.BlockSet, _ = replaceBlockOrAddAfter(r.Blocks.BlockSet, searchInput, reportSearchBlockID, constants.BlockIDWorkspacePBI)
			r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, findReportAction, constants.BlockIDSearchReportInput+stateTag)
			r.Blocks.BlockSet, _ = replaceBlockOrAddAfter(r.Blocks.BlockSet, reportInput, reportInputBlockID, constants.BlockIDSearchReportButton)
			r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, notAllReportsPresentSection, constants.BlockIDSearchReportButton)
		} else {
			searchID := FindBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportInput)
			r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, searchID)
			r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportButton)
			r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.LabelNotAllReportsInList)
			r.Blocks.BlockSet, _ = replaceBlockOrAddAfter(r.Blocks.BlockSet, reportInput, reportBlockID, constants.BlockIDWorkspacePBI)
		}

	case constants.TitleShareReport, constants.TitleScheduleReport, constants.CreateAlertLabel:
		reportPlaceholder := slackcomponents.GetSlackPlainTextBlock(constants.PlaceholderReport)
		reportSelect := buildReportsSelect(rs, reportPlaceholder, constants.ActionIDReport)
		reportAction := slack.NewActionBlock(constants.BlockIDReport+stateTag, reportSelect)

		reportActionBlockID := FindBlock(r.Blocks.BlockSet, constants.BlockIDReport)

		if reportSum >= constants.ReportQuantityReducer {
			r.Blocks.BlockSet, _ = replaceBlockOrAddAfter(r.Blocks.BlockSet, searchInput, reportSearchBlockID, constants.BlockIDWorkspacePBI)
			r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, findReportAction, constants.BlockIDSearchReportInput+stateTag)
			r.Blocks.BlockSet, _ = replaceBlockOrAddAfter(r.Blocks.BlockSet, reportAction, reportActionBlockID, constants.BlockIDSearchReportButton)
			r.Blocks.BlockSet = updateBlockOrAddAfter(r.Blocks.BlockSet, notAllReportsPresentSection, constants.BlockIDSearchReportButton)
		} else {
			searchID := FindBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportInput)

			r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, searchID)
			r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.BlockIDSearchReportButton)
			r.Blocks.BlockSet = RemoveBlock(r.Blocks.BlockSet, constants.LabelNotAllReportsInList)
			r.Blocks.BlockSet, _ = replaceBlockOrAddAfter(r.Blocks.BlockSet, reportAction, reportBlockID, constants.BlockIDWorkspacePBI)
		}
	}

	return r
}

func RemoveBlock(bs []slack.Block, id string) []slack.Block {
	s := []slack.Block(nil)
	for _, b := range bs {
		if blockID(b) != id {
			s = append(s, b)
		}
	}

	return s
}

func replaceBlock(bs []slack.Block, id string, replacement slack.Block) ([]slack.Block, bool) {
	s := []slack.Block(nil)
	done := false
	for _, b := range bs {
		if !done && blockID(b) == id {
			s = append(s, replacement)
			done = true
		} else {
			s = append(s, b)
		}
	}

	return s, done
}

func removeBlockAfter(bs []slack.Block, afterBlockID string) []slack.Block {
	s := []slack.Block(nil)
	for i := 0; i < len(bs); i++ {
		s = append(s, bs[i])
		if blockID(bs[i]) == afterBlockID {
			i++
		}
	}

	return s
}

func addBlockAfter(bs []slack.Block, afterBlockID string, blocks ...slack.Block) ([]slack.Block, bool) {
	s := []slack.Block(nil)
	done := false
	for _, b := range bs {
		if !done && blockID(b) == afterBlockID {
			s = append(s, b)
			s = append(s, blocks...)
			done = true
		} else {
			s = append(s, b)
		}
	}

	return s, done
}

func addBlockBefore(bs []slack.Block, add slack.Block, beforeBlockID string) ([]slack.Block, bool) {
	s := []slack.Block(nil)
	done := false
	for _, b := range bs {
		if !done && blockID(b) == beforeBlockID {
			s = append(s, add, b)
			done = true
		} else {
			s = append(s, b)
		}
	}

	return s, done
}

func replaceBlockOrAddAfter(bs []slack.Block, addOrReplace slack.Block, replaceBlockID, afterBlockID string) ([]slack.Block, bool) {
	s, replaced := replaceBlock(bs, replaceBlockID, addOrReplace)
	if !replaced {
		return addBlockAfter(bs, afterBlockID, addOrReplace)
	}

	return s, replaced
}

func replaceBlockOrAddBefore(bs []slack.Block, addOrReplace slack.Block, replaceBlockID, beforeBlockID string) ([]slack.Block, bool) {
	s, replaced := replaceBlock(bs, replaceBlockID, addOrReplace)
	if !replaced {
		return addBlockBefore(bs, addOrReplace, beforeBlockID)
	}

	return s, replaced
}

func FindBlock(bs []slack.Block, idPrefix string) string {
	for _, b := range bs {
		i := blockID(b)
		if strings.HasPrefix(i, idPrefix) {
			return i
		}
	}

	return ""
}

func blockID(b slack.Block) string {
	switch b.BlockType() {
	case slack.MBTAction:
		return b.(*slack.ActionBlock).BlockID
	case slack.MBTContext:
		return b.(*slack.ContextBlock).BlockID
	case slack.MBTDivider:
		return b.(*slack.DividerBlock).BlockID
	case slack.MBTFile:
		return b.(*slack.FileBlock).BlockID
	case slack.MBTImage:
		return b.(*slack.ImageBlock).BlockID
	case slack.MBTInput:
		return b.(*slack.InputBlock).BlockID
	case slack.MBTSection:
		return b.(*slack.SectionBlock).BlockID
	default:
		return ""
	}
}

func buildWorkspacesSelect(ws []*domain.Group, placeholder *slack.TextBlockObject, actionID string) *slack.SelectBlockElement {
	os := buildWorkspacesList(ws)
	e := slack.NewOptionsGroupSelectBlockElement(
		slack.OptTypeStatic,
		placeholder,
		actionID,
		os...,
	)
	return e
}

func buildReportsSelect(rs domain.GroupedReports, placeholder *slack.TextBlockObject, actionID string) *slack.SelectBlockElement {
	gs := buildReportsGroups(rs)
	e := slack.NewOptionsGroupSelectBlockElement(
		slack.OptTypeStatic,
		placeholder,
		actionID,
		gs...,
	)

	return e
}

func buildReportsGroups(rs domain.GroupedReports) []*slack.OptionGroupBlockObject {
	gs := []*slack.OptionGroupBlockObject(nil)
	for k, v := range rs {
		os := buildReportsOptions(v)
		l := slackcomponents.GetSlackPlainTextBlock(k.Name)
		g := slack.NewOptionGroupBlockElement(l, os...)
		gs = append(gs, g)
	}

	return gs
}

func buildWorkspacesList(ws []*domain.Group) []*slack.OptionGroupBlockObject {
	gs := []*slack.OptionGroupBlockObject(nil)

	os := buildWorkspacesOptions(ws)
	t := slackcomponents.GetSlackPlainTextBlock(constants.LabelPBIWorkspacesList)
	g := slack.NewOptionGroupBlockElement(t, os...)
	gs = append(gs, g)

	return gs
}

func buildWorkspacesOptions(ws []*domain.Group) []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject(nil)
	for _, p := range ws {
		t := slackcomponents.GetSlackPlainTextBlock(p.Name)
		o := slack.NewOptionBlockObject(p.Name, t, nil)
		os = append(os, o)
	}

	return os
}

func buildReportsOptions(rs *domain.ReportsContainer) []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject(nil)
	for _, r := range rs.Value {
		t := slackcomponents.GetSlackPlainTextBlock(r.GetName())
		o := slack.NewOptionBlockObject(r.GetID(), t, nil)
		os = append(os, o)
	}

	return os
}

func buildFiltersSelect(fs []*domain.Filter, placeholder *slack.TextBlockObject, actionID string) *slack.SelectBlockElement {
	os := buildFiltersOptions(fs)
	e := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		placeholder,
		actionID,
		os...,
	)

	return e
}

func buildFiltersOptions(fs []*domain.Filter) []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject(nil)
	for _, f := range fs {
		if len(f.Name) > constants.MaxLenghthOfFilterName {
			f.Name = f.Name[:constants.MaxLenghthOfFilterName]
		}
		t := slackcomponents.GetSlackPlainTextBlock(f.Name)
		o := slack.NewOptionBlockObject(strconv.FormatInt(f.ID, 10), t, nil)
		os = append(os, o)
	}

	return os
}

func buildPagesSelect(actionID string, placeholder *slack.TextBlockObject, ps []*domain.Page) *slack.SelectBlockElement {
	os := buildPagesOptions(ps)
	e := slack.NewOptionsSelectBlockElement(
		slack.MultiOptTypeStatic,
		placeholder,
		actionID,
		os...,
	)

	return e
}

func buildPagesOptions(ps []*domain.Page) []*slack.OptionBlockObject {
	os := []*slack.OptionBlockObject(nil)
	for _, p := range ps {
		t := slackcomponents.GetSlackPlainTextBlock(p.DisplayName)
		o := slack.NewOptionBlockObject(p.Name, t, nil)
		os = append(os, o)
	}

	return os
}

func (m *SelectReportModal) getReportsQuantity() int {
	var sum int
	for _, v := range m.GroupedReports {
		sum += len(v.Value)
	}
	return sum
}
