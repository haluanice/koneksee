// UserController
package controller

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"encoding/json"
	"net/http"
	"requests"
	"responses"
	"service"

	"github.com/drone/routes"
	"golang.org/x/crypto/bcrypt"
)

var (
	globalExecutionUser  atomic.Value
	globalExecutionUsers atomic.Value
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	_, _, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	contactListJson := requests.ContactList{}
	service.DecodeJson(&contactListJson, r.Body)

	chanContactString := mapContactListJson(contactListJson)
	contact := <-chanContactString
	close(chanContactString)

	subQuery := fmt.Sprintf(" mobile_phone NOT IN "+
		" (SELECT friend_mobile_phone FROM users u JOIN friends_relationship fr "+
		" ON u.`mobile_phone` = fr.`user_mobile_phone` WHERE u.`mobile_phone` = '%s' )", mobilePhone)
	sequel := fmt.Sprintf("SELECT user_id, user_name, mobile_phone, profile_picture FROM users WHERE mobile_phone IN "+
		"(%s) AND %s", contact, subQuery)
	rows, err := service.ExecuteChannelSqlRows(sequel)
	if err != nil {
		w.WriteHeader(500)
		routes.ServeJson(w, service.GetErrorMessageType(500, err.Error()))
	} else {
		chanUsers := make(chan responses.GeneralArrMsg)
		go mapUsers(rows, chanUsers)
		select {
		case resChanUsers := <-chanUsers:
			close(chanUsers)
			w.WriteHeader(http.StatusOK)
			routes.ServeJson(w, service.GetGeneralMsgType(http.StatusOK, "success", resChanUsers))
		case <-service.TimeOutInMilis(service.GlobalTimeOutDB):
			close(chanUsers)
			w.WriteHeader(508)
			routes.ServeJson(w, service.GetErrorMessageType(508, "request time out"))
		}
	}
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	urlParams := r.URL.Query()
	id := urlParams.Get(":id")
	user := atomicUser(requests.User{})
	condition := fmt.Sprintf("user_id = %s", id)
	sequel := service.SelectQuery("user_id, user_name, mobile_phone, profile_picture", "users", condition)
	sqlRow, err := service.ExecuteChannelSqlRow(sequel)
	switch {
	case err != nil:
		w.WriteHeader(508)
		routes.ServeJson(w, service.GetErrorMessageType(508, err.Error()))
	default:
		errSqlRow := sqlRow.Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
		statusRow, messageRow := service.CheckScanRowSQL(errSqlRow)
		w.WriteHeader(statusRow)
		if statusRow != 200 {
			routes.ServeJson(w, service.GetErrorMessageType(statusRow, messageRow))
		} else {
			routes.ServeJson(w, service.GetGeneralMsgType(statusRow, messageRow, user))
		}
	}

}

func GenerateNewToken(w http.ResponseWriter, r *http.Request) {
	userTokenJson := requests.MobilePhoneJson{}
	service.DecodeJson(&userTokenJson, r.Body)
	user_id := 0
	sqlRow, err := service.ExecuteChannelSqlRow(getUserIdSQL(userTokenJson.MobilePhone))
	switch {
	case err != nil:
		w.WriteHeader(508)
		routes.ServeJson(w, service.GetErrorMessageType(508, err.Error()))
	default:
		errSqlRow := sqlRow.Scan(&user_id)
		status, message := service.CheckScanRowSQL(errSqlRow)
		if status != 200 {
			w.WriteHeader(status)
			routes.ServeJson(w, service.GetErrorMessageType(status, message))
		} else {
			mobilePhone := userTokenJson.MobilePhone
			resultHashed := hashedMobileNumber(mobilePhone)
			statusInsertToken, messageInsertToken := insertTokenToUsersTable(resultHashed, mobilePhone)
			w.WriteHeader(statusInsertToken)
			if statusInsertToken != 200 {
				routes.ServeJson(w, service.GetErrorMessageType(statusInsertToken, messageInsertToken))
			} else {
				userToken := requests.UserToken{mobilePhone, resultHashed}
				routes.ServeJson(w, service.GetGeneralMsgType(statusInsertToken, messageInsertToken, userToken))
			}
		}
	}
}

