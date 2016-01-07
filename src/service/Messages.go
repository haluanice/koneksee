package service

import (
	"fmt"
	"net/http"
	"responses"
	"strconv"
	"strings"
	"sync/atomic"
)

var (
	globalExecutionSuccessMessage atomic.Value
	globalExecutionErrorMessage   atomic.Value
)

func SetHeaderParameter(w http.ResponseWriter) {
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
}

func GetTokenHeader(authHeader string) (int, string, string, int) {
	splitToken := strings.Split(authHeader, "Asolole ")
	token := splitToken[len(splitToken)-1]

	mobilePhone := ""
	userId := 0
	sequel := fmt.Sprintf("select user_id, mobile_phone from users where token = '%s'", token)

	sqlRow, err := ExecuteChannelSqlRow(sequel)
	switch {
	case err != nil:
		return 508, err.Error(), "", userId
	default:
		errSqlRow := sqlRow.Scan(&userId, &mobilePhone)
		status, _ := CheckScanRowSQL(errSqlRow)
		if status != 200 {
			return 401, "unauthorized", "", userId
		} else {
			return http.StatusOK, "success", mobilePhone, userId
		}
	}
}

func StringtoInt(integer string) int {
	newInteger, _ := strconv.ParseInt(integer, 10, 0)
	return int(newInteger)
}
func GetErrorMessageType(status int, message string) responses.ErrorMessage {
	globalExecutionErrorMessage.Store(responses.ErrorMessage{status, message})
	return globalExecutionErrorMessage.Load().(responses.ErrorMessage)
}

func GetGeneralMsgType(status int, message string, v interface{}) responses.GeneralMsg {
	return responses.GeneralMsg{status, message, v}
}

func DBErrorParser(err string) (string, int64) {
	Parts := strings.Split(err, ":")
	errorMessage := Parts[1]
	Code := strings.Split(Parts[0], "Error")
	errorCode, _ := strconv.ParseInt(Code[1], 10, 32)
	return errorMessage, errorCode
}
