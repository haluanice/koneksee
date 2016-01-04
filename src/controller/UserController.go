// UserController
package controller

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"encoding/json"
	"model"
	"net/http"
	"service"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
)

var globalExecutionUser atomic.Value
var globalExecutionUsers atomic.Value

func atomicUser(user model.User) model.User {
	globalExecutionUser.Store(user)
	dataUser := globalExecutionUser.Load().(model.User)
	return dataUser
}

func atomicUsers(users model.Users) model.Users {
	globalExecutionUsers.Store(users)
	dataUsers := globalExecutionUsers.Load().(model.Users)
	return dataUsers
}

func newUserJson(body io.ReadCloser) model.User {
	decoder := json.NewDecoder(body)
	NewUser := model.User{}
	decoder.Decode(&NewUser)

	//*Get from adiyional params forom URI*//
	//NewUser.Name = r.FormValue("user")

	return NewUser
}

func GetUserId(r http.Request) int {
	urlParams := r.URL.Query()
	idString := urlParams.Get(":id")
	idInt, _ := strconv.Atoi(idString)

	return idInt
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	isValid := service.GetTokenHeader(r)
	service.SetHeaderParameter(w)
	switch isValid {
	case false:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, service.OutputError(500, "token invalid"))
	case true:
		cases := r.FormValue("phone_list")
		sequel := "select user_id, user_name, mobile_phone, profile_picture from users"
		if cases != "" {
			sequel = "select user_id, user_name, mobile_phone, profile_picture from users where mobile_phone in (" + cases + ")"
		}
		rows := service.ExecuteChanelSqlRows(sequel)
		users := atomicUsers(model.Users{})
		chanUser := make(chan model.User)
		chanUsers := make(chan model.Users)
		go func() {
			for rows.Next() {
				go func() {
					user := atomicUser(model.User{})
					rows.Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
					chanUser <- user
				}()
				resChanUser := <-chanUser
				users.Datas = append(users.Datas, resChanUser)
			}
			chanUsers <- users
		}()
		resChanUsers := <-chanUsers
		resChanUsers.Status = 200

		output, _ := json.Marshal(resChanUsers)
		fmt.Fprintln(w, string(output))
	}
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	isValid := service.GetTokenHeader(r)
	service.SetHeaderParameter(w)
	switch isValid {
	case false:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, service.OutputError(500, "token invalid"))
	case true:
		urlParams := r.URL.Query()
		id := urlParams.Get(":id")
		user := atomicUser(model.User{})
		sequel := fmt.Sprintf("select user_id, user_name, mobile_phone, profile_picture from users where user_id = %s", id)
		row := service.ExecuteChanelSqlRow(sequel).Scan(&user.UserId, &user.UserName, &user.MobilePhone, &user.ProfilePicture)
		switch {
		case row == sql.ErrNoRows:
			fmt.Fprintf(w, service.OutputError(400, "user not found"))
		case row != nil:
			fmt.Fprintf(w, service.OutputError(500, row.Error()))
		default:
			fmt.Fprintf(w, service.OutputSuccess(200, "success", user))
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
	create := service.ExecuteChanelSqlResult(SQL)

	switch create {
	case nil:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, service.OutputError(500, "data not created"))
	default:
		affectedRows, _ := create.RowsAffected()
		switch affectedRows < int64(1) {
		case true:
			fmt.Fprintf(w, service.OutputError(500, "data not created"))
		case false:
			newId, _ := create.LastInsertId()
			NewUser.UserId = int(newId)
			output, _ := json.Marshal(NewUser)
			fmt.Fprintln(w, string(output))
		}
	}
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	isValid := service.GetTokenHeader(r)
	service.SetHeaderParameter(w)
	switch isValid {
	case false:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, service.OutputError(500, "token invalid"))
	case true:
		NewUser := atomicUser(newUserJson(r.Body))
		SQL := "UPDATE users SET user_name='" + NewUser.UserName + "', mobile_phone='" + NewUser.MobilePhone + "'"
		updateUserExecutor(w, r, SQL)
	}

}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	isValid := service.GetTokenHeader(r)
	service.SetHeaderParameter(w)
	switch isValid {
	case false:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, service.OutputError(500, "token invalid"))
	case true:
		userId := GetUserId(*r)
		SQL := fmt.Sprintf("Delete FROM users WHERE user_id=%v", userId)
		destroy := service.ExecuteChanelSqlResult(SQL)
		affectedRows, _ := destroy.RowsAffected()
		switch affectedRows < int64(1) {
		case true:
			fmt.Fprintf(w, service.OutputError(422, "data not deleted"))
		case false:
			output, _ := json.Marshal(model.DataDestroy{"deleted", model.UserID{userId}})
			fmt.Fprintf(w, string(output))
		}
	}
}

