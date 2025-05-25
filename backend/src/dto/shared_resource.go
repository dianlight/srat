package dto

type SharedResource struct {
	Name        string       `json:"name,omitempty"  mapper:"mapkey"`
	Disabled    *bool        `json:"disabled,omitempty"`
	Users       []User       `json:"users,omitempty"`
	RoUsers     []User       `json:"ro_users,omitempty"`
	TimeMachine *bool        `json:"timemachine,omitempty"`
	Usage       HAMountUsage `json:"usage,omitempty" enum:"none,backup,media,share,internal"`
	IsHAMounted *bool        `json:"is_ha_mounted,omitempty"`

	//DeviceId       *uint64        `json:"device_id,omitempty"`
	MountPointData *MountPointData `json:"mount_point_data,omitempty"`

	Invalid *bool `json:"invalid,omitempty"`
}
