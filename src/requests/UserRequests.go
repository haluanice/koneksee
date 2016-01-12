// UserRequests
package requests

type User struct {
	UserId         int    `json:"user_id"`
	UserName       string `json:"user_name"`
	PhoneNumber    string `json:"phone_number"`
	ProfilePicture string `json:"profile_picture"`
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

type ActionToFriend struct {
	PhoneNumber string `json:"phone_number"`
}
