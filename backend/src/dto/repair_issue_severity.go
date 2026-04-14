package dto

//go:generate go tool goenums repair_issue_severity.go
type repairIssueSeverity int

const (
	repairIssueSeverityWarning  repairIssueSeverity = iota // "warning"
	repairIssueSeverityError                               // "error"
	repairIssueSeverityCritical                            // "critical"
)
