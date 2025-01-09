package config

import (
	"os"
	"testing"

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
func TestMigrateConfigFromVersion0To1(t *testing.T) {
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

	assert.Len(t, initialConfig.Options.ACL, 2)

	assert.Equal(t, "utente1", (initialConfig.Shares["backup"].Users)[0])
	assert.Equal(t, "utente2", (initialConfig.Shares["ssl"].Users)[0])
	assert.True(t, initialConfig.Shares["ssl"].Disabled)
}
