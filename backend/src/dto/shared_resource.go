package dto

type SharedResource struct {
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
