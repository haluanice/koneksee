package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"responses"
	"strconv"
	"strings"
	"sync/atomic"
)

var globalExecutionSuccessMessage atomic.Value
var globalExecutionErrorMessage atomic.Value

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
	err := ExecuteChanelSqlRow(sequel).Scan(&userId, &mobilePhone)
	status, message := CheckScanRowSQL(err)
	if status != 200 {
		return status, message, "", userId
	} else {
		return 200, "success", mobilePhone, userId
	}
}

func StringtoInt(integer string) int {
	newInteger, _ := strconv.ParseInt(integer, 10, 0)
	return int(newInteger)
}

func OutputError(status int, message string) string {
	globalExecutionErrorMessage.Store(responses.ErrorMessage{status, message})
	dataErrorMessage := globalExecutionErrorMessage.Load().(responses.ErrorMessage)
	output, _ := json.Marshal(dataErrorMessage)
	return string(output)
}

func OutputSuccess(status int, message string, v interface{}) string {
	output, _ := json.Marshal(responses.GeneralMsg{status, message, v})
	return string(output)
}
func DBErrorParser(err string) (string, int64) {
	Parts := strings.Split(err, ":")
	errorMessage := Parts[1]
	Code := strings.Split(Parts[0], "Error")
	errorCode, _ := strconv.ParseInt(Code[1], 10, 32)
	return errorMessage, errorCode
}
