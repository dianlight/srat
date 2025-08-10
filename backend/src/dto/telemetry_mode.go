package dto

//go:generate go tool goenums telemetry_mode.go
type telemetryMode int

const (
	TelemetryModeAsk      telemetryMode = iota // "Ask"
	TelemetryModeAll                           // "All"
	TelemetryModeErrors                        // "Errors"
	TelemetryModeDisabled                      // "Disabled"
)
