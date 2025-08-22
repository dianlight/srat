package homeassistant_test

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
)

// IngressCITestSuite is a test suite for testing Home Assistant Ingress API interactions.
type IngressCITestSuite struct {
	suite.Suite
	ingressClient *ingress.ClientWithResponses
	ctx           context.Context
}

// TestIngressCITestSuite is the test entrypoint
// To run this test, you need to set the following environment variables:
// SUPERVISOR_URL: e.g., http://homeassistant.local/
// SUPERVISOR_TOKEN: Your Home Assistant Supervisor token
func TestIngressCITestSuite(t *testing.T) {
	supervisorURL := os.Getenv("SUPERVISOR_URL")
	supervisorToken := os.Getenv("SUPERVISOR_TOKEN")

	// Use the provided supervisor token if the current one is a placeholder
	if supervisorToken == "root" {
		supervisorToken = "ee13b70e035205be7a977c7c6ae03e4f96524e352f43b17e02538699adf36d0739167059b472e7e2417cbab6c00f2598949f1897bd6be0e5"
		os.Setenv("SUPERVISOR_TOKEN", supervisorToken)
	}

	// Skip test if Supervisor URL or Token are not set or contain placeholder values
	if supervisorURL == "" || supervisorToken == "" ||
		supervisorToken == "<put me here!>" {
		t.Skip("Skipping Ingress integration tests because SUPERVISOR_URL or SUPERVISOR_TOKEN is not properly configured for Home Assistant")
	}

	suite.Run(t, new(IngressCITestSuite))
}

// SetupTest initializes the test suite.
func (suite *IngressCITestSuite) SetupTest() {
	suite.ctx = context.Background()

	supervisorURL := os.Getenv("SUPERVISOR_URL")
	supervisorToken := os.Getenv("SUPERVISOR_TOKEN")

	bearerAuth, err := securityprovider.NewSecurityProviderBearerToken(supervisorToken)
	if err != nil {
		log.Fatalf("Failed to create bearer token security provider: %v", err)
	}

	suite.T().Logf("Supervisor URL: %s", supervisorURL)

	// Create ingress client
	suite.ingressClient, err = ingress.NewClientWithResponses(supervisorURL, ingress.WithRequestEditorFn(bearerAuth.Intercept))
	if err != nil {
		log.Fatalf("Failed to create ingress client: %v", err)
	}
}

// TestGetIngressPanels verifies that we can retrieve the list of ingress panels.
func (suite *IngressCITestSuite) TestGetIngressPanels() {
	resp, err := suite.ingressClient.GetIngressPanelsWithResponse(suite.ctx)
	if err != nil && strings.Contains(err.Error(), "connection refused") {
		suite.T().Skip("Skipping TestGetIngressPanels because ${SUPERVISOR_URL} not working")
	}
	suite.Require().NoError(err, "Error getting ingress panels")
	suite.Require().NotNil(resp.Body, "Get panels response body is nil")
	suite.T().Logf("Get Panels Response: Status=%d, Body=%s", resp.StatusCode(), string(resp.Body))

	suite.Require().Equal(200, resp.StatusCode(), "Expected status code 200 for getting panels but got %d, body: %s", resp.StatusCode(), string(resp.Body))
	suite.Require().NotNil(resp.JSON200, "JSON200 field is nil for get panels response")
	suite.Require().NotNil(resp.JSON200.Data, "Data field in JSON200 is nil for get panels response")
	suite.Require().NotNil(resp.JSON200.Data.Panels, "Panels field is nil in get panels response")
	suite.Require().NotNil(resp.JSON200.Result, "Result field in JSON200 is nil for get panels response")
	suite.Require().Equal("ok", string(*resp.JSON200.Result), "Expected result 'ok' for getting panels")

	// The list of panels can be empty if no addons with ingress are installed,
	// so we can't assert that it's not empty.
	// We can just log the number of panels found.
	suite.T().Logf("Found %d ingress panels.", len(*resp.JSON200.Data.Panels))
}

