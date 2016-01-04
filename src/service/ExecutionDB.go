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

func ExecSQL(sql string, c chan sql.Result) {
	exec, _ := database.Exec(sql)
	c <- exec
}

func QuerySQL(sql string, c chan *sql.Rows) {
	query, _ := database.Query(sql)
	c <- query
}

func QueryRowSQL(sql string, c chan *sql.Row) {
	query := database.QueryRow(sql)
	c <- query
}
