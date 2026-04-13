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
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SettingsHandlerSuite struct {
	suite.Suite
	dirtyService   service.DirtyDataServiceInterface
	settingService service.SettingServiceInterface
	addonsService  service.AddonsServiceInterface
	haComponentSvc service.HomeAssistantComponentServiceInterface
	issueService   service.IssueServiceInterface
	upgradeService service.UpgradeServiceInterface
	repairService  service.RepairServiceInterface
	haService      service.HomeAssistantServiceInterface
	broadcaster    service.BroadcasterServiceInterface
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
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HomeAssistantComponentServiceInterface],
			mock.Mock[service.IssueServiceInterface],
			mock.Mock[service.UpgradeServiceInterface],
			mock.Mock[service.RepairServiceInterface],
			mock.Mock[service.HomeAssistantServiceInterface],
			mock.Mock[service.BroadcasterServiceInterface],
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
		fx.Populate(&suite.addonsService),
		fx.Populate(&suite.haComponentSvc),
		fx.Populate(&suite.issueService),
		fx.Populate(&suite.upgradeService),
		fx.Populate(&suite.repairService),
		fx.Populate(&suite.haService),
		fx.Populate(&suite.broadcaster),
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

func (suite *SettingsHandlerSuite) TestGetAppConfigHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)

	mock.When(suite.addonsService.GetAppConfig(mock.AnyContext())).
		ThenReturn(&dto.AppConfigData{
			Options:         map[string]any{"log_level": "info", "enabled": true},
			RuntimeConfig:   map[string]any{"rendered": true},
			RequiresRestart: true,
		}, nil)
	mock.When(suite.haComponentSvc.DismissAddonConfigIssue(mock.AnyContext())).ThenReturn(nil)

	rr := api.Get("/settings/app-config")
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.AppConfigData
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err)
	suite.Equal("info", res.Options["log_level"])
	suite.True(res.RequiresRestart)
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(0)).DismissAddonConfigIssue(mock.AnyContext())
}

func (suite *SettingsHandlerSuite) TestGetHomeAssistantCustomComponentStatusHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)

	installedVersion := "2026.04.1"
	connectedVersion := "2026.04.2"
	status := &dto.HomeAssistantCustomComponentStatus{
		Component:        dto.HomeAssistantComponentSRAT,
		InstallPath:      "/config/custom_components/srat",
		ManifestPath:     "/config/custom_components/srat/manifest.json",
		Installed:        true,
		CanUpgrade:       true,
		CanUninstall:     true,
		InstalledVersion: &installedVersion,
		Connected:        true,
		ConnectedVersion: &connectedVersion,
	}

	ass := &dto.ReleaseAsset{LastRelease: "2026.04.9"}
	mock.When(suite.haComponentSvc.GetStatus()).ThenReturn(status, nil)
	mock.When(suite.haComponentSvc.SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())).ThenReturn(nil)
	mock.When(suite.upgradeService.GetUpgradeReleaseAsset()).ThenReturn(ass, nil)

	rr := api.Get("/settings/homeassistant/custom-component/status")
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.HomeAssistantCustomComponentStatus
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err)
	suite.True(res.Installed)
	suite.True(res.Connected)
	suite.True(res.CanUpgrade)
	suite.True(res.CanUninstall)
	suite.NotNil(res.InstalledVersion)
	suite.Equal(installedVersion, *res.InstalledVersion)
	suite.NotNil(res.LatestVersion)
	suite.Equal("2026.04.9", *res.LatestVersion)
	_, _ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).GetStatus()
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())
	_, _ = mock.Verify(suite.upgradeService, matchers.Times(1)).GetUpgradeReleaseAsset()
}

func (suite *SettingsHandlerSuite) TestGetHomeAssistantCustomComponentStatusHandler_CreatesIssueOnceWhenMissingDisconnected() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)

	status := &dto.HomeAssistantCustomComponentStatus{
		Component:    dto.HomeAssistantComponentSRAT,
		InstallPath:  "/config/custom_components/srat",
		ManifestPath: "/config/custom_components/srat/manifest.json",
		Installed:    false,
		CanInstall:   true,
		Connected:    false,
	}

	mock.When(suite.haComponentSvc.GetStatus()).ThenReturn(status, nil)
	mock.When(suite.haComponentSvc.SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())).ThenReturn(nil)
	mock.When(suite.upgradeService.GetUpgradeReleaseAsset()).ThenReturn(nil, errors.WithStack(dto.ErrorNoUpdateAvailable))

	rr := api.Get("/settings/homeassistant/custom-component/status")
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	_, _ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).GetStatus()
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())
	_, _ = mock.Verify(suite.upgradeService, matchers.Times(1)).GetUpgradeReleaseAsset()
}

