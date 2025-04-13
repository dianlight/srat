package dto

// Disk defines model for  Disk.
type Disk struct {
	// ConnectionBus Physical connection bus of the drive (USB, etc.).
	ConnectionBus *string `json:"connection_bus,omitempty"`

	// Ejectable Is the drive ejectable by the system?
	Ejectable *bool `json:"ejectable,omitempty"`

	// Partitions A list of filesystem partitions on the drive.
	Partitions *[]Partition `json:"partitions,omitempty"`

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
}

// Partition defines model for Filesystem/Partition.
type Partition struct {
	// Device Special device file for the filesystem (e.g. /dev/sda1).
	Device *string `json:"device,omitempty"`

	// Id Unique and persistent id for filesystem.
	Id *string `json:"id,omitempty"`

	// MountPoints List of paths where the filesystem is mounted on host.
	// MountPoints *[]string `json:"mount_points,omitempty"`

	// Name Name of the filesystem (if known).
	Name *string `json:"name,omitempty"`

	// Size Size of the filesystem in bytes.
	Size *int `json:"size,omitempty"`

	// System true if filesystem considered a system/internal device.
	System *bool `json:"system,omitempty"`

	// MountPointData to mount on the addon-side ( created only if no mountpoint exists on the host side and is not a system/internal device )
	MountPointData *[]MountPointData `json:"mount_point_data,omitempty"`
}
