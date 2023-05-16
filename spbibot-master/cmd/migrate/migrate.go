package main

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	
)

func main() {
	baseProvider, err := config.NewDotenvProvider("./env/base.env")
	if err != nil {
		panic(err.Error())
	}

	conf, err := config.NewBotConfig(baseProvider)
	if err != nil {
		panic(err.Error())
	}

	mysqlConn, err := db.InitDB("mysql", conf.DB)
	if err != nil {
		panic(err.Error())
	}

	defer func() {
		err := mysqlConn.Close()
		if err != nil {
			panic(err.Error())
		}
	}()

	tx, err := mysqlConn.Begin()
	if err != nil {
		panic(err.Error())
	}

	addColumnIsActiveToUsers(tx)
	addColumnIsActiveToWorkspaces(tx)

	err = tx.Commit()
	if err != nil {
		panic(err.Error())
	}
}

func addColumnIsActiveToUsers(tx *sql.Tx) {
	_, err := tx.Exec("ALTER TABLE users " +
		"ADD COLUMN isActive BOOL NOT NULL DEFAULT TRUE AFTER id")
	if err != nil {
		err2 := tx.Rollback()
		if err2 != nil {
			panic(err2.Error())
		}

		panic(err.Error())
	}
}

func addColumnIsActiveToWorkspaces(tx *sql.Tx) {
	_, err := tx.Exec("ALTER TABLE workspaces " +
		"ADD COLUMN isActive BOOL NOT NULL DEFAULT TRUE AFTER id")
	if err != nil {
		err2 := tx.Rollback()
		if err2 != nil {
			panic(err2.Error())
		}

		panic(err.Error())
	}
}
