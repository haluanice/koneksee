// UserController
package controller

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync/atomic"

	"encoding/json"
	"mime/multipart"
	"model"
	"net/http"
	"service"
	"strconv"

	"github.com/drone/routes"
	_ "github.com/go-martini/martini"
	"golang.org/x/crypto/bcrypt"
)

var (
	executionUser  atomic.Value
	executionUsers atomic.Value
	chanFinish     = make(chan bool, 1000)
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	go func() {
		phoneNumber := r.Header.Get("mobile_phone")
		contactListJson := model.ContactList{}
		service.DecodeJson(&contactListJson, r.Body)
		contacts := contactListJson.Contact
		if len(contacts) > service.ReqContactsTreshold {
			return
		}
		chanContactString := mapContactListJson(contacts)
		contact := <-chanContactString
		close(chanContactString)

		condition := fmt.Sprintf("phone_number IN (%s) AND phone_number NOT IN "+
			" (SELECT friend_phone_number FROM users u JOIN friends_relationship fr "+
			" ON u.`phone_number` = fr.`user_phone_number` WHERE u.`phone_number` = '%s' )", contact, phoneNumber)
		sequel := selectUserSQL(condition)
		resultSelectUserSQL(w, sequel)
		chanFinish <- true
	}()
	_ = <-chanFinish
}

func GetUsersBlocked(w http.ResponseWriter, r *http.Request) {
	go func() {
		phoneNumber := r.Header.Get("mobile_phone")
		condition := fmt.Sprintf("phone_number IN "+
			" (SELECT friend_phone_number FROM users u JOIN friends_relationship fr "+
			" ON u.`phone_number` = fr.`user_phone_number` WHERE u.`phone_number` = '%s' )", phoneNumber)
		sequel := selectUserSQL(condition)
		resultSelectUserSQL(w, sequel)
		chanFinish <- true
	}()
	_ = <-chanFinish

}

func GetUser(w http.ResponseWriter, r *http.Request) {
	chanFinish := make(chan bool)
	go func() {
		id := r.Header.Get("user_id")
		user := atomicUser(model.User{})

		condition := fmt.Sprintf("user_id = %v", id)
		sequel := service.SelectQuery("user_id, user_name, phone_number, profile_picture", "users", condition)
		sqlRow, err := service.ExecuteChannelSqlRow(sequel)
		if isErrNotNil(w, 508, err) {
			return
		}
		errSqlRow := sqlRow.Scan(&user.UserId, &user.UserName, &user.PhoneNumber, &user.ProfilePicture)
		statusRow, messageRow := service.CheckScanRowSQL(errSqlRow)
		printResult(w, statusRow, messageRow, user)
		chanFinish <- true
	}()
	_ = <-chanFinish
}

func GenerateNewToken(w http.ResponseWriter, r *http.Request) {
	go func() {
		userId := 0
		userTokenJson := model.PhoneNumberJson{}
		service.DecodeJson(&userTokenJson, r.Body)

		sqlRow, err := service.ExecuteChannelSqlRow(getUserIdSQL(userTokenJson.PhoneNumber))
		if isErrNotNil(w, 508, err) {
			return
		}
		errSqlRow := sqlRow.Scan(&userId)
		status, message := service.CheckScanRowSQL(errSqlRow)
		if isStatusNotOK(w, status, message) {
			return
		}
		phoneNumber := userTokenJson.PhoneNumber
		resultHashed := hashedMobileNumber(phoneNumber)
		statusInsertToken, messageInsertToken := insertTokenToUsersTable(resultHashed, phoneNumber)
		w.WriteHeader(statusInsertToken)
		if isStatusNotOK(w, statusInsertToken, messageInsertToken) {
			return
		}
		userToken := model.UserToken{phoneNumber, resultHashed}
		routes.ServeJson(w, service.GetGeneralMsgType(statusInsertToken, messageInsertToken, userToken))
		chanFinish <- true
	}()
	_ = <-chanFinish
}

