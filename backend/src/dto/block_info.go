package dto

type BlockInfo struct {
	TotalSizeBytes uint64 `json:"total_size_bytes"`
	// Partitions contains an array of pointers to `Partition` structs, one for
	// each partition on any disk drive on the host system.
	Partitions []*BlockPartition `json:"partitions"`
}
