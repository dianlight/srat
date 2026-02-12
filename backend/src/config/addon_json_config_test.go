package config_test

import (
	"os"
	"testing"

	"github.com/dianlight/srat/config"
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
	var config1 config.Config
	e1 := config1.ReadFromFile(tempFile.Name())
	require.NoError(t, e1)
	var config2 config.Config
	e2 := config2.ReadFromFile(tempFile.Name())
	require.NoError(t, e2)
	var config3 config.Config
	e3 := config3.ReadFromFile(tempFile.Name())
	require.NoError(t, e3)

	// Compare the results
	assert.Equal(t, config1, config2)
	assert.Equal(t, config2, config3)
}

func TestConfigToMapWithUnicode(t *testing.T) {
	// Create a Config struct with unicode characters
	config := &config.Config{
		ConfigSpecVersion: 1,
		Shares: config.Shares{
			"unicode": config.Share{
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
	sharePath, ok := (*result)["shares"].(map[string]any)["unicode"].(map[string]any)["path"].(string)
	assert.True(t, ok)
	assert.Equal(t, "/path/to/unicode/文件夹", sharePath)
}

/*
func TestMigrateConfigFromVersion0ToCurrent(t *testing.T) {
	// Create a config with version 0
	initialConfig := &config.Config{
		ConfigSpecVersion: 0,
		Shares:            make(config.Shares),
	}

	// Call migrateConfig
	err := initialConfig.MigrateConfig()
	require.NoError(t, err)

	// Check if the version has been updated
	assert.Equal(t, config.CURRENT_CONFIG_VERSION, initialConfig.ConfigSpecVersion)

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
	assert.Len(t, initialConfig.Shares, len(expectedShares))
}
*/
/*
func TestMigrateConfigSetsVersionToCurrent(t *testing.T) {
	// Create a config with version 0
	initialConfig := &config.Config{
		ConfigSpecVersion: 0,
		Shares:            make(config.Shares),
	}

	// Call migrateConfig
	err := initialConfig.MigrateConfig()
	require.NoError(t, err)

	assert.Equal(t, config.CURRENT_CONFIG_VERSION, initialConfig.ConfigSpecVersion)
}
func TestMigrateConfigWithAllDefaultShares(t *testing.T) {
	// Create a config with version 0 and all default shares already present
	initialConfig := &config.Config{
		ConfigSpecVersion: 0,
		Shares: config.Shares{
			"config":        config.Share{Path: "/config", FS: "native"},
			"addons":        config.Share{Path: "/addons", FS: "native"},
			"ssl":           config.Share{Path: "/ssl", FS: "native"},
			"share":         config.Share{Path: "/share", FS: "native"},
			"backup":        config.Share{Path: "/backup", FS: "native"},
			"media":         config.Share{Path: "/media", FS: "native"},
			"addon_configs": config.Share{Path: "/addon_configs", FS: "native"},
		},
		OtherUsers: []config.User{
			{Username: "utente1"},
			{Username: "utente2"},
		},
		ACL: []config.OptionsAcl{
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
	}

	// Call migrateConfig
	err := initialConfig.MigrateConfig()
	require.NoError(t, err)

	// Check if the version has been updated
	assert.Equal(t, config.CURRENT_CONFIG_VERSION, initialConfig.ConfigSpecVersion)
	// Check if all shares are still present and unchanged
	expectedShares := []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"}
	for _, shareName := range expectedShares {
		share, exists := initialConfig.Shares[shareName]
		assert.True(t, exists)
		assert.Equal(t, share.Path, "/"+shareName)
		assert.Equal(t, "native", share.FS)
	}

	assert.Len(t, initialConfig.Shares, len(expectedShares))

	//assert.Len(t, initialConfig.Options.ACL, 2)

	assert.Equal(t, "utente1", (initialConfig.Shares["backup"].Users)[0])
	assert.Equal(t, "utente2", (initialConfig.Shares["ssl"].Users)[0])
	assert.True(t, initialConfig.Shares["ssl"].Disabled)
}
func TestLoadConfigWithNonExistentFile(t *testing.T) {
	config := &config.Config{}
	err := config.LoadConfig("non_existent_file.json")
	require.Error(t, err)
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

	config := &config.Config{}
	err = config.LoadConfig(tempFile.Name())
	require.Error(t, err)
	// Depending on environment (permissions enforcement), we may get either:
	// - "permission denied" if the file truly cannot be read
	// - "unexpected end of JSON input" if the empty file can be read (e.g., running as root)
	errMsg := err.Error()
	hasPermissionError := strings.Contains(errMsg, "permission denied")
	hasJSONError := strings.Contains(errMsg, "unexpected end of JSON input")
	assert.True(t, hasPermissionError || hasJSONError,
		"Expected error to contain either 'permission denied' or 'unexpected end of JSON input', got: %s", errMsg)
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
	tconfig := &config.Config{}

	// Load the config
	err = tconfig.LoadConfig(tempFile.Name())
	require.NoError(t, err)

	// Check if the config was loaded correctly
	assert.Equal(t, config.CURRENT_CONFIG_VERSION, tconfig.ConfigSpecVersion)
	assert.Contains(t, tconfig.Shares, "test")
	assert.Equal(t, "/test", tconfig.Shares["test"].Path)
	assert.Equal(t, "ext4", tconfig.Shares["test"].FS)

	// Check if additional shares were added during migration
	expectedShares := []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"}
	for _, shareName := range expectedShares {
		assert.Contains(t, tconfig.Shares, shareName)
	}
}
*/
func TestBuildVersion(t *testing.T) {
	// Save original values
	origVersion := config.Version
	origCommit := config.CommitHash
	origTimestamp := config.BuildTimestamp

	// Restore original values after test
	defer func() {
		config.Version = origVersion
		config.CommitHash = origCommit
		config.BuildTimestamp = origTimestamp
	}()

	// Test with default values
	result := config.BuildVersion()
	assert.Contains(t, result, config.Version)
	assert.Contains(t, result, config.CommitHash)
	assert.Contains(t, result, config.BuildTimestamp)

	// Test with custom values
	config.Version = "1.2.3"
	config.CommitHash = "abc123"
	config.BuildTimestamp = "2025-01-01T00:00:00Z"

	result = config.BuildVersion()
	expected := "1.2.3 (abc123 2025-01-01T00:00:00Z)"
	assert.Equal(t, expected, result)
}
