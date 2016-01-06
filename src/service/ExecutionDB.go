// ExecutionDB
package service

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var database = databaseFire()

func databaseFire() *sql.DB {
	database, _ := sql.Open("mysql", "root:@/koneksee")
	return database
}

func ExecSQL(sql string, chanel chan ExecSQLType) chan ExecSQLType {
	exec, err := database.Exec(sql)
	chanel <- ExecSQLType{exec, err}
	return chanel

}

func QuerySQL(sql string, chanel chan QuerySQLType) chan QuerySQLType {
	query, err := database.Query(sql)
	chanel <- QuerySQLType{query, err}
	return chanel
}

func QueryRowSQL(sql string, chanel chan *sql.Row) chan *sql.Row {
	query := database.QueryRow(sql)
	chanel <- query
	return chanel
}
