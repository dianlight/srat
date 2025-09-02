package dto

//go:generate go tool goenums issue_severity.go
type issueSeverity int

const (
	issueSeverityError   issueSeverity = iota // "error"
	issueSeverityWarning                      // "warning"
	issueSeverityInfo                         // "info"
	issueSeveritySuccess                      // "success"
)
