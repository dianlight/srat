package dto

type Settings struct {
	Workgroup         string        `json:"workgroup,omitempty"`
	Mountoptions      []string      `json:"mountoptions,omitempty"`
	AllowHost         []string      `json:"allow_hosts,omitempty" nullable:"false"`
	VetoFiles         []string      `json:"veto_files,omitempty"  nullable:"false"`
	CompatibilityMode bool          `json:"compatibility_mode,omitempty"`
	EnableRecycleBin  bool          `json:"recyle_bin_enabled,omitempty"`
	Interfaces        []string      `json:"interfaces,omitempty" nullable:"false"`
	BindAllInterfaces bool          `json:"bind_all_interfaces,omitempty"`
	LogLevel          string        `json:"log_level,omitempty"`
	MultiChannel      bool          `json:"multi_channel,omitempty"`
	UpdateChannel     UpdateChannel `json:"update_channel,omitempty" enum:"stable,prerelease,none"`
}
