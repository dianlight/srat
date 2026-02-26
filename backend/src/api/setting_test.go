package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/angusgmorrison/logfusc"
	"github.com/danielgtaylor/huma/v2/autopatch"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SettingsHandlerSuite struct {
	suite.Suite
	dirtyService   service.DirtyDataServiceInterface
	settingService service.SettingServiceInterface
	//db           *gorm.DB
	api *api.SettingsHanler
	//config                 config.Config
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
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			/*
				func() *config.DefaultConfig {
					var nconfig config.Config
					buffer, err := templates.Default_Config_content.ReadFile("default_config.json")
					if err != nil {
						log.Fatalf("Cant read default config file %#+v", err)
					}
					err = nconfig.LoadConfigBuffer(buffer) // Assign to existing err
					if err != nil {
						log.Fatalf("Cant load default config from buffer %#+v", err)
					}
					return &config.DefaultConfig{Config: nconfig}
				},
			*/
			dbom.NewDB,
			api.NewSettingsHanler,
			service.NewDirtyDataService,
			service.NewSettingService,
			events.NewEventBus,
			//repository.NewPropertyRepositoryRepository,
			mock.Mock[service.TelemetryServiceInterface],
			//			mock.Mock[service.BroadcasterServiceInterface],
			//			mock.Mock[service.SambaServiceInterface],
			//mock.Mock[service.DirtyDataServiceInterface],
			//mock.Mock[repository.PropertyRepositoryInterface],

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
				sharedResources.DatabasePath = "file::memory:?cache=shared&_pragma=foreign_keys(1)"

				return &sharedResources
			},
			/*
				func() config.Config {
					err := suite.config.LoadConfig("../../test/data/config.json")
					if err != nil {
						suite.T().Errorf("Cant read config file %s", err)
					}
					return suite.config
				},
			*/
		),
		//fx.Populate(&suite.propertyRepository),
		fx.Populate(&suite.dirtyService),
		fx.Populate(&suite.settingService),
		//fx.Populate(&suite.config),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.api),
	)
	suite.app.RequireStart()
	/*
		mock.When(suite.mockPropertyRepository.All()).ThenReturn(dbom.Properties{
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
			"LocalMaster": dbom.Property{
				Key:   "LocalMaster",
				Value: suite.config.LocalMaster,
			},
		}, nil)
	*/
}

// TearDownSuite runs once after all tests in the suite have finished
func (suite *SettingsHandlerSuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup).Wait()
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func TestSettingsHandlerSuite(t *testing.T) {
	suite.Run(t, new(SettingsHandlerSuite))
}

/*
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
*/
func (suite *SettingsHandlerSuite) TestUpdateSettingsHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)
	autopatch.AutoPatch(api)

	glc := dto.Settings{
		Workgroup: "pluto&admin",
	}

	rr := api.Patch("/settings", glc)
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.Settings
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err, "Body %#v", rr.Body.String())

	suite.Equal(glc.Workgroup, res.Workgroup)
	suite.Equal([]string{"10.0.0.0/8", "100.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "fe80::/10", "fc00::/7"}, res.AllowHost)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Settings)

	// Restore original state
	/*
		_, err = suite.mockPropertyRepository.All()
		if err != nil {
			suite.T().Fatalf("Failed to load properties: %v", err)
		}
	*/
}

func (suite *SettingsHandlerSuite) TestUpdateSettingsHandlerWithAllowGuest() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)
	autopatch.AutoPatch(api)

	// Test with AllowGuest enabled
	allowGuestEnabled := true
	glc := dto.Settings{
		Workgroup:  "testworkgroup",
		AllowGuest: &allowGuestEnabled,
	}

	rr := api.Patch("/settings", glc)
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.Settings
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err, "Body %#v", rr.Body.String())

	suite.Equal(glc.Workgroup, res.Workgroup)
	suite.NotNil(res.AllowGuest)
	suite.Equal(allowGuestEnabled, *res.AllowGuest)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Settings)

	// Restore original state
	/*
		_, err = suite.mockPropertyRepository.All()
		if err != nil {
			suite.T().Fatalf("Failed to load properties: %v", err)
		}
	*/
}

func (suite *SettingsHandlerSuite) TestUpdateSettingsHandlerWithSMBoverQUIC() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)
	autopatch.AutoPatch(api)

	// Test with SMBoverQUIC enabled
	smbOverQuicEnabled := true
	glc := dto.Settings{
		Workgroup:   "testworkgroup",
		SMBoverQUIC: &smbOverQuicEnabled,
	}

	rr := api.Patch("/settings", glc)
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.Settings
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err, "Body %#v", rr.Body.String())

	suite.Equal(glc.Workgroup, res.Workgroup)
	suite.NotNil(res.SMBoverQUIC)
	suite.Equal(smbOverQuicEnabled, *res.SMBoverQUIC)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Settings)

	// Restore original state
	/*
		_, err = suite.mockPropertyRepository.All()
		if err != nil {
			suite.T().Fatalf("Failed to load properties: %v", err)
		}
	*/
}

func (suite *SettingsHandlerSuite) TestUpdateSettingsHandler_PreservesHASmbPassword() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)
	autopatch.AutoPatch(api)

	initial := dto.Settings{
		Workgroup:     "initial-workgroup",
		HASmbPassword: logfusc.NewSecret("super-secret"),
	}
	err := suite.settingService.UpdateSettings(&initial)
	suite.Require().NoError(err)

	update := dto.Settings{
		Workgroup: "updated-workgroup",
	}

	rr := api.Patch("/settings", update)
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	loaded, loadErr := suite.settingService.Load()
	suite.Require().NoError(loadErr)
	suite.Equal("updated-workgroup", loaded.Workgroup)
	suite.Equal("super-secret", loaded.HASmbPassword.Expose())
}

func (suite *SettingsHandlerSuite) TestGetSettingsHandler_DoesNotLeakSecrets() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)
	autopatch.AutoPatch(api)

	initial := dto.Settings{
		Workgroup:     "secret-workgroup",
		HASmbPassword: logfusc.NewSecret("top-secret"),
	}
	err := suite.settingService.UpdateSettings(&initial)
	suite.Require().NoError(err)

	resp := api.Get("/settings")
	suite.Require().Equal(http.StatusOK, resp.Code)
	body := resp.Body.String()

	suite.NotContains(body, "HASmbPassword", "Response should not include HASmbPassword field")
	suite.NotContains(body, "top-secret", "Response should not include secret value")
}