func (suite *SettingsHandlerSuite) TestInstallHomeAssistantCustomComponentHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)

	status := &dto.HomeAssistantCustomComponentStatus{
		Component:        dto.HomeAssistantComponentSRAT,
		InstallPath:      "/config/custom_components/srat",
		ManifestPath:     "/config/custom_components/srat/manifest.json",
		Installed:        true,
		CanUpgrade:       true,
		CanUninstall:     true,
		InstalledVersion: new("2026.04.8"),
	}

	mock.When(suite.haComponentSvc.InstallOrUpgrade()).ThenReturn(nil)
	mock.When(suite.haComponentSvc.GetStatus()).ThenReturn(status, nil)
	mock.When(suite.haComponentSvc.SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())).ThenReturn(nil)
	mock.When(suite.haComponentSvc.UpsertRestartRequiredRepair(mock.AnyContext())).ThenReturn(nil)
	mock.When(suite.upgradeService.GetUpgradeReleaseAsset()).ThenReturn(nil, errors.WithStack(dto.ErrorNoUpdateAvailable))

	rr := api.Post("/settings/homeassistant/custom-component/install", map[string]any{})
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).InstallOrUpgrade()
	_, _ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).GetStatus()
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).UpsertRestartRequiredRepair(mock.AnyContext())
}

func (suite *SettingsHandlerSuite) TestUpgradeHomeAssistantCustomComponentHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)

	status := &dto.HomeAssistantCustomComponentStatus{
		Component:        dto.HomeAssistantComponentSRAT,
		InstallPath:      "/config/custom_components/srat",
		ManifestPath:     "/config/custom_components/srat/manifest.json",
		Installed:        true,
		CanUpgrade:       true,
		CanUninstall:     true,
		InstalledVersion: new("2026.04.9"),
	}

	mock.When(suite.haComponentSvc.InstallOrUpgrade()).ThenReturn(nil)
	mock.When(suite.haComponentSvc.GetStatus()).ThenReturn(status, nil)
	mock.When(suite.haComponentSvc.SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())).ThenReturn(nil)
	mock.When(suite.haComponentSvc.UpsertRestartRequiredRepair(mock.AnyContext())).ThenReturn(nil)
	mock.When(suite.upgradeService.GetUpgradeReleaseAsset()).ThenReturn(nil, errors.WithStack(dto.ErrorNoUpdateAvailable))

	rr := api.Post("/settings/homeassistant/custom-component/upgrade", map[string]any{})
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).InstallOrUpgrade()
	_, _ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).GetStatus()
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).UpsertRestartRequiredRepair(mock.AnyContext())
}

func (suite *SettingsHandlerSuite) TestUninstallHomeAssistantCustomComponentHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)

	status := &dto.HomeAssistantCustomComponentStatus{
		Component:    dto.HomeAssistantComponentSRAT,
		InstallPath:  "/config/custom_components/srat",
		ManifestPath: "/config/custom_components/srat/manifest.json",
		Installed:    false,
		Connected:    false,
	}

	mock.When(suite.haComponentSvc.Uninstall()).ThenReturn(nil)
	mock.When(suite.haComponentSvc.GetStatus()).ThenReturn(status, nil)
	mock.When(suite.haComponentSvc.SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())).ThenReturn(nil)
	mock.When(suite.haComponentSvc.UpsertRestartRequiredRepair(mock.AnyContext())).ThenReturn(nil)

	rr := api.Delete("/settings/homeassistant/custom-component")
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).Uninstall()
	_, _ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).GetStatus()
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).SyncIssueStatus(mock.Any[*dto.HomeAssistantCustomComponentStatus]())
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).UpsertRestartRequiredRepair(mock.AnyContext())
}

func (suite *SettingsHandlerSuite) TestGetAppConfigHandler_AutoDismissesRepairWhenRestartNotRequired() {
	_, humaAPI := humatest.New(suite.T())
	suite.api.RegisterSettings(humaAPI)

	mock.When(suite.addonsService.GetAppConfig(mock.AnyContext())).
		ThenReturn(&dto.AppConfigData{
			Options:         map[string]any{"log_level": "info"},
			RuntimeConfig:   map[string]any{"log_level": "info"},
			RequiresRestart: false,
		}, nil)
	mock.When(suite.haComponentSvc.DismissAddonConfigIssue(mock.AnyContext())).
		ThenReturn(nil)

	rr := humaAPI.Get("/settings/app-config")
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).DismissAddonConfigIssue(mock.AnyContext())
}

