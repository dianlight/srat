package dto

// DiskIOStats contains I/O statistics for a single disk.
type DiskIOStats struct {
	DeviceName        string       `json:"device_name"`
	DeviceDescription string       `json:"device_description"`
	ReadIOPS          float64      `json:"read_iops"`
	WriteIOPS         float64      `json:"write_iops"`
	ReadLatency       float64      `json:"read_latency_ms"`
	WriteLatency      float64      `json:"write_latency_ms"`
	SmartData         *SmartStatus `json:"smart_data,omitempty"`
}

// GlobalDiskStats contains the aggregated I/O and latency statistics for all disks.
type GlobalDiskStats struct {
	TotalIOPS         float64 `json:"total_iops"`
	TotalReadLatency  float64 `json:"total_read_latency_ms"`
	TotalWriteLatency float64 `json:"total_write_latency_ms"`
}

// PerPartitionInfo contains per-partition health information such as freespace and fsck support.
type PerPartitionInfo struct {
	Name          string `json:"name,omitempty"`
	MountPoint    string `json:"mount_point"`
	Device        string `json:"device"`
	FSType        string `json:"fstype"`
	FreeSpace     uint64 `json:"free_space_bytes"`
	TotalSpace    uint64 `json:"total_space_bytes"`
	FsckNeeded    bool   `json:"fsck_needed"`
	FsckSupported bool   `json:"fsck_supported"`
}



// PerDiskInfo contains per-disk health and status information.
type PerDiskInfo struct {
	DeviceId     string              `json:"device_id"`
	DevicePath   string              `json:"device_path,omitempty"`
	SmartInfo    *SmartInfo          `json:"smart_info,omitempty"`
	SmartHealth  *SmartHealthStatus  `json:"smart_health,omitempty"`
	HDIdleStatus *HDIdleDeviceStatus `json:"hdidle_status,omitempty"`
}

// DiskHealth contains all disk-related health information.
type DiskHealth struct {
	Global           GlobalDiskStats               `json:"global"`
	PerDiskIO        []DiskIOStats                 `json:"per_disk_io"`
	PerPartitionInfo map[string][]PerPartitionInfo `json:"per_partition_info"`
	PerDiskInfo      map[string]PerDiskInfo        `json:"per_disk_info,omitempty"`
	HDIdleRunning    bool                          `json:"hdidle_running"`
}
