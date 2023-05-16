# SPBIBot

## How to run slack app and service locally

​**Attention** you must have **Chrome** installed!

### Service setup

1. Create `base.env` file which is the same as `base.env.example` file 
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
2. Create database.

   1. Create database `spbibot` (in **MySQL** or **DataGrip** or any other application) using script 1__createDB.sql from folder:  
   `./interfaces/repositories/mysql/scripts/`
   2. Use the [goose](https://gitlab.inyar.ru/spbi/spbibot/-/tree/master#goose-for-migrations) or run all the scripts yourself as you did step before.
3. Run bot: `go run cmd/bot/bot.go`
4. Run [report engine](https://gitlab.inyar.ru/spbi/report-engine)
5. Expose service by ngrok ([how to setup ngrok](https://ngrok.com/docs))
   
   If you use port 80 in your `bot.env` file  
   Run command: `ngrok http 80`
6. ngrok generates url which will be used in the `Slack app setup`
   ​
### Goose for migrations 
Install the [goose](https://pkg.go.dev/github.com/pressly/goose#section-readme), if you don't have one.
You may need to install **gcc**.

   Commands to work with migrations:
   ```
   Run migrations:              goose up
   Downgrade last migration:    goose down
   Check migrations state:      goose status
   Create new migration:        goose create <name>  
   ```
   First migration creates database tables by running all scripts from folder:
   `./interfaces/repositories/mysql/scripts/`  
   
   If you need to change database structure run command `goose create <name>` 
   which creates new file in folder `./db/migrations/`.  
   
   Fill in the function `Up` with the necessary changes and function `Down` with 
   actions to undo the new changes.  
   
   To apply the migration use `goose up`, to cancel migration use `goose down`.  
   
   You can check complete story of applying migrations in the database table `goose_db_version`.

### Slack app setup

For testing this project you need create your own slack app https://api.slack.com/apps
​
1. Go to your slack application settings -> https://api.slack.com/apps/{app_id}
   This documentation help you https://gitlab.inyar.ru/spbi/spbibot/-/wikis/App-settings
2. Go to 'OAuth & Permissions' tab and set same scopes as in [wiki](https://gitlab.inyar.ru/spbi/spbibot/-/wikis/App-settings)
3. Go to 'Slash Commands' tab and set same commands as in [wiki](https://gitlab.inyar.ru/spbi/spbibot/-/wikis/App-settings)
4. Go to 'Events subscription' tab and set same events as in [wiki](https://gitlab.inyar.ru/spbi/spbibot/-/wikis/App-settings)
5. Go to 'Install App' tab and click 'Install App' or 'Reinstall App'

p.s.
Useful sceenshots https://gitlab.inyar.ru/spbi/spbibot/-/wikis/Project-setup#project-setup

### Before merge

1. Format your changes by running `go fmt ./...` from the root
2. Check vet linter by running `go vet ./...` from the root
3. Check golangci-lint linter by running `golangci-lint run -c golangci.yaml ./...` from the root
