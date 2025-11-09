package events

import (
	"github.com/dianlight/srat/dto"
)

type Event struct {
	Type EventType
}

// DiskEvent represents a disk-related event
type DiskEvent struct {
	Event
	Disk *dto.Disk
}

// PartitionEvent represents a partition-related event
type PartitionEvent struct {
	Event
	Partition *dto.Partition
	Disk      *dto.Disk
}

// ShareEvent represents a share-related event
type ShareEvent struct {
	Event
	Share *dto.SharedResource
}

// MountPointEvent represents a mount point event
type MountPointEvent struct {
	Event
	MountPoint *dto.MountPointData
}
