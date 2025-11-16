package service_test

import (
	"context"
	"log"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/templates"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SettingServiceGenericSuite struct {
	suite.Suite
	settingService service.SettingServiceInterface
	propertyRepo   repository.PropertyRepositoryInterface
	app            *fxtest.App
}

func TestSettingServiceGenericSuite(t *testing.T) {
	suite.Run(t, new(SettingServiceGenericSuite))
}

func (suite *SettingServiceGenericSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
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
			service.NewSettingService,
			events.NewEventBus,
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.TelemetryServiceInterface],
		),
		fx.Populate(&suite.settingService),
		fx.Populate(&suite.propertyRepo),
	)
	suite.app.RequireStart()
}

func (suite *SettingServiceGenericSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *SettingServiceGenericSuite) TestGetValueAs_StringFromRepo() {
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("host", nil)

	val, err := service.GetValueAs[string](suite.settingService, "hostname")

	suite.NoError(err)
	suite.Equal("host", val)
}

func (suite *SettingServiceGenericSuite) TestGetValueAs_DefaultFallback() {
	// Not found in repo; should fallback to default (string)
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))

	val, err := service.GetValueAs[string](suite.settingService, "hostname")

	suite.NoError(err)
	suite.NotEmpty(val)
}

func (suite *SettingServiceGenericSuite) TestGetValueAs_TypeMismatch() {
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn(true, nil)

	_, err := service.GetValueAs[string](suite.settingService, "hostname")

	suite.Error(err)
	suite.Contains(err.Error(), "type mismatch")
}

func (suite *SettingServiceGenericSuite) TestSetValueAs_Wrapper() {
	mock.When(suite.propertyRepo.Value("hostname", true)).ThenReturn("old", nil)
	mock.When(suite.propertyRepo.SetValue("hostname", "new")).ThenReturn(nil)

	err := service.SetValueAs(suite.settingService, "hostname", "new")

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("hostname", "new")
}

func (suite *SettingServiceGenericSuite) TestGetValueAs_BoolFromRepo() {
	mock.When(suite.propertyRepo.Value("compatibility_mode", true)).ThenReturn(true, nil)

	val, err := service.GetValueAs[bool](suite.settingService, "compatibility_mode")

	suite.NoError(err)
	suite.True(val)
}

func (suite *SettingServiceGenericSuite) TestGetValueAs_SliceFromRepo() {
	expected := []string{"eth0", "eth1"}
	mock.When(suite.propertyRepo.Value("interfaces", true)).ThenReturn(expected, nil)

	val, err := service.GetValueAs[[]string](suite.settingService, "interfaces")

	suite.NoError(err)
	suite.Equal(expected, val)
}

func (suite *SettingServiceGenericSuite) TestGetValueAs_PointerDefaultFallback() {
	// default for local_master is true per dto.Settings tag
	mock.When(suite.propertyRepo.Value("local_master", true)).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))

	val, err := service.GetValueAs[*bool](suite.settingService, "local_master")

	suite.NoError(err)
	suite.NotNil(val)
	suite.True(*val)
}

func (suite *SettingServiceGenericSuite) TestSetValueAs_Bool() {
	mock.When(suite.propertyRepo.Value("compatibility_mode", true)).ThenReturn(true, nil)
	mock.When(suite.propertyRepo.SetValue("compatibility_mode", false)).ThenReturn(nil)

	err := service.SetValueAs(suite.settingService, "compatibility_mode", false)

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("compatibility_mode", false)
}

func (suite *SettingServiceGenericSuite) TestSetValueAs_Slice() {
	oldSlice := []string{"eth0"}
	newSlice := []string{"eth0", "eth1"}
	mock.When(suite.propertyRepo.Value("interfaces", true)).ThenReturn(oldSlice, nil)
	mock.When(suite.propertyRepo.SetValue("interfaces", newSlice)).ThenReturn(nil)

	err := service.SetValueAs(suite.settingService, "interfaces", newSlice)

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("interfaces", newSlice)
}

func (suite *SettingServiceGenericSuite) TestSetValueAs_Pointer() {
	trueVal := true
	falseVal := false
	mock.When(suite.propertyRepo.Value("local_master", true)).ThenReturn(&trueVal, nil)
	mock.When(suite.propertyRepo.SetValue("local_master", &falseVal)).ThenReturn(nil)

	err := service.SetValueAs(suite.settingService, "local_master", &falseVal)

	suite.NoError(err)
	mock.Verify(suite.propertyRepo, matchers.Times(1)).SetValue("local_master", &falseVal)
}
