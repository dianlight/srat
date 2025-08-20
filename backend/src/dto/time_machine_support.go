package dto

//go:generate go tool goenums time_machine_support.go
type timeMachineSupport int

const (
	unsupported  timeMachineSupport = iota // "unsupported"
	supported                              // "supported"
	experimental                           // "experimental"
	unknown                                // "unknown"

)
