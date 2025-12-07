package dto

//go:generate go tool goenums -l webevent_type.go
type webEventType int

const (
	eventHello        webEventType = iota // "hello"
	eventUpdating                         // "updating"
	eventVolumes                          // "volumes"
	eventHeartbeat                        // "heartbeat"
	eventShare                            // "share"
	eventHDIdleConfig                     // "hdidle_config"
)
