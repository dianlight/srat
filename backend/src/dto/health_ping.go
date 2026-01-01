package dto

import "github.com/dianlight/srat/homeassistant/addons"

type HealthPing struct {
	Alive              bool                   `json:"alive"`
	AliveTime          int64                  `json:"aliveTime"`
	SambaProcessStatus SambaProcessStatus     `json:"samba_process_status"`
	LastError          string                 `json:"last_error"`
	Dirty              DataDirtyTracker       `json:"dirty_tracking"`
	UpdateAvailable    bool                   `json:"update_available"`
	AddonStats         *addons.AddonStatsData `json:"addon_stats"`
	DiskHealth         *DiskHealth            `json:"disk_health"`
	NetworkHealth      *NetworkStats          `json:"network_health"`
	SambaStatus        *SambaStatus           `json:"samba_status"`
	Uptime             int64                  `json:"uptime"`
}
