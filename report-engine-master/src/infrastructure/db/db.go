package db

import (
	"database/sql"
	"fmt"
	"net/url"

)

// InitDB initializes DB
func InitDB(driver string, d *config.DatabaseConfig) (*sql.DB, error) {
	// TODO: update this method so it creates and configures container for repositories

	connectionString := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s%s",
		d.Username,
		d.UserPwd,
		d.Host,
		d.Port,
		d.Name,
		fmt.Sprintf("?loc=UTC&parseTime=true&time_zone=%v", url.QueryEscape("'+00:00'")),
	)

	dbConn, err := sql.Open(driver, connectionString)
	if err != nil {
		return nil, err
	}

	err = dbConn.Ping()
	if err != nil {
		return nil, err
	}

	return dbConn, nil
}
