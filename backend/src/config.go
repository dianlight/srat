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

type Config struct {
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
