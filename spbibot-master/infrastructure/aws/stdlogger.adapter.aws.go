package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
)

type stdLoggerAWSAdapter struct {
	*log.Logger
	level aws.LogLevelType
}

func newStdLoggerAWSAdapter(l *log.Logger, lv aws.LogLevelType) *stdLoggerAWSAdapter {
	return &stdLoggerAWSAdapter{
		Logger: l,
		level:  lv,
	}
}

func (l *stdLoggerAWSAdapter) Log(vs ...interface{}) {
	if l.level.AtLeast(aws.LogDebug) {
		l.Println(vs...)
	}
}
