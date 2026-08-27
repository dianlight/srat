package api

import (
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/require"
	tozderrors "gitlab.com/tozd/go/errors"
)

// wbStubSettings is a hand-written SettingServiceInterface fake for white-box
// tests of requireLabFeature/requireLabMode.
type wbStubSettings struct {
	settings *dto.Settings
	err      tozderrors.E
}

func (s *wbStubSettings) Load() (*dto.Settings, tozderrors.E) { return s.settings, s.err }
func (s *wbStubSettings) UpdateSettings(*dto.Settings) tozderrors.E {
	return nil
}
func (s *wbStubSettings) SetCommandExists(func(cmd []string) bool) {}
func (s *wbStubSettings) DumpTable() (string, tozderrors.E)        { return "", nil }

func newWBHDIdleHandler(settings *dto.Settings, err tozderrors.E) *HDIdleHandler {
	return &HDIdleHandler{
		settingService: &wbStubSettings{settings: settings, err: err},
		labRegistry:    service.NewLabFeatureRegistry(),
	}
}

// TestRequireLabFeatureUnknownKey exercises the registry lookup failure
// branch: an unknown feature key must 404 regardless of environment.
func TestRequireLabFeatureUnknownKey(t *testing.T) {
	h := newWBHDIdleHandler(&dto.Settings{ExperimentalLabMode: true}, nil)
	err := h.requireLabFeature("no_such_feature")
	st, ok := errors.AsType[huma.StatusError](err)
	require.True(t, ok)
	require.Equal(t, 404, st.GetStatus())
}

// TestRequireLabFeatureAlphaBlockedInProduction is the core release gate:
// alpha features must 403 in production builds even with lab mode enabled.
func TestRequireLabFeatureAlphaBlockedInProduction(t *testing.T) {
	config.Version = "1.0.0"
	t.Cleanup(func() { config.Version = "0.0.0-dev.0" })

	h := newWBHDIdleHandler(&dto.Settings{ExperimentalLabMode: true}, nil)
	err := h.requireLabFeature("ha_custom_component")
	st, ok := errors.AsType[huma.StatusError](err)
	require.True(t, ok)
	require.Equal(t, 403, st.GetStatus())
}

// TestRequireLabFeatureAlphaAllowedInPrerelease verifies alpha features pass
// the tier gate outside production, then fall through to the lab-mode check.
func TestRequireLabFeatureAlphaAllowedInPrerelease(t *testing.T) {
	config.Version = "1.0.0-rc.1"
	t.Cleanup(func() { config.Version = "0.0.0-dev.0" })

	// Lab mode on -> allowed.
	h := newWBHDIdleHandler(&dto.Settings{ExperimentalLabMode: true}, nil)
	require.NoError(t, h.requireLabFeature("ha_custom_component"))

	// Lab mode off -> 403 from requireLabMode.
	hOff := newWBHDIdleHandler(&dto.Settings{ExperimentalLabMode: false}, nil)
	st, ok := errors.AsType[huma.StatusError](hOff.requireLabFeature("ha_custom_component"))
	require.True(t, ok)
	require.Equal(t, 403, st.GetStatus())
}

// TestRequireLabFeatureBetaFallsThroughToLabMode verifies beta features
// depend solely on experimental_lab_mode.
func TestRequireLabFeatureBetaFallsThroughToLabMode(t *testing.T) {
	h := newWBHDIdleHandler(&dto.Settings{ExperimentalLabMode: true}, nil)
	require.NoError(t, h.requireLabFeature("hdidle"))

	hOff := newWBHDIdleHandler(&dto.Settings{ExperimentalLabMode: false}, nil)
	err := hOff.requireLabFeature("hdidle")
	st, ok := errors.AsType[huma.StatusError](err)
	require.True(t, ok)
	require.Equal(t, 403, st.GetStatus())
}

// TestRequireLabModeSettingsError covers the 500 path when settings Load
// fails.
func TestRequireLabModeSettingsError(t *testing.T) {
	h := newWBHDIdleHandler(nil, tozderrors.New("storage unavailable"))
	err := h.requireLabMode()
	st, ok := errors.AsType[huma.StatusError](err)
	require.True(t, ok)
	require.Equal(t, 500, st.GetStatus())
}

// TestRequireLabModeNilSettings covers the nil-settings guard.
func TestRequireLabModeNilSettings(t *testing.T) {
	h := newWBHDIdleHandler(nil, nil)
	err := h.requireLabMode()
	st, ok := errors.AsType[huma.StatusError](err)
	require.True(t, ok)
	require.Equal(t, 403, st.GetStatus())
}
