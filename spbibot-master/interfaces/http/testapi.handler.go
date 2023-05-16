package http

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"


)

type testAPIHandler struct {
	reportUsecase usecases.ReportUsecase
	mq            messagequeue.MessageQueue
	config        *config.TestAPIConfig
	logger        *zap.Logger
}

// ConfigureTestAPIHandler adds a test API handler to a request handling pipeline.
func ConfigureTestAPIHandler(
	r *httprouter.Router,
	report usecases.ReportUsecase,
	m messagequeue.MessageQueue,
	c *config.TestAPIConfig,
	l *zap.Logger,
) {
	if !c.Enable {
		return
	}

	h := testAPIHandler{
		reportUsecase: report,
		mq:            m,
		config:        c,
		logger:        l,
	}

	r.POST("/api/test/renderReport", h.handleRenderReport)
}

type filterRequest struct {
	Table                   string `json:"table"`
	Column                  string `json:"column"`
	Value                   string `json:"value"`
	LogicalOperator         string `json:"logicalOperator,omitempty"`
	ConditionOperator       string `json:"conditionOperator"`
	SecondValue             string `json:"secondValue,omitempty"`
	SecondConditionOperator string `json:"secondConditionOperator,omitempty"`
}

type pageRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type postReportRequest struct {
	ReportID    string         `json:"reportID"`
	ReportName  string         `json:"reportName"`
	Filter      *filterRequest `json:"filter,omitempty"`
	Pages       []*pageRequest `json:"pages"`
	UserID      string         `json:"userID"`
	ChannelID   string         `json:"channelID"`
	WorkspaceID string         `json:"workspaceID"`
	SkipPosting bool           `json:"skipPosting"`
}

func (h *testAPIHandler) handleRenderReport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	l := utils.WithContext(r.Context(), h.logger)

	k := r.Header.Get("X-Client-Key")
	if subtle.ConstantTimeCompare([]byte(k), []byte(h.config.ClientKey)) == 0 {
		l.Error("invalid client key", zap.String("clientKey", k))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	c := r.Header.Get(constants.HTTPHeaderContentType)
	if c != constants.MIMETypeJSON {
		l.Error("invalid content type", zap.String("contentType", c))
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	d := json.NewDecoder(r.Body)
	b := postReportRequest{}
	err := d.Decode(&b)
	if err != nil {
		l.Error("invalid request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		_, err2 := w.Write([]byte(err.Error()))
		if err2 != nil {
			l.Error("couldn't write body", zap.Error(err))
		}

		return
	}

	err = h.shareReport(r.Context(), &b)
	if err != nil {
		l.Error("couldn't share report", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		_, err2 := w.Write([]byte(err.Error()))
		if err2 != nil {
			l.Error("couldn't write body", zap.Error(err))
		}

		return
	}

	w.WriteHeader(http.StatusOK)

	l.Debug("shared report")
}

func (h *testAPIHandler) shareReport(ctx context.Context, r *postReportRequest) error {
	l := utils.WithContext(ctx, h.logger)

	pms := []*messagequeue.PageMessage(nil)
	for _, p := range r.Pages {
		pm := messagequeue.PageMessage{
			ID:   p.ID,
			Name: p.Name,
		}
		pms = append(pms, &pm)
	}

	m := messagequeue.PostReportMessage{
		RenderReportMessage: &messagequeue.RenderReportMessage{
			ClientID:    "slack",
			ReportID:    r.ReportID,
			ReportName:  r.ReportName,
			Pages:       pms,
			UserID:      r.UserID,
			ChannelID:   r.ChannelID,
			WorkspaceID: r.WorkspaceID,
			UniqueID:    uuid.New().String(),
		},
		SkipPosting: r.SkipPosting,
	}
	if r.Filter != nil {
		m.Filter = &messagequeue.FilterMessage{
			Table:                   r.Filter.Table,
			Column:                  r.Filter.Column,
			Value:                   r.Filter.Value,
			LogicalOperator:         r.Filter.LogicalOperator,
			ConditionOperator:       r.Filter.ConditionOperator,
			SecondValue:             r.Filter.SecondValue,
			SecondConditionOperator: r.Filter.SecondConditionOperator,
		}
	}

	e := messagequeue.Envelope{
		Kind:    messagequeue.MessagePostReport,
		Body:    m,
		TraceID: utils.RequestID(ctx),
	}
	err := h.mq.Push(ctx, &e, messagequeue.Wait)
	if err != nil {
		l.Error("couldn't enqueue message", zap.Error(err))
	}

	return err
}
