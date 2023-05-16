package logging

import (
	"go.uber.org/zap"
)

type zapMySQLAdapter struct {
	*zap.Logger
}

func newZapMySQLAdapter(l *zap.Logger) *zapMySQLAdapter {
	return &zapMySQLAdapter{
		Logger: l,
	}
}

func (l *zapMySQLAdapter) Print(vs ...interface{}) {
	l.Sugar().Error(vs)
}