func UpdatePhoneNumber(w http.ResponseWriter, r *http.Request) {
	go func() {
		phoneNumber := r.Header.Get("phone_number")
		userTokenJson := model.PhoneNumberJson{}
		service.DecodeJson(&userTokenJson, r.Body)
		newphoneNumber := userTokenJson.PhoneNumber
		if phoneNumber == "" {
			w.WriteHeader(400)
			routes.ServeJson(w, service.GetErrorMessageType(400, "data empty"))
		} else {
			resultHashed := hashedMobileNumber(phoneNumber)
			field := fmt.Sprintf("phone_number = '%s', token = '%s'", newphoneNumber, resultHashed)
			condition := fmt.Sprintf("phone_number = '%s'", phoneNumber)
			statusUpdate, messageUpdate := service.UpdateQuery("users", field, condition)
			userToken := model.UserToken{newphoneNumber, resultHashed}
			printResult(w, statusUpdate, messageUpdate, userToken)
		}
		chanFinish <- true
	}()
	_ = <-chanFinish
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	chanCreate := make(chan model.GeneralMsg)
	go func() {
		NewUser := atomicUser(newUserJson(r.Body))
		if NewUser.PhoneNumber == "" {
			chanCreate <- model.GeneralMsg{422, "phone_number is empty", NewUser}
			return
		}

		mobileBytes := []byte(NewUser.PhoneNumber)
		hashedPassword, err := bcrypt.GenerateFromPassword(mobileBytes, 10)

		if err != nil {
			chanCreate <- model.GeneralMsg{508, err.Error(), NewUser}
			return
		}

		NewUser.Token = fmt.Sprintf("%s", hashedPassword)

		field := fmt.Sprintf("user_name='%s', phone_number='%s', token = '%s', device_id = '%s', device_type = '%s', user_agent = '%s', status = '%s'",
			NewUser.UserName, NewUser.PhoneNumber, NewUser.Token, NewUser.DeviceId, NewUser.DeviceType, NewUser.UserAgent, NewUser.Status)
		SQL := fmt.Sprintf("INSERT INTO users SET %s", field)
		status, message, newId := service.ExecuteInsertSqlResult(SQL)
		switch {
		case status == http.StatusConflict:
			// 1. Update user_name and token in users
			condition := fmt.Sprintf("phone_number = '%s'", NewUser.PhoneNumber)
			statusUpdate, messageUpdate := service.UpdateQuery("users", field, condition)
			if statusUpdate != http.StatusOK {
				chanCreate <- model.GeneralMsg{statusUpdate, messageUpdate, NewUser}
				return
			}
			// 2. Get user_id
			conditionSelect := fmt.Sprintf("phone_number = %s", NewUser.PhoneNumber)
			sequelSelect := service.SelectQuery("user_id", "users", conditionSelect)
			sqlRow, err := service.ExecuteChannelSqlRow(sequelSelect)
			if err != nil {
				chanCreate <- model.GeneralMsg{508, err.Error(), NewUser}
				return
			}
			// 3. Check if result exists
			errSqlRow := sqlRow.Scan(&NewUser.UserId)
			statusRow, messageRow := service.CheckScanRowSQL(errSqlRow)
			if statusUpdate != http.StatusOK {
				chanCreate <- model.GeneralMsg{statusRow, messageRow, NewUser}
				return
			}
			// 4. Return existing mobile_phone with given user_name and new token
			chanCreate <- model.GeneralMsg{statusRow, messageRow, NewUser}
		default:
			NewUser.UserId = int(newId)
			chanCreate <- model.GeneralMsg{status, message, NewUser}
		}
	}()
	resChanCreate := <-chanCreate
	status := resChanCreate.Status
	w.WriteHeader(status)
	routes.ServeJson(w, service.GetGeneralMsgType(status, resChanCreate.Message, resChanCreate.Data))
}

