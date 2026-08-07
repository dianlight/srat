package dto

type User struct {
	_                  struct{}        `json:"-" additionalProperties:"true"`
	Username           string          `json:"username" pattern:"[a-zA-Z0-9 _-]+" maxLength:"30"`
	Password           *Secret[string] `json:"password,omitempty" writeOnly:"true" format:"password"`
	IsAdmin            bool            `json:"is_admin,omitempty" default:"false"`
	IsValid            *bool           `json:"is_valid,omitempty" default:"true"`
	HasDefaultPassword bool            `json:"has_default_password,omitempty" readOnly:"true"`
	RwShares           []string        `json:"rw_shares,omitempty" read-only:"true"`
	RoShares           []string        `json:"ro_shares,omitempty" read-only:"true"`
}
