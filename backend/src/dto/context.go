package dto

import "time"

type ContextState struct {
	AddonIpAddress  string        // IP address of the addon interface
	ReadOnlyMode    bool          // Whether the application is running in read-only mode
	ProtectedMode   bool          // Whether the application is running in an Addon started in protected mode
	SecureMode      bool          // Whether the application is running in secure mode and need authentication for all operations
	HACoreReady     bool          // Whether the Home Assistant Core is ready
	UpdateDataDir   string        // Directory where update files are stored
	UpdateFilePath  string        // Full path to the update file for current update operation. Useful for onplace_update.
	UpdateChannel   UpdateChannel // Current Update Channel
	UpdateAvailable bool          // Whether an update is available
	AutoUpdate      bool          // Whether automatic updates are enabled
	SambaConfigFile string        // Path to the Samba configuration file
	Template        []byte        // Template data for generating configuration files
	DockerInterface string        // Name of the Docker network interface
	DockerNet       string        // Docker network subnet
	Heartbeat       int           // Heartbeat interval in seconds
	SupervisorURL   string        // URL of the Supervisor
	SupervisorToken string        // Authentication token for the Supervisor
	DatabasePath    string        // Path to the database file
	StartTime       time.Time     // Time when the application started
}
