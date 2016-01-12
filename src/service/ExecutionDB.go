// ExecutionDB
package service

import (
	"database/sql"

	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

var database *sql.DB

func NewDatabase(dbName string) {
	dataBaseCredentials := fmt.Sprintf("root:@/%s", dbName)
	database, _ = sql.Open("mysql", dataBaseCredentials)
}

func ExecSQL(sql string, channel chan ExecSQLType) chan ExecSQLType {
	exec, err := database.Exec(sql)
	channel <- ExecSQLType{exec, err}
	close(channel)
	return channel

}

func QuerySQL(sql string, channel chan QuerySQLType) chan QuerySQLType {
	query, err := database.Query(sql)
	channel <- QuerySQLType{query, err}
	close(channel)
	return channel
}

func QueryRowSQL(sql string, channel chan *sql.Row) chan *sql.Row {
	query := database.QueryRow(sql)
	channel <- query
	close(channel)
	return channel
}
