package implementations

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/replaygaming/amplitude"
	"github.com/slack-go/slack"
	"go.uber.org/zap"


)

const (
	slackClient         = "slack"
	teamsClient         = "teams"
	failedReportPattern = "Sorry, we couldn't generate report %v"
)

// Token represents access + refresh token pair.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// GetAccessToken returns AccessToken
func (t Token) GetAccessToken() string {
	return t.AccessToken
}

// GetRefreshToken returns RefreshToken
func (t Token) GetRefreshToken() string {
	return t.RefreshToken
}

// ReportUsecase represent the data-struct for view usecases
type ReportUsecase struct {
	powerBiServiceClient  powerbi.ServiceClient
	workspaceRepository   domain.WorkspaceRepository
	postingTaskRepository domain.PostReportTaskRepository
	userRepository        domain.UserRepository
	mq                    messagequeue.MessageQueue
	dbTimeout             time.Duration
	logger                *zap.Logger
	reportRetryStrategy   reportengine.ReportRetryStrategy
}

// NewReportUsecase creates new an ReportUsecase object representation of domain.ReportUsecase interface
func NewReportUsecase(
	powerBiServiceClient powerbi.ServiceClient,
	workspaceRepository domain.WorkspaceRepository,
	postingTaskRepository domain.PostReportTaskRepository,
	userRepository domain.UserRepository,
	m messagequeue.MessageQueue,
	dbTimeout time.Duration,
	l *zap.Logger,
	r reportengine.ReportRetryStrategy,
) usecases.ReportUsecase {
	return &ReportUsecase{
		powerBiServiceClient:  powerBiServiceClient,
		workspaceRepository:   workspaceRepository,
		postingTaskRepository: postingTaskRepository,
		userRepository:        userRepository,
		mq:                    m,
		dbTimeout:             dbTimeout,
		logger:                l,
		reportRetryStrategy:   r,
	}
}

// ShareReport execute share report job in the background
func (reportUsecase *ReportUsecase) ShareReport(ctx context.Context, user *domain.User, token string, o *utils.ShareOptions) (err error) {
	pis := []string(nil)
	for _, p := range o.Pages {
		pis = append(pis, p.ID)
	}

	switch o.ClientID {
	case slackClient:
		return reportUsecase.shareToSlack(ctx, user, token, o, pis)
	case teamsClient:
		return reportUsecase.shareToTeams(ctx, token, o, pis)
	default:
		return domain.ErrInvalidType
	}
}

func (reportUsecase *ReportUsecase) generateReport(
	ctx *context.Context,
	o *utils.ShareOptions,
	slackUserID *domain.SlackUserID,
	logger *zap.Logger,
	m amplitude.Properties,
) (*domain.Report, *reportengine.RenderedReport, bool, error) {
	renderReportCtx, cancelRender, err := reportengine.DefaultReportEngine().NewContext()
	if err != nil {
		logger.Error("couldn't create context", zap.Error(err))

		return nil, nil, false, err
	}

	// NOTE: Both ctx & renderReportCtx are derived from different immediate parents, so we need to copy values.
	renderReportCtx = utils.WithActivityInfo(renderReportCtx, utils.ActivityInfo(*ctx))
	defer cancelRender()

	go func() {
		select {
		case <-(*ctx).Done():
			cancelRender()

		case <-renderReportCtx.Done():
		}
	}()

	getReportCtx, cancelGetReport := context.WithCancel(*ctx)
	reportChan := make(chan *domain.Report, 1)
	errChan := make(chan error, 1)

	go func() {
		defer cancelGetReport()
		token := Token{
			AccessToken: o.AccessToken,
		}

		var report *domain.Report
		if slackUserID != nil {
			report, err = reportUsecase.powerBiServiceClient.GetReport(*slackUserID, token, o.ReportID)
		} else {
			report, err = reportUsecase.powerBiServiceClient.GetReport(nil, token, o.ReportID)
		}
		if err != nil {
			logger.Error("couldn't get report", zap.Error(err))
			errChan <- err
			cancelRender()

			return
		}

		reportChan <- report
	}()

	renderedReport, err := reportengine.DefaultReportEngine().RenderReport(renderReportCtx, o)
	skipPosting, err := reportUsecase.reportRetryStrategy.Retry(renderReportCtx, o, err)
	if err != nil {
		logger.Error("couldn't render report", zap.Error(err))

		if slackUserID != nil {
			analytics.DefaultAmplitudeClient().Send(analytics.EventKindReportFailedToGenerate, slackUserID.WorkspaceID, slackUserID.ID, slackClient, m)
		} else {
			analytics.DefaultAmplitudeClient().Send(analytics.EventKindReportFailedToGenerate, o.WorkspaceID, o.UserID, teamsClient, m)
		}

		return nil, nil, false, err
	}

	if skipPosting {
		return nil, nil, true, nil
	}

	<-getReportCtx.Done()

	select {
	case err = <-errChan:
		if err != nil {
			return nil, nil, false, err
		}

	default:
	}

	report := <-reportChan

	return report, renderedReport, false, nil
}

