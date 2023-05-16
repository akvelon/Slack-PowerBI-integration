package utils

import (
	"context"
)

type contextKey struct {
	key string
}

var keyRequestID = contextKey{
	key: "requestID",
}

// RequestID extracts a request id from a context.Context.
func RequestID(ctx context.Context) string {
	return getString(ctx, keyRequestID)
}

var keyRequestInfo = &contextKey{
	key: "requestInfo",
}

// RequestInfo extracts request info from a context.Context.
func RequestInfo(ctx context.Context) string {
	return getString(ctx, keyRequestInfo)
}

var keySlackInfo = &contextKey{
	key: "slackInfo",
}

// SlackInfo extracts Slack-related info from a context.Context.
func SlackInfo(ctx context.Context) StringSet {
	return getStringMap(ctx, keySlackInfo)
}

var keyTaskID = &contextKey{
	key: "taskID",
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
