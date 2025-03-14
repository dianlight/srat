package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
)

type SettingsHandlerSuite struct {
	suite.Suite
	dirtyService service.DirtyDataServiceInterface
}

func TestSettingsHandlerSuite(t *testing.T) {
	csuite := new(SettingsHandlerSuite)
	csuite.dirtyService = service.NewDirtyDataService(testContext)
	suite.Run(t, csuite)
}
func (suite *SettingsHandlerSuite) TestGetSettingsHandler() {
	settings := api.NewSettingsHanler(&apiContextState, suite.dirtyService)
	_, api := humatest.New(suite.T())
	settings.RegisterSettings(api)

	resp := api.Get("/settings")
	suite.Equal(http.StatusOK, resp.Code)

	var config config.Config
	err := config.FromContext(testContext)
	suite.Require().NoError(err)
	var expected dto.Settings
	var conv converter.ConfigToDtoConverterImpl
	err = conv.ConfigToSettings(config, &expected)
	suite.Require().NoError(err)

	var returned dto.Settings
	jsonError := json.Unmarshal(resp.Body.Bytes(), &returned)
	suite.Require().NoError(jsonError)

	suite.Equal(expected, returned)

	suite.False(suite.dirtyService.GetDirtyDataTracker().Settings)
}

func (suite *SettingsHandlerSuite) TestUpdateSettingsHandler() {
	settings := api.NewSettingsHanler(&apiContextState, suite.dirtyService)
	_, api := humatest.New(suite.T())
	settings.RegisterSettings(api)

	glc := dto.Settings{
		Workgroup: "pluto&admin",
	}

	rr := api.Put("/settings", glc)
	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.Settings
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err, "Body %#v", rr.Body.String())

	suite.Equal(glc.Workgroup, res.Workgroup)
	suite.EqualValues([]string{"10.0.0.0/8", "100.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "fe80::/10", "fc00::/7"}, res.AllowHost)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Settings)

	// Restore original state
	var properties dbom.Properties
	if err := properties.Load(); err != nil {
		suite.T().Fatalf("Failed to load properties: %v", err)
	}
	if err := properties.SetValue("Workgroup", "WORKGROUP"); err != nil {
		suite.T().Fatalf("Failed to add workgroup property: %v", err)
	}
}
