package service

import (
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyStandardShareNamesPolicy verifies that the standard share name
// policy rewrites legacy share paths to the new directories and exposes only
// the names selected by the mode (issue #898).
func TestApplyStandardShareNamesPolicy(t *testing.T) {
	// newShares returns a fresh map for each subtest so that delete
	// operations in one subtest cannot leak into the next.
	newShares := func() config.Shares {
		return config.Shares{
			"addons":        {Name: "addons", Path: "/addons"},
			"addon_configs": {Name: "addon_configs", Path: "/addon_configs"},
			"local_apps":    {Name: "local_apps", Path: "/local_apps"},
			"app_configs":   {Name: "app_configs", Path: "/app_configs"},
			"other":         {Name: "other", Path: "/x"},
		}
	}

	t.Run("both mode keeps all names and rewrites legacy paths", func(t *testing.T) {
		tconfig := &config.Config{Shares: newShares()}

		applyStandardShareNamesPolicy(tconfig, dto.StandardShareNamesModeBoth)

		require.Len(t, tconfig.Shares, 5)
		assert.Equal(t, "/local_apps", tconfig.Shares["addons"].Path)
		assert.Equal(t, "/app_configs", tconfig.Shares["addon_configs"].Path)
		assert.Equal(t, "/local_apps", tconfig.Shares["local_apps"].Path)
		assert.Equal(t, "/app_configs", tconfig.Shares["app_configs"].Path)
		assert.Equal(t, "/x", tconfig.Shares["other"].Path)
	})

	t.Run("empty mode defaults to both", func(t *testing.T) {
		tconfig := &config.Config{Shares: newShares()}

		applyStandardShareNamesPolicy(tconfig, "")

		require.Len(t, tconfig.Shares, 5)
		assert.Equal(t, "/local_apps", tconfig.Shares["addons"].Path)
		assert.Equal(t, "/app_configs", tconfig.Shares["addon_configs"].Path)
	})

	t.Run("old mode exposes only legacy names with new paths", func(t *testing.T) {
		tconfig := &config.Config{Shares: newShares()}

		applyStandardShareNamesPolicy(tconfig, dto.StandardShareNamesModeOld)

		require.Len(t, tconfig.Shares, 3)
		assert.Equal(t, "/local_apps", tconfig.Shares["addons"].Path)
		assert.Equal(t, "/app_configs", tconfig.Shares["addon_configs"].Path)
		assert.Equal(t, "/x", tconfig.Shares["other"].Path)
		assert.NotContains(t, tconfig.Shares, "local_apps")
		assert.NotContains(t, tconfig.Shares, "app_configs")
	})

	t.Run("new mode exposes only new names", func(t *testing.T) {
		tconfig := &config.Config{Shares: newShares()}

		applyStandardShareNamesPolicy(tconfig, dto.StandardShareNamesModeNew)

		require.Len(t, tconfig.Shares, 3)
		assert.Equal(t, "/local_apps", tconfig.Shares["local_apps"].Path)
		assert.Equal(t, "/app_configs", tconfig.Shares["app_configs"].Path)
		assert.Equal(t, "/x", tconfig.Shares["other"].Path)
		assert.NotContains(t, tconfig.Shares, "addons")
		assert.NotContains(t, tconfig.Shares, "addon_configs")
	})

	t.Run("uppercase keys are left untouched", func(t *testing.T) {
		tconfig := &config.Config{Shares: config.Shares{
			"ADDONS": {Name: "ADDONS", Path: "/addons"},
		}}

		applyStandardShareNamesPolicy(tconfig, dto.StandardShareNamesModeOld)

		require.Len(t, tconfig.Shares, 1)
		assert.Equal(t, "/addons", tconfig.Shares["ADDONS"].Path)
	})

	t.Run("nil shares map is safe", func(t *testing.T) {
		tconfig := &config.Config{}

		applyStandardShareNamesPolicy(tconfig, dto.StandardShareNamesModeNew)

		assert.Empty(t, tconfig.Shares)
	})
}
