package dto

type SharedResource struct {
	_                  struct{}     `json:"-" additionalProperties:"true"`
	Name               string       `json:"name,omitempty"  mapper:"mapkey" maxLength:"128"`
	Disabled           *bool        `json:"disabled,omitempty"`
	Users              []User       `json:"users,omitempty"`
	RoUsers            []User       `json:"ro_users,omitempty"`
	TimeMachine        *bool        `json:"timemachine,omitempty"`
	RecycleBin         *bool        `json:"recycle_bin_enabled,omitempty"`
	GuestOk            *bool        `json:"guest_ok,omitempty"`
	TimeMachineMaxSize *string      `json:"timemachine_max_size,omitempty"`
	Usage              HAMountUsage `json:"usage,omitempty" enum:"none,backup,media,share,internal"`
	VetoFiles          []string     `json:"veto_files,omitempty"  nullable:"false"`
	// Subfolder is the folder inside the volume that the share exposes,
	// relative to the mount point (issue #184). Empty means the volume root.
	// A pointer so an update that omits it keeps the stored value instead of
	// silently widening the share to the whole volume.
	Subfolder      *string               `json:"subfolder,omitempty" maxLength:"1024" doc:"Folder inside the volume exposed by the share, relative to the mount point. Empty means the volume root. Omit on update to keep the current value."`
	MountPointData *MountPointData       `json:"mount_point_data,omitempty"`
	Status         *SharedResourceStatus `json:"status,omitempty" read-only:"true"`
}

type SharedResourceStatus struct {
	IsValid     bool `json:"is_valid" default:"false" read-only:"true"`
	IsHAMounted bool `json:"is_ha_mounted,omitempty" default:"false" read-only:"true"`
	// IsHidden marks standard shares hidden by the standard_share_names mode
	// (issue #1142). Annotated, never persisted; zero value visible.
	IsHidden bool `json:"is_hidden" default:"false" read-only:"true"`
}
