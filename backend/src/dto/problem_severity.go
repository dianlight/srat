package dto

//go:generate go tool goenums problem_severity.go
type problemSeverity int

const (
	problemSeverityInfo     problemSeverity = iota // "info"
	problemSeverityWarning                         // "warning"
	problemSeverityError                           // "error"
	problemSeverityCritical                        // "critical"
)
