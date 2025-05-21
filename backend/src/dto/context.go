package dto

type ContextState struct {
	AddonIpAddress  string
	ReadOnlyMode    bool
	UpdateFilePath  string
	SambaConfigFile string
	Template        []byte
	DockerInterface string
	DockerNet       string
	Heartbeat       int
	SupervisorURL   string
}
