package dto

type MountPointData struct {
	Path               string           `json:"path" read-only:"true"`
	PathHash           string           `json:"path_hash,omitempty" read-only:"true"`
	Type               string           `json:"type" read-only:"true" enum:"HOST,ADDON"` // Type of the mountpoint.
	FSType             *string          `json:"fstype,omitempty"`
	Flags              *MountFlags      `json:"flags,omitempty"`
	CustomFlags        *MountFlags      `json:"custom_flags,omitempty"`
	Device             string           `json:"device,omitempty" read-only:"true"` // Source Device source of the filesystem (e.g. /dev/sda1).
	IsMounted          bool             `json:"is_mounted,omitempty" read-only:"true"`
	IsInvalid          bool             `json:"invalid,omitempty" read-only:"true"`
	IsToMountAtStartup *bool            `json:"is_to_mount_at_startup,omitempty"` // If true, mount point should be mounted at startup.
	InvalidError       *string          `json:"invalid_error,omitempty" read-only:"true"`
	Warnings           *string          `json:"warnings,omitempty" read-only:"true"`
	Shares             []SharedResource `json:"shares,omitempty" read-only:"true"` // Shares that are mounted on this mount point.
}
