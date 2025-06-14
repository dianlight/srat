package dto

type Welcome struct {
	Message         string `json:"message"`
	SupportedEvents string `json:"supported_events" enum:"hello,updating,volumes,heartbeat,share,dirty"`
	UpdateChannel   string `json:"update_channel" enum:"None,Develop,Release,Prerelease"`
}