func UpdateUserName(w http.ResponseWriter, r *http.Request) {
	go func() {
		phoneNumber := r.Header.Get("phone_number")
		updateUserName := model.UpdateUserName{}
		service.DecodeJson(&updateUserName, r.Body)
		userName := updateUserName.UserName
		table := "users"
		field := fmt.Sprintf("user_name='%s'", userName)
		condition := fmt.Sprintf("phone_number = '%s'", phoneNumber)
		statusUpdate, messageUpdate := service.UpdateQuery(table, field, condition)
		printResult(w, statusUpdate, messageUpdate, model.UserUpdateType{userName, phoneNumber})
		chanFinish <- true
	}()
	_ = <-chanFinish

}

func UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	go func() {
		phoneNumber := r.Header.Get("phone_number")
		updateStatus := model.UpdateUserStatus{}
		service.DecodeJson(&updateStatus, r.Body)
		userStatus := updateStatus.Status
		table := "users"
		field := fmt.Sprintf("status='%s'", userStatus)
		condition := fmt.Sprintf("phone_number = '%s'", phoneNumber)
		statusUpdate, messageUpdate := service.UpdateQuery(table, field, condition)
		printResult(w, statusUpdate, messageUpdate, model.RespUpdateUserStatus{phoneNumber, userStatus})
		chanFinish <- true
	}()
	_ = <-chanFinish
}
func UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	go func() {
		id, _ := strconv.Atoi(r.Header.Get("user_id"))
		file, header, err := r.FormFile("file")
		if isErrNotNil(w, http.StatusNotAcceptable, err) {
			return
		}
		fileType := header.Header.Get("Content-Type")
		status, info := uploadCloudinary(file, fileType)
		if isStatusNotOK(w, status, info) {
			return
		}
		phoneNumber := r.Header.Get("phone_number")
		cloudinaryPath := info
		userName := r.FormValue("user_name")
		userStatus := r.FormValue("status")
		field := fmt.Sprintf("user_name = '%s', profile_picture = '%s', status = '%s'", userName, cloudinaryPath, userStatus)
		condition := fmt.Sprintf("phone_number = '%s'", phoneNumber)
		statusUpdate, messageUpdate := service.UpdateQuery("users", field, condition)
		profileUpdate := model.ProfileUpdateType{id, userName, userStatus, cloudinaryPath}
		printResult(w, statusUpdate, messageUpdate, profileUpdate)
		chanFinish <- true
	}()
	_ = <-chanFinish
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	go func() {
		phoneNumber := r.Header.Get("phone_number")

		file, header, err := r.FormFile("file")
		if isErrNotNil(w, http.StatusNotAcceptable, err) {
			return
		}

		fileType := header.Header.Get("Content-Type")
		status, info := uploadCloudinary(file, fileType)
		if isStatusNotOK(w, status, info) {
			return
		}

		cloudinaryPath := info
		field := fmt.Sprintf("profile_picture = '%s'", cloudinaryPath)
		condition := fmt.Sprintf("phone_number = '%s'", phoneNumber)
		statusUpdate, messageUpdate := service.UpdateQuery("users", field, condition)
		profilePictureUser := model.UserProfilePictureType{phoneNumber, cloudinaryPath}
		printResult(w, statusUpdate, messageUpdate, profilePictureUser)
		chanFinish <- true
	}()
	_ = <-chanFinish
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	go func() {
		phoneNumber := r.Header.Get("phone_number")
		SQL := fmt.Sprintf("Delete FROM users WHERE phone_number = '%s'", phoneNumber)

		statusResult, messageResult := service.ExecuteChannelSqlResult(SQL)
		printResult(w, statusResult, messageResult, model.PhoneNumberJson{phoneNumber})
		chanFinish <- true
	}()
	_ = <-chanFinish
}

func BlockFriend(w http.ResponseWriter, r *http.Request) {
	go func() {
		block := 0
		status, phoneNumber := getStatusphoneNumber(r)
		friendPhoneNumber := decodeActionFriendMobilePhone(r.Body)
		status, message := blockFriend(phoneNumber, friendPhoneNumber, block)
		printDefaultMessage(w, status, message)
		chanFinish <- true
	}()
	_ = <-chanFinish
}

