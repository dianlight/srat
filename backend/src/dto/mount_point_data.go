package dto

type MountPointData struct {
	Path        string        `json:"path"`
	DefaultPath string        `json:"default_path"`
	Label       string        `json:"label"`
	Name        string        `json:"name"`
	FSType      string        `json:"fstype"`
	Flags       MounDataFlags `json:"flags"`
	Data        string        `json:"data,omitempty"`
}
