### Service setup

1. Create `base.env` file which is the same as `base.env.example` [file](https://gitlab.inyar.ru/spbi/spbibot/-/blob/master/base.env.example) 
and fill in the following fields: 
   - `DB_USERNAME_PWD` - Install **MySQL** if you do not have it installed.
   Get the **password** from the root client 
   - `CLIENT_ID`- Register your application for [Power BI](https://gitlab.inyar.ru/spbi/spbibot/-/wikis/App-settings#register-via-power-bi)
   - `CLIENT_SECRET` 
   - `SLACK_CLIENT_ID` - Start creating an app in [Slack](https://gitlab.inyar.ru/spbi/spbibot/-/tree/master#slack-app-setup). At [this](https://gitlab.inyar.ru/spbi/spbibot/-/wikis/App-settings#obtain-bot-access-token) step, take the necessary information.
   - `SLACK_CLIENT_SECRET`
   - `SLACK_SIGN_IN_SECRET`
   - `MQ_URL'-You need to create your own
queue on [Amazon SQS services](https://aws.amazon.com/en/sqs). In the settings, select **FIFO** and **duplication based duplication**.
   - `AWS_ACCESSKEYID`
   - `AWS_ACCESSKEY`
2. Run report engine: `reportengine.go`

### Before merge

1. Format your changes by running `go fmt ./...` from the root
2. Check vet linter by running `go vet ./...` from the root
3. Check golangci-lint linter by running `golangci-lint run -c golangci.yaml ./...` from the root