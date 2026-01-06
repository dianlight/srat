package dto

import "gitlab.com/tozd/go/errors"

type DiskMap map[string]*Disk

// AddOrUpdate inserts or updates a Disk in the map using its Id as the key.
// It initializes the map if it is nil. Returns an error if the disk Id is nil or empty.
func (m *DiskMap) AddOrUpdate(d *Disk) error {
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

// AddOrUpdateMountPoint inserts or updates a MountPointData in the specified partition of the specified disk.
// The mount point is keyed by its Path field. Returns an error if inputs are invalid or the target disk/partition is missing.
func (m *DiskMap) AddOrUpdateMountPoint(diskID, partitionID string, mpd MountPointData) error {
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

// AddMountPointShare sets the Share field on the mount point.
// Extracts diskID, partitionID, and path from the share's MountPointData.
// If partition info is nil in the share, searches existing mount points for the partition.
// Returns an error if the share, mount point data, or disk/partition is not found.
func (m *DiskMap) AddMountPointShare(share *SharedResource) (*Disk, error) {
	if m == nil || *m == nil {
		return nil, errors.WithDetails(ErrorNotFound, "Message", "disk map is nil or empty")
	}
	if share == nil {
		return nil, errors.WithDetails(ErrorInvalidParameter, "Message", "share is nil")
	}
	if share.MountPointData == nil {
		return nil, errors.WithDetails(ErrorInvalidParameter, "Message", "share mount point data is nil")
	}
	if share.MountPointData.Path == "" {
		return nil, errors.WithDetails(ErrorInvalidParameter, "Message", "mount point path is empty")
	}

	path := share.MountPointData.Path
	var diskID, partitionID string

	// If partition info is provided in share, use it
	if share.MountPointData.Partition != nil {
		if share.MountPointData.Partition.DiskId == nil || *share.MountPointData.Partition.DiskId == "" {
			return nil, errors.WithDetails(ErrorInvalidParameter, "Message", "partition disk id is nil or empty")
		}
		if share.MountPointData.Partition.Id == nil || *share.MountPointData.Partition.Id == "" {
			return nil, errors.WithDetails(ErrorInvalidParameter, "Message", "partition id is nil or empty")
		}
		diskID = *share.MountPointData.Partition.DiskId
		partitionID = *share.MountPointData.Partition.Id
	} else {
		// Search for the mount point in existing disks/partitions to find disk and partition info
		found := false
		for dID, d := range *m {
			if d.Partitions == nil {
				continue
			}
			for pID, part := range *d.Partitions {
				if part.MountPointData == nil {
					continue
				}
				if _, ok := (*part.MountPointData)[path]; ok {
					diskID = dID
					partitionID = pID
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return nil, errors.WithDetails(ErrorNotFound, "Message", "mount point not found", "Path", path)
		}
	}

	d, ok := (*m)[diskID]
	if !ok {
		return nil, errors.WithDetails(ErrorNotFound, "Message", "disk not found", "DiskId", diskID)
	}
	if d.Partitions == nil {
		return nil, errors.WithDetails(ErrorNotFound, "Message", "disk has no partitions", "DiskId", diskID)
	}
	part, ok := (*d.Partitions)[partitionID]
	if !ok {
		return nil, errors.WithDetails(ErrorNotFound, "Message", "partition not found", "DiskId", diskID, "PartitionId", partitionID)
	}
	if part.MountPointData == nil {
		return nil, errors.WithDetails(ErrorNotFound, "Message", "partition has no mount points", "DiskId", diskID, "PartitionId", partitionID)
	}
	mp, ok := (*part.MountPointData)[path]
	if !ok {
		return nil, errors.WithDetails(ErrorNotFound, "Message", "mount point not found", "DiskId", diskID, "PartitionId", partitionID, "Path", path)
	}

	mp.Share = share
	(*part.MountPointData)[path] = mp
	(*d.Partitions)[partitionID] = part
	(*m)[diskID] = d
	return d, nil
}

// RemoveMountPointShare removes the Share field from the mount point that has a share with the given name (sets it to nil).
// Searches all disks and partitions for a mount point with a matching share name.
// Returns true if the mount point with the matching share was found and removed, false otherwise.
func (m *DiskMap) RemoveMountPointShare(shareName string) (bool, *Disk) {
	if m == nil || *m == nil || shareName == "" {
		return false, nil
	}
	for diskID, d := range *m {
		if d.Partitions == nil {
			continue
		}
		for partitionID, part := range *d.Partitions {
			if part.MountPointData == nil {
				continue
			}
			for mpPath, mp := range *part.MountPointData {
				if mp.Share != nil && mp.Share.Name == shareName {
					mp.Share = nil
					(*part.MountPointData)[mpPath] = mp
					(*d.Partitions)[partitionID] = part
					(*m)[diskID] = d
					return true, d
				}
			}
		}
	}
	return false, nil
}

// AddHDIdleDevice sets the HDIdleDevice for the specified disk.
// Returns an error if the disk map is nil, diskID is empty, or the disk is not found.
func (m *DiskMap) AddHDIdleDevice(hdIdleDevice *HDIdleDevice) error {
	if m == nil || *m == nil {
		return errors.WithDetails(ErrorNotFound, "Message", "disk map is nil or empty")
	}
	if hdIdleDevice.DiskId == "" {
		return errors.WithDetails(ErrorInvalidParameter, "Message", "disk id is empty")
	}

	d, ok := (*m)[hdIdleDevice.DiskId]
	if !ok {
		return errors.WithDetails(ErrorNotFound, "Message", "disk not found", "DiskId", hdIdleDevice.DiskId)
	}

	d.HDIdleDevice = hdIdleDevice
	(*m)[hdIdleDevice.DiskId] = d
	return nil
}

// AddSmartInfo sets the SmartInfo for the specified disk.
// Returns an error if the disk map is nil, diskID is empty, or the disk is not found.
func (m *DiskMap) AddSmartInfo(smartInfo *SmartInfo) error {
	if m == nil || *m == nil {
		return errors.WithDetails(ErrorNotFound, "Message", "disk map is nil or empty")
	}
	if smartInfo.DiskId == "" {
		return errors.WithDetails(ErrorInvalidParameter, "Message", "disk id is empty")
	}

	d, ok := (*m)[smartInfo.DiskId]
	if !ok {
		return errors.WithDetails(ErrorNotFound, "Message", "disk not found", "DiskId", smartInfo.DiskId)
	}

	d.SmartInfo = smartInfo
	(*m)[smartInfo.DiskId] = d
	return nil
}
