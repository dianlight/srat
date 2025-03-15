package dto

type Welcome struct {
	Message         string `json:"message"`
	SupportedEvents string `json:"supported_events" enum:"hello,update,updating,volumes,heartbeat,share,dirty"`
}
