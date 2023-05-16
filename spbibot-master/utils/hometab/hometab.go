package hometab

import (
	"github.com/slack-go/slack"


)

// PublishHomeTab build and publish hometab view
func PublishHomeTab(user domain.User, botAccessToken, authURL string, c *config.FeatureTogglesConfig) (err error) {
	api := slack.New(botAccessToken)
	view := hometab.GetHomeTabViewRequest(user, authURL, c)

	if _, e := api.PublishView(user.ID, view, ""); e != nil {
		return domain.ErrUpdatingView(e)
	}

	return
}
