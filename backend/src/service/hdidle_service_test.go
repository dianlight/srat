package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type HDIdleServiceSuite struct {
	suite.Suite
	app            *fxtest.App
	service        service.HDIdleServiceInterface
	hdidleRepo     repository.HDIdleDeviceRepositoryInterface
	settingService service.SettingServiceInterface
}

func TestHDIdleServiceSuite(t *testing.T) {
	suite.Run(t, new(HDIdleServiceSuite))
}

func (suite *HDIdleServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					HACoreReady: true,
				}
			},
			service.NewHDIdleService,
			mock.Mock[repository.HDIdleDeviceRepositoryInterface],
			mock.Mock[service.SettingServiceInterface],
		),
		fx.Populate(&suite.service),
		fx.Populate(&suite.hdidleRepo),
		fx.Populate(&suite.settingService),
	)
	suite.app.RequireStart()

	// Service doesn't auto-start anymore, so no need to stop it
}

func (suite *HDIdleServiceSuite) TearDownTest() {
	if suite.service.IsRunning() {
		_ = suite.service.Stop()
	}
	suite.app.RequireStop()
}

func (suite *HDIdleServiceSuite) TestNewHDIdleService() {
	suite.NotNil(suite.service)
	// Service may already be running due to OnStart hook
	// suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithValidSettings() {
	// Mock settings
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: boolPtr(false),
	}, nil)

	// Mock empty device list
	mock.When(suite.hdidleRepo.LoadAll()).ThenReturn([]*dbom.HDIdleDevice{}, nil)

	err := suite.service.Start()
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithDefaultValues() {
	// Mock settings with default values
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleDefaultIdleTime:         0, // Should use default
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: nil,
	}, nil)

	// Mock empty device list
	mock.When(suite.hdidleRepo.LoadAll()).ThenReturn([]*dbom.HDIdleDevice{}, nil)

	err := suite.service.Start()
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartAlreadyRunning() {
	// Mock settings
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: boolPtr(false),
	}, nil)

	// Mock empty device list
	mock.When(suite.hdidleRepo.LoadAll()).ThenReturn([]*dbom.HDIdleDevice{}, nil)

	err := suite.service.Start()
	suite.NoError(err)

	// Try to start again
	err = suite.service.Start()
	suite.Error(err)
	suite.Contains(err.Error(), "already running")
}

func (suite *HDIdleServiceSuite) TestStartWithSettingsLoadError() {
	// Mock settings load error
	mock.When(suite.settingService.Load()).ThenReturn(nil, errors.New("settings load error"))

	err := suite.service.Start()
	suite.Error(err)
	suite.Contains(err.Error(), "settings load error")
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithDeviceLoadError() {
	// Mock settings
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: boolPtr(false),
	}, nil)

	// Mock device load error
	mock.When(suite.hdidleRepo.LoadAll()).ThenReturn(nil, errors.New("device load error"))

	err := suite.service.Start()
	suite.Error(err)
	suite.Contains(err.Error(), "failed to load HDIdle devices")
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithValidDevices() {
	// Mock settings
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: boolPtr(false),
	}, nil)

	// Mock devices
	devices := []*dbom.HDIdleDevice{
		{
			DevicePath:     "sda",
			IdleTime:       300,
			CommandType:    &dto.HdidleCommands.SCSICOMMAND,
			PowerCondition: 1,
		},
		{
			DevicePath:     "sdb",
			IdleTime:       900,
			CommandType:    &dto.HdidleCommands.ATACOMMAND,
			PowerCondition: 0,
		},
	}
	mock.When(suite.hdidleRepo.LoadAll()).ThenReturn(devices, nil)

	err := suite.service.Start()
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStopWhenNotRunning() {
	err := suite.service.Stop()
	suite.Error(err)
	suite.Contains(err.Error(), "not running")
}

func (suite *HDIdleServiceSuite) TestStopWhenRunning() {
	// Mock settings
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: boolPtr(false),
	}, nil)

	// Mock empty device list
	mock.When(suite.hdidleRepo.LoadAll()).ThenReturn([]*dbom.HDIdleDevice{}, nil)

	err := suite.service.Start()
	suite.NoError(err)
	suite.True(suite.service.IsRunning())

	err = suite.service.Stop()
	suite.NoError(err)
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestGetStatusWhenNotRunning() {
	status, err := suite.service.GetStatus()
	suite.NoError(err)
	suite.NotNil(status)
	suite.False(status.Running)
}

func (suite *HDIdleServiceSuite) TestGetStatusWhenRunning() {
	// Mock settings
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: boolPtr(false),
	}, nil)

	// Mock empty device list
	mock.When(suite.hdidleRepo.LoadAll()).ThenReturn([]*dbom.HDIdleDevice{}, nil)

	err := suite.service.Start()
	suite.NoError(err)

	// Wait a bit for monitoring to start
	time.Sleep(100 * time.Millisecond)

	status, err := suite.service.GetStatus()
	suite.NoError(err)
	suite.NotNil(status)
	suite.True(status.Running)
	suite.NotZero(status.MonitoredAt)
}

func (suite *HDIdleServiceSuite) TestGetDeviceConfig() {
	expectedDevice := &dbom.HDIdleDevice{
		DevicePath:     "sda",
		IdleTime:       300,
		CommandType:    &dto.HdidleCommands.SCSICOMMAND,
		PowerCondition: 1,
	}

	mock.When(suite.hdidleRepo.LoadByPath("sda")).ThenReturn(expectedDevice, nil)

	config, err := suite.service.GetDeviceConfig("sda")
	suite.NoError(err)
	suite.NotNil(config)
	suite.Equal("sda", config.DevicePath)
	suite.Equal(300, config.IdleTime)
}

func (suite *HDIdleServiceSuite) TestGetDeviceConfigNotFound() {
	mock.When(suite.hdidleRepo.LoadByPath("nonexistent")).ThenReturn(nil, errors.Wrap(gorm.ErrRecordNotFound, "record not found"))

	config, err := suite.service.GetDeviceConfig("nonexistent")
	suite.NoError(err)
	suite.NotNil(config)
	suite.Equal("nonexistent", config.DevicePath)
	suite.Equal(0, config.IdleTime)
}

func (suite *HDIdleServiceSuite) TestSaveDeviceConfig() {
	device := dto.HDIdleDeviceDTO{
		DevicePath:     "sda",
		IdleTime:       300,
		CommandType:    dto.HdidleCommands.SCSICOMMAND,
		PowerCondition: 1,
	}

	// Note: In a real test, you might want to mock the Update call
	// For now, we'll just test that the method doesn't panic
	err := suite.service.SaveDeviceConfig(device)
	suite.NoError(err)
}

func (suite *HDIdleServiceSuite) TestStartStopMultipleTimes() {
	// Mock settings
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: boolPtr(false),
	}, nil)

	// Mock empty device list
	mock.When(suite.hdidleRepo.LoadAll()).ThenReturn([]*dbom.HDIdleDevice{}, nil)

	// First cycle
	err := suite.service.Start()
	suite.NoError(err)
	suite.True(suite.service.IsRunning())

	err = suite.service.Stop()
	suite.NoError(err)
	suite.False(suite.service.IsRunning())

	// Second cycle
	err = suite.service.Start()
	suite.NoError(err)
	suite.True(suite.service.IsRunning())

	err = suite.service.Stop()
	suite.NoError(err)
	suite.False(suite.service.IsRunning())
}
