package service

import (
	"fmt"
	"net/http"
	"os"
	"responses"
	"strconv"
	"strings"
	"sync/atomic"

	"golang.org/x/crypto/bcrypt"
)

var (
	globalExecutionSuccessMessage atomic.Value
	globalExecutionErrorMessage   atomic.Value
	ReqContactsTreshold           = 100
)

func GetRootPath() string {
	rootPath, _ := os.Getwd()
	return rootPath
}
func SetHeaderParameterJson(w http.ResponseWriter) {
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
}

func GetTokenHeader(authHeader string) (status int, message string, mobilePhone string) {
	splitToken := strings.Split(authHeader, "Bearer ")
	token := splitToken[len(splitToken)-1]

	sequel := fmt.Sprintf("select phone_number from users where token = '%s'", token)

	sqlRow, err := ExecuteChannelSqlRow(sequel)
	switch {
	case err != nil:
		return 508, err.Error(), ""
	default:
		_ = sqlRow.Scan(&mobilePhone)
		byetMobilePhone := []byte(mobilePhone)
		byteToken := []byte(token)
		authorized := (bcrypt.CompareHashAndPassword(byteToken, byetMobilePhone) == nil)
		if authorized {
			return http.StatusOK, "success", mobilePhone
		} else {
			return http.StatusUnauthorized, "unauthorized", ""
		}
	}
}

func StringtoInt(integer string) int {
	newInteger, _ := strconv.ParseInt(integer, 10, 0)
	return int(newInteger)
}
func GetDefaultMessage(status int, message string) responses.DefaultMessage {
	globalExecutionErrorMessage.Store(responses.DefaultMessage{status, message})
	return globalExecutionErrorMessage.Load().(responses.DefaultMessage)
}
func GetErrorMessageType(status int, message string) responses.DefaultMessage {
	return GetDefaultMessage(status, message)
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
