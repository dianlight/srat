package config

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfigConsistency(t *testing.T) {
	// Create a temporary file with some sample config data
	tempFile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	sampleConfig := `{"version": 1, "shares": {"test": {"path": "/test", "fs": "ext4"}}}`
	if _, err := tempFile.Write([]byte(sampleConfig)); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tempFile.Close()

	// Call readConfig multiple times
	config1, e1 := readConfigFile(tempFile.Name())
	if e1 != nil {
		t.Fatalf("Failed to read config: %v", e1)
	}
	config2, e2 := readConfigFile(tempFile.Name())
	if e2 != nil {
		t.Fatalf("Failed to read config: %v", e2)
	}
	config3, e3 := readConfigFile(tempFile.Name())
	if e3 != nil {
		t.Fatalf("Failed to read config: %v", e3)
	}

	// Compare the results
	if !reflect.DeepEqual(config1, config2) || !reflect.DeepEqual(config2, config3) {
		t.Errorf("readConfig returned inconsistent results for the same input file")
	}
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

	// Check if the result is not nil
	if result == nil {
		t.Fatalf("configToMap returned nil for valid input")
	}

	// Check if the unicode characters are preserved
	sharePath, ok := (*result)["shares"].(map[string]interface{})["unicode"].(map[string]interface{})["path"].(string)
	if !ok || sharePath != "/path/to/unicode/文件夹" {
		t.Errorf("Unicode path not preserved, got: %v", sharePath)
	}

}
func TestMigrateConfigFromVersion0To1(t *testing.T) {
	// Create a config with version 0
	initialConfig := &Config{
		ConfigSpecVersion: 0,
		Shares:            make(Shares),
	}

	// Call migrateConfig
	migratedConfig := MigrateConfig(initialConfig)

	// Check if the version has been updated
	if migratedConfig.ConfigSpecVersion != CURRENT_CONFIG_VERSION {
		t.Errorf("Expected ConfigSpecVersion to be %d, got %d", CURRENT_CONFIG_VERSION, migratedConfig.ConfigSpecVersion)
	}

	// Check if all required shares have been added
	expectedShares := []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"}
	for _, shareName := range expectedShares {
		share, exists := migratedConfig.Shares[shareName]
		if !exists {
			t.Errorf("Expected share '%s' to be added, but it wasn't", shareName)
		} else {
			expectedPath := "/" + shareName
			if share.Path != expectedPath {
				t.Errorf("Expected share '%s' to have path '%s', got '%s'", shareName, expectedPath, share.Path)
			}
			if share.FS != "native" {
				t.Errorf("Expected share '%s' to have FS 'native', got '%s'", shareName, share.FS)
			}
		}
	}

	// Check that no extra shares were added
	if len(migratedConfig.Shares) != len(expectedShares) {
		t.Errorf("Expected %d shares, got %d", len(expectedShares), len(migratedConfig.Shares))
	}
}
func TestMigrateConfigCurrentVersion(t *testing.T) {
	// Create a config with the current version
	initialConfig := &Config{
		ConfigSpecVersion: CURRENT_CONFIG_VERSION,
		Shares:            make(Shares),
	}

	// Call migrateConfig
	migratedConfig := MigrateConfig(initialConfig)

	// Check if the config is unchanged
	if !reflect.DeepEqual(initialConfig, migratedConfig) {
		t.Errorf("migrateConfig modified a config that was already at the current version")
	}

	// Verify that no shares were added
	if len(migratedConfig.Shares) != 0 {
		t.Errorf("Expected 0 shares, got %d", len(migratedConfig.Shares))
	}
}
func TestMigrateConfigSetsVersionToCurrent(t *testing.T) {
	// Create a config with version 0
	initialConfig := &Config{
		ConfigSpecVersion: 0,
		Shares:            make(Shares),
	}

	// Call migrateConfig
	migratedConfig := MigrateConfig(initialConfig)

	// Check if the version has been updated to 1
	if migratedConfig.ConfigSpecVersion != CURRENT_CONFIG_VERSION {
		t.Errorf("Expected ConfigSpecVersion to be %d after migration, got %d", CURRENT_CONFIG_VERSION, migratedConfig.ConfigSpecVersion)
	}
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
	migratedConfig := MigrateConfig(initialConfig)

	// Check if the version has been updated
	if migratedConfig.ConfigSpecVersion != CURRENT_CONFIG_VERSION {
		t.Errorf("Expected ConfigSpecVersion to be %d, got %d", CURRENT_CONFIG_VERSION, migratedConfig.ConfigSpecVersion)
	}

	// Check if all shares are still present and unchanged
	expectedShares := []string{"config", "addons", "ssl", "share", "backup", "media", "addon_configs"}
	for _, shareName := range expectedShares {
		share, exists := migratedConfig.Shares[shareName]
		if !exists {
			t.Errorf("Expected share '%s' to be present, but it wasn't", shareName)
		} else {
			expectedPath := "/" + shareName
			if share.Path != expectedPath {
				t.Errorf("Expected share '%s' to have path '%s', got '%s'", shareName, expectedPath, share.Path)
			}
			if share.FS != "native" {
				t.Errorf("Expected share '%s' to have FS 'native', got '%s'", shareName, share.FS)
			}
		}
	}

	// Check that no extra shares were added
	if len(migratedConfig.Shares) != len(expectedShares) {
		t.Errorf("Expected %d shares, got %d", len(expectedShares), len(migratedConfig.Shares))
	}

	// Check if old acl are empty
	if len(migratedConfig.ACL) != 0 {
		t.Error("Expected no ACLs, got some")
	}

	// Check if acl are as share attributes
	//t.Log(pretty.Sprint(migratedConfig.Shares))
	assert.Equal(t, "utente1", (migratedConfig.Shares["backup"].Users)[0])
	assert.Equal(t, "utente2", (migratedConfig.Shares["ssl"].Users)[0])
	assert.True(t, migratedConfig.Shares["ssl"].Disabled)
}
