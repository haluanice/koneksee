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

	"golang.org/x/crypto/bcrypt"
)

var (
	globalExecutionUser  atomic.Value
	globalExecutionUsers atomic.Value
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	service.SetHeaderParameter(w)
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		cases := r.FormValue("phone_list")
		sequel := fmt.Sprintf("SELECT user_id, user_name, mobile_phone, profile_picture FROM users WHERE mobile_phone IN "+
			"(%s) AND mobile_phone NOT IN "+
			" (SELECT friend FROM users u JOIN friends_relationship fr "+
			" ON u.`mobile_phone` = fr.`user` WHERE u.`mobile_phone` = '%s' )", cases, mobilePhone)
		if cases == "" {
			sequel = "select user_id, user_name, mobile_phone, profile_picture from users"
		}
		rows, err := service.ExecuteChannelSqlRows(sequel)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, service.OutputError(500, err.Error()))
		} else {
			chanUsers := make(chan responses.GeneralArrMsg)
			go mapUsers(rows, chanUsers)
			select {
			case resChanUsers := <-chanUsers:
				fmt.Fprintf(w, service.OutputSuccess(200, "success", resChanUsers))
			case <-service.TimeOutInMilis(service.GlobalTimeOutDB):
				close(chanUsers)
				fmt.Fprintf(w, service.OutputError(500, "request time out"))
			}
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
		condition := fmt.Sprintf("user_id = %v", id)
		sequel := service.SelectQuery("user_id, user_name, mobile_phone, profile_picture", "users", condition)
		sqlRow, err := service.ExecuteChannelSqlRow(sequel)
		switch {
		case err != nil:
			w.WriteHeader(508)
			fmt.Fprintf(w, service.OutputError(508, err.Error()))
		default:
			errSqlRow := sqlRow.Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
			status, message := service.CheckScanRowSQL(errSqlRow)
			if status != 200 {
				w.WriteHeader(status)
				fmt.Fprintf(w, service.OutputError(status, message))
			} else {
				w.WriteHeader(status)
				fmt.Fprintf(w, service.OutputSuccess(status, message, user))
			}
		}
	}
}

func GenerateNewToken(w http.ResponseWriter, r *http.Request) {
	service.SetHeaderParameter(w)

	userTokenJson := requests.UserTokenJson{}
	service.DecodeJson(&userTokenJson, r.Body)
	user_id := 0
	sqlRow, err := service.ExecuteChannelSqlRow(getUserIdSQL(userTokenJson.MobilePhone))
	switch {
	case err != nil:
		w.WriteHeader(508)
		fmt.Fprintf(w, service.OutputError(508, err.Error()))
	default:
		errSqlRow := sqlRow.Scan(&user_id)
		status, message := service.CheckScanRowSQL(errSqlRow)
		if status != 200 {
			w.WriteHeader(status)
			fmt.Fprintf(w, service.OutputError(status, message))
		} else {
			mobilePhone := userTokenJson.MobilePhone
			resultHashed := hashedMobileNumber(mobilePhone)
			statusInsertToken, messageInsertToken := insertTokenToUsersTable(resultHashed, mobilePhone)
			if statusInsertToken != 200 {
				w.WriteHeader(status)
				fmt.Fprintln(w, service.OutputError(statusInsertToken, messageInsertToken))
			} else {
				w.WriteHeader(status)
				userToken := requests.UserToken{user_id, resultHashed}
				fmt.Fprintf(w, service.OutputSuccess(statusInsertToken, messageInsertToken, userToken))
			}
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
		table := "users"
		field := fmt.Sprintf("user_name='%s'", userName)
		condition := fmt.Sprintf("mobile_phone = '%s'", mobilePhone)
		sequel := service.UpdateQuery(table, field, condition)
		status, message := updateUserExecutor(sequel)
		if status != 200 {
			fmt.Fprintf(w, service.OutputError(status, message))
		} else {
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

		status, message := service.ExecuteChannelSqlResult(SQL)
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

		fileType := service.GetFileType(header.Filename)

		if !allowedImageType(fileType) {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			fmt.Fprintf(w, service.OutputError(415, "type is not allowed"))
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
			status, message := updateUserExecutor(sequel)
			if status != 200 {
				fmt.Fprintf(w, service.OutputError(status, message))
			} else {
				fmt.Fprintf(w, service.OutputSuccess(status, message, requests.UserProfilePictureType{mobilePhone, newFilePath}))
			}

		case <-service.TimeOutInMilis(service.GlobalTimeOutIO):
			close(chanCopyFile)
			fmt.Fprintf(w, service.OutputError(408, "request time out"))
		}
	}
}

func BlockFriend(w http.ResponseWriter, r *http.Request) {
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

func HideFriend(w http.ResponseWriter, r *http.Request) {
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

func UnBlockFriend(w http.ResponseWriter, r *http.Request) {
	block := 0
	service.SetHeaderParameter(w)

	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		friendMobilePhone := decodeActionFriendMobilePhone(r.Body)

		status, message := sqlDeleteFriendRelationship(mobilePhone, friendMobilePhone, block)
		w.WriteHeader(status)
		fmt.Fprintln(w, service.OutputError(status, message))
	}
}

func UnHideFriend(w http.ResponseWriter, r *http.Request) {
	hide := 1
	service.SetHeaderParameter(w)

	status, message, mobilePhone, _ := service.GetTokenHeader(r.Header.Get("Authorization"))
	switch {
	case status != 200:
		w.WriteHeader(status)
		fmt.Fprintf(w, service.OutputError(status, message))
	case true:
		friendMobilePhone := decodeActionFriendMobilePhone(r.Body)

		status, message := sqlDeleteFriendRelationship(mobilePhone, friendMobilePhone, hide)
		w.WriteHeader(status)
		fmt.Fprintln(w, service.OutputError(status, message))
	}
}

//User Controller Private Function

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

func hideOrBlockFriend(mobilePhone string, friendMobilePhone string, status int) (int, string) {
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
			if statusInsert == 409 {
				table := "friends_relationship"
				field := fmt.Sprintf("status =  %v", status)
				condition := fmt.Sprintf("user_mobile_phone = '%s' and friend_mobile_phone = '%s'", mobilePhone,
					friendMobilePhone)
				sequel := service.UpdateQuery(table, field, condition)
				return service.ExecuteChannelSqlResult(sequel)
			}
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
	chanUsers <- users
	close(chanUsers)
	return chanUsers
}

func assignedMapedUsers(rows *sql.Rows, chanUser chan requests.User) chan requests.User {
	user := atomicUser(requests.User{})
	rows.Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
	chanUser <- user
	close(chanUser)
	return chanUser
}

func insertTokenToUsersTable(token string, mobilePhone string) (int, string) {
	SQL := fmt.Sprintf("UPDATE users SET token = '%s' where mobile_phone = '%s'", token, mobilePhone)

	status, message := service.ExecuteChannelSqlResult(SQL)
	return status, message
}

func hashedMobileNumber(mobilePhone string) string {
	mobileBytes := []byte(mobilePhone)
	hashedPassword, err := bcrypt.GenerateFromPassword(mobileBytes, 10)
	if err != nil {
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
		fmt.Fprintf(w, service.OutputError(406, err.Error()))
	}
}

func getUserIdSQL(mobilePhone string) string {
	condition := fmt.Sprintf(" mobile_phone = '%s'", mobilePhone)
	return service.SelectQuery("user_id", "users", condition)
}
