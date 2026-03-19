package dto

import "time"

// HomeAssistantComponentConnection stores runtime metadata about the currently
// connected SRAT Home Assistant custom component.
type HomeAssistantComponentConnection struct {
	Component   string    // Reported custom component identity, e.g. "srat"
	Version     string    // Reported integration version from manifest.json
	HAVersion   *string   // Optional Home Assistant core version from the client
	EntryID     *string   // Optional Home Assistant config entry identifier
	ConnectedAt time.Time // Time when the backend accepted the current handshake
}

type ContextState struct {
	AddonIpAddress string // IP address of the addon interface
	ServerPort     int    // Port on which the HTTP server is listening
	ReadOnlyMode   bool   // Whether the application is running in read-only mode
	ProtectedMode  bool   // Whether the application is running in an Addon started in protected mode
	SecureMode     bool   // Whether the application is running in secure mode and need authentication for all operations
	HACoreReady    bool   // Whether the Home Assistant Core is ready
	UpdateDataDir  string // Directory where update files are stored
	//UpdateFilePath  string        // Full path to the update file for current update operation. Useful for onplace_update.
	UpdateChannel   UpdateChannel                     // Current Update Channel
	UpdateAvailable bool                              // Whether an update is available
	AutoUpdate      bool                              // Whether automatic updates are enabled
	SambaConfigFile string                            // Path to the Samba configuration file
	Template        []byte                            // Template data for generating configuration files
	DockerInterface string                            // Name of the Docker network interface
	DockerNet       string                            // Docker network subnet
	Heartbeat       int                               // Heartbeat interval in seconds
	SupervisorURL   string                            // URL of the Supervisor
	SupervisorToken string                            // Authentication token for the Supervisor
	DatabasePath    string                            // Path to the database file
	StartTime       time.Time                         // Time when the application started
	HAWsComponent   *HomeAssistantComponentConnection // Connected Home Assistant custom component metadata for the active /ws session
}
