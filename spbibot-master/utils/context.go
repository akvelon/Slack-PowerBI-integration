package utils

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/segmentio/ksuid"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"go.uber.org/zap"
)

type contextKey struct {
	key string
}

var keyRequestID = contextKey{
	key: "requestID",
}

// WithRequestID adds a request id to a context.Context.
func WithRequestID(ctx context.Context) context.Context {
	id, err := ksuid.NewRandom()
	if err != nil {
		zap.L().Error("couldn't generate request id", zap.Error(err))
	}

	return context.WithValue(ctx, keyRequestID, id.String())
}

// RequestID extracts a request id from a context.Context.
func RequestID(ctx context.Context) string {
	return getString(ctx, keyRequestID)
}

var keyRequestInfo = &contextKey{
	key: "requestInfo",
}

// WithRequestInfo adds request info to a context.Context.
func WithRequestInfo(ctx context.Context, r *http.Request) context.Context {
	p := fmt.Sprintf("%v %v", r.Method, r.RequestURI)

	return context.WithValue(ctx, keyRequestInfo, p)
}

// RequestInfo extracts request info from a context.Context.
func RequestInfo(ctx context.Context) string {
	return getString(ctx, keyRequestInfo)
}

var keySlackInfo = &contextKey{
	key: "slackInfo",
}

// WithSlashCommand adds useful identifiers from a slack.SlashCommand to a context.Context.
func WithSlashCommand(ctx context.Context, c *slack.SlashCommand) context.Context {
	i := StringSet{
		"userID":       c.UserID,
		"teamID":       c.TeamID,
		"enterpriseID": c.EnterpriseID,
		"channelID":    c.ChannelID,
		"command":      c.Command,
		"commandText":  c.Text,
	}

	return context.WithValue(ctx, keySlackInfo, i)
}

// WithInteractionPayload adds useful identifiers from a slack.InteractionCallback to a context.Context.
func WithInteractionPayload(ctx context.Context, c *slack.InteractionCallback) context.Context {
	i := StringSet{
		"payloadType":  string(c.Type),
		"teamID":       c.Team.ID,
		"userID":       c.User.ID,
		"callbackID":   c.View.CallbackID,
		"enterpriseID": c.User.Enterprise.EnterpriseID,
	}

	return context.WithValue(ctx, keySlackInfo, i)
}

// WithAPIEvent adds useful identifiers from a slackevents.EventsAPIEvent to a context.Context.
func WithAPIEvent(ctx context.Context, c *slackevents.EventsAPIEvent) context.Context {
	i := StringSet{
		"teamID":    c.TeamID,
		"eventType": c.Type,
	}

	return context.WithValue(ctx, keySlackInfo, i)
}

// SlackInfo extracts Slack-related info from a context.Context.
func SlackInfo(ctx context.Context) StringSet {
	return getStringMap(ctx, keySlackInfo)
}

var keyTaskID = &contextKey{
	key: "taskID",
}

// WithTaskID adds a task id to a context.Context.
func WithTaskID(ctx context.Context, t *Task) context.Context {
	i := strconv.FormatInt(t.ID, 10)

	return context.WithValue(ctx, keyTaskID, i)
}

// TaskID extracts a task id from a context.Context.
func TaskID(ctx context.Context) string {
	return getString(ctx, keyTaskID)
}

var keyActivityInfo = &contextKey{
	key: "activityInfo",
}

// WithActivityInfo adds an arbitrary string set to a context.Context.
func WithActivityInfo(ctx context.Context, s StringSet) context.Context {
	i := ActivityInfo(ctx)
	for k, v := range s {
		i[k] = v
	}

	return context.WithValue(ctx, keyActivityInfo, i)
}

// ActivityInfo extracts an arbitrary string set from a context.Context.
func ActivityInfo(ctx context.Context) StringSet {
	return getStringMap(ctx, keyActivityInfo)
}

func getString(ctx context.Context, k interface{}) string {
	s, ok := ctx.Value(k).(string)
	if ok {
		return s
	}

	return ""
}

func getStringMap(ctx context.Context, k interface{}) StringSet {
	m, ok := ctx.Value(k).(StringSet)
	if ok {
		return m
	}

	return StringSet{}
}
