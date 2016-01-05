// UserResponses
package responses

type ErrorMessage struct {
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

type UserCreated struct {
	UserId      int    `json:"user_id"`
	UserName    string `json:"user_name"`
	MobilePhone string `json:"mobile_phone"`
	Token       string `json:"token"`
}
