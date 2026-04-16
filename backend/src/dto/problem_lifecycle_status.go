package dto

//go:generate go tool goenums problem_lifecycle_status.go
type problemLifecycleStatus int

const (
	problemLifecycleStatusCreated   problemLifecycleStatus = iota // "created"
	problemLifecycleStatusUpdated                                 // "updated"
	problemLifecycleStatusIgnored                                 // "ignored"
	problemLifecycleStatusFixed                                   // "fixed"
	problemLifecycleStatusDismissed                               // "dismissed"
	problemLifecycleStatusDeleted                                 // "deleted"
	problemLifecycleStatusError                                   // "error"
)
