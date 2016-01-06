// UserController
package controller

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"runtime"
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
	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	service.SetHeaderParameter(w)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		cases := r.FormValue("phone_list")
		sequel := fmt.Sprintf("select user_id, user_name, mobile_phone, profile_picture from users where mobile_phone in "+
			"(%s) and mobile_phone not in "+
			" (select friend from users u join friends_relationship fr " +
			" on u.`mobile_phone` = fr.`user` where u.`mobile_phone` = '%s' )", cases, mobilePhone)
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

			fmt.Fprintf(w, service.OutputSuccess(200, "success", resChanUsers))
		}
	}
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	status, message, _, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
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
	status, message, newId := service.ExecuteInsertSqlResult(SQL)
	w.WriteHeader(status)

	switch {
	case status != 200:
		fmt.Fprintln(w, service.OutputError(status, message))
	default:
		userCreated := responses.UserCreated{int(newId), NewUser.UserName, NewUser.MobilePhone, fmt.Sprintf("%s", hashedPassword)}
		fmt.Fprintf(w, service.OutputSuccess(200, "user created", userCreated))
	}
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	service.SetHeaderParameter(w)

	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		updateUserName := requests.UpdateUserName{}
		service.DecodeJson(&updateUserName, r.Body)
		userName := updateUserName.UserName
		SQL := fmt.Sprintf("UPDATE users SET user_name='%s' WHERE mobile_phone = '%s'", userName, mobilePhone)
		status, message := updateUserExecutor(SQL)
		if status != 200 {
			fmt.Fprintf(w, service.OutputError(status, message))
		}else{
			fmt.Fprintf(w, service.OutputSuccess(status, message, requests.UserUpdateType{userName, mobilePhone}))
		}
	}
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	service.SetHeaderParameter(w)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		SQL := fmt.Sprintf("Delete FROM users WHERE mobile_phone = '%s'", mobilePhone)

		status, message := service.ExecuteChanelSqlResult(SQL)
		w.WriteHeader(status)

		switch {
		case status != 200:
			fmt.Fprintln(w, service.OutputError(status, message))
		default:
			fmt.Fprintf(w, service.OutputSuccess(status, "user deleted", requests.UserMobilePhone{mobilePhone}))
		}
	}
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	service.SetHeaderParameter(w)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		file, header, err := r.FormFile("file")

		if err != nil {
			printUploadError(w, err)
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
		errFindOrCreateDir := findOrCreateDirectory(targetPath)
		if errFindOrCreateDir != nil {
			printUploadError(w, err)
			return
		}

		pathFile, nameFile, err := service.GenerateNewPath(targetPath, fileType)

		if err != nil {
			printUploadError(w, err)
			return
		}

		out, err := service.CreateFile(pathFile)

		if err != nil {
			printUploadError(w, err)
			return
		}

		go service.ExecuteCopyFile(out, file)
		runtime.Gosched()

		rcvChannelCopyFile := <-service.ChanelCopyFile
		out.Close()
		file.Close()

		errCopy := rcvChannelCopyFile.Err
		if errCopy != nil {
			printUploadError(w, err)
			return
		}

		newFilePath := fmt.Sprintf("%s%s", staticPath, nameFile)
		sequel := fmt.Sprintf("UPDATE users SET profile_picture='%s' WHERE mobile_phone = '%s'", newFilePath, mobilePhone)
		status, message := updateUserExecutor(sequel)
		if status != 200 {
			fmt.Fprintf(w, service.OutputError(status, message))
		}else{
			fmt.Fprintf(w, service.OutputSuccess(status, message, requests.UserProfilePictureType{mobilePhone, newFilePath}))
		}
	}
}

func BlockFriend(w http.ResponseWriter, r *http.Request){
	service.SetHeaderParameter(w)
	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	block := 0
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		friendMobilePhone := decodeActionFriendMobilePhone(r.Body)
		
		status, message := hideOrBlockFriend(mobilePhone, friendMobilePhone, block)
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))

	}
}

func HideFriend(w http.ResponseWriter, r *http.Request){
	service.SetHeaderParameter(w)
	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	hide := 1
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		friendMobilePhone := decodeActionFriendMobilePhone(r.Body)
		
		status, message := hideOrBlockFriend(mobilePhone, friendMobilePhone, hide)
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))

	}
}