func (reportUsecase *ReportUsecase) shareToSlack(ctx context.Context, user *domain.User, slackToken string, o *utils.ShareOptions, pis []string) error {
	slackUserID := user.GetSlackUserID()

	// NOTE: We aren't actually working in request context (it completes early to satisfy Slack's timing requirements), so we need to populate it again.
	ctx = utils.WithActivityInfo(ctx, utils.StringSet{
		"activityKind": "shareReport",
		"reportID":     o.ReportID,
		"pageIDs":      strings.Join(pis, ", "),
		"channelID":    o.ChannelID,
		"userID":       slackUserID.ID,
		"workspaceID":  slackUserID.WorkspaceID,
	})
	logger := utils.WithContext(ctx, reportUsecase.logger)

	startedAt := time.Now().UTC()
	logger.Debug("started sharing report")

	reportProperty := json.RawMessage(fmt.Sprintf(`{"isScheduled": %v, "reportID": "%v"}`, o.IsScheduled, o.ReportID))
	m := amplitude.Properties{
		"report": &reportProperty,
	}

	report, renderedReport, skipPosting, err := reportUsecase.generateReport(&ctx, o, slackUserID, logger, m)
	if err != nil {
		logger.Error("couldn't generate report", zap.Error(err))

		api := slack.New(slackToken)
		errText := fmt.Sprintf(failedReportPattern, o.ReportName)

		_, _, err := api.PostMessage(
			o.ChannelID,
			slack.MsgOptionText(errText, false),
			slack.MsgOptionAsUser(true),
		)

		return err
	}

	if skipPosting {
		return nil
	}

	if !o.SkipPosting {
		api := slack.New(slackToken)

		slackUser, err := api.GetUserInfo(slackUserID.ID)
		if err != nil {
			if err.Error() == constants.ErrorAccountInactive {
				err = reportUsecase.workspaceRepository.DeleteSoft(ctx, user.WorkspaceID)
				if err != nil {
					logger.Error("couldn't remove workspace", zap.Error(err))
					return err
				}
				logger.Info("workspace had been deactivated, removing it", zap.String("slackID", user.ID))

				analytics.DefaultAmplitudeClient().Send(analytics.EventKindWorkspaceDeleted, user.WorkspaceID, user.ID, slackClient, nil)

				return nil
			}

			if err.Error() == constants.ErrorNotInChannel {
				err = reportUsecase.postingTaskRepository.DeleteBySlackInfo(ctx, slackUserID, o.ChannelID)
				if err != nil {
					logger.Error("couldn't remove scheduled report", zap.Error(err))
					return err
				}
				logger.Info("channel had been deactivated, removing related scheduled tasks", zap.String("slackID", user.ID))

				analytics.DefaultAmplitudeClient().Send(analytics.EventKindChannelDeleted, user.WorkspaceID, user.ID, slackClient, nil)

				return nil
			}

			logger.Error("couldn't get user info", zap.Error(err))
			return err
		}

		if slackUser.Deleted && user.IsActive {
			err = reportUsecase.userRepository.Deactivate(ctx, slackUserID)
			if err != nil {
				logger.Error("couldn't deactivate user", zap.Error(err))

				return err
			}
			logger.Info("deactivating user account", zap.String("slackID", user.ID))

			analytics.DefaultAmplitudeClient().Send(analytics.EventKindUserDeactivated, user.WorkspaceID, user.ID, slackClient, nil)

			return nil
		}
		if !slackUser.Deleted && !user.IsActive {
			err = reportUsecase.userRepository.Reactivate(ctx, slackUserID)
			if err != nil {
				logger.Error("couldn't activate user", zap.Error(err))

				return err
			}
			logger.Info("reactivating user account", zap.String("slackID", user.ID))

			analytics.DefaultAmplitudeClient().Send(analytics.EventKindUserReactivated, user.WorkspaceID, user.ID, slackClient, nil)
		}

		for _, page := range renderedReport.Pages {
			title := ""
			if o.Filter != nil {
				title = constants.FormatMessageTitleWithFilter(o.ReportName, o.Filter.String(), page.Name)
			} else {
				title = constants.FormatMessageTitle(o.ReportName, page.Name)
			}

			comment := constants.FormatPageURL(report.GetWebURL(), page.ID)
			if o.IsScheduled {
				comment = fmt.Sprintf("<@%v>, %v", o.UserID, comment)
			}

			file := bytes.NewReader(page.ImageData)
			uploadPage := slack.FileUploadParameters{
				Title:    title,
				Filename: page.Filename,
				Reader:   file,
				Channels: []string{
					o.ChannelID,
				},
				InitialComment: comment,
			}
			_, err := api.UploadFile(uploadPage)
			if err != nil {
				logger.Error("couldn't upload page", zap.Error(err), zap.String("pageID", page.ID))

				analytics.DefaultAmplitudeClient().Send(analytics.EventKindSendReportMessageFailed, slackUserID.WorkspaceID, slackUserID.ID, slackClient, m)
				return err
			}
		}

		filterProperty := json.RawMessage(fmt.Sprintf(`{"withFilter": %v}`, o.Filter != nil))
		m["filter"] = &filterProperty

		analytics.DefaultAmplitudeClient().Send(analytics.EventKindReportGenerated, slackUserID.WorkspaceID, slackUserID.ID, slackClient, m)
	}

	if renderedReport != nil {
		completedIn := time.Now().UTC().Sub(startedAt)
		logger.Info("completed sharing report", zap.Duration("completedIn", completedIn), zap.Int("totalPages", len(renderedReport.Pages)))
	}

	return nil
}

