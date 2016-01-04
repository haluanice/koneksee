package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"model"
	"net/http"
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

func GetTokenHeader(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	splitToken := strings.Split(authHeader, "Asolole ")
	token := splitToken[len(splitToken)-1]

	mobilePhone := ""
	sequel := fmt.Sprintf("select mobile_phone from users where token = '%s'", token)
	row := ExecuteChanelSqlRow(sequel).Scan(&mobilePhone)
	switch {
	case row == sql.ErrNoRows:
		fmt.Println("no rows")
		return false
	case row != nil:
		fmt.Println(row.Error())
		return false
	default:
		hashedTokenBytes := []byte(token)
		mobileBytes := []byte(mobilePhone)
		return bcrypt.CompareHashAndPassword(hashedTokenBytes, mobileBytes) == nil
	}
}

func StringtoInt(integer string) int {
	newInteger, _ := strconv.ParseInt(integer, 10, 0)
	return int(newInteger)
}

func OutputError(status int, message string) string {
	globalExecutionErrorMessage.Store(model.ErrorMessage{status, message})
	dataErrorMessage := globalExecutionErrorMessage.Load().(model.ErrorMessage)
	output, _ := json.Marshal(dataErrorMessage)
	return string(output)
}

func OutputSuccess(status int, message string, user model.User) string {
	globalExecutionSuccessMessage.Store(model.SuccessMessage{status, message, user})
	SuccessMessageMessage := globalExecutionSuccessMessage.Load().(model.SuccessMessage)
	output, _ := json.Marshal(SuccessMessageMessage)
	return string(output)
}

func DBErrorParser(err string) (string, int64) {
	Parts := strings.Split(err, ":")
	errorMessage := Parts[1]
	Code := strings.Split(Parts[0], "Error")
	errorCode, _ := strconv.ParseInt(Code[1], 10, 32)
	return errorMessage, errorCode
}
