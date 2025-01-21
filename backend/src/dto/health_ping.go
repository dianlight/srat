package dto

type HealthPing struct {
	Alive     bool   `json:"alive"`
	ReadOnly  bool   `json:"read_only"`
	Samba     int32  `json:"samba_pid"`
	LastError string `json:"last_error"`
}
