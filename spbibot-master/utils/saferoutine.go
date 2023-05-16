package utils

import (
	"go.uber.org/zap"
)

// SafeRoutine wraps goroutine func with recover
func SafeRoutine(routineFn func()) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				zap.L().Error("goroutine panicked", zap.Any("panic", p))
			}
		}()

		routineFn()
	}()
}
