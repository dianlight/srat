package dto

//go:generate go tool goenums hdidle_enabled.go
type hdidleEnabled int

const (
	customEnabled hdidleEnabled = iota // "custom"
	noEnabled                          // "no"
)
