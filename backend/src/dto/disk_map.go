package dto

import "gitlab.com/tozd/go/errors"

type DiskMap map[string]*Disk

// Add inserts or updates a Disk in the map using its Id as the key.
// It initializes the map if it is nil. Returns an error if the disk Id is nil or empty.
func (m *DiskMap) Add(d *Disk) error {
	if d.Id == nil || *d.Id == "" {
		return errors.WithDetails(ErrorInvalidParameter, "Message", "disk id is nil or empty")
	}
	//	if *m == nil {
	//		*m = make(DiskMap)
	//	}
	(*m)[*d.Id] = d
	return nil
}

// Remove deletes a Disk from the map by its id.
// It returns true if the disk was present and removed, false otherwise.
func (m *DiskMap) Remove(id string) bool {
	if m == nil || *m == nil || id == "" {
		return false
	}
	if _, ok := (*m)[id]; ok {
		delete(*m, id)
		return true
	}
	return false
}

// Get returns the Disk for the given id and a boolean indicating if it exists.
func (m *DiskMap) Get(id string) (*Disk, bool) {
	if m == nil || *m == nil || id == "" {
		return nil, false
	}
	d, ok := (*m)[id]
	return d, ok
}

// AddMountPoint inserts or updates a MountPointData in the specified partition of the specified disk.
// The mount point is keyed by its Path field. Returns an error if inputs are invalid or the target disk/partition is missing.
func (m *DiskMap) AddMountPoint(diskID, partitionID string, mpd MountPointData) error {
	if m == nil || *m == nil {
		return errors.WithDetails(ErrorNotFound, "Message", "disk map is nil or empty")
	}
	if diskID == "" {
		return errors.WithDetails(ErrorInvalidParameter, "Message", "disk id is empty")
	}
	if partitionID == "" {
		return errors.WithDetails(ErrorInvalidParameter, "Message", "partition id is empty")
	}
	if mpd.Path == "" {
		return errors.WithDetails(ErrorInvalidParameter, "Message", "mount point path is empty")
	}

	d, ok := (*m)[diskID]
	if !ok {
		return errors.WithDetails(ErrorNotFound, "Message", "disk not found", "DiskId", diskID)
	}
	if d.Partitions == nil {
		return errors.WithDetails(ErrorNotFound, "Message", "disk has no partitions", "DiskId", diskID)
	}
	part, ok := (*d.Partitions)[partitionID]
	if !ok {
		return errors.WithDetails(ErrorNotFound, "Message", "partition not found", "DiskId", diskID, "PartitionId", partitionID)
	}

	if part.MountPointData == nil {
		mp := make(map[string]MountPointData)
		part.MountPointData = &mp
	}

	// Ensure DeviceId matches the partition id if not set
	if mpd.DeviceId == "" && part.Id != nil {
		mpd.DeviceId = *part.Id
	}
	// Optionally associate partition reference
	if mpd.Partition == nil {
		tmp := part // copy to avoid referencing future mutated value inadvertently
		mpd.Partition = &tmp
	}

	(*part.MountPointData)[mpd.Path] = mpd
	(*d.Partitions)[partitionID] = part
	(*m)[diskID] = d
	return nil
}

// RemoveMountPoint deletes a MountPointData by path from the specified partition of the specified disk.
// Returns true if the mount point existed and was removed.
func (m *DiskMap) RemoveMountPoint(diskID, partitionID, path string) bool {
	if m == nil || *m == nil || diskID == "" || partitionID == "" || path == "" {
		return false
	}
	d, ok := (*m)[diskID]
	if !ok || d.Partitions == nil {
		return false
	}
	part, ok := (*d.Partitions)[partitionID]
	if !ok || part.MountPointData == nil {
		return false
	}
	if _, ok := (*part.MountPointData)[path]; ok {
		delete(*part.MountPointData, path)
		(*d.Partitions)[partitionID] = part
		(*m)[diskID] = d
		return true
	}
	return false
}

