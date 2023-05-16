package logging

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"


)

type timeLayout string

const (
	layoutISO8601 timeLayout = "2006-01-02T15:04:05.000Z0700"
)

type key string

const (
	keyTime key = "time"
)

// Builder configures a zap.Logger.
type Builder struct {
	cloudWatchLogs *cloudwatchlogs.CloudWatchLogs
	hostConfig     *config.HostConfig
	loggerConfig   *config.LoggerConfig
	fallbackLogger *log.Logger
}

// NewBuilder creates a Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// WithHostConfig adds a config.HostConfig.
func (b *Builder) WithHostConfig(h *config.HostConfig) *Builder {
	b.hostConfig = h

	return b
}

// WithCloudWatchLogs adds a cloudwatchlogs.CloudWatchLogs.
func (b *Builder) WithCloudWatchLogs(l *cloudwatchlogs.CloudWatchLogs) *Builder {
	b.cloudWatchLogs = l

	return b
}

// WithLoggerConfig adds a config.LoggerConfig.
func (b *Builder) WithLoggerConfig(l *config.LoggerConfig) *Builder {
	b.loggerConfig = l

	return b
}

// WithFallbackLogger adds a log.Logger.
func (b *Builder) WithFallbackLogger(l *log.Logger) *Builder {
	b.fallbackLogger = l

	return b
}

// NewLogger creates a zap.Logger.
func (b *Builder) NewLogger() (*zap.Logger, func(), error) {
	loggerConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(b.loggerConfig.Level),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          b.loggerConfig.Encoding,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:       "message",
			LevelKey:         "level",
			TimeKey:          string(keyTime),
			NameKey:          "name",
			CallerKey:        "caller",
			FunctionKey:      zapcore.OmitKey,
			StacktraceKey:    "stacktrace",
			LineEnding:       zapcore.DefaultLineEnding,
			EncodeLevel:      b.loggerConfig.LevelEncoder,
			EncodeTime:       zapcore.TimeEncoderOfLayout(string(layoutISO8601)),
			EncodeDuration:   zapcore.StringDurationEncoder,
			EncodeCaller:     zapcore.ShortCallerEncoder,
			EncodeName:       zapcore.FullNameEncoder,
			ConsoleSeparator: "\t",
		},
		OutputPaths:      b.loggerConfig.Sinks,
		ErrorOutputPaths: b.loggerConfig.ErrorSinks,
		InitialFields:    nil,
	}
	switch b.hostConfig.Environment {
	case config.EnvironmentDevelopment:
		loggerConfig.Development = true

	case config.EnvironmentProduction:
		loggerConfig.Development = false

	default:
		return nil, nil, fmt.Errorf("unknown environment: %v", b.hostConfig.Environment)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, err
	}

	o := pathOptions{
		Host: hostname,
	}

	err = zap.RegisterSink("lumberjack", lumberjackSinkFactory(b.loggerConfig, &o))
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't register sink")
	}

	if b.cloudWatchLogs != nil {
		err = zap.RegisterSink("cloudwatch", cloudwatchSinkFactory(b.cloudWatchLogs, b.loggerConfig, &o, b.fallbackLogger))
		if err != nil {
			return nil, nil, errors.Wrap(err, "couldn't register sink")
		}
	}

	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't build logger")
	}

	syncLogger := func() {
		err := logger.Sync()
		if err != nil {
			b.fallbackLogger.Println("couldn't sync logger:", err)
		}
	}

	zap.ReplaceGlobals(logger)

	_, err = zap.RedirectStdLogAt(logger.Named("std"), zapcore.InfoLevel)
	if err != nil {
		syncLogger()

		return nil, nil, errors.Wrap(err, "couldn't redirect std logger")
	}

	err = mysql.SetLogger(newZapMySQLAdapter(logger.Named("mysql")))
	if err != nil {
		syncLogger()

		return nil, nil, errors.Wrap(err, "couldn't set DB logger")
	}

	return logger, syncLogger, nil
}
