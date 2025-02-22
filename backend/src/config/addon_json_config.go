package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/jinzhu/copier"
	"github.com/ztrue/tracerr"
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
	ConfigSpecVersion int `json:"version,omitempty,default=0"`
	// Options
	Workgroup        string   `json:"workgroup"`
	Username         string   `json:"username"`
	Password         string   `json:"password"`
	Automount        bool     `json:"automount"`
	Moredisks        []string `json:"moredisks"`
	Mountoptions     []string `json:"mountoptions"`
	AvailableDiskLog bool     `json:"available_disks_log"`
	Medialibrary     struct {
		Enable bool   `json:"enable"`
		SSHKEY string `json:"ssh_private_key"`
	} `json:"medialibrary"`
	AllowHost         []string `json:"allow_hosts"`
	VetoFiles         []string `json:"veto_files"`
	CompatibilityMode bool     `json:"compatibility_mode"`
	EnableRecycleBin  bool     `json:"recyle_bin_enabled"`
	WSDD              bool     `json:"wsdd"`
	WSDD2             bool     `json:"wsdd2"`
	HDDIdle           int      `json:"hdd_idle_seconds"`
	Smart             bool     `json:"enable_smart"`
	MQTTNextGen       bool     `json:"mqtt_nexgen_entities"`
	MQTTEnable        bool     `json:"mqtt_enable"`
	MQTTHost          string   `json:"mqtt_host"`
	MQTTUsername      string   `json:"mqtt_username"`
	MQTTPassword      string   `json:"mqtt_password"`
	MQTTPort          string   `json:"mqtt_port"`
	MQTTTopic         string   `json:"mqtt_topic"`
	Autodiscovery     struct {
		DisableDiscovery  bool `json:"disable_discovery"`
		DisablePersistent bool `json:"disable_persistent"`
		DisableAutoremove bool `json:"disable_autoremove"`
	} `json:"autodiscovery"`
	OtherUsers        []User       `json:"other_users,omitempty"`
	ACL               []OptionsAcl `json:"acl,omitempty"`
	Interfaces        []string     `json:"interfaces"`
	BindAllInterfaces bool         `json:"bind_all_interfaces"`
	LogLevel          string       `json:"log_level"`
	MOF               string       `json:"meaning_of_life"`
	MultiChannel      bool         `json:"multi_channel"`
	// End Options
	Shares          Shares `json:"shares"`
	DockerInterface string `json:"docker_interface"`
	DockerNet       string `json:"docker_net"`
	UpdateChannel   string `json:"update_channel"`
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
		return tracerr.Wrap(err)
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
				in.Shares[share] = Share{Name: share, Path: "/" + share, FS: "native", Disabled: false, Usage: "internal", Users: []string{in.Username}}
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
				if len(share.Users) == 0 {
					share.Users = append(share.Users, in.Username)
				}
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
			/*
				for ix, user := range share.Users {
					switch user.(type) {
					case string:
						{
							ux := funk.Find(in.OtherUsers, func(u User) bool { return u.Username == user.(string) })
							if ux != nil {
								share.Users[ix] = ux
							} else if user.(string) == in.Username {
								share.Users[ix] = User{Username: in.Username, Password: in.Password}
							} else {
								share.Users[ix] = User{Username: user.(string), Password: "<invalid password>"}
							}
						}
					case User:
					default:
						{
							log.Printf("Unknown user type in share %s: %T", shareName, user)
						}
					}
				}
				for ix, user := range share.RoUsers {
					switch user.(type) {
					case string:
						{
							ux := funk.Find(in.OtherUsers, func(u User) bool { return u.Username == user.(string) })
							if ux != nil {
								share.RoUsers[ix] = ux
							} else if user.(string) == in.Username {
								share.RoUsers[ix] = User{Username: in.Username, Password: in.Password}
							} else {
								share.RoUsers[ix] = User{Username: user.(string), Password: "<invalid password>"}
							}
						}
					case User:
					default:
						{
							log.Printf("Unknown rouser type in share %s: %T", shareName, user)
						}
					}
				}
			*/
			if share.Usage == "" && in.Automount {
				share.Usage = "media"
				in.Shares[shareName] = share
			}
		}
	}

	if in.ConfigSpecVersion != CURRENT_CONFIG_VERSION {
		return fmt.Errorf("unsupported config version: %d (expected %d)", in.ConfigSpecVersion, CURRENT_CONFIG_VERSION)
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
		return tracerr.Wrap(err)
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

// Mapping Functions
/*
func (self Config) To(ctx context.Context, dst any) (bool, error) {
	switch dst.(type) {
	case *[]dto.User:
		*dst.(*[]dto.User) = append((*dst.(*[]dto.User)), dto.User{
			Username: pointer.String(self.Username),
			Password: pointer.String(self.Password),
			IsAdmin:  pointer.Bool(true),
		})
		for _, user := range self.OtherUsers {
			*dst.(*[]dto.User) = append((*dst.(*[]dto.User)), dto.User{
				Username: pointer.String(user.Username),
				Password: pointer.String(user.Password),
			})
		}
		return true, nil
	case *[]dto.SharedResource:
		for _, share := range self.Shares {
			var sr dto.SharedResource
			err := mapper.Map(context.Background(), &sr, share)
			if err != nil {
				return false, err
			}
			*dst.(*[]dto.SharedResource) = append((*dst.(*[]dto.SharedResource)), sr)
		}
		return true, nil
	default:
		return false, nil
	}
}
*/

/*
func (m Shares) To(dst any) (bool, error) {
	switch v := dst.(type) {
	case *[]dto.SharedResource:
		var shr dto.SharedResource
		err := mapper.Map(&shr, m)
		if err != nil {
			return false, err
		}
		*v = append(*v, shr)
		return true, nil
	default:
		return false, nil
	}
}
*/

/*
func (m *Shares) From(src any) (bool, error) {
	switch v := src.(type) {
	case [string]:
		m.Username = v
		return true, nil
	default:
		return false, nil
	}
}
*/
