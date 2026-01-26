package dto

import "github.com/angusgmorrison/logfusc"

type Settings struct {
	Hostname  string `json:"hostname,omitempty" default:"homeassistant"`
	Workgroup string `json:"workgroup,omitempty" default:"WORKGROUP"`
	//Mountoptions      []string `json:"mountoptions,omitempty"`
	AllowHost         []string `json:"allow_hosts,omitempty" nullable:"false" default:"[\"10.0.0.0/8\",\"100.0.0.0/8\",\"172.16.0.0/12\",\"192.168.0.0/16\",\"169.254.0.0/16\",\"fe80::/10\",\"fc00::/7\"]"`
	CompatibilityMode bool     `json:"compatibility_mode,omitempty" default:"false"`
	Interfaces        []string `json:"interfaces,omitempty" nullable:"false" default:"[]"`
	BindAllInterfaces bool     `json:"bind_all_interfaces,omitempty" default:"true"`
	//LogLevel                      string                 `json:"log_level,omitempty"`
	MultiChannel                  bool                   `json:"multi_channel,omitempty" default:"true"`
	AllowGuest                    *bool                  `json:"allow_guest,omitempty" default:"false"`
	TelemetryMode                 TelemetryMode          `json:"telemetry_mode,omitempty" enum:"Ask,All,Errors,Disabled"`
	LocalMaster                   *bool                  `json:"local_master,omitempty" default:"true"`
	ExportStatsToHA               *bool                  `json:"export_stats_to_ha,omitempty" default:"false"`
	HAUseNFS                      *bool                  `json:"ha_use_nfs,omitempty" default:"false"`
	HASmbPassword                 logfusc.Secret[string] `json:"-"`
	SMBoverQUIC                   *bool                  `json:"smb_over_quic,omitempty" default:"false"`
	HDIdleEnabled                 *bool                  `json:"hdidle_enabled,omitempty" default:"false"`
	HDIdleDefaultIdleTime         int                    `json:"hdidle_default_idle_time,omitempty"` // seconds
	HDIdleDefaultCommandType      HdidleCommand          `json:"hdidle_default_command_type,omitempty" enum:"scsi,ata"`
	HDIdleDefaultPowerCondition   uint8                  `json:"hdidle_default_power_condition,omitempty"` // 0-15
	HDIdleIgnoreSpinDownDetection bool                   `json:"hdidle_ignore_spin_down_detection,omitempty" default:"false"`
}
