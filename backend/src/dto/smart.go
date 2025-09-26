package dto

type SmartRangeValue struct {
	Code       int `json:"code,omitempty"`
	Value      int `json:"value"`
	Min        int `json:"min,omitempty"`
	Worst      int `json:"worst,omitempty"`
	Thresholds int `json:"thresholds,omitempty"`
}

type SmartTempValue struct {
	Value           int `json:"value"`
	Min             int `json:"min,omitempty"`
	Max             int `json:"max,omitempty"`
	OvertempCounter int `json:"overtemp_counter,omitempty"`
}

type SmartInfo struct {
	DiskType        string                     `json:"disk_type,omitempty" enum:"SATA,NVMe,SCSI,Unknown"`
	Temperature     SmartTempValue             `json:"temperature"`
	PowerOnHours    SmartRangeValue            `json:"power_on_hours"`
	PowerCycleCount SmartRangeValue            `json:"power_cycle_count"`
	Additional      map[string]SmartRangeValue `json:"others,omitempty"`
}
