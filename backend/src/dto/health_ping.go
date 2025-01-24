package dto

type HealthPing struct {
	Alive              bool               `json:"alive"`
	ReadOnly           bool               `json:"read_only"`
	SambaProcessStatus SambaProcessStatus `json:"samba_process_status"`
	LastError          string             `json:"last_error"`
}
