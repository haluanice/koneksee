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

func ExecSQL(sql string, c chan ExecSQLType) {
	exec, err := database.Exec(sql)
	c <- ExecSQLType{exec, err}
}

func QuerySQL(sql string, c chan QuerySQLType) {
	query, err := database.Query(sql)
	c <- QuerySQLType{query, err}
}

func QueryRowSQL(sql string, c chan *sql.Row) {
	query := database.QueryRow(sql)
	c <- query
}
