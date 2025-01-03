package dto

import (
	"net/http"

	"github.com/dianlight/srat/data"
	"github.com/jinzhu/copier"
)

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
	PartitionFlags data.MounDataFlags `json:"partition_flags"`
	// MountFlags contains the mount flags for the partition.
	MountFlags data.MounDataFlags `json:"mount_flags"`
	// MountData contains additional data associated with the partition.
	MountData string `json:"mount_data"`
}

func (self *BlockPartition) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *BlockPartition) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self BlockPartition) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self BlockPartition) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self BlockPartition) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self BlockPartition) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *BlockPartition) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
