package dto

type EventType string

const (
	EventHello     EventType = "hello"
	EventUpdate    EventType = "update"
	EventHeartbeat EventType = "heartbeat"
	EventShare     EventType = "share"
	EventVolumes   EventType = "volumes"
	EventDirty     EventType = "dirty"
)

var EventTypes = []string{
	string(EventHello),
	string(EventUpdate),
	string(EventHeartbeat),
	string(EventShare),
	string(EventVolumes),
}
