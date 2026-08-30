package service

// LabFeatureStatus classifies the maturity tier of a lab feature.
type LabFeatureStatus string

const (
	// StatusAlpha marks features available only in development and
	// pre-release builds. They are never exposed in release (production)
	// builds, regardless of experimental_lab_mode.
	StatusAlpha LabFeatureStatus = "alpha"
	// StatusBeta marks features available in any build when
	// experimental_lab_mode is enabled.
	StatusBeta LabFeatureStatus = "beta"
)

type LabFeature struct {
	Key         string
	Name        string
	Description string
	Status      LabFeatureStatus
}

// LabFeatureRegistry is the single source of truth for lab features.
// Add new lab features here; the API layer derives availability from the
// status tier plus the current environment/build.
type LabFeatureRegistry struct {
	features map[string]LabFeature
}

func NewLabFeatureRegistry() *LabFeatureRegistry {
	entries := []LabFeature{
		{Key: "hdidle", Name: "HDIdle per-disk control",
			Description: "Per-disk spin-down configuration, dashboard suggestion badge and HDIdle API routes.",
			Status:      StatusBeta},
		{Key: "smb_conf", Name: "smb.conf view",
			Description: "Read-only view of the generated smb.conf.",
			Status:      StatusBeta},
		{Key: "ha_use_nfs", Name: "Use NFS for Home Assistant",
			Description: "Mount Home Assistant shares with NFS instead of SMB/CIFS.",
			Status:      StatusBeta},
		{Key: "ha_custom_component", Name: "Home Assistant custom component tools",
			Description: "Install, upgrade and uninstall the SRAT custom component.",
			Status:      StatusAlpha},
		{Key: "smb_over_quic", Name: "SMB over QUIC",
			Description: "Expose SMB shares over the QUIC transport.",
			Status:      StatusBeta},
		{Key: "addon_mdns", Name: "Addon-side mDNS registration",
			Description: "Zeroconf mDNS registration of the Samba service directly from the addon.",
			Status:      StatusBeta},
	}
	m := make(map[string]LabFeature, len(entries))
	for _, f := range entries {
		m[f.Key] = f
	}
	return &LabFeatureRegistry{features: m}
}

func (r *LabFeatureRegistry) Get(key string) (LabFeature, bool) {
	f, ok := r.features[key]
	return f, ok
}

func (r *LabFeatureRegistry) All() []LabFeature {
	out := make([]LabFeature, 0, len(r.features))
	for _, f := range r.features {
		out = append(out, f)
	}
	return out
}
