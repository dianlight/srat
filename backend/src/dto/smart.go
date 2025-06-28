package dto

type SmartInfo struct {
	Temperature     uint64 `json:"temperature"`
	PowerOnHours    uint64 `json:"power_on_hours"`
	PowerCycleCount uint64 `json:"power_cycle_count"`
}
