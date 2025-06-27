package dto

import "github.com/dianlight/srat/homeassistant/addons"

type HealthPing struct {
	Alive              bool                   `json:"alive"`
	AliveTime          int64                  `json:"aliveTime"`
	StartTime          int64                  `json:"startTime"`
	ReadOnly           bool                   `json:"read_only"`
	SambaProcessStatus SambaProcessStatus     `json:"samba_process_status"`
	LastError          string                 `json:"last_error"`
	Dirty              DataDirtyTracker       `json:"dirty_tracking"`
	LastRelease        ReleaseAsset           `json:"last_release"`
	SecureMode         bool                   `json:"secure_mode"`
	ProtectedMode      bool                   `json:"protected_mode"`
	BuildVersion       string                 `json:"build_version"`
	AddonStats         *addons.AddonStatsData `json:"addon_stats"`
	DiskHealth         DiskHealth             `json:"disk_health"`
}
