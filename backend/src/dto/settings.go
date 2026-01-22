package dto

type Settings struct {
	Hostname          string   `json:"hostname,omitempty"`
	Workgroup         string   `json:"workgroup,omitempty"`
	Mountoptions      []string `json:"mountoptions,omitempty"`
	AllowHost         []string `json:"allow_hosts,omitempty" nullable:"false"`
	CompatibilityMode bool     `json:"compatibility_mode,omitempty"`
	Interfaces        []string `json:"interfaces,omitempty" nullable:"false"`
	BindAllInterfaces bool     `json:"bind_all_interfaces,omitempty"`
	LogLevel          string   `json:"log_level,omitempty"`
	MultiChannel      bool     `json:"multi_channel,omitempty"`
	AllowGuest        *bool    `json:"allow_guest,omitempty" default:"false"`
	//UpdateChannel                 UpdateChannel `json:"update_channel,omitempty" enum:"None,Develop,Release,Prerelease"`
	TelemetryMode                 TelemetryMode `json:"telemetry_mode,omitempty" enum:"Ask,All,Errors,Disabled"`
	LocalMaster                   *bool         `json:"local_master,omitempty" default:"true"`
	ExportStatsToHA               *bool         `json:"export_stats_to_ha,omitempty" default:"true"`
	HAUseNFS                      *bool         `json:"ha_use_nfs,omitempty" default:"false"`
	SMBoverQUIC                   *bool         `json:"smb_over_quic,omitempty" default:"true"`
	HDIdleEnabled                 *bool         `json:"hdidle_enabled,omitempty" default:"false"`
	HDIdleDefaultIdleTime         int           `json:"hdidle_default_idle_time,omitempty"` // seconds
	HDIdleDefaultCommandType      HdidleCommand `json:"hdidle_default_command_type,omitempty" enum:"scsi,ata"`
	HDIdleDefaultPowerCondition   uint8         `json:"hdidle_default_power_condition,omitempty"` // 0-15
	HDIdleIgnoreSpinDownDetection bool          `json:"hdidle_ignore_spin_down_detection,omitempty" default:"false"`
}
