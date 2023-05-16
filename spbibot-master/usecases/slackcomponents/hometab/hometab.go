package hometab

import (
	"github.com/slack-go/slack"

)

const (
	appNewsDescription = ":warning: *" + constants.NoteLabel + ":*" +
		"\nWe are planing to change out payments model. <https://manage-bi/payments-changes|Read for more details>."
	appDescription = "*" + constants.AppName + "*" +
		"\nThis application allows you to share your Power BI reports with your teammates."
	signInText                 = "Please sign in to get full access to the application's features."
	signInButtonText           = "Sign in to Power BI account"
	addAppToChannelDescription = "Please add this application to channels where you want to use."
)

// GetHomeTabViewRequest returns hometab view request
func GetHomeTabViewRequest(user domain.User, authURL string, c *config.FeatureTogglesConfig) slack.HomeTabViewRequest {
	var blocks slack.Blocks
	appDescriptionTextBlock := slackcomponents.GetSlackMarkdownTextBlock(appDescription)
	appDescriptionSection := slack.NewSectionBlock(appDescriptionTextBlock, nil, nil)

	addAppToChannelDescriptionTExtBlock := slackcomponents.GetSlackPlainTextBlock(addAppToChannelDescription)
	addAppToChannelDescriptionSection := slack.NewSectionBlock(addAppToChannelDescriptionTExtBlock, nil, nil)

	if user.AccessToken != "" {
		shareReportButtons := []*slack.ButtonBlockElement{
			slackcomponents.NewSlackButtonElement(constants.CallbackIDShareReport, "📈 "+constants.TitleShareReport, ""),
			slackcomponents.NewSlackButtonElement(constants.CallbackIDManageFilters, "⚙ "+constants.TitleManageFilters, ""),
		}
		shareReportButtonsBlock := slackcomponents.GetSlackButtonBlock(shareReportButtons)

		alertButtons := []*slack.ButtonBlockElement{
			slackcomponents.NewSlackButtonElement(constants.CallbackIDSaveAlert, "🔔 "+constants.CreateAlertLabel, ""),
			slackcomponents.NewSlackButtonElement(constants.CallbackIDManageAlerts, "⚙ "+constants.TitleManageAlerts, ""),
		}

		alertButtonsBlock := slackcomponents.GetSlackButtonBlock(alertButtons)

		var scheduleReportsButtons []*slack.ButtonBlockElement
		if c.ReportScheduling {
			scheduleReportsButtons = append(scheduleReportsButtons, slackcomponents.NewSlackButtonElement(constants.CallbackIDScheduleReport, "⏰ "+constants.TitleScheduleReport, ""))
		}
		scheduleReportsButtons = append(scheduleReportsButtons, slackcomponents.NewSlackButtonElement(constants.CallbackIDManageScheduledReports, "⚙ "+constants.TitleManageScheduledReports, ""))

		scheduleReportsButtonsBlock := slackcomponents.GetSlackButtonBlock(scheduleReportsButtons)

		accountButtons := []*slack.ButtonBlockElement{
			slackcomponents.NewSlackButtonElement(constants.HomeSignOutID, "🛑 "+constants.DisconnectLabel, ""),
		}
		accountButtonsBlock := slackcomponents.GetSlackButtonBlock(accountButtons)

		appNewsDescriptionTextBlock := slackcomponents.GetSlackMarkdownTextBlock(appNewsDescription)
		appNewsDescriptionSection := slack.NewSectionBlock(appNewsDescriptionTextBlock, nil, nil)

		blockSet := []slack.Block{
			appDescriptionSection,
			addAppToChannelDescriptionSection,
			shareReportButtonsBlock,
			slack.NewDividerBlock(),
			scheduleReportsButtonsBlock,
			slack.NewDividerBlock(),
			alertButtonsBlock,
			slack.NewDividerBlock(),
			accountButtonsBlock,
		}
		if c.PaymentIntroduction {
			blockSet = append(blockSet, appNewsDescriptionSection)
		}

		blocks = slack.Blocks{
			BlockSet: blockSet,
		}
	} else {
		signInButton := []*slack.ButtonBlockElement{
			slackcomponents.NewSlackButtonElement(constants.ConnectActionID, "🚪 "+signInButtonText, authURL),
		}
		buttonBlock := slackcomponents.GetSlackButtonBlock(signInButton)

		signInTextBlock := slackcomponents.GetSlackMarkdownTextBlock(signInText)
		signInSection := slack.NewSectionBlock(signInTextBlock, nil, nil)

		blocks = slack.Blocks{
			BlockSet: []slack.Block{
				appDescriptionSection,
				signInSection,
				buttonBlock,
				addAppToChannelDescriptionSection,
			},
		}
	}

	return slack.HomeTabViewRequest{
		Type:   slack.VTHomeTab,
		Blocks: blocks,
	}
}
