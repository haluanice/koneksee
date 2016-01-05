package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"responses"
	"strconv"
	"strings"
	"sync/atomic"

	"golang.org/x/crypto/bcrypt"
)

var globalExecutionSuccessMessage atomic.Value
var globalExecutionErrorMessage atomic.Value

func SetHeaderParameter(w http.ResponseWriter) {
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
}

func GetTokenHeader(r *http.Request) (bool, int, string, string) {
	authHeader := r.Header.Get("Authorization")
	splitToken := strings.Split(authHeader, "Asolole ")
	token := splitToken[len(splitToken)-1]

	mobilePhone := ""
	sequel := fmt.Sprintf("select mobile_phone from users where token = '%s'", token)
	row := ExecuteChanelSqlRow(sequel).Scan(&mobilePhone)
	switch {
	case row != nil:
		messageArr := strings.Split(row.Error(), ": ")
		message := messageArr[len(messageArr)-1]
		return false, 500, message, ""
	case row == sql.ErrNoRows:
		return false, 404, "token not satisfied to any credential", ""
	default:
		hashedTokenBytes := []byte(token)
		mobileBytes := []byte(mobilePhone)
		switch bcrypt.CompareHashAndPassword(hashedTokenBytes, mobileBytes) == nil {
		case false:
			return false, 401, "invalid token", ""
		default:
			return true, 200, "welcome to the club", mobilePhone

		}
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
	output, _ := json.Marshal(responses.GeneralMessage{status, message, v})
	return string(output)
}
func DBErrorParser(err string) (string, int64) {
	Parts := strings.Split(err, ":")
	errorMessage := Parts[1]
	Code := strings.Split(Parts[0], "Error")
	errorCode, _ := strconv.ParseInt(Code[1], 10, 32)
	return errorMessage, errorCode
}
