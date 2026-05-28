package dto

// MdnsRegisterNotification is sent over WebSocket to the Home Assistant custom component
// to instruct it to register or deregister the Samba server via Home Assistant's zeroconf (mDNS) integration.
type MdnsRegisterNotification struct {
	// Hostname is the Samba server hostname to advertise.
	Hostname string `json:"hostname"`
	// Port is the SMB port to advertise (typically 445).
	Port int `json:"port"`
	// Enabled indicates whether the custom component should register (true) or deregister (false).
	Enabled bool `json:"enabled"`
}
