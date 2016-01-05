// UserController
package controller

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"encoding/json"
	"net/http"
	"requests"
	"responses"
	"service"

	"golang.org/x/crypto/bcrypt"
)

var globalExecutionUser atomic.Value
var globalExecutionUsers atomic.Value

func GetUsers(w http.ResponseWriter, r *http.Request) {
	status, message, _ := service.GetTokenHeader(r)
	service.SetHeaderParameter(w)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		cases := r.FormValue("phone_list")

		sequel := "select user_id, user_name, mobile_phone, profile_picture from users where mobile_phone in (" + cases + ")"
		if cases == "" {
			sequel = "select user_id, user_name, mobile_phone, profile_picture from users"
		}
		rows, err := service.ExecuteChanelSqlRows(sequel)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, service.OutputError(500, err.Error()))
		} else {
			chanUsers := make(chan responses.GeneralArrMsg)
			go mapUsers(rows, chanUsers)
			resChanUsers := <-chanUsers

			fmt.Fprintf(w, service.OutputSuccess(status, message, resChanUsers))
		}
	}
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	status, message, _ := service.GetTokenHeader(r)
	service.SetHeaderParameter(w)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		urlParams := r.URL.Query()
		id := urlParams.Get(":id")
		user := atomicUser(requests.User{})
		sequel := fmt.Sprintf("select user_id, user_name, mobile_phone, profile_picture from users where user_id = %s", id)
		err := service.ExecuteChanelSqlRow(sequel).Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
		status, message := service.CheckScanRowSQL(err)
		if status != 200 {
			w.WriteHeader(status)
			fmt.Fprintf(w, service.OutputError(status, message))
		} else {
			w.WriteHeader(status)
			fmt.Fprintf(w, service.OutputSuccess(status, message, user))
		}
	}
}

func GenerateNewToken(w http.ResponseWriter, r *http.Request) {
	service.SetHeaderParameter(w)

	userTokenJson := requests.UserTokenJson{}
	service.DecodeJson(&userTokenJson, r.Body)

	user_id := 0
	sequel := fmt.Sprintf("select user_id from users where mobile_phone = %s", userTokenJson.MobilePhone)
	err := service.ExecuteChanelSqlRow(sequel).Scan(&user_id)
	status, message := service.CheckScanRowSQL(err)
	if status != 200 {
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	} else {
		mobilePhone := userTokenJson.MobilePhone
		resultHashed := hashedMobileNumber(mobilePhone)
		statusInsertToken, messageInsertToken := insertTokenToUsersTable(resultHashed, mobilePhone)
		if status != 200 {
			w.WriteHeader(status)
			fmt.Fprintln(w, service.OutputError(statusInsertToken, messageInsertToken))
		} else {
			w.WriteHeader(status)
			userToken := requests.UserToken{user_id, resultHashed}
			fmt.Fprintf(w, service.OutputSuccess(statusInsertToken, messageInsertToken, userToken))
		}
	}
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	service.SetHeaderParameter(w)
	NewUser := atomicUser(newUserJson(r.Body))

	mobileBytes := []byte(NewUser.MobilePhone)
	hashedPassword, err := bcrypt.GenerateFromPassword(mobileBytes, 10)
	if err != nil {
		fmt.Fprintf(w, service.OutputError(500, err.Error()))
		return
	}

	SQL := fmt.Sprintf("INSERT INTO users SET user_name='%s', mobile_phone='%s'"+
		", token = '%s'", NewUser.UserName, NewUser.MobilePhone, hashedPassword)
	create, err := service.ExecuteChanelSqlResult(SQL)

	switch {
	case err != nil:
		status, message := service.ErrorMessageDB(err.Error())
		w.WriteHeader(status)
		fmt.Fprintln(w, service.OutputError(status, message))
	default:
		affectedRows, _ := create.RowsAffected()
		switch affectedRows < int64(1) {
		case true:
			fmt.Fprintf(w, service.OutputError(500, "data not created"))
		case false:
			newId, _ := create.LastInsertId()

			userCreated := responses.UserCreated{int(newId), NewUser.UserName, NewUser.MobilePhone, fmt.Sprintf("%s", hashedPassword)}

			fmt.Fprintf(w, service.OutputSuccess(200, "user created", userCreated))
		}
	}
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	service.SetHeaderParameter(w)

	status, message, mobilePhone := service.GetTokenHeader(r)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		updateUserName := requests.UpdateUserName{}
		service.DecodeJson(&updateUserName, r.Body)
		SQL := "UPDATE users SET user_name='" + updateUserName.UserName + "'"
		updateUserExecutor(w, r, SQL, mobilePhone)
	}

}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	status, message, mobilePhone := service.GetTokenHeader(r)
	service.SetHeaderParameter(w)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		SQL := fmt.Sprintf("Delete FROM users WHERE mobile_phone = '%s'", mobilePhone)

		_, err := service.ExecuteChanelSqlResult(SQL)
		switch {
		case err != nil:
			fmt.Fprintf(w, service.OutputError(404, "user not found"))
		default:
			fmt.Fprintf(w, service.OutputSuccess(200, "deleted", requests.UserMobilePhone{mobilePhone}))
		}
	}
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	status, message, mobilePhone := service.GetTokenHeader(r)
	service.SetHeaderParameter(w)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		file, header, err := r.FormFile("file")

		if err != nil {
			printError(w, err)
			return
		}

		fileName := header.Filename
		fileTypeArr := strings.Split(fileName, ".")
		fileType := fileTypeArr[len(fileTypeArr)-1]

		if !allowedImageType(fileType) {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			infoMessage := "type is not allowed"
			infoError := infoMessage
			fmt.Fprintf(w, service.OutputError(415, infoError))
			return
		}

		pwd, _ := os.Getwd()

		staticPath := "/static/"

		targetPath := pwd + staticPath
		isDirectoryExists(targetPath)

		pathFile, nameFile, err := service.GenerateNewPath(targetPath, fileType)

		if err != nil {
			printError(w, err)
			return
		}

		out, err := service.CreateFile(pathFile)

		if err != nil {
			printError(w, err)
			return
		}

		go service.ExecuteCopyFile(out, file)
		runtime.Gosched()

		rcvChannelCopyFile := <-service.ChannelCopyFile
		out.Close()
		file.Close()

		errCopy := rcvChannelCopyFile.Err
		if errCopy != nil {
			printError(w, err)
			return
		}

		newFilePath := fmt.Sprintf("%s%s", staticPath, nameFile)
		sequel := fmt.Sprintf("UPDATE users SET profile_picture='%s'", newFilePath)
		updateUserExecutor(w, r, sequel, mobilePhone)
	}
}

