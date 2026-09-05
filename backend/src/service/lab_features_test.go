package service_test

import (
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/require"
	tozderrors "gitlab.com/tozd/go/errors"
)

func TestNewLabFeatureRegistryContainsAllTiers(t *testing.T) {
	r := service.NewLabFeatureRegistry()

	beta, ok := r.Get("hdidle")
	require.True(t, ok)
	require.Equal(t, service.StatusBeta, beta.Status)
	require.NotEmpty(t, beta.Name)
	require.NotEmpty(t, beta.Description)

	alpha, ok := r.Get("ha_custom_component")
	require.True(t, ok)
	require.Equal(t, service.StatusAlpha, alpha.Status)
}

func TestLabFeatureRegistryGetUnknownKey(t *testing.T) {
	r := service.NewLabFeatureRegistry()

	_, ok := r.Get("does_not_exist")
	require.False(t, ok)
}

func TestLabFeatureRegistryAllReturnsEveryFeature(t *testing.T) {
	r := service.NewLabFeatureRegistry()

	all := r.All()
	require.Len(t, all, 6)

	keys := make(map[string]service.LabFeatureStatus, len(all))
	for _, f := range all {
		keys[f.Key] = f.Status
	}
	require.Equal(t, service.StatusBeta, keys["hdidle"])
	require.Equal(t, service.StatusBeta, keys["smb_conf"])
	require.Equal(t, service.StatusBeta, keys["ha_use_nfs"])
	require.Equal(t, service.StatusAlpha, keys["ha_custom_component"])
	require.Equal(t, service.StatusBeta, keys["smb_over_quic"])
	require.Equal(t, service.StatusBeta, keys["addon_mdns"])
}

type stubLabSettingLoader struct {
	settings *dto.Settings
	err      tozderrors.E
}

func (s *stubLabSettingLoader) Load() (*dto.Settings, tozderrors.E) {
	return s.settings, s.err
}

func (s *stubLabSettingLoader) UpdateSettings(*dto.Settings) tozderrors.E {
	return nil
}

func (s *stubLabSettingLoader) SetCommandExists(func(cmd []string) bool) {}

func (s *stubLabSettingLoader) DumpTable() (string, tozderrors.E) { return "", nil }

func TestIsHaCustomComponentProblemKey(t *testing.T) {
	require.True(t, service.IsHaCustomComponentProblemKey("custom_component_missing"))
	require.True(t, service.IsHaCustomComponentProblemKey("custom_component_restart_required"))
	require.True(t, service.IsHaCustomComponentProblemKey("  custom_component_x  "))
	require.False(t, service.IsHaCustomComponentProblemKey("disk_error"))
	require.False(t, service.IsHaCustomComponentProblemKey(""))
}

func TestIsHaCustomComponentLabEnabled(t *testing.T) {
	oldVersion := config.Version
	t.Cleanup(func() { config.Version = oldVersion })

	// Production always disabled, even with lab on.
	config.Version = "1.0.0"
	require.False(t, service.IsHaCustomComponentLabEnabled(&stubLabSettingLoader{settings: &dto.Settings{ExperimentalLabMode: true}}))

	// Non-production follows lab mode.
	config.Version = "0.0.0-dev.0"
	require.True(t, service.IsHaCustomComponentLabEnabled(&stubLabSettingLoader{settings: &dto.Settings{ExperimentalLabMode: true}}))
	require.False(t, service.IsHaCustomComponentLabEnabled(&stubLabSettingLoader{settings: &dto.Settings{ExperimentalLabMode: false}}))

	// Fail-closed on nil service, load error and nil settings.
	require.False(t, service.IsHaCustomComponentLabEnabled(nil))
	require.False(t, service.IsHaCustomComponentLabEnabled(&stubLabSettingLoader{settings: nil, err: tozderrors.New("boom")}))
	require.False(t, service.IsHaCustomComponentLabEnabled(&stubLabSettingLoader{settings: nil}))
}