// TestCreateAndValidateIngressSession verifies that we can create and then validate an ingress session.
func (suite *IngressCITestSuite) TestCreateAndValidateIngressSession() {
	suite.T().Skip("Skipping TestCreateAndValidateIngressSession Addons don't have the right permissions to create ingress sessions. ")
	// 1. Create an Ingress Session
	createReqBody := ingress.CreateIngressSessionJSONRequestBody{
		UserId: pointer.String("2a68fe2cc380467eaf1e71c1c14f2230"),
	}
	createResp, err := suite.ingressClient.CreateIngressSessionWithResponse(suite.ctx, createReqBody)
	suite.Require().NoError(err, "Error creating ingress session")
	suite.Require().NotNil(createResp.Body, "Create session response body is nil")
	suite.T().Logf("Create Session Response: Status=%d, Body=%s", createResp.StatusCode(), string(createResp.Body))

	suite.Require().Equal(200, createResp.StatusCode(), "Expected status code 200 for session creation but got %d, body: %s", createResp.StatusCode(), string(createResp.Body))
	suite.Require().NotNil(createResp.JSON200, "JSON200 field is nil for create session response")
	suite.Require().NotNil(createResp.JSON200.Data, "Data field in JSON200 is nil for create session response")
	suite.Require().NotNil(createResp.JSON200.Data.Session, "Session ID is nil in create session response")
	suite.Require().Equal("ok", string(*createResp.JSON200.Result), "Expected result 'ok' for session creation")

	sessionID := *createResp.JSON200.Data.Session
	suite.Require().NotEmpty(sessionID, "Session ID should not be empty")
	suite.T().Logf("Successfully created ingress session with ID: %s", sessionID)

	// 2. Validate the Ingress Session
	validateReqBody := ingress.ValidateIngressSessionJSONRequestBody{
		Session: &sessionID,
	}
	validateResp, err := suite.ingressClient.ValidateIngressSessionWithResponse(suite.ctx, validateReqBody)
	suite.Require().NoError(err, "Error validating ingress session")
	suite.Require().NotNil(validateResp.HTTPResponse, "Validate session HTTP response is nil")
	suite.T().Logf("Validate Session Response: Status=%d, Body=%s", validateResp.StatusCode(), string(validateResp.Body))

	suite.Require().Equal(200, validateResp.StatusCode(), "Expected status code 200 for session validation but got %d, body: %s", validateResp.StatusCode(), string(validateResp.Body))
	suite.T().Logf("Successfully validated ingress session with ID: %s", sessionID)
}

// TestValidateIngressSession_Invalid verifies that an invalid session fails validation.
func (suite *IngressCITestSuite) TestValidateIngressSession_Invalid() {
	suite.T().Skip("Skipping TestCreateAndValidateIngressSession Addons don't have the right permissions to create ingress sessions. ")
	invalidSessionID := "invalid-session-id-12345" // A clearly invalid session ID

	validateReqBody := ingress.ValidateIngressSessionJSONRequestBody{
		Session: &invalidSessionID,
	}
	validateResp, err := suite.ingressClient.ValidateIngressSessionWithResponse(suite.ctx, validateReqBody)
	suite.Require().NoError(err, "Error should not occur at HTTP level for invalid session")
	suite.Require().NotNil(validateResp.HTTPResponse, "Validate session HTTP response is nil")
	suite.T().Logf("Validate Invalid Session Response: Status=%d, Body=%s", validateResp.StatusCode(), string(validateResp.Body))

	// According to ingress.yaml, a 400 is expected for invalid request/session.
	// The HA middleware might convert this to 401, but the direct API call should return 400.
	suite.Require().Equal(400, validateResp.StatusCode(), "Expected status code 400 for invalid session but got %d, body: %s", validateResp.StatusCode(), string(validateResp.Body))
	suite.T().Logf("Invalid ingress session correctly rejected with status %d", validateResp.StatusCode())
}
