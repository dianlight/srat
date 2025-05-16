package dto

type User struct {
	_        struct{} `json:"-" additionalProperties:"true"`
	Username string   `json:"username" pattern:"[a-z]+" maxLength:"30"`
	Password string   `json:"password,omitempty"`
	IsAdmin  bool     `json:"is_admin,omitempty" default:"false"`
	RwShares []string `json:"rw_shares,omitempty" read-only:"true"`
	RoShares []string `json:"ro_shares,omitempty" read-only:"true"`
}
