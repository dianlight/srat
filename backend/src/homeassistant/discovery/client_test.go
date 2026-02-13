package discovery_test

import (
	"testing"

	"github.com/dianlight/srat/homeassistant/discovery"
	"github.com/stretchr/testify/assert"
)

func TestDiscoveryClientCompiles(t *testing.T) {
	// This test just verifies the generated client compiles and can be instantiated
	client, err := discovery.NewClientWithResponses("http://supervisor")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestDiscoveryRequestStructsCompile(t *testing.T) {
	// Verify request/response structures compile
	req := discovery.CreateDiscoveryServiceJSONRequestBody{
		Service: "test",
		Config: map[string]any{
			"host": "test-host",
			"port": 8099,
		},
	}
	assert.Equal(t, "test", req.Service)
	assert.NotNil(t, req.Config)

	// Verify response structures exist
	var resp discovery.CreateDiscoveryServiceResponse
	_ = resp

	var listResp discovery.ListDiscoveryServicesResponse
	_ = listResp

	var getResp discovery.GetDiscoveryServiceResponse
	_ = getResp

	var delResp discovery.DeleteDiscoveryServiceResponse
	_ = delResp
}

func TestDiscoveryServiceStruct(t *testing.T) {
	// Verify DiscoveryService struct compiles and has expected fields
	var svc discovery.DiscoveryService
	assert.NotNil(t, &svc)

	// Fields should be pointers as per OpenAPI spec
	addon := "test-addon"
	svc.Addon = &addon
	assert.Equal(t, "test-addon", *svc.Addon)
}
