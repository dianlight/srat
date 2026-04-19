package config

// AddonOptionsFilePath is the path to the Home Assistant Supervisor addon options file.
// Declared as a variable (not a constant) so tests may substitute a temporary path.
var AddonOptionsFilePath = "/data/options.json"

type OptionsAcl struct {
	Share       string   `json:"share,omitempty"`
	Disabled    bool     `json:"disabled,omitempty"`
	Users       []string `json:"users,omitempty"`
	RoUsers     []string `json:"ro_users,omitempty"`
	TimeMachine bool     `json:"timemachine,omitempty"`
	Usage       string   `json:"usage,omitempty"`
}

type User struct {
	Username string `json:"username"`
	//Password string `json:"password"`
}