//User Controller Private Function
func mapUsers(rows *sql.Rows, chanUsers chan responses.GeneralArrMsg) {
	users := atomicUsers(responses.GeneralArrMsg{})
	chanUser := make(chan requests.User)
	runtime.Gosched()

	for rows.Next() {
		go assignedMapedUsers(rows, chanUser)
		resChanUser := <-chanUser
		users.Datas = append(users.Datas, resChanUser)
	}

	users.Status = 200
	users.Message = "success"
	chanUsers <- users
}

func assignedMapedUsers(rows *sql.Rows, chanUser chan requests.User) {
	runtime.Gosched()
	user := atomicUser(requests.User{})
	rows.Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
	chanUser <- user
}

func insertTokenToUsersTable(token string, mobilePhone string) (int, string) {
	SQL := fmt.Sprintf("UPDATE users SET token = '%s' where mobile_phone = '%s'", token, mobilePhone)
	_, err := service.ExecuteChanelSqlResult(SQL)
	switch {
	case err != nil:
		status, message := service.ErrorMessageDB(err.Error())
		return status, message
	default:
		return 200, "token created"
	}
}

func hashedMobileNumber(mobilePhone string) string {
	mobileBytes := []byte(mobilePhone)
	hashedPassword, err := bcrypt.GenerateFromPassword(mobileBytes, 10)
	if err != nil {
		//fmt.Fprintf(w, service.OutputError(500, err.Error()))
		return err.Error()
	}
	hashedResult := fmt.Sprintf("%s", hashedPassword)
	return hashedResult
}

func allowedImageType(contentType string) bool {
	m := map[string]bool{
		"png":  true,
		"jpeg": true,
		"jpg":  true,
		"gif":  true,
	}
	_, isImageAllowed := m[contentType]
	return isImageAllowed
}

func printError(w http.ResponseWriter, err error) {
	if err != nil {
		w.WriteHeader(406)
		fmt.Fprintf(w, service.OutputError(406, err.Error()))
	}
}

func isDirectoryExists(targetPath string) {
	_, err := os.Stat(targetPath)
	if err != nil || os.IsNotExist(err) {
		os.Mkdir(targetPath, 0777)
	}
}

func updateUserExecutor(w http.ResponseWriter, r *http.Request, sequel string, mobilePhone string) {
	user := atomicUser(requests.User{})

	sequel += fmt.Sprintf(" WHERE mobile_phone = '%s'", mobilePhone)

	update, err := service.ExecuteChanelSqlResult(sequel)
	if err != nil {
		status, message := service.ErrorMessageDB(err.Error())
		w.WriteHeader(status)
		fmt.Fprintln(w, service.OutputError(status, message))
		return
	}

	affectedRows, _ := update.RowsAffected()

	switch affectedRows < int64(1) {
	case true:
		w.WriteHeader(422)
		fmt.Fprintf(w, service.OutputError(422, "data not updated"))
	case false:
		showUserSQL := fmt.Sprintf("SELECT user_id, user_name, mobile_phone, profile_picture "+
			"FROM users WHERE mobile_phone = '%s'", mobilePhone)
		err := service.ExecuteChanelSqlRow(showUserSQL).Scan(&user.UserId,
			&user.UserName, &user.MobilePhone, &user.ProfilePicture)
		status, message := service.CheckScanRowSQL(err)
		if status != 200 {
			w.WriteHeader(status)
			fmt.Fprintf(w, service.OutputError(status, message))
		} else {
			w.WriteHeader(status)
			fmt.Fprintf(w, service.OutputSuccess(status, "user updated", user))
		}
	}
}

func atomicUser(user requests.User) requests.User {
	globalExecutionUser.Store(user)
	dataUser := globalExecutionUser.Load().(requests.User)
	return dataUser
}

func atomicUsers(users responses.GeneralArrMsg) responses.GeneralArrMsg {
	globalExecutionUsers.Store(users)
	dataUsers := globalExecutionUsers.Load().(responses.GeneralArrMsg)
	return dataUsers
}

func newUserJson(body io.ReadCloser) requests.User {
	decoder := json.NewDecoder(body)
	NewUser := requests.User{}
	decoder.Decode(&NewUser)
	return NewUser
}

func getUserId(r http.Request) int {
	urlParams := r.URL.Query()
	idString := urlParams.Get(":id")
	idInt, _ := strconv.Atoi(idString)
	return idInt
}
