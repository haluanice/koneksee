// Model
package model

type User struct {
	UserId         int    `json:"user_id"`
	UserName       string `json:"user_name"`
	MobilePhone    string `json:"mobile_phone"`
	ProfilePicture string `json:"profile_picture"`
}

type Users struct {
	Status int    `json:"status"`
	Datas  []User `json:"users"`
}

type Job struct {
	AffectedRow  bool
	LastInsertId int64
}

type ErrorMessage struct {
	Status  int    `json:status`
	Message string `json:"message"`
}

type SuccessMessage struct {
	Status  int    `json:status`
	Message string `json:"message"`
	UserObj User   `json:"data"`
}
type UserID struct {
	ID int `json:"id"`
}
type DataDestroy struct {
	Message string `json:"message"`
	User    UserID `json:"data"`
}
