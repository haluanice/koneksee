// Model
package model

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type ExecSQLType struct {
	SqlResult sql.Result
	Err       string
}

type Job struct {
	AffectedRow  bool
	LastInsertId int64
}

type User struct {
	UserId         int    `json:"user_id"`
	UserName       string `json:"user_name"`
	PhoneNumber    string `json:"phone_number"`
	Status         string `json:"status"`
	ProfilePicture string `json:"profile_picture"`
	DeviceId       string `json:"device_id"`
	DeviceType     string `json:"device_type"`
	UserAgent      string `json:"user_agent"`
	Token          string `json:"token"`
}

type UserUpdateType struct {
	UserName    string `json:"user_name"`
	PhoneNumber string `json:"phone_number"`
}

type UserProfilePictureType struct {
	PhoneNumber    string `json:"phone_number"`
	ProfilePicture string `json:"profile_picture"`
}

type PhoneNumberJson struct {
	PhoneNumber string `json:"phone_number"`
}

type UserToken struct {
	PhoneNumber string `json:"phone_number"`
	Token       string `json:"token"`
}

type ContactList struct {
	Contact []string `json:"phone_number"`
}

type UpdateUserName struct {
	UserName string `json:"user_name"`
}

type UpdateUserStatus struct {
	Status string `json:"status"`
}

type RespUpdateUserStatus struct {
	PhoneNumber string `json:"phone_number"`
	Status      string `json:"status"`
}

type ActionToFriend struct {
	PhoneNumber string `json:"phone_number"`
}
type DefaultMessage struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type GeneralMsg struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type GeneralArrMsg struct {
	Datas []interface{} `json:"data"`
}

type ProfileUpdateType struct {
	UserId         int    `json:"user_id"`
	UserName       string `json:"user_name"`
	Status         string `json:"status"`
	ProfilePicture string `json:"profile_picture"`
}

type UserCreated struct {
	UserId      int    `json:"user_id"`
	UserName    string `json:"user_name"`
	PhoneNumber string `json:"phone_number"`
	Token       string `json:"token"`
}
