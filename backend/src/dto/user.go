package dto

//type Users []User

type User struct {
	_        struct{} `json:"-" additionalProperties:"true"`
	Username *string  `json:"username"`
	Password *string  `json:"password,omitempty"`
	IsAdmin  *bool    `json:"is_admin,omitempty"`
}
