package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/autopatch"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SettingsHandlerSuite struct {
	suite.Suite
	dirtyService           service.DirtyDataServiceInterface
	mockPropertyRepository repository.PropertyRepositoryInterface
	api                    *api.SettingsHanler
	config                 config.Config
	//
	ctx    context.Context
	cancel context.CancelFunc
	app    *fxtest.App
}

// SetupSuite runs once before the tests in the suite are run
func (suite *SettingsHandlerSuite) SetupTest() {

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			fx.Annotate(
				func() bool { return false },
				fx.ResultTags(`name:"ha_mode"`),
			),
			api.NewSettingsHanler,
			service.NewDirtyDataService,
			//			mock.Mock[service.BroadcasterServiceInterface],
			//			mock.Mock[service.SambaServiceInterface],
			//mock.Mock[service.DirtyDataServiceInterface],
			mock.Mock[repository.PropertyRepositoryInterface],

			func() *dto.ContextState {
				sharedResources := dto.ContextState{}
				sharedResources.ReadOnlyMode = false
				sharedResources.Heartbeat = 1
				sharedResources.DockerInterface = "hassio"
				sharedResources.DockerNet = "172.30.32.0/23"
				var err error
				sharedResources.Template, err = os.ReadFile("../templates/smb.gtpl")
				if err != nil {
					suite.T().Errorf("Cant read template file %s", err)
				}

				return &sharedResources
			},
			func() config.Config {
				err := suite.config.LoadConfig("../../test/data/config.json")
				if err != nil {
					suite.T().Errorf("Cant read config file %s", err)
				}
				return suite.config
			},
		),
		fx.Populate(&suite.mockPropertyRepository),
		fx.Populate(&suite.dirtyService),
		fx.Populate(&suite.config),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.api),
	)
	suite.app.RequireStart()

	mock.When(suite.mockPropertyRepository.All(mock.Any[bool]())).ThenReturn(dbom.Properties{
		"Hostname": dbom.Property{
			Key:   "Hostname",
			Value: suite.config.Hostname,
		},
		"Workgroup": dbom.Property{
			Key:   "Workgroup",
			Value: suite.config.Workgroup,
		},
		"Mountoptions": dbom.Property{
			Key:   "Mountoptions",
			Value: suite.config.Mountoptions,
		},
		"AllowHost": dbom.Property{
			Key:   "AllowHost",
			Value: suite.config.AllowHost,
		},
		"VetoFiles": dbom.Property{
			Key:   "VetoFiles",
			Value: suite.config.VetoFiles,
		},
		"Interfaces": dbom.Property{
			Key:   "Interfaces",
			Value: suite.config.Interfaces,
		},
		"BindAllInterfaces": dbom.Property{
			Key:   "BindAllInterfaces",
			Value: suite.config.BindAllInterfaces,
		},
		"UpdateChannel": dbom.Property{
			Key:   "UpdateChannel",
			Value: suite.config.UpdateChannel,
		},
		"WSDD": dbom.Property{
			Key:   "WSDD",
			Value: "none",
		},
	}, nil)

}

// TearDownSuite runs once after all tests in the suite have finished
func (suite *SettingsHandlerSuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
}

func TestSettingsHandlerSuite(t *testing.T) {
	suite.Run(t, new(SettingsHandlerSuite))
}

func (suite *SettingsHandlerSuite) TestGetSettingsHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)

	resp := api.Get("/settings")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var expected dto.Settings
	var conv converter.ConfigToDtoConverterImpl
	err := conv.ConfigToSettings(suite.config, &expected)
	suite.Require().NoError(err)

	var returned dto.Settings
	jsonError := json.Unmarshal(resp.Body.Bytes(), &returned)
	suite.Require().NoError(jsonError)

	suite.Equal(expected, returned)

	suite.False(suite.dirtyService.GetDirtyDataTracker().Settings)
}

func (suite *SettingsHandlerSuite) TestUpdateSettingsHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)
	autopatch.AutoPatch(api)

	glc := dto.Settings{
		Workgroup:     "pluto&admin",
		UpdateChannel: dto.UpdateChannels.RELEASE,
	}

	rr := api.Patch("/settings", glc)
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.Settings
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err, "Body %#v", rr.Body.String())

	suite.Equal(glc.Workgroup, res.Workgroup)
	suite.Equal([]string{"10.0.0.0/8", "100.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "fe80::/10", "fc00::/7"}, res.AllowHost)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Settings)
	suite.Equal(dto.UpdateChannels.RELEASE, res.UpdateChannel)

	// Restore original state
	_, err = suite.mockPropertyRepository.All(false)
	if err != nil {
		suite.T().Fatalf("Failed to load properties: %v", err)
	}
	//if err := properties.SetValue("Workgroup", "WORKGROUP"); err != nil {
	//	suite.T().Fatalf("Failed to add workgroup property: %v", err)
	//}
}
