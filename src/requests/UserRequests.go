// UserRequests
package requests

type User struct {
	UserId         int    `json:"user_id"`
	UserName       string `json:"user_name"`
	MobilePhone    string `json:"mobile_phone"`
	ProfilePicture string `json:"profile_picture"`
}

type UserUpdateType struct {
	UserName    string `json:"user_name"`
	MobilePhone string `json:"mobile_phone"`
}

type UserProfilePictureType struct {
	MobilePhone    string `json:"mobile_phone"`
	ProfilePicture string `json:"profile_picture"`
}

type MobilePhoneJson struct {
	MobilePhone string `json:"mobile_phone"`
}

type UserToken struct {
	MobilePhone string `json:"mobile_phone"`
	Token       string `json:"token"`
}

type ContactList struct {
	Contact []MobilePhoneJson `json:"data"`
}

type UpdateUserName struct {
	UserName string `json:"user_name"`
}

type ActionToFriend struct {
	MobilePhone string `json:"mobile_phone"`
}
