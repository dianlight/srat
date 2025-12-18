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

// Static Info about S.M.A.R.T. capabilities of a disk
type SmartInfo struct {
	Supported    bool   `json:"supported"`
	DiskType     string `json:"disk_type,omitempty" enum:"SATA,NVMe,SCSI,Unknown"`
	ModelFamily  string `json:"model_family,omitempty"`
	ModelName    string `json:"model_name,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	Firmware     string `json:"firmware_version,omitempty"`
	RotationRate int    `json:"rotation_rate,omitempty"` // RPM, only populated if > 0 (HDDs)
	//Additional   map[string]SmartRangeValue `json:"others,omitempty"`
}

// SmartStatus represents current S.M.A.R.T. status of a disk
type SmartStatus struct {
	Enabled         bool                       `json:"enabled"`
	InStandby       bool                       `json:"in_standby,omitempty"` // True if device is in standby/sleep mode (ATA only)
	Temperature     SmartTempValue             `json:"temperature"`
	PowerOnHours    SmartRangeValue            `json:"power_on_hours"`
	PowerCycleCount SmartRangeValue            `json:"power_cycle_count"`
	Additional      map[string]SmartRangeValue `json:"others,omitempty"`
}

// SmartTestType represents the type of SMART test to execute
type SmartTestType string

const (
	SmartTestTypeShort      SmartTestType = "short"
	SmartTestTypeLong       SmartTestType = "long"
	SmartTestTypeConveyance SmartTestType = "conveyance"
)

// SmartTestStatus represents the status of a SMART test
type SmartTestStatus struct {
	Status          string `json:"status"`                     // "idle", "running", "completed", "failed"
	TestType        string `json:"test_type"`                  // Type of test
	PercentComplete int    `json:"percent_complete,omitempty"` // Percentage complete (0-100)
	LBAOfFirstError string `json:"lba_of_first_error,omitempty"`
}

// SmartHealthStatus represents the overall health status of a disk
type SmartHealthStatus struct {
	Passed            bool     `json:"passed"`
	FailingAttributes []string `json:"failing_attributes,omitempty"`
	OverallStatus     string   `json:"overall_status"` // "healthy", "warning", "failing"
}
