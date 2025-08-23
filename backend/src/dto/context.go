package dto

type ContextState struct {
	AddonIpAddress  string
	ReadOnlyMode    bool
	ProtectedMode   bool
	SecureMode      bool
	HACoreReady     bool
	UpdateFilePath  string
	UpdateChannel   UpdateChannel
	SambaConfigFile string
	Template        []byte
	DockerInterface string
	DockerNet       string
	Heartbeat       int
	SupervisorURL   string
	SupervisorToken string
	DatabasePath    string
}
