package homeassistant_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/dianlight/srat/homeassistant/core"
	"github.com/dianlight/srat/homeassistant/core_api"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"github.com/stretchr/testify/suite"
)

// SupervisorCITestSuite is a test suite for testing Supervisor API interactions.
type SupervisorCITestSuite struct {
	suite.Suite
	coreClient     *core.ClientWithResponses
	coreAPIClient  *core_api.ClientWithResponses
	hardwareClient *hardware.ClientWithResponses
	mountClient    *mount.ClientWithResponses
	ctx            context.Context
}

// TestSupervisorCITestSuite is the test entrypoint
func TestSupervisorCITestSuite(t *testing.T) {
	// Skip test if Supervisor URL or Token are not set
	if os.Getenv("SUPERVISOR_URL") == "" || os.Getenv("SUPERVISOR_TOKEN") == "" {
		t.Skip("Skipping Supervisor integration tests because SUPERVISOR_URL or SUPERVISOR_TOKEN is not set")
	}
	suite.Run(t, new(SupervisorCITestSuite))
}

// SetupTest initializes the test suite.
func (suite *SupervisorCITestSuite) SetupTest() {
	suite.ctx = context.Background()

	supervisorURL := os.Getenv("SUPERVISOR_URL")
	supervisorToken := os.Getenv("SUPERVISOR_TOKEN")

	bearerAuth, err := securityprovider.NewSecurityProviderBearerToken(supervisorToken)
	if err != nil {
		log.Fatal(err)
	}

	suite.T().Logf("Supervisor URL: %s", supervisorURL)

	// Create clients
	suite.coreClient, err = core.NewClientWithResponses(supervisorURL, core.WithRequestEditorFn(bearerAuth.Intercept))
	if err != nil {
		log.Fatal(err)
	}

	suite.coreAPIClient, err = core_api.NewClientWithResponses(supervisorURL, core_api.WithRequestEditorFn(bearerAuth.Intercept))
	if err != nil {
		log.Fatal(err)
	}

	suite.hardwareClient, err = hardware.NewClientWithResponses(supervisorURL, hardware.WithRequestEditorFn(bearerAuth.Intercept))
	if err != nil {
		log.Fatal(err)
	}

	suite.mountClient, err = mount.NewClientWithResponses(supervisorURL, mount.WithRequestEditorFn(bearerAuth.Intercept))
	if err != nil {
		log.Fatal(err)
	}
}

// TestCoreInfo verifies that we can retrieve core info.
func (suite *SupervisorCITestSuite) TestCoreInfo() {
	resp, err := suite.coreClient.GetCoreInfoWithResponse(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp.Body)
	suite.T().Log(string(resp.Body[:]))
	suite.Require().Equal(200, resp.StatusCode(), "Expected status code 200 but got %d body %s", resp.StatusCode(), resp.Status())
	suite.Require().NotNil(resp.JSON200, "Response body is %v", resp.Body)
	suite.Require().NotNil(resp.JSON200.Data.Version, "Response object is %v", resp.JSON200)
}

// TestCoreApiGetApi verify that core api are reached
func (suite *SupervisorCITestSuite) TestCoreApiGetApi() {
	resp, err := suite.coreAPIClient.GetApiWithResponse(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(200, resp.StatusCode(), "Expected status code 200 but got %d body %s", resp.StatusCode(), resp.Status())
	suite.Require().NotNil(resp.JSON200)
}

// TestCoreApiGetEntityState verifies that we can retrieve entity state.
func (suite *SupervisorCITestSuite) TestCoreApiGetEntityState() {
	entityId := "sun.sun" // Example entity ID, adjust as needed
	resp, err := suite.coreAPIClient.GetEntityStateWithResponse(suite.ctx, entityId)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp.Body)
	suite.T().Log(string(resp.Body[:]))
	suite.Require().Equal(200, resp.StatusCode(), "Expected status code 200 but got %d body %s", resp.StatusCode(), resp.Status())
	suite.Require().NotNil(resp.JSON200)
	suite.Require().NotNil(resp.JSON200.EntityId)
	suite.Equal(entityId, *resp.JSON200.EntityId)
}

// TestGetHardware verifies that we can get hardware info
func (suite *SupervisorCITestSuite) TestGetHardware() {
	resp, err := suite.hardwareClient.GetHardwareInfoWithResponse(suite.ctx)
	suite.Require().NoError(err)
	suite.T().Log(string(resp.Body[:]))
	suite.Require().NotNil(resp.Body)
	suite.Require().Equal(200, resp.StatusCode(), "Expected status code 200 but got %d body %s", resp.StatusCode(), resp.Status())
	suite.T().Logf("%#v", *resp.JSON200)
	suite.Require().NotNil(resp.JSON200)
	suite.Require().NotNil(resp.JSON200.Data.Devices)
	suite.Require().NotNil(resp.JSON200.Data.Drives)
	//	suite.Fail("xx")

	// suite.Require().Greater(len(*resp.JSON200.Data), 1)
	// suite.Require().NotNil((*resp.JSON200.Data)[0].Name)
}

// TestGetMounts verifies that we can get mounts info
func (suite *SupervisorCITestSuite) TestGetMounts() {
	resp, err := suite.mountClient.GetMountsWithResponse(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp.Body)
	suite.T().Log(string(resp.Body[:]))
	suite.Require().Equal(200, resp.StatusCode(), "Expected status code 200 but got %d body %s", resp.StatusCode(), resp.Status())
	suite.Require().NotNil(resp.JSON200)
	suite.NotEmpty(*resp.JSON200)
}