// AddPartition inserts or updates a Partition in the specified disk.
// Returns an error if the disk is not found or the partition id is invalid.
func (m *DiskMap) AddPartition(diskID string, p Partition) error {
	if m == nil || *m == nil {
		return errors.WithDetails(ErrorNotFound, "Message", "disk map is nil or empty")
	}
	if diskID == "" {
		return errors.WithDetails(ErrorInvalidParameter, "Message", "disk id is empty")
	}
	if p.Id == nil || *p.Id == "" {
		return errors.WithDetails(ErrorInvalidParameter, "Message", "partition id is nil or empty")
	}

	d, ok := (*m)[diskID]
	if !ok {
		return errors.WithDetails(ErrorNotFound, "Message", "disk not found", "DiskId", diskID)
	}

	if d.Partitions == nil {
		mp := make(map[string]Partition)
		d.Partitions = &mp
	}

	// Ensure partition has proper DiskId
	if p.DiskId == nil || *p.DiskId != diskID {
		p.DiskId = &diskID
	}

	(*d.Partitions)[*p.Id] = p
	(*m)[diskID] = d
	return nil
}

// RemovePartition deletes a Partition from the specified disk by partition id.
// It returns true if the partition was present and removed, false otherwise.
func (m *DiskMap) RemovePartition(diskID, partitionID string) bool {
	if m == nil || *m == nil || diskID == "" || partitionID == "" {
		return false
	}
	d, ok := (*m)[diskID]
	if !ok || d.Partitions == nil {
		return false
	}
	if _, ok := (*d.Partitions)[partitionID]; ok {
		delete(*d.Partitions, partitionID)
		(*m)[diskID] = d
		return true
	}
	return false
}

// GetPartition retrieves the specified partition from the given disk.
// Returns the partition value and true if it exists; otherwise returns false.
func (m *DiskMap) GetPartition(diskID, partitionID string) (Partition, bool) {
	if m == nil || *m == nil || diskID == "" || partitionID == "" {
		return Partition{}, false
	}
	if d, ok := (*m)[diskID]; ok && d.Partitions != nil {
		if p, ok := (*d.Partitions)[partitionID]; ok {
			return p, true
		}
	}
	return Partition{}, false
}

// GetMountPoint retrieves a mount point from the specified disk partition by path.
// Returns the mount point data and true if it exists; otherwise returns false.
func (m *DiskMap) GetMountPoint(diskID, partitionID, path string) (*MountPointData, bool) {
	if path == "" {
		return nil, false
	}
	partition, ok := m.GetPartition(diskID, partitionID)
	if !ok || partition.MountPointData == nil {
		return nil, false
	}
	if mp, ok := (*partition.MountPointData)[path]; ok {
		return &mp, true
	}
	return nil, false
}

// GetMountPointByPath searches all disks and partitions for a mount point matching the given path.
// Returns the mount point data and true if found, otherwise returns false.
func (m *DiskMap) GetMountPointByPath(path string) (*MountPointData, bool) {
	if m == nil || *m == nil || path == "" {
		return nil, false
	}
	for _, d := range *m {
		if d.Partitions == nil {
			continue
		}
		for _, part := range *d.Partitions {
			if part.MountPointData == nil {
				continue
			}
			if mp, ok := (*part.MountPointData)[path]; ok {
				return &mp, true
			}
		}
	}
	return nil, false
}

func (m *DiskMap) GetAllMountPoints() []*MountPointData {
	var result []*MountPointData
	if m == nil || *m == nil {
		return result
	}
	for _, d := range *m {
		if d.Partitions == nil {
			continue
		}
		for _, part := range *d.Partitions {
			if part.MountPointData == nil {
				continue
			}
			for _, mp := range *part.MountPointData {
				result = append(result, &mp)
			}
		}
	}
	return result
}

