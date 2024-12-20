package main

import (
	"encoding/json"
	"log"
	"os"
	"slices"

	"github.com/jinzhu/copier"
)

type Share struct {
	Path        string   `json:"path"`
	FS          string   `json:"fs"`
	Disabled    bool     `json:"disabled,omitempty"`
	Users       []string `json:"users,omitempty"`
	RoUsers     []string `json:"ro_users,omitempty"`
	TimeMachine bool     `json:"timemachine,omitempty"`
	Usage       string   `json:"usage,omitempty"`
}

type Shares map[string]Share

const CURRENT_CONFIG_VERSION = 3

type Config struct {
	ConfigSpecVersion int8 `json:"version,omitempty,default=0"`
	Options
	Shares Shares `json:"shares"`
	// "_interfaces":["eth0","eth1"],
	DockerInterface string `json:"docker_interface"`
	DockerNet       string `json:"docker_net"`
	// "_moredisks":["mnt/EFI","mnt/LIBRARY","mnt/Updater"],
	// Redefinitions and new config elements
	Users []User `json:"users"`
}

func readConfig(file string) *Config {
	if file == "" {
		return readConfigPipe()
	} else {
		return readConfigFile(file)
	}
}

func readConfigPipe() *Config {
	var config Config
	defer os.Stdin.Close()
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		err := json.NewDecoder(os.Stdin).Decode(&config)
		if err != nil {
			log.Fatal(err)
		}
	}
	return &config
}

func readConfigFile(file string) *Config {
	configFile, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	// Parse json
	return readConfigBuffer(configFile)
}

func readConfigBuffer(buffer []byte) *Config {
	var config Config
	// Parse json
	err := json.Unmarshal(buffer, &config)
	if err != nil {
		log.Fatal(err)
	}

	return &config
}

func configToMap(in *Config) *map[string]interface{} {
	var nconfig map[string]interface{}

	// Parse json
	buffer, err := json.Marshal(in)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	err_2 := json.Unmarshal(buffer, &nconfig)
	if err_2 != nil {
		log.Fatal(err_2)
		return nil
	}

	return &nconfig
}

func migrateConfig(in *Config) *Config {
	if in.ConfigSpecVersion == CURRENT_CONFIG_VERSION {
		return in
	}

	// From version 0 to version 1 - Default shares ain config
	if in.ConfigSpecVersion == 0 {
		log.Printf("Migrating config from version 0 to version 1")
		in.ConfigSpecVersion = 1
		for _, share := range []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"} {
			_, ok := in.Shares[share]
			if !ok {
				in.Shares[share] = Share{Path: "/" + share, FS: "native", Disabled: false, Usage: "native"}
				log.Printf("Added share: %s", share)
			}
		}
	}
	// From version 1 to version 2 - ACL in Share object
	if in.ConfigSpecVersion == 1 {
		log.Printf("Migrating config from version 1 to version 2")
		in.ConfigSpecVersion = 2
		for shareName, share := range in.Shares {
			i := slices.IndexFunc(in.ACL, func(a OptionsAcl) bool { return a.Share == shareName })
			if i > -1 {
				//share.OptionsAcl = OptionsAcl{}
				//log.Printf("ACL found for share %v", in.ACL[i])
				copier.Copy(&share, &in.ACL[i])
				//log.Printf("ACL found for dest %v", share)
				in.ACL = slices.Delete(in.ACL, i, i+1)
				in.Shares[shareName] = share
			}
		}
		//log.Printf("Shares %v", in.Shares)
	}

	// From version 2 to version 3 - Users in share
	if in.ConfigSpecVersion == 2 {
		log.Printf("Migrating config from version 2 to version 3")
		in.ConfigSpecVersion = 3
		for shareName, share := range in.Shares {
			if share.Users == nil {
				share.Users = append(share.Users, in.Username)
				in.Shares[shareName] = share
			}
			if share.Usage == "" && in.Automount {
				share.Usage = "media"
				in.Shares[shareName] = share
			}
		}
		//log.Printf("Shares %v", in.Shares)
	}

	return in
}
