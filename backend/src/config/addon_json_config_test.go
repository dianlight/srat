package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/mapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfigConsistency(t *testing.T) {
	// Create a temporary file with some sample config data
	tempFile, err := os.CreateTemp("", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	sampleConfig := `{"version": 1, "shares": {"test": {"path": "/test", "fs": "ext4"}}}`
	if _, err := tempFile.Write([]byte(sampleConfig)); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tempFile.Close()

	// Call readConfig multiple times
	var config1 Config
	e1 := config1.ReadFromFile(tempFile.Name())
	require.NoError(t, e1)
	var config2 Config
	e2 := config2.ReadFromFile(tempFile.Name())
	require.NoError(t, e2)
	var config3 Config
	e3 := config3.ReadFromFile(tempFile.Name())
	require.NoError(t, e3)

	// Compare the results
	assert.Equal(t, config1, config2)
	assert.Equal(t, config2, config3)
}

func TestConfigToMapWithUnicode(t *testing.T) {
	// Create a Config struct with unicode characters
	config := &Config{
		ConfigSpecVersion: 1,
		Shares: Shares{
			"unicode": Share{
				Path: "/path/to/unicode/文件夹",
				FS:   "ext4",
			},
		},
		DockerInterface: "eth0",
		DockerNet:       "172.17.0.0/16",
	}

	// Call configToMap
	result := config.ConfigToMap()
	assert.NotNil(t, result)

	// Check if the unicode characters are preserved
	sharePath, ok := (*result)["shares"].(map[string]interface{})["unicode"].(map[string]interface{})["path"].(string)
	assert.True(t, ok)
	assert.Equal(t, "/path/to/unicode/文件夹", sharePath)
}
func TestMigrateConfigFromVersion0ToCurrent(t *testing.T) {
	// Create a config with version 0
	initialConfig := &Config{
		ConfigSpecVersion: 0,
		Shares:            make(Shares),
	}

	// Call migrateConfig
	err := initialConfig.MigrateConfig()
	require.NoError(t, err)

	// Check if the version has been updated
	assert.Equal(t, CURRENT_CONFIG_VERSION, initialConfig.ConfigSpecVersion)

	// Check if all required shares have been added
	expectedShares := []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"}
	for _, shareName := range expectedShares {
		share, exists := initialConfig.Shares[shareName]
		assert.True(t, exists)
		expectedPath := "/" + shareName
		assert.Equal(t, "native", share.FS)
		assert.Equal(t, expectedPath, share.Path)
	}

	// Check that no extra shares were added
	assert.Equal(t, len(initialConfig.Shares), len(expectedShares))
}

func TestMigrateConfigSetsVersionToCurrent(t *testing.T) {
	// Create a config with version 0
	initialConfig := &Config{
		ConfigSpecVersion: 0,
		Shares:            make(Shares),
	}

	// Call migrateConfig
	err := initialConfig.MigrateConfig()
	require.NoError(t, err)

	assert.Equal(t, CURRENT_CONFIG_VERSION, initialConfig.ConfigSpecVersion)
}
func TestMigrateConfigWithAllDefaultShares(t *testing.T) {
	// Create a config with version 0 and all default shares already present
	initialConfig := &Config{
		ConfigSpecVersion: 0,
		Shares: Shares{
			"config":        Share{Path: "/config", FS: "native"},
			"addons":        Share{Path: "/addons", FS: "native"},
			"ssl":           Share{Path: "/ssl", FS: "native"},
			"share":         Share{Path: "/share", FS: "native"},
			"backup":        Share{Path: "/backup", FS: "native"},
			"media":         Share{Path: "/media", FS: "native"},
			"addon_configs": Share{Path: "/addon_configs", FS: "native"},
		},
		Options: Options{
			OtherUsers: []User{
				{Username: "utente1", Password: "Test Password"},
				{Username: "utente2", Password: "Test Password"},
			},
			ACL: []OptionsAcl{
				{
					Share:    "config",
					Disabled: false,
					Users:    []string{"utente1"},
				},
				{
					Share:    "backup",
					Disabled: false,
					Users:    []string{"utente1"},
				},
				{
					Share:    "ssl",
					Disabled: true,
					Users:    []string{"utente2"},
				},
			},
		},
	}

	// Call migrateConfig
	err := initialConfig.MigrateConfig()
	require.NoError(t, err)

	// Check if the version has been updated
	assert.Equal(t, CURRENT_CONFIG_VERSION, initialConfig.ConfigSpecVersion)
	// Check if all shares are still present and unchanged
	expectedShares := []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"}
	for _, shareName := range expectedShares {
		share, exists := initialConfig.Shares[shareName]
		assert.True(t, exists)
		assert.Equal(t, share.Path, "/"+shareName)
		assert.Equal(t, "native", share.FS)
	}

	assert.Equal(t, len(initialConfig.Shares), len(expectedShares))

	//assert.Len(t, initialConfig.Options.ACL, 2)

	assert.Equal(t, User{Username: "utente1", Password: "Test Password"}, (initialConfig.Shares["backup"].Users)[0])
	assert.Equal(t, User{Username: "utente2", Password: "Test Password"}, (initialConfig.Shares["ssl"].Users)[0])
	assert.True(t, initialConfig.Shares["ssl"].Disabled)
}
func TestLoadConfigWithNonExistentFile(t *testing.T) {
	config := &Config{}
	err := config.LoadConfig("non_existent_file.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}
func TestLoadConfigWithNonReadableFile(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Change permissions to make the file non-readable
	err = os.Chmod(tempFile.Name(), 0000)
	require.NoError(t, err)

	config := &Config{}
	err = config.LoadConfig(tempFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}
func TestLoadConfigWithValidFile(t *testing.T) {
	// Create a temporary file with a valid config
	tempFile, err := os.CreateTemp("", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	validConfig := `{"version": 0, "shares": {"test": {"path": "/test", "fs": "ext4"}}}`
	_, err = tempFile.Write([]byte(validConfig))
	require.NoError(t, err)
	tempFile.Close()

	// Create a new Config instance
	config := &Config{}

	// Load the config
	err = config.LoadConfig(tempFile.Name())
	require.NoError(t, err)

	// Check if the config was loaded correctly
	assert.Equal(t, CURRENT_CONFIG_VERSION, config.ConfigSpecVersion)
	assert.Contains(t, config.Shares, "test")
	assert.Equal(t, "/test", config.Shares["test"].Path)
	assert.Equal(t, "ext4", config.Shares["test"].FS)

	// Check if additional shares were added during migration
	expectedShares := []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"}
	for _, shareName := range expectedShares {
		assert.Contains(t, config.Shares, shareName)
	}
}
func TestMappableToDtoSettings(t *testing.T) {
	// Create a new Config instance
	config := &Config{}

	// Load the config
	err := config.LoadConfig("../../test/data/config.json")
	require.NoError(t, err)

	dto := dto.Settings{}
	err = mapper.Map(&dto, config)
	require.NoError(t, err)

	//bdata, err := os.ReadFile("../../test/data/config.json")
	//require.NoError(t, err)
	//data := make(map[string]interface{})
	//require.NoError(t,json.Unmarshal(bdata,data))
	assert.Equal(t, dto.Workgroup, "WORKGROUP")
	assert.Equal(t, dto.AllowHost, []string{"10.0.0.0/8",
		"100.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"fe80::/10",
		"fc00::/7",
	})
	assert.Equal(t, dto.Mountoptions, []string{"nosuid", "relatime", "noexec"})
	assert.Equal(t, dto.VetoFiles, []string{
		"._*",
		".DS_Store",
		"Thumbs.db",
		"icon?",
		".Trashes"})

	assert.Equal(t, dto.CompatibilityMode, false)
	assert.Equal(t, dto.EnableRecycleBin, false)
	assert.Equal(t, dto.BindAllInterfaces, true)
	assert.Equal(t, dto.MultiChannel, false)
}

func TestMappableToDtoUsers(t *testing.T) {
	// Create a new Config instance
	config := &Config{}

	// Load the config
	err := config.LoadConfig("../../test/data/config.json")
	require.NoError(t, err)

	_dto := []dto.User{}
	err = mapper.Map(&_dto, config)
	require.NoError(t, err)

	//bdata, err := os.ReadFile("../../test/data/config.json")
	//require.NoError(t, err)
	//data := make(map[string]interface{})
	//require.NoError(t,json.Unmarshal(bdata,data))

	assert.Len(t, _dto, 4)
	assert.Contains(t, _dto, dto.User{Username: "backupuser", Password: "\u003cbackupuser secret password\u003e", IsAdmin: false})
	assert.Contains(t, fmt.Sprintf("%v", _dto), "utente2")
	assert.Contains(t, fmt.Sprintf("%v", _dto), "rouser")
	assert.Contains(t, _dto, dto.User{Username: "dianlight", Password: "hassio2010", IsAdmin: true})
}

func TestMappableToDtoSharedResources(t *testing.T) {
	// Create a new Config instance
	config := &Config{}

	// Load the config
	err := config.LoadConfig("../../test/data/config.json")
	require.NoError(t, err)

	_dto := make([]dto.SharedResource, 0, 20)
	err = mapper.Map(&_dto, config)
	require.NoError(t, err)

	//bdata, err := os.ReadFile("../../test/data/config.json")
	//require.NoError(t, err)
	//data := make(map[string]interface{})
	//require.NoError(t,json.Unmarshal(bdata,data))

	assert.Len(t, _dto, 10, "Size of shared resources is not as expected %#v", _dto)
	assert.Contains(t, _dto, dto.SharedResource{ID: (*uint)(nil), Name: "addons", Path: "/addons", FS: "native", Disabled: false, Users: []dto.User{{Username: "", Password: "", IsAdmin: false}}, RoUsers: []dto.User(nil), TimeMachine: false, Usage: "none", DirtyStatus: false, DeviceId: (*uint64)(nil), Invalid: false})
	// assert.Contains(t, fmt.Sprintf("%v", _dto), "utente2")
	// assert.Contains(t, fmt.Sprintf("%v", _dto), "rouser")
	// assert.Contains(t, _dto, dto.User{Username: "dianlight", Password: "hassio2010", IsAdmin: true})
}
