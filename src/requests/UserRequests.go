// UserRequests
package requests

type User struct {
	UserId         int    `json:"user_id"`
	UserName       string `json:"user_name"`
	MobilePhone    string `json:"mobile_phone"`
	ProfilePicture string `json:"profile_picture"`
}

type UserTokenJson struct {
	MobilePhone string `json:"mobile_phone"`
}

type UserToken struct {
	UserID int    `json:"user_id"`
	Token  string `json:"token"`
}

type UserMobilePhone struct {
	MobilePhone string `json:"mobile_phone"`
}

type UpdateUserName struct {
	UserName string `json:"user_name"`
}
