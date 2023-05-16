package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"

)

type postReportTaskRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewMySQLPostReportTaskRepository creates a domain.PostReportTaskRepository.
func NewMySQLPostReportTaskRepository(db *sql.DB, l *zap.Logger) domain.PostReportTaskRepository {
	return &postReportTaskRepository{
		db:     db,
		logger: l,
	}
}

func (r *postReportTaskRepository) Add(ctx context.Context, t *domain.PostReportTask) error {
	pageIDsJSON, err := json.Marshal(t.PageIDs)
	if err != nil {
		return err
	}

	var dayOfWeek, dayOfMonth interface{}
	if t.IsEveryDay || t.IsEveryHour {
		dayOfWeek = nil
		dayOfMonth = nil
	} else if t.DayOfMonth == 0 {
		dayOfWeek = t.DayOfWeek
		dayOfMonth = nil
	} else {
		dayOfWeek = nil
		dayOfMonth = t.DayOfMonth
	}

	query := `INSERT INTO postReportTasks SET id=?, workspaceID=?, userID=?, reportID=?, pageIDs=?, channelID=?, taskTime=?, dayOfWeek=?, dayOfMonth=?, isEveryDay=?, tz=?, completedAt=?, isActive=?, isEveryHour=?`
	res, err := r.execute(
		ctx,
		true,
		query,
		t.ID,
		t.WorkspaceID,
		t.UserID,
		t.ReportID,
		pageIDsJSON,
		t.ChannelID,
		t.TaskTime,
		dayOfWeek,
		dayOfMonth,
		t.IsEveryDay,
		t.TZ,
		sql.NullTime{},
		t.IsActive,
		t.IsEveryHour,
	)
	mysqlErr, ok := err.(*mysql.MySQLError)
	if ok && mysqlErr.Number == errorCodeDuplicateEntry {
		return domain.ErrConflict
	}

	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	t.ID = id

	return err
}

