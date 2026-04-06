package dto

// SanitizeWebEventData clones mutable/cyclic payloads before they are broadcast to
// WebSocket clients so JSON encoding sees a stable, acyclic snapshot.
func SanitizeWebEventData(data any) any {
	switch v := data.(type) {
	case []*Disk:
		return snapshotDiskSlice(v)
	case []SharedResource:
		return snapshotShareSlice(v)
	case SharedResource:
		return snapshotSharedResource(v, true)
	case *SharedResource:
		if v == nil {
			return v
		}
		cloned := snapshotSharedResource(*v, true)
		return &cloned
	case MountPointData:
		return snapshotMountPointData(v, true)
	case *MountPointData:
		if v == nil {
			return v
		}
		cloned := snapshotMountPointData(*v, true)
		return &cloned
	default:
		return data
	}
}

func snapshotDiskSlice(disks []*Disk) []*Disk {
	if disks == nil {
		return nil
	}

	cloned := make([]*Disk, 0, len(disks))
	for _, disk := range disks {
		cloned = append(cloned, snapshotDisk(disk))
	}
	return cloned
}

func snapshotDisk(src *Disk) *Disk {
	if src == nil {
		return nil
	}

	cloned := *src
	if src.Partitions != nil {
		partitions := make(map[string]Partition, len(*src.Partitions))
		for key, partition := range *src.Partitions {
			partitions[key] = snapshotPartition(partition)
		}
		cloned.Partitions = &partitions
	}
	return &cloned
}

func snapshotPartition(src Partition) Partition {
	cloned := src
	if src.MountPointData != nil {
		mountPoints := make(map[string]MountPointData, len(*src.MountPointData))
		for key, mountPoint := range *src.MountPointData {
			mountPoints[key] = snapshotMountPointData(mountPoint, true)
		}
		cloned.MountPointData = &mountPoints
	}
	if src.HostMountPointData != nil {
		hostMountPoints := make(map[string]MountPointData, len(*src.HostMountPointData))
		for key, mountPoint := range *src.HostMountPointData {
			hostMountPoints[key] = snapshotMountPointData(mountPoint, true)
		}
		cloned.HostMountPointData = &hostMountPoints
	}
	return cloned
}

func snapshotMountPointData(src MountPointData, includeShare bool) MountPointData {
	cloned := src
	cloned.Partition = nil
	if src.Share != nil && includeShare {
		share := snapshotSharedResource(*src.Share, false)
		cloned.Share = &share
	} else {
		cloned.Share = nil
	}
	return cloned
}

func snapshotShareSlice(shares []SharedResource) []SharedResource {
	if shares == nil {
		return nil
	}

	cloned := make([]SharedResource, 0, len(shares))
	for _, share := range shares {
		cloned = append(cloned, snapshotSharedResource(share, true))
	}
	return cloned
}

func snapshotSharedResource(src SharedResource, includeMountPoint bool) SharedResource {
	cloned := src
	cloned.Users = append([]User(nil), src.Users...)
	cloned.RoUsers = append([]User(nil), src.RoUsers...)
	cloned.VetoFiles = append([]string(nil), src.VetoFiles...)

	if src.MountPointData != nil && includeMountPoint {
		mountPoint := snapshotMountPointData(*src.MountPointData, false)
		cloned.MountPointData = &mountPoint
	} else if !includeMountPoint {
		cloned.MountPointData = nil
	}

	return cloned
}