func UpdatePhoneNumber(w http.ResponseWriter, r *http.Request) {
	_, _, _, userId := service.GetTokenHeader(r.Header.Get("Authorization"))
	userTokenJson := requests.MobilePhoneJson{}
	service.DecodeJson(&userTokenJson, r.Body)
	mobilePhone := userTokenJson.MobilePhone
	if mobilePhone == "" {
		routes.ServeJson(w, service.GetErrorMessageType(400, "data empty"))
	} else {
		resultHashed := hashedMobileNumber(mobilePhone)
		field := fmt.Sprintf("mobile_phone = '%s', token = '%s'", mobilePhone, resultHashed)
		condition := fmt.Sprintf("user_id = %v", userId)
		sequel := service.UpdateQuery("users", field, condition)
		statusResult, messageResult := service.ExecuteChannelSqlResult(sequel)
		w.WriteHeader(statusResult)
		if statusResult != 200 {
			routes.ServeJson(w, service.GetErrorMessageType(statusResult, messageResult))
		} else {
			userToken := requests.UserToken{mobilePhone, resultHashed}
			routes.ServeJson(w, service.GetGeneralMsgType(statusResult, messageResult, userToken))
		}
	}

}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	NewUser := atomicUser(newUserJson(r.Body))

	mobileBytes := []byte(NewUser.MobilePhone)
	hashedPassword, err := bcrypt.GenerateFromPassword(mobileBytes, 10)
	if err != nil {
		routes.ServeJson(w, service.GetErrorMessageType(500, err.Error()))
		return
	}

	SQL := fmt.Sprintf("INSERT INTO users SET user_name='%s', mobile_phone='%s'"+
		", token = '%s'", NewUser.UserName, NewUser.MobilePhone, hashedPassword)
	status, message, newId := service.ExecuteInsertSqlResult(SQL)
	w.WriteHeader(status)

	switch {
	case status != 200:
		routes.ServeJson(w, service.GetErrorMessageType(status, message))
	default:
		userCreated := responses.UserCreated{int(newId), NewUser.UserName, NewUser.MobilePhone, fmt.Sprintf("%s", hashedPassword)}
		routes.ServeJson(w, service.GetGeneralMsgType(status, "user created", userCreated))
	}
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	_, _, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	updateUserName := requests.UpdateUserName{}
	service.DecodeJson(&updateUserName, r.Body)
	userName := updateUserName.UserName
	table := "users"
	field := fmt.Sprintf("user_name='%s'", userName)
	condition := fmt.Sprintf("mobile_phone = '%s'", mobilePhone)
	sequel := service.UpdateQuery(table, field, condition)
	statusUpdate, messageUpdate := updateUserExecutor(sequel)
	w.WriteHeader(statusUpdate)
	if statusUpdate != 200 {
		routes.ServeJson(w, service.GetErrorMessageType(statusUpdate, messageUpdate))
	} else {
		routes.ServeJson(w, service.GetGeneralMsgType(statusUpdate, messageUpdate, requests.UserUpdateType{userName, mobilePhone}))
	}

}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	_, _, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	SQL := fmt.Sprintf("Delete FROM users WHERE mobile_phone = '%s'", mobilePhone)

	statusResult, messageResult := service.ExecuteChannelSqlResult(SQL)
	w.WriteHeader(statusResult)

	switch {
	case statusResult != 200:
		routes.ServeJson(w, service.GetErrorMessageType(statusResult, messageResult))
	default:
		routes.ServeJson(w, service.GetGeneralMsgType(statusResult, "user deleted", requests.MobilePhoneJson{mobilePhone}))
	}
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	_, _, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	file, header, err := r.FormFile("file")

	if err != nil {
		printUploadError(w, err)
		return
	}

	fileType := service.GetFileType(header.Filename)

	if !allowedImageType(fileType) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		routes.ServeJson(w, service.GetErrorMessageType(415, "type is not allowed"))
		return
	}

	staticPath := "/static/"
	targetPath, errorAlocatePath := service.AllocateNewPath(staticPath)
	if errorAlocatePath != nil {
		printUploadError(w, errorAlocatePath)
		return
	}

	errFindOrCreateDir := findOrCreateDirectory(targetPath)
	if errFindOrCreateDir != nil {
		printUploadError(w, errFindOrCreateDir)
		return
	}

	pathFile, nameFile, errNewPath := service.GenerateNewPath(targetPath, fileType)
	if errNewPath != nil {
		printUploadError(w, errNewPath)
		return
	}

	out, errCreateFile := service.CreateFile(pathFile)
	if errCreateFile != nil {
		printUploadError(w, errCreateFile)
		return
	}

	chanCopyFile := make(chan service.CopyFileType)
	go service.ExecuteCopyFile(out, file, chanCopyFile)
	select {
	case rcvChannelCopyFile := <-chanCopyFile:
		out.Close()
		file.Close()
		errCopy := rcvChannelCopyFile.Err
		if errCopy != nil {
			printUploadError(w, err)
			return
		}
		newFilePath := fmt.Sprintf("%s%s", staticPath, nameFile)

		table := "users"
		field := fmt.Sprintf("profile_picture = '%s'", newFilePath)
		condition := fmt.Sprintf("mobile_phone = '%s'", mobilePhone)
		sequel := service.UpdateQuery(table, field, condition)
		statusUpdate, messageUpdate := updateUserExecutor(sequel)
		w.WriteHeader(statusUpdate)
		if statusUpdate != 200 {
			routes.ServeJson(w, service.GetErrorMessageType(statusUpdate, messageUpdate))
		} else {
			routes.ServeJson(w, service.GetGeneralMsgType(statusUpdate, messageUpdate, requests.UserProfilePictureType{mobilePhone, newFilePath}))
		}

	case <-service.TimeOutInMilis(service.GlobalTimeOutIO):
		close(chanCopyFile)
		routes.ServeJson(w, service.GetErrorMessageType(408, "request time out"))
	}
}

