package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"slices"

	"github.com/dianlight/srat/dm"
	"github.com/jinzhu/copier"
)

type Share struct {
	Name        string   `json:"name,omitempty"`
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
	CurrentFile       string
	ConfigSpecVersion int8 `json:"version,omitempty,default=0"`
	Options
	Shares          Shares           `json:"shares"`
	DockerInterface string           `json:"docker_interface"`
	DockerNet       string           `json:"docker_net"`
	Users           []User           `json:"users"`
	UpdateChannel   dm.UpdateChannel `json:"update_channel"`
}

// readConfigFile reads and parses a configuration file.
//
// It takes the path to a configuration file, reads its contents, and then
// passes the data to readConfigBuffer for parsing into a Config struct.
//
// Parameters:
//   - file: A string representing the path to the configuration file to be read.
//
// Returns:
//   - *Config: A pointer to the parsed Config struct.
//     If reading the file fails, the function will log a fatal error and terminate the program.
func readConfigFile(file string) (*Config, error) {
	configFile, err := os.ReadFile(file)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	// Parse json
	config, err := readConfigBuffer(configFile)
	if err != nil {
		return nil, err
	}
	config.CurrentFile = file
	return config, nil
}

// readConfigBuffer parses a JSON-encoded byte slice into a Config struct.
//
// This function takes a byte slice containing JSON data and attempts to unmarshal it
// into a Config struct. If the unmarshaling process fails, the function will log
// a fatal error and terminate the program.
//
// Parameters:
//   - buffer: A byte slice containing the JSON-encoded configuration data to be parsed.
//
// Returns:
//   - *Config: A pointer to the parsed Config struct.
//     If parsing fails, the function will log a fatal error and terminate the program.
func readConfigBuffer(buffer []byte) (*Config, error) {
	var config Config
	// Parse json
	err := json.Unmarshal(buffer, &config)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &config, nil
}

// ConfigToMap converts a Config struct to a map[string]interface{}.
// This function is useful for converting a strongly-typed Config object
// into a more flexible map representation.
//
// Parameters:
//   - in: A pointer to the Config struct to be converted.
//
// Returns:
//   - *map[string]interface{}: A pointer to the resulting map.
//     If the conversion process fails at any step, the function returns nil.
func ConfigToMap(in *Config) *map[string]interface{} {
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

// MigrateConfig upgrades the configuration to the latest version.
// It performs a series of migrations based on the current ConfigSpecVersion,
// updating the configuration structure and data as needed.
//
// Parameters:
//   - in: A pointer to the Config struct that needs to be migrated.
//
// Returns:
//   - *Config: A pointer to the migrated Config struct. If the input config
//     is already at the latest version, it returns the input unchanged.
func MigrateConfig(in *Config) *Config {
	if in.ConfigSpecVersion == CURRENT_CONFIG_VERSION {
		return in
	}

	// From version 0 to version 1 - Default shares ain config
	if in.ConfigSpecVersion == 0 {
		log.Printf("Migrating config from version 0 to version 1")
		in.ConfigSpecVersion = 1
		in.UpdateChannel = dm.Stable
		if in.Shares == nil {
			in.Shares = make(Shares)
		}
		for _, share := range []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"} {
			_, ok := in.Shares[share]
			if !ok {
				in.Shares[share] = Share{Path: "/" + share, FS: "native", Disabled: false, Usage: "native"}
			}
		}
	}
	// From version 1 to version 2 - ACL in Share object
	if in.ConfigSpecVersion == 1 {
		log.Printf("Migrating config from version 1 to version 2")
		in.ConfigSpecVersion = 2
		for shareName, share := range in.Shares {
			share.Name = shareName
			i := slices.IndexFunc(in.ACL, func(a OptionsAcl) bool { return a.Share == shareName })
			if i > -1 {
				copier.Copy(&share, &in.ACL[i])
				in.ACL = slices.Delete(in.ACL, i, i+1)
			}
			in.Shares[shareName] = share
		}
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
	}

	return in
}

// LoadConfig reads a configuration file, parses it, and migrates it to the latest version.
//
// This function takes a file path, reads the configuration from that file,
// and then applies any necessary migrations to ensure the configuration
// is up-to-date with the current version.
//
// Parameters:
//   - file: A string representing the path to the configuration file to be loaded.
//
// Returns:
//   - *Config: A pointer to the loaded and migrated Config struct.
//   - error: An error if the file couldn't be read or parsed. If successful, this will be nil.
func LoadConfig(file string) (*Config, error) {
	config, err := readConfigFile(file)
	if err != nil {
		return nil, err
	}
	config = MigrateConfig(config)
	return config, nil
}

func SaveConfig(in *Config) (*Config, error) {
	// TODO: Implement save config
	return in, nil
}

// RollbackConfig reverts the configuration to the last saved state.
//
// This function attempts to reload the configuration from the file specified
// in the CurrentFile field of the input Config. If CurrentFile is empty,
// it returns an error indicating that the current file was not found.
//
// Parameters:
//   - in: A pointer to the Config struct containing the current configuration state.
//
// Returns:
//   - *Config: A pointer to the reloaded Config struct if successful.
//   - error: An error if the rollback fails, either due to a missing CurrentFile
//     or issues with reloading the configuration.
func RollbackConfig(in *Config) (*Config, error) {
	if in.CurrentFile == "" {
		return nil, errors.New("current file not found")
	}
	return LoadConfig(in.CurrentFile)
}
