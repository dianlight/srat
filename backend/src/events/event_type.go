package events

//go:generate go tool goenums -l event_type.go
type eventType int

const (
	Add    eventType = iota // "add"
	Remove                  // "remove"
	Update                  // "update"
	Error                   // "error"
)
