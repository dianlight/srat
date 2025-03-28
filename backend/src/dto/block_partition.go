package dto

type BlockPartition struct {
	// Name is the system name given to the partition, e.g. "sda1".
	Name string `json:"name"`
	// Label is the human-readable label given to the partition. On Linux, this
	// is derived from the `ID_PART_ENTRY_NAME` udev entry.
	Label string `json:"label"`
	// MountPoint is the path where this partition is mounted.
	MountPoint string `json:"mount_point"`
	// MountPoint is the path where this partition is mounted last time
	DefaultMountPoint string `json:"default_mount_point"`
	// SizeBytes contains the total amount of storage, in bytes, this partition
	// can consume.
	SizeBytes uint64 `json:"size_bytes"`
	// Type contains the type of the partition.
	Type string `json:"type"`
	// IsReadOnly indicates if the partition is marked read-only.
	IsReadOnly bool `json:"read_only"`
	// UUID is the universally-unique identifier (UUID) for the partition.
	// This will be volume UUID on Darwin, PartUUID on linux, empty on Windows.
	UUID string `json:"uuid"`
	// FilesystemLabel is the label of the filesystem contained on the
	// partition. On Linux, this is derived from the `ID_FS_NAME` udev entry.
	FilesystemLabel string `json:"filesystem_label"`
	// PartiionFlags contains the mount flags for the partition.
	PartitionFlags []string `json:"partition_flags" enum:"MS_RDONLY,MS_NOSUID,MS_NODEV,MS_NOEXEC,MS_SYNCHRONOUS,MS_REMOUNT,MS_MANDLOCK,MS_NOATIME,MS_NODIRATIME,MS_BIND,MS_LAZYTIME,MS_NOUSER,MS_RELATIME"`
	// MountFlags contains the mount flags for the partition.
	MountFlags []string `json:"mount_flags" enum:"MS_RDONLY,MS_NOSUID,MS_NODEV,MS_NOEXEC,MS_SYNCHRONOUS,MS_REMOUNT,MS_MANDLOCK,MS_NOATIME,MS_NODIRATIME,MS_BIND,MS_LAZYTIME,MS_NOUSER,MS_RELATIME"`
	// MountData contains additional data associated with the partition.
	MountData string `json:"mount_data"`
	// DeviceId is the ID of the block device this partition is on.
	DeviceId *uint64 `json:"device_id"`
	// Relative MountPointData
	MountPointData MountPointData `json:"mount_point_data"`
}
