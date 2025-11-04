package dto

//go:generate go tool goenums hdidle_enabled.go
type hdidleEnabled int

const (
	yesEnabled    hdidleEnabled = iota // "yes"
	customEnabled                      // "custom"
	noEnabled                          // "no"
)