func (suite *SettingsHandlerSuite) TestGetAppConfigSchemaHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)

	mock.When(suite.addonsService.GetAppConfigSchema(mock.AnyContext())).
		ThenReturn(&dto.AppConfigSchema{
			Description:     "Example app",
			LongDescription: "Example long description",
			RequiresRestart: true,
			Fields: []dto.AppConfigSchemaField{
				{Name: "log_level", Constraint: "str", Description: "Logging level", Optional: false, Options: []string{"debug", "info"}},
			},
		}, nil)

	rr := api.Get("/settings/app-config/schema")
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.AppConfigSchema
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err)
	suite.Equal("Example app", res.Description)
	suite.Len(res.Fields, 1)
	suite.Equal("log_level", res.Fields[0].Name)
	suite.Equal("str", res.Fields[0].Constraint)
	suite.ElementsMatch([]string{"debug", "info"}, res.Fields[0].Options)
}

func (suite *SettingsHandlerSuite) TestUpdateAppConfigHandler() {
	_, api := humatest.New(suite.T())
	suite.api.RegisterSettings(api)
	autopatch.AutoPatch(api)
	suite.dirtyService.ResetDirtyDataTracker()

	request := dto.AppConfigUpdateRequest{
		Options: map[string]any{"log_level": "debug", "enabled": true},
	}

	mock.When(suite.addonsService.SetAppConfig(mock.AnyContext(), mock.Any[map[string]any]())).
		ThenReturn(nil)
	mock.When(suite.addonsService.GetAppConfig(mock.AnyContext())).
		ThenReturn(&dto.AppConfigData{
			Options:         request.Options,
			RuntimeConfig:   map[string]any{"rendered": true},
			RequiresRestart: true,
		}, nil)
	mock.When(suite.haComponentSvc.DismissAddonConfigIssue(mock.AnyContext())).
		ThenReturn(nil)

	rr := api.Put("/settings/app-config", request)
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.AppConfigData
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	suite.Require().NoError(err)
	suite.Equal("debug", res.Options["log_level"])

	tracker := suite.dirtyService.GetDirtyDataTracker()
	suite.True(tracker.AppConfig)
	suite.False(tracker.Settings)

	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).DismissAddonConfigIssue(mock.AnyContext())
}

func (suite *SettingsHandlerSuite) TestUpdateAppConfigHandler_FallbackDismissPersistentNotificationWhenRepairServiceNil() {
	_, humaAPI := humatest.New(suite.T())
	eventBus := events.NewEventBus(suite.ctx)
	handler := api.NewSettingsHanler(&dto.ContextState{}, suite.settingService, suite.addonsService, suite.haComponentSvc, suite.upgradeService, eventBus, nil, suite.haService, suite.broadcaster)
	handler.RegisterSettings(humaAPI)
	autopatch.AutoPatch(humaAPI)

	request := dto.AppConfigUpdateRequest{
		Options: map[string]any{"log_level": "debug"},
	}

	mock.When(suite.addonsService.SetAppConfig(mock.AnyContext(), mock.Any[map[string]any]())).
		ThenReturn(nil)
	mock.When(suite.addonsService.GetAppConfig(mock.AnyContext())).
		ThenReturn(&dto.AppConfigData{
			Options:         request.Options,
			RuntimeConfig:   map[string]any{"rendered": true},
			RequiresRestart: true,
		}, nil)
	mock.When(suite.haComponentSvc.DismissAddonConfigIssue(mock.AnyContext())).
		ThenReturn(nil)

	rr := humaAPI.Put("/settings/app-config", request)
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).DismissAddonConfigIssue(mock.AnyContext())
}

func (suite *SettingsHandlerSuite) TestRestartAddonHandler() {
	_, humaAPI := humatest.New(suite.T())
	suite.api.RegisterSettings(humaAPI)

	mock.When(suite.addonsService.RestartSelfApp(mock.AnyContext())).
		ThenReturn(nil)
	mock.When(suite.haComponentSvc.DismissRestartRequiredRepair(mock.AnyContext())).ThenReturn(nil)

	rr := humaAPI.Put("/restart", map[string]any{})
	suite.Require().Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())
	_ = mock.Verify(suite.addonsService, matchers.Times(1)).RestartSelfApp(mock.AnyContext())
	_ = mock.Verify(suite.haComponentSvc, matchers.Times(1)).DismissRestartRequiredRepair(mock.AnyContext())
}

func (suite *SettingsHandlerSuite) TestRestartAddonHandler_FailsWhenServiceFails() {
	_, humaAPI := humatest.New(suite.T())
	suite.api.RegisterSettings(humaAPI)

	mock.When(suite.addonsService.RestartSelfApp(mock.AnyContext())).
		ThenReturn(errors.New("restart failed"))

	rr := humaAPI.Put("/restart", map[string]any{})
	suite.Require().Equal(http.StatusInternalServerError, rr.Code, "Response body: %s", rr.Body.String())
	_ = mock.Verify(suite.addonsService, matchers.Times(1)).RestartSelfApp(mock.AnyContext())
}