func (reportUsecase *ReportUsecase) shareToTeams(ctx context.Context, token string, o *utils.ShareOptions, pis []string) error {
	ctx = utils.WithActivityInfo(ctx, utils.StringSet{
		"activityKind": "shareReport",
		"reportID":     o.ReportID,
		"pageIDs":      strings.Join(pis, ", "),
		"channelID":    o.ChannelID,
		"userID":       o.UserID,
	})
	logger := utils.WithContext(ctx, reportUsecase.logger)

	startedAt := time.Now().UTC()
	logger.Debug("started sharing report")

	reportProperty := json.RawMessage(fmt.Sprintf(`{"isScheduled": %v, "reportID": "%v"}`, o.IsScheduled, o.ReportID))
	m := amplitude.Properties{
		"report": &reportProperty,
	}

	report, renderedReport, skipPosting, err := reportUsecase.generateReport(&ctx, o, nil, logger, m)
	if err != nil {
		logger.Error("couldn't generate report", zap.Error(err))
		_ = teams.SendFailedMessage(o, token)

		return err
	}

	if skipPosting {
		return nil
	}

	for _, page := range renderedReport.Pages {
		comment := constants.FormatPageURL(report.GetWebURL(), page.ID)
		if o.IsScheduled {
			comment = fmt.Sprintf("<@%v>, %v", o.UserID, comment)
		}

		encodedReport := base64.StdEncoding.EncodeToString(page.ImageData)
		err = teams.SendMessage(encodedReport, o, token)
		if err != nil {
			logger.Error("couldn't upload page", zap.Error(err), zap.String("pageID", page.ID))

			analytics.DefaultAmplitudeClient().Send(analytics.EventKindSendReportMessageFailed, o.WorkspaceID, o.UserID, teamsClient, m)
			return err
		}
	}

	completedIn := time.Now().UTC().Sub(startedAt)
	logger.Info("completed sharing report", zap.Duration("completedIn", completedIn), zap.Int("totalPages", len(renderedReport.Pages)))

	filterProperty := json.RawMessage(fmt.Sprintf(`{"withFilter": %v}`, o.Filter != nil))
	m["filter"] = &filterProperty

	analytics.DefaultAmplitudeClient().Send(analytics.EventKindReportGenerated, o.WorkspaceID, o.UserID, teamsClient, m)
	return nil
}
