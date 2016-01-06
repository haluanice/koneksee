// ResponseService
package service

import (
	"database/sql"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func ErrorMessageDB(errorMessage string) (int, string) {
	messageArr := strings.Split(errorMessage, ": ")
	message := messageArr[len(messageArr)-1]
	status := 500
	isDuplicate := messageArr[0] == "Error 1062"
	if isDuplicate {
		status = 409
	}
	return status, message
}

func CheckScanRowSQL(err error) (int, string) {
	switch {
	case err == sql.ErrNoRows:
		return 404, "data not found"
	case err != nil:
		statusDB, messageDB := ErrorMessageDB(err.Error())
		return statusDB, messageDB
	default:
		return 200, "success"
	}
}
