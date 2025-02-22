package dto

type HAMountUsage string // https://developers.home-assistant.io/docs/api/supervisor/models#mount

const (
	UsageAsNone     HAMountUsage = "none"
	UsageAsBackup   HAMountUsage = "backup"
	UsageAsMedia    HAMountUsage = "media"
	UsageAsShare    HAMountUsage = "share"
	UsageAsInternal HAMountUsage = "internal"
)

type SharedResource struct {
	ID          *uint        `json:"id,omitempty"`
	Name        string       `json:"name,omitempty"  mapper:"mapkey"`
	Disabled    *bool        `json:"disabled,omitempty"`
	Users       []User       `json:"users"`
	RoUsers     []User       `json:"ro_users"`
	TimeMachine *bool        `json:"timemachine,omitempty"`
	Usage       HAMountUsage `json:"usage,omitempty"`

	//DeviceId       *uint64        `json:"device_id,omitempty"`
	MountPointData *MountPointData `json:"mount_point_data,omitempty"`

	Invalid *bool `json:"invalid,omitempty"`
}
