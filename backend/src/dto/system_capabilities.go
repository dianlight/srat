package dto

// SystemCapabilities represents the system capabilities available.
type SystemCapabilities struct {
	SupportsQUIC            bool     `json:"supports_quic" doc:"Whether SMB over QUIC is supported"`
	HasKernelModule         bool     `json:"has_kernel_module" doc:"Whether QUIC kernel module is loaded"`
	SambaVersion            string   `json:"samba_version" doc:"Installed Samba version"`
	SambaVersionSufficient  bool     `json:"samba_version_sufficient" doc:"Whether Samba version >= 4.23.0"`
	UnsupportedReason       string   `json:"unsupported_reason,omitempty" doc:"Reason why QUIC is not supported"`
	SupportNFS              bool     `json:"support_nfs" doc:"Whether NFS is supported"`
	LibSmartAvailable       bool     `json:"lib_smart_available" doc:"Whether the lib SMART backend (libsmartmon_go.so) is available at runtime"`
	AvailableMDNSInterfaces []string `json:"available_mdns_interfaces,omitempty" doc:"Network interfaces eligible for addon-side direct mDNS registration"`
}