func BlockFriend(w http.ResponseWriter, r *http.Request) {
	status, _, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	w.WriteHeader(status)
	block := 0
	friendMobilePhone := decodeActionFriendMobilePhone(r.Body)
	routes.ServeJson(w, service.GetErrorMessageType(blockFriend(mobilePhone, friendMobilePhone, block)))
}

func HideFriend(w http.ResponseWriter, r *http.Request) {
	status, _, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	hide := 1
	w.WriteHeader(status)
	friendMobilePhone := decodeActionFriendMobilePhone(r.Body)
	routes.ServeJson(w, service.GetErrorMessageType(blockFriend(mobilePhone, friendMobilePhone, hide)))

}

func UnBlockFriend(w http.ResponseWriter, r *http.Request) {
	block := 0

	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	w.WriteHeader(status)
	switch {
	case status != 200:
		routes.ServeJson(w, service.GetErrorMessageType(status, message))
	case true:
		friendMobilePhone := decodeActionFriendMobilePhone(r.Body)
		routes.ServeJson(w, service.GetErrorMessageType(sqlDeleteFriendRelationship(mobilePhone, friendMobilePhone, block)))
	}
}

func UnHideFriend(w http.ResponseWriter, r *http.Request) {
	hide := 1
	_, _, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	friendMobilePhone := decodeActionFriendMobilePhone(r.Body)
	routes.ServeJson(w, service.GetErrorMessageType(sqlDeleteFriendRelationship(mobilePhone, friendMobilePhone, hide)))
}

