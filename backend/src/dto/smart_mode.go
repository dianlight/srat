package dto

//go:generate go tool goenums smart_mode.go
type smartMode int

const (
	SmartModeNone   smartMode = iota // "none"
	SmartModeLegacy                  // "legacy"
	SmartModeDirect                  // "direct"
)
