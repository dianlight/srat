package dto

import "github.com/dianlight/srat/dm"

type Settings struct {
	Workgroup         string           `json:"workgroup"`
	Mountoptions      []string         `json:"mountoptions"`
	AllowHost         []string         `json:"allow_hosts"`
	VetoFiles         []string         `json:"veto_files"`
	CompatibilityMode bool             `json:"compatibility_mode"`
	EnableRecycleBin  bool             `json:"recyle_bin_enabled"`
	Interfaces        []string         `json:"interfaces"`
	BindAllInterfaces bool             `json:"bind_all_interfaces"`
	LogLevel          string           `json:"log_level"`
	MultiChannel      bool             `json:"multi_channel"`
	UpdateChannel     dm.UpdateChannel `json:"update_channel"`
}
