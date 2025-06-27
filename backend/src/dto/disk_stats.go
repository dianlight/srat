package dto

// DiskIOStats contains I/O statistics for a single disk.
type DiskIOStats struct {
	DeviceName   string  `json:"device_name"`
	ReadIOPS     float64 `json:"read_iops"`
	WriteIOPS    float64 `json:"write_iops"`
	ReadLatency  float64 `json:"read_latency_ms"`
	WriteLatency float64 `json:"write_latency_ms"`
}

// GlobalDiskStats contains the aggregated I/O and latency statistics for all disks.
type GlobalDiskStats struct {
	TotalIOPS         float64 `json:"total_iops"`
	TotalReadLatency  float64 `json:"total_read_latency_ms"`
	TotalWriteLatency float64 `json:"total_write_latency_ms"`
}

// DiskHealth contains all disk-related health information.
type DiskHealth struct {
	Global    GlobalDiskStats `json:"global"`
	PerDiskIO []DiskIOStats   `json:"per_disk_io"`
}
