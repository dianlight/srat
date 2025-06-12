package dto

type Settings struct {
	Hostname          string        `json:"hostname,omitempty"`
	Workgroup         string        `json:"workgroup,omitempty"`
	Mountoptions      []string      `json:"mountoptions,omitempty"`
	AllowHost         []string      `json:"allow_hosts,omitempty" nullable:"false"`
	VetoFiles         []string      `json:"veto_files,omitempty"  nullable:"false"`
	CompatibilityMode bool          `json:"compatibility_mode,omitempty"`
	Interfaces        []string      `json:"interfaces,omitempty" nullable:"false"`
	BindAllInterfaces bool          `json:"bind_all_interfaces,omitempty"`
	LogLevel          string        `json:"log_level,omitempty"`
	MultiChannel      bool          `json:"multi_channel,omitempty"`
	UpdateChannel     UpdateChannel `json:"update_channel,omitempty" enum:"stable,prerelease,none"`
	WSDD              WSDDSettings  `json:"wsdd,omitempty" enum:"none,wsdd,wsdd2"`
}

type WSDDSettings string

const (
	NoWSDD WSDDSettings = "none"
	WSDD   WSDDSettings = "wsdd"
	WSDD2  WSDDSettings = "wsdd2"
)
