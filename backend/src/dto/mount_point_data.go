package dto

type MountPointData struct {
	// DeviceId Unique and persistent id for filesystem.
	//ID string `json:"id"`
	// Path Mount point path.
	Path string `json:"path"`
	//PrimaryPath string   `json:"primary_path,omitempty"`
	FSType string   `json:"fstype,omitempty"`
	Flags  []string `json:"flags,omitempty" enum:"MS_RDONLY,MS_NOSUID,MS_NODEV,MS_NOEXEC,MS_SYNCHRONOUS,MS_REMOUNT,MS_MANDLOCK,MS_NOATIME,MS_NODIRATIME,MS_BIND,MS_LAZYTIME,MS_NOUSER,MS_RELATIME"`
	// Source Device source of the filesystem (e.g. /dev/sda1).
	Device       string  `json:"device,omitempty"`
	IsMounted    bool    `json:"is_mounted,omitempty"`
	IsInvalid    bool    `json:"invalid,omitempty"`
	InvalidError *string `json:"invalid_error,omitempty"`
	Warnings     *string `json:"warnings,omitempty"`
}
