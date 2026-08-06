package dto

import "github.com/angusgmorrison/logfusc"

type Settings struct {
	Hostname            string                 `json:"hostname,omitempty"`
	Workgroup           string                 `json:"workgroup,omitempty"`
	AllowHost           []string               `json:"allow_hosts,omitempty" nullable:"false" default:"[\"10.0.0.0/8\",\"100.0.0.0/8\",\"172.16.0.0/12\",\"192.168.0.0/16\",\"169.254.0.0/16\",\"fe80::/10\",\"fc00::/7\"]"`
	CompatibilityMode   bool                   `json:"compatibility_mode,omitempty" default:"false"`
	Interfaces          []string               `json:"interfaces,omitempty" nullable:"false" default:"[]"`
	BindAllInterfaces   bool                   `json:"bind_all_interfaces,omitempty" default:"true"`
	MultiChannel        bool                   `json:"multi_channel,omitempty" default:"true"`
	AllowGuest          *bool                  `json:"allow_guest,omitempty" default:"false"`
	TelemetryMode       TelemetryMode          `json:"telemetry_mode" enum:"Ask,All,Errors,Disabled"`
	LocalMaster         *bool                  `json:"local_master,omitempty" default:"true"`
	ExportStatsToHA     *bool                  `json:"export_stats_to_ha,omitempty" default:"false"`
	HAUseNFS            *bool                  `json:"ha_use_nfs,omitempty" default:"false"`
	HASmbPassword       logfusc.Secret[string] `json:"-"`
	SMBoverQUIC         *bool                  `json:"smb_over_quic,omitempty" default:"false"`
	SmartMode           SmartMode              `json:"smart_mode" enum:"none,legacy,direct"`
	ExperimentalLabMode bool                   `json:"experimental_lab_mode"`
	// MDNSRegistration is the master switch that enables or disables mDNS
	// registration of the Samba service (Samba mDNS Announce).
	MDNSRegistration *bool `json:"mdns_registration,omitempty" default:"false"`
	// UseComponentMDNSProxy selects the implementation: true uses the Home
	// Assistant custom component proxy, false uses SRAT's direct zeroconf
	// registration. Defaults to true (component) for backward compatibility.
	UseComponentMDNSProxy *bool `json:"use_component_mdns_proxy,omitempty" default:"true"`
	// StandardShareNames controls which standard share names Samba exposes:
	// "old" (addons, addon_configs), "new" (local_apps, app_configs), or
	// "both". Defaults to both for backward compatibility.
	StandardShareNames StandardShareNamesMode `json:"standard_share_names" enum:"old,new,both" default:"both"`
}
