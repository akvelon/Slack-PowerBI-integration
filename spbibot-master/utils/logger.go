package utils

import (
	"context"

	"go.uber.org/zap"
)

// WithContext extracts specific values from a context.Context & attaches them as fields to a zap.Logger.
func WithContext(ctx context.Context, l *zap.Logger) *zap.Logger {
	return l.With(fromContext(ctx)...)
}

func fromContext(ctx context.Context) []zap.Field {
	s := cleanupStringSets(
		StringSet{
			"requestID":   RequestID(ctx),
			"requestInfo": RequestInfo(ctx),
			"taskID":      TaskID(ctx),
		},
		SlackInfo(ctx),
		ActivityInfo(ctx),
	)

	return toZapFields(s)
}

// StringSet is a set of arbitrary string fields.
type StringSet map[string]string

func cleanupStringSets(ss ...StringSet) StringSet {
	fs := StringSet{}
	for _, s := range ss {
		for k, v := range s {
			if v != "" && k != "" {
				fs[k] = v
			}
		}
	}

	return fs
}

func toZapFields(s StringSet) []zap.Field {
	fs := []zap.Field(nil)
	for k, v := range s {
		fs = append(fs, zap.String(k, v))
	}

	return fs
}
