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

// HDIdleDeviceStatus represents the HD idle status for a single disk.
type HDIdleDeviceStatus struct {
	SpunDown       bool   `json:"spun_down"`
	LastIOAt       string `json:"last_io_at,omitempty"`       // ISO8601 timestamp
	SpinDownAt     string `json:"spin_down_at,omitempty"`     // ISO8601 timestamp
	SpinUpAt       string `json:"spin_up_at,omitempty"`       // ISO8601 timestamp
	IdleTimeMillis int64  `json:"idle_time_millis,omitempty"` // Configured idle time in milliseconds
	CommandType    string `json:"command_type,omitempty"`     // "scsi" or "ata"
	Supported      bool   `json:"supported"`                  // Whether HD idle is supported for this device
	Enabled        bool   `json:"enabled"`                    // Whether HD idle monitoring is enabled for this device
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
