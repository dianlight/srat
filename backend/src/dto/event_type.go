package dto

//go:generate go tool goenums -l event_type.go
type eventType int

const (
	eventHello          eventType = iota // "hello"
	eventUpdating                        // "updating"
	eventVolumes                         // "volumes"
	eventHeartbeat                       // "heartbeat"
	eventShare                           // "share"
	eventHDIdleConfig                    // "hdidle_config"
)
