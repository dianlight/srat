package service_test

import (
	"testing"

	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/require"
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
