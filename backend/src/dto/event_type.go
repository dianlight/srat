package dto

//go:generate go run github.com/zarldev/goenums@v0.3.8 event_type.go
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
