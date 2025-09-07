package dto

type MountPointData struct {
	DiskLabel          *string             `json:"disk_label,omitempty" read-only:"true"`
	DiskSerial         *string             `json:"disk_serial,omitempty" read-only:"true"`
	DiskSize           *uint64             `json:"disk_size,omitempty" read-only:"true"` // Size of the disk in bytes.
	Path               string              `json:"path" read-only:"true"`
	PathHash           string              `json:"path_hash,omitempty" read-only:"true"`
	Type               string              `json:"type" read-only:"true" enum:"HOST,ADDON"` // Type of the mountpoint.
	FSType             *string             `json:"fstype,omitempty"`
	Flags              *MountFlags         `json:"flags,omitempty"`
	CustomFlags        *MountFlags         `json:"custom_flags,omitempty"`
	DeviceId           string              `json:"device_id,omitempty" read-only:"true"` // Source Device source of the filesystem (e.g. /dev/sda1).
	Partition          *Partition          `json:"partition,omitempty" read-only:"true"` // Partition object ephemeral
	IsMounted          bool                `json:"is_mounted,omitempty" read-only:"true"`
	IsInvalid          bool                `json:"invalid,omitempty" read-only:"true"`
	IsToMountAtStartup *bool               `json:"is_to_mount_at_startup,omitempty"`              // If true, mount point should be mounted at startup.
	IsWriteSupported   *bool               `json:"is_write_supported,omitempty" read-only:"true"` // If true, write operations are supported on this mount point.
	TimeMachineSupport *TimeMachineSupport `json:"time_machine_support,omitempty" read-only:"true" enum:"unsupported,supported,experimental,unknown"`
	InvalidError       *string             `json:"invalid_error,omitempty" read-only:"true"`
	Warnings           *string             `json:"warnings,omitempty" read-only:"true"`
	Shares             []SharedResource    `json:"shares,omitempty" read-only:"true"` // Shares that are mounted on this mount point.
}
