package dto

type MountPointData struct {
	ID           uint     `json:"id"`
	Path         string   `json:"path"`
	PrimaryPath  string   `json:"primary_path,omitempty"`
	FSType       string   `json:"fstype,omitempty"`
	Flags        []string `json:"flags,omitempty" enum:"MS_RDONLY,MS_NOSUID,MS_NODEV,MS_NOEXEC,MS_SYNCHRONOUS,MS_REMOUNT,MS_MANDLOCK,MS_NOATIME,MS_NODIRATIME,MS_BIND,MS_LAZYTIME,MS_NOUSER,MS_RELATIME"`
	Source       string   `json:"source,omitempty"`
	IsMounted    bool     `json:"is_mounted,omitempty"`
	IsInvalid    bool     `json:"invalid,omitempty"`
	InvalidError *string  `json:"invalid_error,omitempty"`
	Warnings     *string  `json:"warnings,omitempty"`
}
