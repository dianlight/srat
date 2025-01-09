package config

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"slices"

	"github.com/jinzhu/copier"
	"github.com/thoas/go-funk"
)

type Share struct {
	Name        string `json:"name,omitempty"`
	Path        string `json:"path"`
	FS          string `json:"fs"`
	Disabled    bool   `json:"disabled,omitempty"`
	Users       []any  `json:"users,omitempty"`
	RoUsers     []any  `json:"ro_users,omitempty"`
	TimeMachine bool   `json:"timemachine,omitempty"`
	Usage       string `json:"usage,omitempty"`
}

type Shares map[string]Share

const CURRENT_CONFIG_VERSION = 3

type Config struct {
	CurrentFile       string
	ConfigSpecVersion int8 `json:"version,omitempty,default=0"`
	Options
	Shares          Shares `json:"shares"`
	DockerInterface string `json:"docker_interface"`
	DockerNet       string `json:"docker_net"`
	//Users           []User           `json:"users"`
	UpdateChannel string `json:"update_channel"`
}

// ReadConfigBuffer reads and parses a configuration file.
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
func (self *Config) ReadFromFile(file string) error {
	configFile, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	// Parse json
	return self.ReadConfigBuffer(configFile)
}

// ReadConfigBuffer parses a JSON-encoded byte slice into a Config struct.
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
func (self *Config) ReadConfigBuffer(buffer []byte) error {
	return json.Unmarshal(buffer, &self)
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
func (in *Config) ConfigToMap() *map[string]interface{} {
	var nconfig map[string]interface{}

	//log.Println(pretty.Sprint("New Config:", in))

	// Parse json
	buffer, err := json.Marshal(&in)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	err_2 := json.Unmarshal(buffer, &nconfig)
	if err_2 != nil {
		log.Fatal(err_2)
		return nil
	}

	//log.Println(pretty.Sprint("New Config2:", nconfig))

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
func (in *Config) MigrateConfig() error {
	if in.ConfigSpecVersion == CURRENT_CONFIG_VERSION {
		return nil
	}

	// From version 0 to version 1 - Default shares ain config
	if in.ConfigSpecVersion == 0 {
		log.Printf("Migrating config from version 0 to version 1")
		in.ConfigSpecVersion = 1
		in.UpdateChannel = "stable"
		if in.Shares == nil {
			in.Shares = make(Shares)
		}
		for _, share := range []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"} {
			_, ok := in.Shares[share]
			if !ok {
				in.Shares[share] = Share{Path: "/" + share, FS: "native", Disabled: false, Usage: "none", Users: []any{in.Username}}
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
			if len(share.Users) == 0 {
				share.Users = append(share.Users, User{
					Username: in.Username,
					Password: in.Password,
				})
			}
			if share.Usage == "" {
				if in.Medialibrary.Enable {
					share.Usage = "media"
				} else {
					share.Usage = "share"
				}
			}
			in.Shares[shareName] = share
		}
	}

	// From version 2 to version 3 - Users in share
	if in.ConfigSpecVersion == 2 {
		log.Printf("Migrating config from version 2 to version 3")
		in.ConfigSpecVersion = 3
		for shareName, share := range in.Shares {
			for ix, user := range share.Users {
				switch user.(type) {
				case string:
					{
						share.Users[ix] = funk.Find(in.OtherUsers, func(u User) bool { return u.Username == user.(string) })
					}
				case User:
					{
						share.Users[ix] = user
					}
				default:
					{
						log.Printf("Unknown user type in share %s: %T", shareName, user)
					}
				}
			}

			//			if share.Users == nil {
			//				share.Users = []any{
			//					in.Username,
			//				}
			//				in.Shares[shareName] = share
			//			}
			if share.Usage == "" && in.Automount {
				share.Usage = "media"
				in.Shares[shareName] = share
			}
		}
	}

	return nil
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
func (self *Config) LoadConfig(file string) error {
	err := self.ReadFromFile(file)
	if err != nil {
		return err
	}
	self.MigrateConfig()
	return nil
}

func (self *Config) FromContext(ctx context.Context) error {
	*self = *ctx.Value("samba_json_config").(*Config)
	return nil
}

func (self *Config) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "samba_json_config", self)
}
