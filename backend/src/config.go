package main

import (
	"encoding/json"
	"log"
	"os"
)

type Share struct {
	Path string `json:"path"`
	FS   string `json:"fs"`
}

type Shares map[string]Share

const CURRENT_CONFIG_VERSION = 1

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

	// From version 0 to version 1
	if in.ConfigSpecVersion == 0 {
		log.Printf("Migrating config from version 0 to version 1")
		in.ConfigSpecVersion = 1
		for _, share := range []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"} {
			_, ok := in.Shares[share]
			if !ok {
				in.Shares[share] = Share{Path: "/" + share + share, FS: "native"}
				log.Printf("Added share: %s", share)
			}
		}
	}

	return in
}
