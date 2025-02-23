package dto

type ContextState struct {
	ReadOnlyMode    bool
	UpdateFilePath  string
	SambaConfigFile string
	Template        []byte
	DockerInterface string
	DockerNet       string
	Heartbeat       int
}
