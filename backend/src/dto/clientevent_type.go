package dto

//go:generate go tool goenums -l clientevent_type.go
type clientEventType int

const (
	clientEventTypeHelo             clientEventType = iota // "helo"
	clientEventTypeRepairLifecycle                         // "repair_lifecycle"
	clientEventTypeProblemLifecycle                        // "problem_lifecycle"
)
