package dto

type Settings struct {
	Workgroup         string        `json:"workgroup,omitempty"`
	Mountoptions      []string      `json:"mountoptions,omitempty"`
	AllowHost         []string      `json:"allow_hosts,omitempty"`
	VetoFiles         []string      `json:"veto_files,omitempty"`
	CompatibilityMode bool          `json:"compatibility_mode,omitempty"`
	EnableRecycleBin  bool          `json:"recyle_bin_enabled,omitempty"`
	Interfaces        []string      `json:"interfaces,omitempty"`
	BindAllInterfaces bool          `json:"bind_all_interfaces,omitempty"`
	LogLevel          string        `json:"log_level,omitempty"`
	MultiChannel      bool          `json:"multi_channel,omitempty"`
	UpdateChannel     UpdateChannel `json:"update_channel,omitempty"`
}
