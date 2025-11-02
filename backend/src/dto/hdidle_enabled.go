package dto

//go:generate go tool goenums hdidle_enabled.go
type hdidleEnabled int

const (
	defaultEnabled hdidleEnabled = iota // "default"
	yesEnabled                          // "yes"
	noEnabled                           // "no"
)
