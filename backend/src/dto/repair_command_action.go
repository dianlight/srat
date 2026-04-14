package dto

//go:generate go tool goenums repair_command_action.go
type repairCommandAction int

const (
	repairCommandActionUpsert    repairCommandAction = iota // "upsert"
	repairCommandActionDelete                               // "delete"
	repairCommandActionReconcile                            // "reconcile"
)
