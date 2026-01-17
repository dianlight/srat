package config

import (
	"encoding/json"
	"log"
	"os"
)

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

type Options struct {
	Workgroup string `json:"workgroup"`
	Username  string `json:"username"`
	//Password         string   `json:"password"`
	Automount        bool     `json:"automount"`
	Moredisks        []string `json:"moredisks"`
	Mountoptions     []string `json:"mountoptions"`
	AvailableDiskLog bool     `json:"available_disks_log"`
	Medialibrary     struct {
		Enable bool `json:"enable"`
		//SSHKEY string `json:"ssh_private_key"`
	} `json:"medialibrary"`
	AllowHost         []string `json:"allow_hosts"`
	VetoFiles         []string `json:"veto_files"`
	CompatibilityMode bool     `json:"compatibility_mode"`
	EnableRecycleBin  bool     `json:"recyle_bin_enabled"`
	//WSDD              bool     `json:"wsdd"`
	//WSDD2             bool     `json:"wsdd2"`
	//HDDIdle           int      `json:"hdd_idle_seconds"`
	Smart bool `json:"enable_smart"`
	/* MQTTNextGen       bool     `json:"mqtt_nexgen_entities"`
	MQTTEnable        bool     `json:"mqtt_enable"`
	MQTTHost          string   `json:"mqtt_host"`
	MQTTUsername      string   `json:"mqtt_username"`
	MQTTPassword      string   `json:"mqtt_password"`
	MQTTPort          string   `json:"mqtt_port"`
	MQTTTopic   */string `json:"mqtt_topic"`
	Autodiscovery        struct {
		DisableDiscovery  bool `json:"disable_discovery"`
		DisablePersistent bool `json:"disable_persistent"`
		DisableAutoremove bool `json:"disable_autoremove"`
	} `json:"autodiscovery"`
	OtherUsers        []User       `json:"other_users,omitempty"`
	ACL               []OptionsAcl `json:"acl,omitempty"`
	Interfaces        []string     `json:"interfaces"`
	BindAllInterfaces bool         `json:"bind_all_interfaces"`
	LogLevel          string       `json:"log_level"`
	MOF               int          `json:"meaning_of_life"`
	MultiChannel      bool         `json:"multi_channel"`
}

func ReadOptionsFile(file string) *Options {
	optionsFile, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	// Parse json
	return readOptionsBuffer(optionsFile)
}

func readOptionsBuffer(buffer []byte) *Options {
	var options Options

	// Parse json
	err := json.Unmarshal(buffer, &options)
	if err != nil {
		log.Fatal(err)
	}

	return &options
}