//User Controller Private Function
func mapContactListJson(contactListJson requests.ContactList) chan string {
	chanListContact := make(chan string)
	go func() {
		listContact := ""
		contact := contactListJson.Contact
		for i, value := range contact {
			if i >= (len(contact) - 1) {
				listContact += value.MobilePhone
			} else {
				listContact += value.MobilePhone + ", "
			}
		}
		chanListContact <- listContact
	}()
	return chanListContact
}

func sqlDeleteFriendRelationship(mobilePhone string, friendMobilePhone string, friendshipStatus int) (status int, message string) {
	sequel := fmt.Sprintf("Delete FROM friends_relationship WHERE user_mobile_phone = '%s' and friend_mobile_phone = '%s' and status = %v",
		mobilePhone, friendMobilePhone, friendshipStatus)
	status, message = service.ExecuteChannelSqlResult(sequel)
	return
}

func decodeActionFriendMobilePhone(body io.ReadCloser) string {
	actionToFriend := requests.ActionToFriend{}
	service.DecodeJson(&actionToFriend, body)
	return actionToFriend.MobilePhone
}

func blockFriend(mobilePhone string, friendMobilePhone string, status int) (int, string) {
	friendUserId := 0
	sqlRow, err := service.ExecuteChannelSqlRow(getUserIdSQL(friendMobilePhone))
	switch {
	case err != nil:
		return 508, err.Error()
	default:
		errSqlRow := sqlRow.Scan(&friendUserId)
		statusRow, messageRow := service.CheckScanRowSQL(errSqlRow)
		switch {
		case statusRow == 404:
			return statusRow, "phone number doesn't exists"
		case statusRow != 200:
			return statusRow, messageRow
		default:
			sequel := fmt.Sprintf("INSERT INTO friends_relationship SET user_mobile_phone =  '%s', friend_mobile_phone = '%s', status = %v",
				mobilePhone, friendMobilePhone, status)

			statusInsert, messageInsert := service.ExecuteChannelSqlResult(sequel)
			return statusInsert, messageInsert
		}
	}
}

func mapUsers(rows *sql.Rows, chanUsers chan responses.GeneralArrMsg) chan responses.GeneralArrMsg {
	users := atomicUsers(responses.GeneralArrMsg{})
	chanUser := make(chan requests.User)
	for rows.Next() {
		go assignedMapedUsers(rows, chanUser)
		resChanUser := <-chanUser
		users.Datas = append(users.Datas, resChanUser)
	}
	close(chanUser)
	chanUsers <- users
	return chanUsers
}

func assignedMapedUsers(rows *sql.Rows, chanUser chan requests.User) chan requests.User {
	user := atomicUser(requests.User{})
	rows.Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
	chanUser <- user
	return chanUser
}

func insertTokenToUsersTable(token string, mobilePhone string) (int, string) {
	field := fmt.Sprintf("token = '%s'", token)
	condition := fmt.Sprintf("mobile_phone = '%s'", mobilePhone)
	sequel := service.UpdateQuery("users", field, condition)
	return service.ExecuteChannelSqlResult(sequel)
}

func hashedMobileNumber(mobilePhone string) string {
	mobileBytes := []byte(mobilePhone)
	hashedPassword, err := bcrypt.GenerateFromPassword(mobileBytes, 10)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s", hashedPassword)
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
	return service.ExecuteUpdateSqlResult(sequel)
}

func atomicUser(user requests.User) requests.User {
	service.MutexTime()
	globalExecutionUser.Store(user)
	dataUser := globalExecutionUser.Load().(requests.User)
	return dataUser
}

func atomicUsers(users responses.GeneralArrMsg) responses.GeneralArrMsg {
	service.MutexTime()
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
		routes.ServeJson(w, service.GetErrorMessageType(406, err.Error()))
	}
}

func getUserIdSQL(mobilePhone string) string {
	condition := fmt.Sprintf(" mobile_phone = '%s'", mobilePhone)
	return service.SelectQuery("user_id", "users", condition)
}
