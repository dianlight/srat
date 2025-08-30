package dto

type Welcome struct {
	Message         string      `json:"message"`
	ActiveClients   int32       `json:"active_clients"`
	SupportedEvents []EventType `json:"supported_events" enum:"hello,updating,volumes,heartbeat,share"`
	UpdateChannel   string      `json:"update_channel" enum:"None,Develop,Release,Prerelease"`
}
