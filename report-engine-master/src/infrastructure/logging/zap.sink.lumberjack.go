package logging

import (
	"fmt"
	"net/url"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"

)

type lumberjackSink struct {
	*lumberjack.Logger
}

func newLumberjackSink(l *lumberjack.Logger) *lumberjackSink {
	return &lumberjackSink{
		Logger: l,
	}
}

func (*lumberjackSink) Sync() error {
	return nil
}

func lumberjackSinkFactory(c *config.LoggerConfig, o *pathOptions) func(u *url.URL) (zap.Sink, error) {
	return func(u *url.URL) (zap.Sink, error) {
		u, err := o.useWith(u)
		if err != nil {
			return nil, err
		}

		if u.Host != "localhost" {
			return nil, fmt.Errorf("host must be localhost")
		}

		l := lumberjack.Logger{
			Filename:   strings.TrimPrefix(u.Path, "/"),
			MaxSize:    c.MaxSizeMB,
			MaxAge:     c.MaxAgeDays,
			MaxBackups: c.MaxBackups,
			LocalTime:  false,
			Compress:   false,
		}
		sink := newLumberjackSink(&l)

		return sink, nil
	}
}
