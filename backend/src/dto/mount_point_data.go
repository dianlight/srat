package dto

type MountPointData struct {
	Path         string     `json:"path"`
	PathHash     string     `json:"path_hash,omitempty" read-only:"true"`
	Type         string     `json:"type" enum:"HOST,ADDON"` // Type of the mountpoint.
	FSType       string     `json:"fstype,omitempty"`
	Flags        MountFlags `json:"flags,omitempty"`
	CustomFlags  MountFlags `json:"custom_flags,omitempty"`
	Device       string     `json:"device,omitempty"` // Source Device source of the filesystem (e.g. /dev/sda1).
	IsMounted    bool       `json:"is_mounted,omitempty"`
	IsInvalid    bool       `json:"invalid,omitempty"`
	InvalidError *string    `json:"invalid_error,omitempty"`
	Warnings     *string    `json:"warnings,omitempty"`
}
