package mq

import (
	"context"
	"encoding/json"
	"strings"

	"go.uber.org/zap"

	
)

const (
	slackClient = "slack"
)

type reportWorker struct {
	reportUsecase    usecases.ReportUsecase
	userUsecase      usecases.UserUsecase
	workspaceUsecase usecases.WorkspaceUsecase
	logger           *zap.Logger
}

// NewReportWorker creates a Worker capable of report handling.
func NewReportWorker(
	r usecases.ReportUsecase,
	u usecases.UserUsecase,
	w usecases.WorkspaceUsecase,
	l *zap.Logger,
) Worker {
	return &reportWorker{
		reportUsecase:    r,
		userUsecase:      u,
		workspaceUsecase: w,
		logger:           l,
	}
}

func (w *reportWorker) SupportedMessages() []messagequeue.MessageKind {
	return []messagequeue.MessageKind{
		messagequeue.MessagePostReport,
	}
}

func (w *reportWorker) Handle(ctx context.Context, e *messagequeue.Envelope) error {
	l := utils.WithContext(ctx, w.logger)

	p, err := e.Unpack(func(j json.RawMessage) (interface{}, error) {
		p := messagequeue.PostReportMessage{}
		err := json.Unmarshal(j, &p)
		if err != nil {
			return nil, err
		}

		return &p, nil
	})
	if err != nil {
		l.Error("couldn't unpack envelope body", zap.Error(err))

		return err
	}

	err = w.shareReport(ctx, p.(*messagequeue.PostReportMessage))
	if err != nil {
		l.Error("couldn't share report", zap.Error(err))
	}

	return err
}

func (w *reportWorker) shareReport(ctx context.Context, r *messagequeue.PostReportMessage) error {
	pis := []string(nil)
	for _, p := range r.Pages {
		pis = append(pis, p.ID)
	}

	ctx = utils.WithActivityInfo(ctx, map[string]string{
		"reportID":    r.ReportID,
		"reportName":  r.ReportName,
		"pageIDs":     strings.Join(pis, ", "),
		"userID":      r.UserID,
		"channelID":   r.ChannelID,
		"workspaceID": r.WorkspaceID,
	})
	l := utils.WithContext(ctx, w.logger)

	pos := []*utils.PageOptions(nil)
	for _, pr := range r.Pages {
		po := utils.PageOptions{
			Name: pr.Name,
			ID:   pr.ID,
		}
		pos = append(pos, &po)
	}

	o := &utils.ShareOptions{
		ClientID:          r.ClientID,
		ReportID:          r.ReportID,
		ReportName:        r.ReportName,
		Pages:             pos,
		ChannelID:         r.ChannelID,
		WorkspaceID:       r.WorkspaceID,
		UserID:            r.UserID,
		IsScheduled:       r.IsScheduled,
		SkipPosting:       r.SkipPosting,
		AccessToken:       r.Token.PowerBIToken,
		RetryAttempt:      r.RetryAttempt,
		PostReportMessage: r,
	}
	var accessToken string
	var usrPtr *domain.User

	if r.ClientID == slackClient {
		u, err := w.userUsecase.GetByID(ctx, &domain.SlackUserID{
			WorkspaceID: r.WorkspaceID,
			ID:          r.UserID,
		})
		if err != nil {
			l.Error("couldn't get user", zap.Error(err))

			return err
		}
		usrPtr = &u

		s, err := w.workspaceUsecase.Get(ctx, r.WorkspaceID)
		if err != nil {
			l.Error("couldn't get workspace", zap.Error(err))

			return err
		}

		o = utils.WithAccessToken(*o, u.AccessToken)

		if r.Filter != nil {
			o.Filter = &utils.FilterOptions{
				Table:                   r.Filter.Table,
				Column:                  r.Filter.Column,
				Value:                   r.Filter.Value,
				LogicalOperator:         r.Filter.LogicalOperator,
				ConditionOperator:       r.Filter.ConditionOperator,
				SecondValue:             r.Filter.SecondValue,
				SecondConditionOperator: r.Filter.SecondConditionOperator,
			}
		}

		accessToken = s.BotAccessToken
	} else {
		accessToken = r.Token.BotAccessToken
		usrPtr = nil
	}

	err := w.reportUsecase.ShareReport(ctx, usrPtr, accessToken, o)
	if err != nil {
		l.Error("couldn't share report", zap.Error(err))
	}

	return err
}
