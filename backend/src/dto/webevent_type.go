package dto

//go:generate go tool goenums -l webevent_type.go
type webEventType int

const (
	eventHello           webEventType = iota // "hello"
	eventUpdating                            // "updating"
	eventVolumes                             // "volumes"
	eventHeartbeat                           // "heartbeat"
	eventShares                              // "shares"
	eventDirtyTracker                        // "dirty_data_tracker"
	eventSmartTestStatus                     // "smart_test_status"
)

var WebEventMap = map[string]any{
	WebEventTypes.EVENTHELLO.String():           Welcome{},
	WebEventTypes.EVENTUPDATING.String():        UpdateProgress{},
	WebEventTypes.EVENTVOLUMES.String():         []*Disk{},
	WebEventTypes.EVENTHEARTBEAT.String():       HealthPing{},
	WebEventTypes.EVENTSHARES.String():          []SharedResource{},
	WebEventTypes.EVENTDIRTYTRACKER.String():    DataDirtyTracker{},
	WebEventTypes.EVENTSMARTTESTSTATUS.String(): SmartTestStatus{},
}
