package dto

// SystemCapabilities represents the system capabilities available.
type SystemCapabilities struct {
	SupportsQUIC              bool   `json:"supports_quic" doc:"Whether SMB over QUIC is supported"`
	HasKernelModule           bool   `json:"has_kernel_module" doc:"Whether QUIC kernel module is loaded"`
	SambaVersion              string `json:"samba_version" doc:"Installed Samba version"`
	SambaVersionSufficient    bool   `json:"samba_version_sufficient" doc:"Whether Samba version >= 4.23.0"`
	UnsupportedReason         string `json:"unsupported_reason,omitempty" doc:"Reason why QUIC is not supported"`
	SupportNFS                bool   `json:"support_nfs" doc:"Whether NFS is supported"`
	LibSmartAvailable         bool   `json:"lib_smart_available" doc:"Whether the lib SMART backend (libsmartmon_go.so) is available at runtime"`
	LibSmartUnavailableReason string `json:"lib_smart_unavailable_reason,omitempty" doc:"Reason why the lib SMART backend is unavailable (empty when lib_smart_available is true)"`
	// SmartBackend is the active SMART backend, computed at startup and
	// read-only: "direct" when the lib backend is loaded, otherwise "legacy"
	// (smartctl exec subprocesses).
	SmartBackend string `json:"smart_backend" doc:"Active SMART backend: direct (lib) or legacy (smartctl exec)"`
}
