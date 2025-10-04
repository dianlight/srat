package dto

// SystemCapabilities represents the system capabilities available.
type SystemCapabilities struct {
	SupportsQUIC           bool   `json:"supports_quic" doc:"Whether SMB over QUIC is supported"`
	HasKernelModule        bool   `json:"has_kernel_module" doc:"Whether QUIC kernel module is loaded"`
	HasLibngtcp2           bool   `json:"has_libngtcp2" doc:"Whether libngtcp2 library is available"`
	SambaVersion           string `json:"samba_version" doc:"Installed Samba version"`
	SambaVersionSufficient bool   `json:"samba_version_sufficient" doc:"Whether Samba version >= 4.23.0"`
	UnsupportedReason      string `json:"unsupported_reason,omitempty" doc:"Reason why QUIC is not supported"`
}
