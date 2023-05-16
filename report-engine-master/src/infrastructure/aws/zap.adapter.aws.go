package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"go.uber.org/zap"
)

type zapAWSAdapter struct {
	*zap.Logger
	level aws.LogLevelType
}

func newZapAWSAdapter(l *zap.Logger, lv aws.LogLevelType) *zapAWSAdapter {
	return &zapAWSAdapter{
		Logger: l,
		level:  lv,
	}
}

func (l *zapAWSAdapter) Log(vs ...interface{}) {
	if l.level.AtLeast(aws.LogDebug) {
		l.Sugar().Debug(vs)
	}
}
