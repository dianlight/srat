package dto

// Disk defines model for  Disk.
type Disk struct {
	// Device Special device file for the drive (e.g. sda).
	LegacyDeviceName *string `json:"legacy_device_name,omitempty"`

	// Device Special device file for the drive (e.g. /dev/sda).
	LegacyDevicePath *string `json:"legacy_device_path,omitempty"`

	// Device Special device file for the drive (e.g. /dev/disk/by-id/).
	DevicePath *string `json:"device_path,omitempty"`

	// ConnectionBus Physical connection bus of the drive (USB, etc.).
	ConnectionBus *string `json:"connection_bus,omitempty"`

	// Ejectable Is the drive ejectable by the system?
	Ejectable *bool `json:"ejectable,omitempty"`

	// Partitions A map of filesystem partitions on the drive keyed by Partition.Id.
	// Using a map allows O(1) lookups and stable identification without relying on array order.
	Partitions *map[string]Partition `json:"partitions,omitempty"`

	// Id Unique and persistent id for drive.
	Id *string `json:"id,omitempty"`

	// Model Drive model.
	Model *string `json:"model,omitempty"`

	// Removable Is the drive removable by the user?
	Removable *bool `json:"removable,omitempty"`

	// Revision Drive revisio.
	Revision *string `json:"revision,omitempty"`

	// Seat Identifier of seat drive is plugged into.
	Seat *string `json:"seat,omitempty"`

	// Serial Drive serial number.
	Serial *string `json:"serial,omitempty"`

	// Size Size of the drive in bytes.
	Size *int `json:"size,omitempty"`

	// TimeDetected Time drive was detected by system.
	//TimeDetected *time.Time `json:"time_detected,omitempty"`

	// Vendor Drive vendor.
	Vendor *string `json:"vendor,omitempty"`

	// S.M.A.R.T. info, if available.
	SmartInfo *SmartInfo `json:"smart_info,omitempty" readonly:"true"`

	// HDIdleStatus contains current HDIdle configuration snapshot for this disk, if available.
	HDIdleStatus *HDIdleDevice `json:"hdidle_status,omitempty" readonly:"true"`

	// Refresh version counter to indicate when the disk info was last refreshed.
	RefreshVersion uint32 `json:"refresh_version,omitempty" readonly:"true"`
}

// Partition defines model for Filesystem/Partition.
type Partition struct {
	// Device Special device file for the filesystem (e.g. /dev/sda1).
	LegacyDevicePath *string `json:"legacy_device_path,omitempty"`

	// Device Special device file for the filesystem (e.g. sda1).
	LegacyDeviceName *string `json:"legacy_device_name,omitempty"`

	// Device Special device file for the filesystem (e.g. /dev/disk/by-id/).
	DevicePath *string `json:"device_path,omitempty"`

	// Id Unique and persistent id for disk partition.
	Id *string `json:"id,omitempty"`

	// uuid UUID of the filesystem, if available uniquely identifying it.
	Uuid *string `json:"uuid,omitempty"`

	// DiskId Unique and persistent id for parent drive.
	DiskId *string `json:"disk_id,omitempty"`

	// FsType Filesystem type (e.g. ext4, ntfs, etc.).
	FsType *string `json:"fs_type,omitempty"`

	// Name Name of the filesystem (if known).
	Name *string `json:"name,omitempty"`

	// Size Size of the filesystem in bytes.
	Size *int `json:"size,omitempty"`

	// System true if filesystem considered a system/internal device.
	System *bool `json:"system,omitempty"`

	// HostMountPointData A map of mount points on the host-side keyed by MountPointData.Path.
	// Using a map allows O(1) lookups and stable identification by path.
	HostMountPointData *map[string]MountPointData `json:"host_mount_point_data,omitempty"`

	// MountPointData A map of mount points on the addon-side keyed by MountPointData.Path.
	// Using a map allows O(1) lookups and stable identification by path.
	MountPointData *map[string]MountPointData `json:"mount_point_data,omitempty"`

	// Refresh version counter to indicate when the partition info was last refreshed.
	RefreshVersion uint32 `json:"refresh_version,omitempty" readonly:"true"`
}