func (r *postReportTaskRepository) GetScheduledReports(ctx context.Context, u domain.SlackUserID, reportID string) ([]*domain.PostReportTask, error) {
	query := `SELECT id, workspaceID, userID, reportID, pageIDs, channelID, taskTime, IFNULL(dayOfWeek, 0), IFNULL(dayOfMonth, 0), isEveryDay, tz, completedAt, isActive, isEveryHour
			  FROM postReportTasks
              WHERE workspaceID=? and userID=? and reportID=?`
	reports, err := r.fetch(ctx, true, query, u.WorkspaceID, u.ID, reportID)
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func (r *postReportTaskRepository) GetPowerBIReportIDsByUser(ctx context.Context, u domain.SlackUserID) ([]string, error) {
	query := `SELECT DISTINCT reportID
			  FROM postReportTasks
              WHERE workspaceID=? and userID=?`
	reports, err := r.fetchReportIDs(ctx, true, query, u.WorkspaceID, u.ID)
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func (r *postReportTaskRepository) GetActualScheduledReports(ctx context.Context) ([]*domain.PostReportTask, error) {
	query := `SELECT id, workspaceID, userID, reportID, pageIDs, channelID, taskTime, IFNULL(dayOfWeek, 0), IFNULL(dayOfMonth, 0), isEveryDay, tz, completedAt, isActive, isEveryHour
 			  FROM postReportTasks
			  WHERE ADDTIME(UTC_TIME(), '-0:30') < TIME(taskTime) AND UTC_TIME() > TIME(taskTime)
    			AND (isEveryHour = true OR isEveryDay = true OR DAYOFWEEK(UTC_TIMESTAMP()) = dayOfWeek OR DAYOFMONTH(UTC_TIMESTAMP()) = dayOfMonth
    			OR dayOfMonth = 32 AND DAYOFMONTH(last_day(UTC_TIMESTAMP())) = DAYOFMONTH(UTC_TIMESTAMP()) OR dayOfMonth = -1 AND DAYOFMONTH(UTC_TIMESTAMP()) = 1) AND isActive = true`

	reports, err := r.fetch(ctx, false, query)
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *postReportTaskRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM postReportTasks WHERE id=?`
	_, err := r.execute(ctx, true, query, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *postReportTaskRepository) DeleteBySlackInfo(ctx context.Context, u *domain.SlackUserID, channelID string) error {
	query := `DELETE FROM postReportTasks WHERE workspaceID=? AND userID=? AND channelID=?`
	_, err := r.execute(ctx, true, query, u.WorkspaceID, u.ID, channelID)
	if err != nil {
		return err
	}
	return nil
}

func (r *postReportTaskRepository) UpdateCompletionStatus(ctx context.Context, id int64) (bool, error) {
	query := `SELECT id, workspaceID, userID, reportID, pageIDs, channelID, taskTime, IFNULL(dayOfWeek, 0), IFNULL(dayOfMonth, 0), isEveryDay, tz, completedAt, isActive, isEveryHour
 			  FROM postReportTasks WHERE id=?`
	result, err := r.fetch(ctx, true, query, id)
	if err != nil {
		return false, err
	}
	newStatus := !result[0].IsActive

	query = `UPDATE postReportTasks SET isActive=? WHERE id=?`
	_, err = r.execute(ctx, true, query, newStatus, id)
	if err != nil {
		return false, err
	}
	return newStatus, nil
}

func (r *postReportTaskRepository) UpdateHourlyReports(ctx context.Context, id int64) error {
	hours := time.Now().UTC().Hour()
	newTime := strconv.Itoa(hours) + ":55"
	query := `UPDATE postReportTasks SET taskTime=? WHERE id=?`
	_, err := r.execute(ctx, true, query, newTime, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *postReportTaskRepository) Update(ctx context.Context, t *domain.PostReportTask) error {
	completedAtNull := sql.NullTime{}
	if !t.CompletedAt.IsZero() {
		completedAtNull.Time = t.CompletedAt
		completedAtNull.Valid = true
	}

	query := `UPDATE postReportTasks SET completedAt=?, isActive=? WHERE id=?`
	res, err := r.execute(
		ctx,
		true,
		query,
		completedAtNull,
		t.IsActive,
		t.ID,
	)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		return domain.ErrNotUpdated
	}

	return nil
}

func (r *postReportTaskRepository) CheckIfReportScheduledAlready(ctx context.Context, t *domain.PostReportTask) (bool, error) {
	query := `SELECT EXISTS
	(
		SELECT id
		FROM postReportTasks
		WHERE workspaceID = ?
		  AND userID = ?
		  AND reportID = ?
		  AND channelID = ?
		  AND taskTime = ?
		  AND IFNULL(dayOfWeek, 0) = ?
		  AND IFNULL(dayOfMonth, 0) = ?
		  AND isEveryDay = ?
		  AND isEveryHour = ?
	)`

	isExist, err := queryRowContextWithRetry(
		ctx,
		true,
		r.logger,
		r.db,
		query,
		t.WorkspaceID,
		t.UserID,
		t.ReportID,
		t.ChannelID,
		t.TaskTime,
		t.DayOfWeek,
		t.DayOfMonth,
		t.IsEveryDay,
		t.IsEveryHour,
	)
	if err != nil {
		r.logger.Error("couldn't execute query", zap.Error(err))

		return false, err
	}
	if isExist == int64(1) {
		return true, nil
	}

	return false, nil
}

func (r *postReportTaskRepository) execute(ctx context.Context, isFastRetry bool, query string, args ...interface{}) (sql.Result, error) {
	l := utils.
		WithContext(ctx, r.logger).
		With(zap.String("query", query))

	s, err := prepareContextWithRetry(ctx, isFastRetry, r.logger, r.db, query)
	if err != nil {
		l.Error("couldn't create prepared statement", zap.Error(err))

		return nil, err
	}

	e, err := execContextWithRetry(ctx, isFastRetry, r.logger, s, args...)
	if err != nil {
		l.Error("couldn't execute prepared statement", zap.Error(err))

		return nil, err
	}

	return e, nil
}

func (r *postReportTaskRepository) fetch(ctx context.Context, isFastRetry bool, query string, args ...interface{}) ([]*domain.PostReportTask, error) {
	l := utils.
		WithContext(ctx, r.logger).
		With(zap.String("query", query))

	rows, err := queryContextWithRetry(ctx, isFastRetry, r.logger, r.db, query, args...)
	if err != nil {
		l.Error("couldn't execute query", zap.Error(err))

		return nil, err
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			l.Error("couldn't close rows", zap.Error(err))
		}
	}()

	result := []*domain.PostReportTask(nil)
	for rows.Next() {
		task := domain.PostReportTask{}
		completedAtNull := sql.NullTime{}
		pageIDsJSON := sql.RawBytes{}
		err := rows.Scan(
			&task.ID,
			&task.WorkspaceID,
			&task.UserID,
			&task.ReportID,
			&pageIDsJSON,
			&task.ChannelID,
			&task.TaskTime,
			&task.DayOfWeek,
			&task.DayOfMonth,
			&task.IsEveryDay,
			&task.TZ,
			&completedAtNull,
			&task.IsActive,
			&task.IsEveryHour,
		)
		if err != nil {
			l.Error("couldn't scan row", zap.Error(err))

			return nil, err
		}

		if completedAtNull.Valid {
			task.CompletedAt = completedAtNull.Time
		} else {
			task.CompletedAt = time.Time{}
		}

		pageIDs := []string(nil)
		err = json.Unmarshal(pageIDsJSON, &pageIDs)
		if err != nil {
			l.Error("couldn't unmarshal page ids", zap.Error(err))

			return nil, err
		}

		task.PageIDs = pageIDs

		result = append(result, &task)
	}

	return result, nil
}

func (r *postReportTaskRepository) fetchReportIDs(ctx context.Context, isFastRetry bool, query string, args ...interface{}) ([]string, error) {
	l := utils.
		WithContext(ctx, r.logger).
		With(zap.String("query", query))

	rows, err := queryContextWithRetry(ctx, isFastRetry, r.logger, r.db, query, args...)
	if err != nil {
		l.Error("couldn't execute query", zap.Error(err))

		return nil, err
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			l.Error("couldn't close rows", zap.Error(err))
		}
	}()

	result := []string(nil)
	for rows.Next() {
		var reportID string
		err := rows.Scan(
			&reportID,
		)
		if err != nil {
			l.Error("couldn't scan row", zap.Error(err))

			return nil, err
		}

		result = append(result, reportID)
	}

	return result, nil
}
