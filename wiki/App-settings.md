# Slack Application Setup

### **_App Home_**

#### **_Show Tabs_**

Enable **_Home Tab_** & **_Messages Tab_**.

### **_Slash Commands_**

| **_Command_** | **_Short Description_** | **_Request URL_** |
| --- | --- | --- |
| `/pbi-sign-in` | `Connect a Power BI account.` | `{service_url}/slash` (e.g.: `https://{ngrok_url}/slash`) |
| `/pbi-share-report` | `Share report as image.` | `{service_url}/slash` |
| `/pbi-sign-out` | `Sign-out from Power BI account.` | `{service_url}/slash` |
| `/pbi-create-alert`  | `Create alert notification.` | `{service_url}/slash` |
| `/pbi-manage-filters`  | `Manage your filters.` | `{service_url}/slash` |
| `/pbi-manage-scheduled-reports`  | `Manage scheduled reports.` | `{service_url}/slash` |
| `/pbi-manage-alerts` | `Manage alerts.` | `{service_url}/slash` |
| `/pbi-schedule-report` | `Schedule automatic report posting.` | `{service_url}/slash` |

⚠ Corresponding functionality is intentionally disabled by default. You can enable it by using respective feature toggles (put these in `.env`):

```dotenv
FEATURETOGGLES_REPORTSCHEDULING=true
```

⚠ Enabling these on production environment should be done w/ extra care:
* `REPORTSCHEDULING` will pose performance issues on our current machine.

However, you can play w/ these features all you like on your local development environment.   

### **_OAuth & Permissions_**

#### **_Redirect URLs_**

| **_Redirect URLs_** |
| --- |
| `{service_url}/bot_authorization_response` |

#### **_Scopes_**

| **_Bot token scopes_** |
| --- |
| `files:write` |
| `commands` |
| `channels:read` |
| `chat:write` |
| `users:read` |
| `groups:read` |
| `im:read` |
| `users:read.email` |

### **_Interactivity & Shortcuts_**

Enable **_Interactivity_**.

| **_Request URL_** |
| --- |
| `{service_url}/interaction` |

### **_Event Subscriptions_**

Enable **_Events_**.

| **_Request URL_** |
| --- |
| `{service_url}/events` |

#### **_Subscribe to bot events_**

| **_Event Name_** |
| --- |
| `app_home_opened` | 

# Fill Slack app basic information

1. Go to Basic Information tab
2. Look at "Building Apps for Slack" section. Make sure "Manage distribution" is green
3. Scroll down to "Installing Your App" section
4. Choose "Install from App Directory"
5. Set url {service_url}/add_to_slack
6. Push "Save changes"

# Obtain Bot Access Token 

1. Add redirect URLs at OAuth & Permissions
2. In .env file set SLACK_CLIENT_ID and SLACK_CLIENT_SECRET variables
3. Start service 
4. Go to Manage Distribution page and copy Sharable URL
5. Paste Sharable URL to browser and Allow

# Enable event subscription and home tab

1. Go to "App Home" section in slack app dashboard
2. Turn on "Home tab" in Show Tabs 
3. Start service with implemented events.handler
4. Go to Event Subscriptions section in slack app dashboard
5. Turn on "Enable Events" and paste Redirect URL. Wait until it verify your URL
6. In "Subscribe to bot events" add "app_home_opened" bot user event 

# Azure Application Setup

### Azure Application Registration

##### Register via Power BI 

1. Go to [Register your application for Power BI page](https://dev.powerbi.com/apps)
2. Sign in to Power BI
3. Register your application 
    - Application Name: e.g. Slack bot Power BI integration
    - Application Type: Server-side web application (for web apps or web APIs)
    - Home Page URL: e.g. http://localhost/interaction 
    - Redirect URL: e.g. http://localhost/authorization_response
    - API Access: Read all reports, dashboards, datasets and workspaces
4. Save Application ID and Application secret somewhere (this matches the values CLIENT_ID and CLIENT_SECRET in .env)

##### Register via Azure AD

1. Go to [Azure Portal](https://portal.azure.com/) -> Azure Active Directory -> App Registrations
2. Click on New registration
3. Register an application:
    - Name: e.g. Slack bot Power BI integration
    - Supported account types: 	Accounts in any organizational directory
    - Redirect URI: Web e.g. http://localhost/authorization_response
    - Click on Register button

### Setup Azure app

1. Go to Azure Portal -> Azure Active Directory -> App Registrations -> <Your App Name> -> Integration assistant
2. Choose Web API and check that the "Is this application calling APIs?" checkbox is enabled 
3. Evaluate my app registration
4. If you registered your app via Power BI **skip ** this step.
If you registered your app via Azure App:
    - Go to  Certificates & secrets
    - Click on New client secret
    - Fill Description with "Client sign in secret" for example
    - Choose the expires date 
    - Click on Add
    - Copy the new secret value and save (will be needed for connection)
    - Go to API permissions (on the left)
    - Click on Add a permission -> choose Power BI Service -> Delegated permissions
    - Check `Dashboard.Read.All`, `Report.Read.All`, `Dataset.Read.All`, & `Workspace.Read.All` are enabled
    - Click Add permissions
    - Make sure you have Microsoft Graph User.Read permission
5. Go to the Authentication page
6. In the Implicit grant section choose ID tokens as tokens that are issued by the authorization endpoint
7. Save changes