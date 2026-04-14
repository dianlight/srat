package dto

//go:generate go tool goenums repair_lifecycle_status.go
type repairLifecycleStatus int

const (
	repairLifecycleStatusCreated   repairLifecycleStatus = iota // "created"
	repairLifecycleStatusUpdated                                // "updated"
	repairLifecycleStatusIgnored                                // "ignored"
	repairLifecycleStatusFixed                                  // "fixed"
	repairLifecycleStatusDismissed                              // "dismissed"
	repairLifecycleStatusDeleted                                // "deleted"
	repairLifecycleStatusError                                  // "error"
)