func UnBlockFriend(w http.ResponseWriter, r *http.Request) {
	go func() {
		block := 0
		status, phoneNumber := getStatusphoneNumber(r)
		friendPhoneNumber := decodeActionFriendphoneNumber(r.Body)
		status, message := sqlDeleteFriendRelationship(phoneNumber, friendPhoneNumber, block)
		printDefaultMessage(w, status, message)
		chanFinish <- true
	}()
	_ = <-chanFinish
}

//User Controller Private Function
func uploadCloudinary(file multipart.File, fileType string) (int, string) {
	statusNotAcceptable := http.StatusNotAcceptable
	// 1. Check file content type

	if !allowedImageType(fileType) {
		return http.StatusUnsupportedMediaType, "type is not allowed"
	}
	// 2. Generate new filename
	nameFile, errNewPath := service.GenerateNewPath()
	if errNewPath != nil {
		return statusNotAcceptable, errNewPath.Error()
	}
	// 3. Read multipart file

	buff, errReadFile := ioutil.ReadAll(file)
	if errReadFile != nil {
		return statusNotAcceptable, errReadFile.Error()
	}
	//4. Upload to cloudinary
	resChannelUpload := service.UploadImage(nameFile, buff)
	cloudinaryInfo := <-resChannelUpload
	close(resChannelUpload)
	if cloudinaryInfo.Err != nil {
		internalServerStatus := http.StatusInternalServerError
		return internalServerStatus, "internal server error with cloudinary"
	}
	// 5. Return cludinary path
	cloudinaryPath := cloudinaryInfo.FilePath
	return http.StatusOK, cloudinaryPath
}
func printDefaultMessage(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	routes.ServeJson(w, service.GetDefaultMessage(status, message))
}
func decodeActionFriendMobilePhone(body io.ReadCloser) string {
	actionToFriend := model.ActionToFriend{}
	service.DecodeJson(&actionToFriend, body)
	return actionToFriend.PhoneNumber
}

func isErrNotNil(w http.ResponseWriter, status int, err error) bool {
	if err != nil {
		w.WriteHeader(status)
		routes.ServeJson(w, service.GetErrorMessageType(status, err.Error()))
		return true
	}
	return false
}

func isStatusNotOK(w http.ResponseWriter, status int, message string) bool {
	if status != http.StatusOK {
		w.WriteHeader(status)
		routes.ServeJson(w, service.GetErrorMessageType(status, message))
		return true
	}
	return false
}

func printResult(w http.ResponseWriter, status int, message string, valueType interface{}) {
	if isStatusNotOK(w, status, message) {
		return
	} else {
		w.WriteHeader(status)
		routes.ServeJson(w, service.GetGeneralMsgType(status, message, valueType))
	}
}

func selectUserSQL(condition string) string {
	return service.SelectQuery("user_id, user_name, phone_number, profile_picture", "users", condition)
}

func resultSelectUserSQL(w http.ResponseWriter, sequel string) {
	rows, err := service.ExecuteChannelSqlRows(sequel)
	internalServerStatus := http.StatusInternalServerError
	if isErrNotNil(w, internalServerStatus, err) {
		w.WriteHeader(internalServerStatus)
		routes.ServeJson(w, service.GetErrorMessageType(internalServerStatus, err.Error()))
		return
	}
	select {
	case resChanUsers := <-mapUsers(rows):
		if resChanUsers.Datas == nil {
			betterEmptyThanNil := make([]interface{}, 0)
			resChanUsers.Datas = betterEmptyThanNil
		}
		statusOK := http.StatusOK
		w.WriteHeader(statusOK)
		routes.ServeJson(w, service.GetGeneralMsgType(statusOK, "success", resChanUsers))
	case <-service.TimeOutInMilis(service.GlobalTimeOutDB):
		printDefaultMessage(w, 508, "request timeout")
	}
}

func getStatusphoneNumber(r *http.Request) (status int, phoneNumber string) {
	status, _ = strconv.Atoi(r.Header.Get("status_filter"))
	phoneNumber = r.Header.Get("phone_number")
	return
}