var channelCopyFile = make(chan int64)

func allowedImageType(contentType string) bool {
	m := map[string]bool{
		"png":  true,
		"jpeg": true,
		"jpg":  true,
		"gif":  true,
	}
	_, isAllowed := m[contentType]
	return isAllowed

}
func UploadFile(w http.ResponseWriter, r *http.Request) {
	service.SetHeaderParameter(w)

	file, header, err := r.FormFile("file")
	printError(w, err)
	if err != nil {
		return
	}
	defer file.Close()

	fileName := header.Filename
	fileTypeArr := strings.Split(fileName, ".")
	fileType := fileTypeArr[len(fileTypeArr)-1]
	if !allowedImageType(fileType) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		infoError := fileType + " is not allowed"
		fmt.Fprintf(w, service.OutputError(415, infoError))
		return
	}

	pwd, _ := os.Getwd()
	targetPath := pwd + "/static/"
	pathFile, nameFile, err := service.GenerateNewPath(targetPath)
	printError(w, err)
	if err != nil {
		return
	}

	directoryExists(targetPath)

	out, err := service.CreateFile(pathFile)
	printError(w, err)
	if err != nil {
		return
	}
	defer out.Close()

	go executeCopyFile(w, out, file)
	_ = <-channelCopyFile

	newFilePath := fmt.Sprintf("/static/%s", nameFile)
	sequel := fmt.Sprintf("UPDATE users SET profile_picture='%s'", newFilePath)
	updateUserExecutor(w, r, sequel)
}

func executeCopyFile(w http.ResponseWriter, out *os.File, file multipart.File) {
	copied, err := io.Copy(out, file)
	printError(w, err)
	channelCopyFile <- copied
}
func printError(w http.ResponseWriter, err error) {
	if err != nil {
		fmt.Fprintf(w, service.OutputError(422, err.Error()))
	}
}

func directoryExists(targetPath string) {
	_, err := os.Stat(targetPath)
	if err != nil || os.IsNotExist(err) {
		os.Mkdir(targetPath, 0777)
	}
}

func updateUserExecutor(w http.ResponseWriter, r *http.Request, sequel string) {
	userId := GetUserId(*r)
	user := atomicUser(model.User{})

	sequel += fmt.Sprintf(" WHERE user_id = %v", userId)

	update := service.ExecuteChanelSqlResult(sequel)
	if update == nil {
		fmt.Fprintln(w, service.OutputError(500, "Internal Server Error"))
		return
	}

	affectedRows, _ := update.RowsAffected()

	switch affectedRows < int64(1) {
	case true:
		fmt.Fprintf(w, service.OutputError(422, "data not updated"))
	case false:
		showUserSQL := fmt.Sprintf("SELECT user_id, user_name, mobile_phone, profile_picture FROM users WHERE user_id = %v", userId)
		_ = service.ExecuteChanelSqlRow(showUserSQL).Scan(&user.UserId,
			&user.UserName, &user.MobilePhone, &user.ProfilePicture)
		output, _ := json.Marshal(user)
		fmt.Fprintln(w, string(output))
	}
}
