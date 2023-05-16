package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	
)

// UserRepository represent the data-struct for msql user repoitory
type UserRepository struct {
	conn   *sql.DB
	logger *zap.Logger
}

// NewMysqlUserRepository will create an object that represent the user.Repository interface
func NewMysqlUserRepository(conn *sql.DB, l *zap.Logger) domain.UserRepository {
	return &UserRepository{
		conn:   conn,
		logger: l,
	}
}

func (mysqlUserRepo *UserRepository) fetch(ctx context.Context, isFastRetry bool, query string, args ...interface{}) ([]domain.User, error) {
	l := utils.
		WithContext(ctx, mysqlUserRepo.logger).
		With(zap.String("query", query))

	rows, err := queryContextWithRetry(ctx, isFastRetry, mysqlUserRepo.logger, mysqlUserRepo.conn, query, args...)
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

	res := make([]domain.User, 0)
	for rows.Next() {
		user := domain.User{}
		err = rows.Scan(
			&user.WorkspaceID,
			&user.ID,
			&user.IsActive,
			&user.HashID,
			&user.AccessToken,
			&user.RefreshToken,
		)
		if err != nil {
			l.Error("couldn't scan row", zap.Error(err))

			return nil, err
		}

		res = append(res, user)
	}

	return res, nil
}

// GetByID method returns a user by id from storage
func (mysqlUserRepo *UserRepository) GetByID(ctx context.Context, id *domain.SlackUserID) (domain.User, error) {
	query := `SELECT workspaceID, id, isActive, hashID, accessToken, refreshToken FROM users WHERE (workspaceID=?) AND (id=?)`
	list, err := mysqlUserRepo.fetch(ctx, true, query, id.WorkspaceID, id.ID)
	if err != nil {
		return domain.User{}, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return domain.User{}, domain.ErrNotFound
}

// GetByHash method returns a user by hash from storage
func (mysqlUserRepo *UserRepository) GetByHash(ctx context.Context, hash string) (domain.User, error) {
	query := `SELECT workspaceID, id, isActive, hashID, accessToken, refreshToken FROM users WHERE hashID=?`
	list, err := mysqlUserRepo.fetch(ctx, true, query, hash)
	if err != nil {
		return domain.User{}, err
	}

	if len(list) > 0 {
		return list[0], nil
	}

	return domain.User{}, domain.ErrNotFound
}

// Store method inserts the user to mysql db
func (mysqlUserRepo *UserRepository) Store(ctx context.Context, user *domain.User) error {
	query := `INSERT users SET workspaceID=?, id=?, isActive=?, email=?, hashID=?, accessToken=?, refreshToken=?`
	_, err := mysqlUserRepo.execute(ctx, true, query, user.WorkspaceID, user.ID, true, user.Email, user.HashID, user.AccessToken, user.RefreshToken)

	return err
}

// Deactivate method deactivates user
func (mysqlUserRepo *UserRepository) Deactivate(ctx context.Context, id *domain.SlackUserID) error {
	query := `UPDATE users SET isActive = FALSE WHERE workspaceID=? AND id=?`
	_, err := mysqlUserRepo.execute(ctx, true, query, id.WorkspaceID, id.ID)

	return err
}

// Reactivate method make user active again
func (mysqlUserRepo *UserRepository) Reactivate(ctx context.Context, id *domain.SlackUserID) error {
	query := `UPDATE users SET isActive = TRUE WHERE workspaceID=? AND id=?`
	_, err := mysqlUserRepo.execute(ctx, true, query, id.WorkspaceID, id.ID)

	return err
}

// Update method updates the user to mysql db
func (mysqlUserRepo *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `UPDATE users set accessToken=?, refreshToken=? WHERE (workspaceID=?) AND (id=?)`
	res, err := mysqlUserRepo.execute(ctx, true, query, user.AccessToken, user.RefreshToken, user.WorkspaceID, user.ID)
	if err != nil {
		return err
	}

	affect, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affect != 1 {
		err = fmt.Errorf("weird  Behavior. Total Affected: %d", affect)

		return err
	}

	return nil
}

func (mysqlUserRepo *UserRepository) MigrateEnterpriseUserToUseTeamID(ctx context.Context, user *domain.User) error {
	if strings.HasPrefix(user.ID, "W") {
		selectEnterpriseUserQuery := `SELECT workspaceID, id, isActive, hashID, accessToken, refreshToken FROM users WHERE (users.id=?)`
		enterpriseUser, err := mysqlUserRepo.fetch(ctx, true, selectEnterpriseUserQuery, user.ID)
		if err != nil {
			return err
		}
		if enterpriseUser[0].WorkspaceID == user.WorkspaceID {
			return nil
		}
		mysqlWorkspaceRepo := NewMysqlWorkspaceRepository(mysqlUserRepo.conn, mysqlUserRepo.logger)
		workspaceData, err := mysqlWorkspaceRepo.GetByID(ctx, enterpriseUser[0].WorkspaceID)
		if err != nil {
			return err
		}

		newWorkspace := domain.Workspace{
			ID:             user.WorkspaceID,
			IsActive:       workspaceData.IsActive,
			BotAccessToken: workspaceData.BotAccessToken,
		}

		tx, err := mysqlUserRepo.conn.Begin()
		if err != nil {
			return tx.Rollback()
		}

		upsertingErr := mysqlWorkspaceRepo.Upsert(ctx, &newWorkspace)
		if upsertingErr != nil {
			return upsertingErr
		}

		insertNewEnterpriseUserQuery := `UPDATE users SET workspaceID=?  WHERE id=? and workspaceID=?`
		_, err1 := mysqlUserRepo.execute(ctx, true, insertNewEnterpriseUserQuery, newWorkspace.ID, user.ID, user.WorkspaceID)

		if err1 != nil {
			return tx.Rollback()
		}
		return tx.Commit()
	}
	return nil
}

func (mysqlUserRepo *UserRepository) execute(ctx context.Context, isFastRetry bool, query string, args ...interface{}) (sql.Result, error) {
	l := utils.
		WithContext(ctx, mysqlUserRepo.logger).
		With(zap.String("query", query))

	stmt, err := prepareContextWithRetry(ctx, isFastRetry, mysqlUserRepo.logger, mysqlUserRepo.conn, query)
	if err != nil {
		l.Error("couldn't create prepared statement", zap.Error(err))

		return nil, err
	}

	res, err := execContextWithRetry(ctx, isFastRetry, mysqlUserRepo.logger, stmt, args...)
	if err != nil {
		l.Error("couldn't execute prepared statement", zap.Error(err))

		return nil, err
	}

	return res, nil
}
