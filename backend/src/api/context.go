package api

type ContextState struct {
	ReadOnlyMode   bool
	UpdateFilePath string
	//DataDirtyTracker dto.DataDirtyTracker
	SambaConfigFile string
	Template        []byte
	DockerInterface string
	DockerNet       string
	Heartbeat       int
	//SSEBroker        BrokerInterface
}
