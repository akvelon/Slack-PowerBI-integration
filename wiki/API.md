# Microsoft Flow for send data from PowerBi to Slack
[PowerBi + Slack](https://flow.microsoft.com/en-us/galleries/public/templates/97aa04aea66a4f4d9c0c353294237161/post-a-message-to-a-slack-channel-when-a-power-bi-data-alert-is-triggered/) - Only for Alerts in PowerBI

# OAuth2 for Go
[Github golang/oauth2](https://github.com/golang/oauth2)

# URL for request access

https://login.microsoftonline.com/common/oauth2/authorize?client_id=af4ba4bf-bfee-4fbc-ab06-8fca7570ce2d&response_mode=query&response_type=code&scope=openid%20profile%20https%3A%2F%2Fanalysis.windows.net%2Fpowerbi%2Fapi

* endpoint = https://login.microsoftonline.com/common/oauth2/authorize
* client_id = af4ba4bf-bfee-4fbc-ab06-8fca7570ce2d
* response_mode = query
* response_type = code
* scope = openid, profile, https://analysis.windows.net/powerbi/api


# Simple App from Microsoft for connecting to PowerBi
[Github microsoft/PowerBI-Developer-Samples](https://github.com/microsoft/PowerBI-Developer-Samples/tree/master/NodeJS)
Update config/config.json
1. *"clientId" *-> Open you app on Azure portal. On tab "Overview" copy "Application (client) ID"
2. *"workspaceId"* -> should be guid -> https ://app.powerbi.com/groups/**00000000-0000-0000-0000-000000000000**/reports/00000000-0000-0000-0000-000000000000/ReportSection
3. *"reportId"* -> https ://app.powerbi.com/groups/00000000-0000-0000-0000-000000000000/reports/**00000000-0000-0000-0000-000000000000**/ReportSection
4. *"pbiUsername"* -> your email
5. *"pbiPassword"* -> your password
6. *"clientSecret"* -> Open you app on Azure portal. On tab "Certificates & secrets" copy "Client secrets"
7. *"tenantId"* -> Open you app on Azure portal. On tab "Overview" copy "Directory (tenant) ID"

# [IN PROGRESS] Simple App for API 

1. Add code in main.go file
2. go get golang.org/x/oauth2
3. start -> go run main.go
```
package main

import (
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
	"golang.org/x/oauth2"
)

var (
	slackClient   *slack.Client
	powerBiConfig = &oauth2.Config{
		ClientID:     "<YOUR GUID>",
		ClientSecret: "<YOUR SECRET KEY>",
		// RedirectURL:  "https://localhost",
		Scopes: []string{"https://analysis.windows.net/powerbi/api"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/token",
		},
	}
	randomeState = "random"
)

func handleHome(w http.ResponseWriter, req *http.Request) {
	var html = `<html><body><a href="/login"> Login </a></body></html>`
	fmt.Fprint(w, html)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	url := powerBiConfig.AuthCodeURL(randomeState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("state") != randomeState {
		fmt.Println("state is not valid" + r.FormValue("state"))
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

		return
	}

	token, err := powerBiConfig.Exchange(oauth2.NoContext, r.FormValue("code"))

	if err != nil {
		fmt.Printf("Could not get token: %s\n" + err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

		return
	}

	fmt.Println("state is not valid", token)

	// res. err := http.Get("")
}

func main() {
	url := powerBiConfig.AuthCodeURL("state", oauth2.AccessTypeOnline)
	fmt.Printf("Visit the URL for the auth dialog: %v", url)

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)
	http.ListenAndServe(":8080", nil)
}
```

# Simple app for connecting to your slack bot
Add your SLACK_API_TOKEN
1. Open https://api.slack.com/apps
2. Choose your app
3. Click "Install App"
4. Copy "Bot User OAuth Access Token"
```
package main

import (
	"fmt"
	"os"

	"github.com/slack-go/slack"
)

var (
	slackClient *slack.Client
)

func main() {
	os.Setenv("SLACK_API_TOKEN", "xoxb-00000000000000000000000000000000000000")
	slackClient = slack.New(os.Getenv("SLACK_API_TOKEN"))
	fmt.Printf(os.Getenv("SLACK_API_TOKEN"))
	rtm := slackClient.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			go handleMessage(ev)
		}
	}
}

func handleMessage(ev *slack.MessageEvent) {
	fmt.Printf("%v\n", ev)
}
```