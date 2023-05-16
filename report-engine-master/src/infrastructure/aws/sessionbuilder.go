package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"go.uber.org/zap"

)

// SessionBuilder configures a client.ConfigProvider.
type SessionBuilder struct {
	awsConfig *config.AWSConfig
	stdLogger *log.Logger
	zapLogger *zap.Logger
}

// NewSessionBuilder creates a SessionBuilder.
func NewSessionBuilder() *SessionBuilder {
	return &SessionBuilder{}
}

// WithAWSConfig adds a config.AWSConfig.
func (b *SessionBuilder) WithAWSConfig(a *config.AWSConfig) *SessionBuilder {
	b.awsConfig = a

	return b
}

// WithStdLogger adds a log.Logger.
func (b *SessionBuilder) WithStdLogger(l *log.Logger) *SessionBuilder {
	b.stdLogger = l

	return b
}

// WithZapLogger adds a zap.Logger.
func (b *SessionBuilder) WithZapLogger(l *zap.Logger) *SessionBuilder {
	b.zapLogger = l

	return b
}

// NewSession creates a client.ConfigProvider.
func (b *SessionBuilder) NewSession() (client.ConfigProvider, error) {
	lv := aws.LogOff
	if b.awsConfig.LogRequests {
		lv = aws.LogDebug | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors
	}

	c := aws.NewConfig()
	if b.zapLogger != nil {
		c = c.WithLogger(newZapAWSAdapter(b.zapLogger.Named("aws"), lv))
	} else if b.stdLogger != nil {
		c = c.WithLogger(newStdLoggerAWSAdapter(b.stdLogger, lv))
	} else if lv != aws.LogOff {
		return nil, fmt.Errorf("logger must be set when requuest logging is enabled")
	}

	c = c.WithLogLevel(lv)
	c = c.WithCredentials(credentials.NewStaticCredentials(b.awsConfig.AccessKeyID, b.awsConfig.AccessKey, ""))

	s, err := session.NewSession(c, aws.NewConfig().WithRegion(b.awsConfig.Region))
	if err != nil {
		return nil, err
	}

	return s, err
}
