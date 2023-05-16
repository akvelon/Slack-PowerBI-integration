package mysql

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"


)

const errorCodeDuplicateEntry = 1062

type filterRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewMySQLFilterRepository creates a domain.filterRepository.
func NewMySQLFilterRepository(db *sql.DB, l *zap.Logger) domain.FilterRepository {
	return &filterRepository{
		db:     db,
		logger: l,
	}
}

func (r *filterRepository) Get(ctx context.Context, id int64) (*domain.Filter, error) {
	query := `SELECT id, workspaceID, userID, reportID, name, kind, definition FROM filters WHERE id=?`
	reports, err := r.fetch(ctx, true, query, id)
	if err != nil {
		return nil, err
	}

	if len(reports) == 1 {
		return reports[0], nil
	}

	return nil, domain.ErrNotFound
}

func (r *filterRepository) ListByReportID(ctx context.Context, reportID string) ([]*domain.Filter, error) {
	query := `SELECT id, workspaceID, userID, reportID, name, kind, definition FROM filters WHERE reportId=?`

	return r.fetch(ctx, true, query, reportID)
}

func (r *filterRepository) Store(ctx context.Context, filter *domain.Filter) error {
	definitionJSON, err := json.Marshal(filter.Definition)
	if err != nil {
		return err
	}

	query := `INSERT INTO filters SET id=?, workspaceID=?, userID=?, reportID=?, name=?, kind=?, definition=?`
	_, err = r.execute(ctx, true, query, filter.ID, filter.WorkspaceID, filter.UserID, filter.ReportID, filter.Name, filter.Kind, definitionJSON)
	mysqlErr, ok := err.(*mysql.MySQLError)
	if ok && mysqlErr.Number == errorCodeDuplicateEntry {
		return domain.ErrConflict
	}

	return err
}

func (r *filterRepository) Update(ctx context.Context, filter *domain.Filter, oldName string) error {
	definitionJSON, err := json.Marshal(filter.Definition)
	if err != nil {
		return err
	}

	query := `UPDATE filters SET name=?, definition=? WHERE name=?`
	_, err = r.execute(ctx, true, query, filter.Name, definitionJSON, oldName)
	mysqlErr, ok := err.(*mysql.MySQLError)
	if ok && mysqlErr.Number == errorCodeDuplicateEntry {
		return domain.ErrConflict
	}

	return err
}

func (r *filterRepository) Delete(ctx context.Context, filterID string) error {
	query := `DELETE FROM filters WHERE id=?`
	_, err := r.execute(ctx, true, query, filterID)
	mysqlErr, ok := err.(*mysql.MySQLError)
	if ok && mysqlErr.Number == errorCodeDuplicateEntry {
		return domain.ErrConflict
	}

	return err
}

func (r *filterRepository) execute(ctx context.Context, isFastRetry bool, query string, args ...interface{}) (sql.Result, error) {
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

func (r *filterRepository) fetch(ctx context.Context, isFastRetry bool, query string, args ...interface{}) ([]*domain.Filter, error) {
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

	result := []*domain.Filter{}
	for rows.Next() {
		filter := domain.Filter{}
		definitionJSON := sql.RawBytes{}
		err := rows.Scan(
			&filter.ID,
			&filter.WorkspaceID,
			&filter.UserID,
			&filter.ReportID,
			&filter.Name,
			&filter.Kind,
			&definitionJSON,
		)
		if err != nil {
			l.Error("couldn't scan row", zap.Error(err))

			return nil, err
		}

		switch filter.Kind {
		case domain.FilterKindIn:
			definition := utils.FilterOptions{}
			err := json.Unmarshal(definitionJSON, &definition)
			if err != nil {
				l.Error("couldn't unmarshal definition", zap.Error(err))

				return nil, err
			}

			filter.Definition = &definition

		default:
			l.Error("unsupported filter kind", zap.String("kind", string(filter.Kind)), zap.Int64("filterID", filter.ID))
		}

		result = append(result, &filter)
	}

	return result, nil
}
