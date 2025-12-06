package dto

type SharedResource struct {
	_                  struct{}              `json:"-" additionalProperties:"true"`
	Name               string                `json:"name,omitempty"  mapper:"mapkey"`
	Disabled           *bool                 `json:"disabled,omitempty"`
	Users              []User                `json:"users,omitempty"`
	RoUsers            []User                `json:"ro_users,omitempty"`
	TimeMachine        *bool                 `json:"timemachine,omitempty"`
	RecycleBin         *bool                 `json:"recycle_bin_enabled,omitempty"`
	GuestOk            *bool                 `json:"guest_ok,omitempty"`
	TimeMachineMaxSize *string               `json:"timemachine_max_size,omitempty"`
	Usage              HAMountUsage          `json:"usage,omitempty" enum:"none,backup,media,share,internal"`
	VetoFiles          []string              `json:"veto_files,omitempty"  nullable:"false"`
	MountPointData     *MountPointData       `json:"mount_point_data,omitempty"`
	Status             *SharedResourceStatus `json:"status,omitempty" read-only:"true"`
}

type SharedResourceStatus struct {
	IsValid     bool `json:"is_valid,omitempty" default:"false" read-only:"true"`
	IsHAMounted bool `json:"is_ha_mounted,omitempty" default:"false" read-only:"true"`
}
