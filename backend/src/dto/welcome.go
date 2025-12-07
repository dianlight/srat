package dto

type Welcome struct {
	Message         string         `json:"message"`
	ActiveClients   int32          `json:"active_clients"`
	SupportedEvents []WebEventType `json:"supported_events" enum:"hello,updating,volumes,heartbeat,share"`
	UpdateChannel   string         `json:"update_channel" enum:"None,Develop,Release,Prerelease"`
	MachineId       *string        `json:"machine_id,omitempty"`
	BuildVersion    string         `json:"build_version"`
	SecureMode      bool           `json:"secure_mode"`
	ProtectedMode   bool           `json:"protected_mode"`
	ReadOnly        bool           `json:"read_only"`
	StartTime       int64          `json:"startTime"`
}
