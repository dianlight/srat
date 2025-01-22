package dto

type MountPointData struct {
	ID          uint   `json:"id"`
	Path        string `json:"path"`
	PrimaryPath string `json:"primary_path"`
	//DefaultPath string `json:"default_path"`
	//Label       string        `json:"label"`
	FSType string        `json:"fstype"`
	Flags  MounDataFlags `json:"flags,omitempty"`
	//Data         string        `json:"data,omitempty"`
	//DeviceId     uint64 `json:"device_id,omitempty"`
	Source       string  `json:"source,omitempty"`
	Invalid      bool    `json:"invalid,omitempty"`
	InvalidError *string `json:"invalid_error,omitempty"`
	Warnings     *string `json:"warnings,omitempty"`
}
