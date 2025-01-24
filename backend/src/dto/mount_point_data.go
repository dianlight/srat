package dto

type MountPointData struct {
	ID           uint          `json:"id"`
	Path         string        `json:"path"`
	PrimaryPath  string        `json:"primary_path"`
	FSType       string        `json:"fstype"`
	Flags        MounDataFlags `json:"flags,omitempty"`
	Source       string        `json:"source,omitempty"`
	IsMounted    bool          `json:"is_mounted,omitempty"`
	IsInvalid    bool          `json:"invalid,omitempty"`
	InvalidError *string       `json:"invalid_error,omitempty"`
	Warnings     *string       `json:"warnings,omitempty"`
}