func mapContactListJson(contacts []string) chan string {
	chanListContact := make(chan string)
	go func() {
		var listContact string
		sizeContacts := len(contacts)
		for i, value := range contacts {
			if i >= (sizeContacts - 1) {
				listContact += value
			} else {
				listContact += value + ", "
			}
		}
		chanListContact <- listContact
	}()
	return chanListContact
}

func sqlDeleteFriendRelationship(phoneNumber string, friendphoneNumber string, friendshipStatus int) (status int, message string) {
	sequel := fmt.Sprintf("Delete FROM friends_relationship WHERE user_phone_number = '%s' and friend_phone_number = '%s' and status = %v",
		phoneNumber, friendphoneNumber, friendshipStatus)
	status, message = service.ExecuteChannelSqlResult(sequel)
	return
}

func decodeActionFriendphoneNumber(body io.ReadCloser) string {
	actionToFriend := model.ActionToFriend{}
	service.DecodeJson(&actionToFriend, body)
	return actionToFriend.PhoneNumber
}

func blockFriend(phoneNumber string, friendPhoneNumber string, status int) (int, string) {
	var friendUserId int
	sqlRow, err := service.ExecuteChannelSqlRow(getUserIdSQL(friendPhoneNumber))

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
			sequel := fmt.Sprintf("INSERT INTO friends_relationship SET user_phone_number =  '%s', friend_phone_number = '%s', status = %v",
				phoneNumber, friendPhoneNumber, status)

			statusInsert, messageInsert := service.ExecuteChannelSqlResult(sequel)
			return statusInsert, messageInsert
		}
	}
}

func mapUsers(rows *sql.Rows) chan model.GeneralArrMsg {
	users := atomicUsers(model.GeneralArrMsg{})
	chanUsers := make(chan model.GeneralArrMsg)
	go func() {
		chanUser := make(chan model.User)
		for rows.Next() {
			go assignedMapedUsers(rows, chanUser)
			resChanUser := <-chanUser
			users.Datas = append(users.Datas, resChanUser)
		}
		close(chanUser)
		chanUsers <- users
	}()
	return chanUsers
}

func assignedMapedUsers(rows *sql.Rows, chanUser chan model.User) chan model.User {
	user := atomicUser(model.User{})
	rows.Scan(&user.UserId, &user.UserName, &user.PhoneNumber, &user.ProfilePicture)
	chanUser <- user
	return chanUser
}

func insertTokenToUsersTable(token string, phoneNumber string) (int, string) {
	field := fmt.Sprintf("token = '%s'", token)
	condition := fmt.Sprintf("phone_number = '%s'", phoneNumber)
	return service.UpdateQuery("users", field, condition)
}

func hashedMobileNumber(phoneNumber string) string {
	mobileBytes := []byte(phoneNumber)
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
		"image/png":  true,
		"image/jpeg": true,
		"image/jpg":  true,
		"image/gif":  true,
	}
	_, isImageAllowed := m[contentType]
	return isImageAllowed
}

func updateUserExecutor(sequel string) (int, string) {
	return service.ExecuteUpdateSqlResult(sequel)
}

func atomicUser(user model.User) model.User {
	service.MutexTime()
	executionUser.Store(user)
	dataUser := executionUser.Load().(model.User)
	return dataUser
}

func atomicUsers(users model.GeneralArrMsg) model.GeneralArrMsg {
	service.MutexTime()
	executionUsers.Store(users)
	dataUsers := executionUsers.Load().(model.GeneralArrMsg)
	return dataUsers
}

func newUserJson(body io.ReadCloser) model.User {
	decoder := json.NewDecoder(body)
	NewUser := model.User{}
	decoder.Decode(&NewUser)
	return NewUser
}

func getUserIdSQL(phoneNumber string) string {
	condition := fmt.Sprintf(" phone_number = '%s'", phoneNumber)
	return service.SelectQuery("user_id", "users", condition)
}