func UnBlockFriend(w http.ResponseWriter, r *http.Request){
	block := 0
	service.SetHeaderParameter(w)
	
	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		friendMobilePhone := decodeActionFriendMobilePhone(r.Body)
		SQL := fmt.Sprintf("Delete FROM friends_relationship WHERE user = '%s' and friend = '%s' and status = %v", 
			mobilePhone, friendMobilePhone, block)

		status, message := service.ExecuteChanelSqlResult(SQL)
		w.WriteHeader(status)
		fmt.Fprintln(w, service.OutputError(status, message))
	}
}

func UnHideFriend(w http.ResponseWriter, r *http.Request){
	hide := 1
	service.SetHeaderParameter(w)

	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		friendMobilePhone := decodeActionFriendMobilePhone(r.Body)
		SQL := fmt.Sprintf("Delete FROM friends_relationship WHERE user = '%s' and friend = '%s' and status = %v", 
			mobilePhone, friendMobilePhone, hide)

		status, message := service.ExecuteChanelSqlResult(SQL)
		w.WriteHeader(status)
		fmt.Fprintln(w, service.OutputError(status, message))
	}
}

//User Controller Private Function

func decodeActionFriendMobilePhone(body io.ReadCloser) string{
	actionToFriend := requests.ActionToFriend{}
	service.DecodeJson(&actionToFriend, body)
	return actionToFriend.MobilePhone
}

func hideOrBlockFriend(mobilePhone string, friendMobilePhone string, status int) (int, string){
		friendUserId := 0
		
		sequel := fmt.Sprintf("SELECT user_id FROM users where mobile_phone = '%s'", friendMobilePhone)
		err := service.ExecuteChanelSqlRow(sequel).Scan(&friendUserId)
		statusRow, messageRow := service.CheckScanRowSQL(err)
		switch{
		case statusRow == 404 :
			return statusRow, "phone number doesn't exists"
		case statusRow != 200:
			return statusRow, messageRow	
		default:
			sequel := fmt.Sprintf("INSERT INTO friends_relationship SET user =  '%s', friend = '%s', status = %v", 
				mobilePhone, friendMobilePhone, status)

			statusInsert, messageInsert := service.ExecuteChanelSqlResult(sequel)
			if statusInsert == 409 {
				sequel := fmt.Sprintf("UPDATE friends_relationship SET status =  %v where user = '%s' and friend = '%s'", 
				status, mobilePhone, friendMobilePhone)

				statusUpdate, messageUpdate := service.ExecuteChanelSqlResult(sequel)
				return statusUpdate, messageUpdate
			}			
			return statusInsert, messageInsert
		}
}

func mapUsers(rows *sql.Rows, chanUsers chan responses.GeneralArrMsg) chan responses.GeneralArrMsg {
	users := atomicUsers(responses.GeneralArrMsg{})
	chanUser := make(chan requests.User)
	runtime.Gosched()

	for rows.Next() {
		go assignedMapedUsers(rows, chanUser)
		resChanUser := <-chanUser
		users.Datas = append(users.Datas, resChanUser)
		runtime.Gosched()
	}
	chanUsers <- users
	return chanUsers
}

func assignedMapedUsers(rows *sql.Rows, chanUser chan requests.User) chan requests.User {
	runtime.Gosched()
	user := atomicUser(requests.User{})
	rows.Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
	chanUser <- user
	return chanUser
}

func insertTokenToUsersTable(token string, mobilePhone string) (int, string) {
	SQL := fmt.Sprintf("UPDATE users SET token = '%s' where mobile_phone = '%s'", token, mobilePhone)
	
	status, message := service.ExecuteChanelSqlResult(SQL)
	return status, message
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


func findOrCreateDirectory(targetPath string) error {
	_, err := os.Stat(targetPath)
	if err != nil || os.IsNotExist(err) {
		err := os.Mkdir(targetPath, 0777)
		return err
	}
	return nil
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


func updateUserExecutor(sequel string) (int, string) {
	status, message := service.ExecuteChanelSqlResult(sequel)
	switch {
	case status == 404:
		return 422, "data not updated"
	case status == 200:
		return 200, "user updated"
	default:
		return status, message
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


func printUploadError(w http.ResponseWriter, err error) {
	if err != nil {
		w.WriteHeader(406)
		fmt.Fprintf(w, service.OutputError(406, err.Error()))
	}
}

