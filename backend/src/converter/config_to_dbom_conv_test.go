package converter

import (
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertiesToConfig_AllFieldsAreMapped(t *testing.T) {
	conv := ConfigToDbomConverterImpl{}

	medialibrary := config.Config{}.Medialibrary
	medialibrary.Enable = true
	medialibrary.SSHKEY = "ssh-key"

	autodiscovery := config.Config{}.Autodiscovery
	autodiscovery.DisableDiscovery = true
	autodiscovery.DisableAutoremove = true

	expectedShare := config.Share{
		Name:        "media",
		Path:        "/media",
		FS:          "nfs",
		Disabled:    true,
		Users:       []string{"alice"},
		RoUsers:     []string{"bob"},
		TimeMachine: true,
		Usage:       "custom",
		VetoFiles:   []string{".DS_Store"},
	}

	properties := dbom.Properties{
		"CurrentFile":       {Key: "CurrentFile", Value: "/tmp/config.json"},
		"ConfigSpecVersion": {Key: "ConfigSpecVersion", Value: 5},
		"Hostname":          {Key: "Hostname", Value: "ha"},
		"Workgroup":         {Key: "Workgroup", Value: "WORKGROUP"},
		"LocalMaster":       {Key: "LocalMaster", Value: true},
		"Username":          {Key: "Username", Value: "admin"},
		"Automount":         {Key: "Automount", Value: true},
		"Moredisks":         {Key: "Moredisks", Value: []any{"sda", "sdb"}},
		"Mountoptions":      {Key: "Mountoptions", Value: []string{"noatime", "ro"}},
		"AvailableDiskLog":  {Key: "AvailableDiskLog", Value: true},
		"Medialibrary":      {Key: "Medialibrary", Value: medialibrary},
		"AllowHost":         {Key: "AllowHost", Value: []any{"host1", "host2"}},
		"VetoFiles":         {Key: "VetoFiles", Value: []string{"Thumbs.db"}},
		"CompatibilityMode": {Key: "CompatibilityMode", Value: true},
		"HDDIdle":           {Key: "HDDIdle", Value: 300},
		"Smart":             {Key: "Smart", Value: true},
		"Autodiscovery":     {Key: "Autodiscovery", Value: autodiscovery},
		"OtherUsers":        {Key: "OtherUsers", Value: []config.User{{Username: "carol"}}},
		"ACL":               {Key: "ACL", Value: []config.OptionsAcl{{Share: "media", Users: []string{"admin"}}}},
		"Interfaces":        {Key: "Interfaces", Value: []any{"eth0", "eth1"}},
		"BindAllInterfaces": {Key: "BindAllInterfaces", Value: true},
		"LogLevel":          {Key: "LogLevel", Value: "debug"},
		"MOF":               {Key: "MOF", Value: 42},
		"MultiChannel":      {Key: "MultiChannel", Value: true},
		"AllowGuest":        {Key: "AllowGuest", Value: true},
		"Shares":            {Key: "Shares", Value: config.Shares{"media": expectedShare}},
		"DockerInterface":   {Key: "DockerInterface", Value: "eth0"},
		"DockerNet":         {Key: "DockerNet", Value: "bridge"},
		"UpdateChannel":     {Key: "UpdateChannel", Value: "beta"},
		"TelemetryMode":     {Key: "TelemetryMode", Value: "opt-in"},
	}

	var cfg config.Config

	err := conv.PropertiesToConfig(properties, &cfg)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/config.json", cfg.CurrentFile)
	assert.Equal(t, 5, cfg.ConfigSpecVersion)
	assert.Equal(t, "ha", cfg.Hostname)
	assert.Equal(t, "WORKGROUP", cfg.Workgroup)
	assert.True(t, cfg.LocalMaster)
	assert.Equal(t, "admin", cfg.Username)
	assert.True(t, cfg.Automount)
	assert.ElementsMatch(t, []string{"sda", "sdb"}, cfg.Moredisks)
	assert.Equal(t, []string{"noatime", "ro"}, cfg.Mountoptions)
	assert.True(t, cfg.AvailableDiskLog)
	assert.Equal(t, medialibrary, cfg.Medialibrary)
	assert.ElementsMatch(t, []string{"host1", "host2"}, cfg.AllowHost)
	assert.Equal(t, []string{"Thumbs.db"}, cfg.VetoFiles)
	assert.True(t, cfg.CompatibilityMode)
	assert.Equal(t, 300, cfg.HDDIdle)
	assert.True(t, cfg.Smart)
	assert.Equal(t, autodiscovery, cfg.Autodiscovery)
	assert.Equal(t, []config.User{{Username: "carol"}}, cfg.OtherUsers)
	assert.Equal(t, []config.OptionsAcl{{Share: "media", Users: []string{"admin"}}}, cfg.ACL)
	assert.ElementsMatch(t, []string{"eth0", "eth1"}, cfg.Interfaces)
	assert.True(t, cfg.BindAllInterfaces)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 42, cfg.MOF)
	assert.True(t, cfg.MultiChannel)
	assert.True(t, cfg.AllowGuest)
	if assert.Contains(t, cfg.Shares, "media") {
		assert.Equal(t, expectedShare, cfg.Shares["media"])
	}
	assert.Equal(t, "eth0", cfg.DockerInterface)
	assert.Equal(t, "bridge", cfg.DockerNet)
	assert.Equal(t, "beta", cfg.UpdateChannel)
	assert.Equal(t, "opt-in", cfg.TelemetryMode)
}

func TestPropertiesToConfig_StringBoolsAndNilValues(t *testing.T) {
	conv := ConfigToDbomConverterImpl{}

	cfg := config.Config{
		Hostname:    "initial",
		Moredisks:   []string{"keep"},
		LocalMaster: false,
		AllowGuest:  true,
	}

	properties := dbom.Properties{
		"LocalMaster": {Key: "LocalMaster", Value: "enabled"},
		"AllowGuest":  {Key: "AllowGuest", Value: "0"},
		"Hostname":    {Key: "Hostname", Value: nil},
		"Moredisks":   {Key: "Moredisks", Value: nil},
	}

	err := conv.PropertiesToConfig(properties, &cfg)
	require.NoError(t, err)

	assert.True(t, cfg.LocalMaster)
	assert.False(t, cfg.AllowGuest)
	assert.Empty(t, cfg.Hostname)
	assert.Nil(t, cfg.Moredisks)
}
