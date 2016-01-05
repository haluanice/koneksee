// Model
package model

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type ExecSQLType struct {
	SqlResult sql.Result
	Err       string
}

type Job struct {
	AffectedRow  bool
	LastInsertId int64
}
