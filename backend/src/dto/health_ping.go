package dto

type HealthPing struct {
	Alive              bool               `json:"alive"`
	AliveTime          int64              `json:"aliveTime"`
	ReadOnly           bool               `json:"read_only"`
	SambaProcessStatus SambaProcessStatus `json:"samba_process_status"`
	LastError          string             `json:"last_error"`
	Dirty              DataDirtyTracker   `json:"dirty_tracking"`
	LastRelease        ReleaseAsset       `json:"last_release"`
}
