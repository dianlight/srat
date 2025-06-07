package dto

// g.o:generate go run github.com/zarldev/goenums@v0.4.0 event_type.go
//
//go:generate go tool goenums event_type.go
type eventType int // Name[string]

const (
	eventHello     eventType = iota // "hello"
	eventUpdate                     // "update"
	eventUpdating                   // "updating"
	eventVolumes                    // "volumes"
	eventHeartbeat                  // "heartbeat"
	eventShare                      // "share"
	eventDirty                      // "dirty"
)
