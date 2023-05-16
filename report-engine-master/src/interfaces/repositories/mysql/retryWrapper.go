package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

func queryContextWithRetry(ctx context.Context, isFastRetry bool, logger *zap.Logger, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err == nil {
		return rows, nil
	}
	logger.Error("couldn't execute query first time, try to use retry strategy", zap.Error(err))

	timeUnit, timeouts := getRetryStrategy(isFastRetry)
	for i, timeout := range timeouts {
		time.Sleep(timeout * timeUnit)
		rows, err := db.QueryContext(ctx, query, args...)
		if err == nil {
			return rows, nil
		}
		logger.Error(fmt.Sprintf("couldn't execute query with retry %v", i+1), zap.Error(err))
	}

	return nil, err
}

func queryRowContextWithRetry(ctx context.Context, isFastRetry bool, logger *zap.Logger, db *sql.DB, query string, args ...interface{}) (interface{}, error) {
	var result interface{}
	err := db.QueryRowContext(ctx, query, args...).Scan(&result)
	if err == nil {
		return result, nil
	}
	logger.Error("couldn't execute query first time, try to use retry strategy", zap.Error(err))

	timeUnit, timeouts := getRetryStrategy(isFastRetry)
	for i, timeout := range timeouts {
		time.Sleep(timeout * timeUnit)
		err := db.QueryRowContext(ctx, query, args...).Scan(&result)
		if err == nil {
			return result, nil
		}
		logger.Error(fmt.Sprintf("couldn't execute query with retry %v", i+1), zap.Error(err))
	}

	return nil, err
}

func prepareContextWithRetry(ctx context.Context, isFastRetry bool, logger *zap.Logger, db *sql.DB, query string) (*sql.Stmt, error) {
	stmt, err := db.PrepareContext(ctx, query)
	if err == nil {
		return stmt, nil
	}
	logger.Error("couldn't prepare context first time, try to use retry strategy", zap.Error(err))

	timeUnit, timeouts := getRetryStrategy(isFastRetry)
	for i, timeout := range timeouts {
		time.Sleep(timeout * timeUnit)
		stmt, err := db.PrepareContext(ctx, query)
		if err == nil {
			return stmt, nil
		}
		logger.Error(fmt.Sprintf("couldn't prepare context with retry %v", i+1), zap.Error(err))
	}

	return nil, err
}

func execContextWithRetry(ctx context.Context, isFastRetry bool, logger *zap.Logger, stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	res, err := stmt.ExecContext(ctx, args...)
	if err == nil {
		return res, nil
	}
	logger.Error("couldn't execute context first time, try to use retry strategy", zap.Error(err))

	timeUnit, timeouts := getRetryStrategy(isFastRetry)
	for i, timeout := range timeouts {
		time.Sleep(timeout * timeUnit)
		res, err := stmt.ExecContext(ctx, args...)
		if err == nil {
			return res, nil
		}
		logger.Error(fmt.Sprintf("couldn't execute context with retry %v", i+1), zap.Error(err))
	}

	return nil, err
}

func getRetryStrategy(isFastRetry bool) (time.Duration, []time.Duration) {
	timeUnit := time.Second
	timeouts := []time.Duration{2, 4, 8, 16, 32}
	if isFastRetry {
		timeUnit = time.Millisecond
		timeouts = []time.Duration{250, 500}
	}
	return timeUnit, timeouts
}
