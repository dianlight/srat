package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx/fxtest"
)

type HDIdleServiceSuite struct {
	suite.Suite
	app     *fxtest.App
	service service.HDIdleServiceInterface
	ctx     context.Context
	cancel  context.CancelFunc
}

func TestHDIdleServiceSuite(t *testing.T) {
	suite.Run(t, new(HDIdleServiceSuite))
}

func (suite *HDIdleServiceSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))

	state := &dto.ContextState{
		HACoreReady: true,
	}

	params := service.HDIdleServiceParams{
		ApiContext:       suite.ctx,
		ApiContextCancel: suite.cancel,
		State:            state,
	}

	suite.service = service.NewHDIdleService(params)
}

func (suite *HDIdleServiceSuite) TearDownTest() {
	if suite.service.IsRunning() {
		_ = suite.service.Stop()
	}
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value("wg").(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func (suite *HDIdleServiceSuite) TestNewHDIdleService() {
	suite.NotNil(suite.service)
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithValidConfig() {
	config := &service.HDIdleConfig{
		DefaultIdleTime:    600,
		DefaultCommandType: "scsi",
		Debug:              false,
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithDefaultValues() {
	config := &service.HDIdleConfig{}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartAlreadyRunning() {
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
	}

	err := suite.service.Start(config)
	suite.NoError(err)

	// Try to start again
	err = suite.service.Start(config)
	suite.Error(err)
	suite.Contains(err.Error(), "already running")
}

func (suite *HDIdleServiceSuite) TestStartWithNilConfig() {
	err := suite.service.Start(nil)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid configuration")
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithInvalidCommandType() {
	config := &service.HDIdleConfig{
		DefaultCommandType: "invalid",
	}

	err := suite.service.Start(config)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid configuration")
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithDevicesNoName() {
	config := &service.HDIdleConfig{
		Devices: []service.HDIdleDeviceConfig{
			{
				Name: "",
			},
		},
	}

	err := suite.service.Start(config)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid configuration")
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithDevicesInvalidCommandType() {
	config := &service.HDIdleConfig{
		Devices: []service.HDIdleDeviceConfig{
			{
				Name:        "sda",
				CommandType: "invalid",
			},
		},
	}

	err := suite.service.Start(config)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid configuration")
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithValidDevices() {
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
		Devices: []service.HDIdleDeviceConfig{
			{
				Name:        "sda",
				IdleTime:    300,
				CommandType: "scsi",
			},
			{
				Name:        "sdb",
				IdleTime:    900,
				CommandType: "ata",
			},
		},
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStopWhenNotRunning() {
	err := suite.service.Stop()
	suite.Error(err)
	suite.Contains(err.Error(), "not running")
}

func (suite *HDIdleServiceSuite) TestStopWhenRunning() {
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
	}

	err := suite.service.Start(config)
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
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
	}

	err := suite.service.Start(config)
	suite.NoError(err)

	// Wait a bit for monitoring to start
	time.Sleep(100 * time.Millisecond)

	status, err := suite.service.GetStatus()
	suite.NoError(err)
	suite.NotNil(status)
	suite.True(status.Running)
	suite.NotZero(status.MonitoredAt)
}

func (suite *HDIdleServiceSuite) TestStartWithScsiCommandType() {
	config := &service.HDIdleConfig{
		DefaultCommandType: "scsi",
		Devices: []service.HDIdleDeviceConfig{
			{
				Name:        "sda",
				CommandType: "scsi",
			},
		},
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithAtaCommandType() {
	config := &service.HDIdleConfig{
		DefaultCommandType: "ata",
		Devices: []service.HDIdleDeviceConfig{
			{
				Name:        "sda",
				CommandType: "ata",
			},
		},
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithSymlinkPolicy() {
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
		SymlinkPolicy:   1,
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithDebugEnabled() {
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
		Debug:           true,
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithLogFile() {
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
		LogFile:         "/tmp/hdidle-test.log",
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithIgnoreSpinDownDetection() {
	config := &service.HDIdleConfig{
		DefaultIdleTime:         600,
		IgnoreSpinDownDetection: true,
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartWithPowerCondition() {
	config := &service.HDIdleConfig{
		DefaultIdleTime:       600,
		DefaultPowerCondition: 3,
		Devices: []service.HDIdleDeviceConfig{
			{
				Name:           "sda",
				PowerCondition: 5,
			},
		},
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStartStopMultipleTimes() {
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
	}

	// First cycle
	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())

	err = suite.service.Stop()
	suite.NoError(err)
	suite.False(suite.service.IsRunning())

	// Second cycle
	err = suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())

	err = suite.service.Stop()
	suite.NoError(err)
	suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestContextCancellation() {
	config := &service.HDIdleConfig{
		DefaultIdleTime: 600,
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())

	// Cancel the context
	suite.cancel()
	time.Sleep(100 * time.Millisecond)

	// Service should still report running (doesn't auto-stop on context cancel)
	// but monitoring loop should exit
	suite.True(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestMultipleDevicesWithDifferentSettings() {
	config := &service.HDIdleConfig{
		DefaultIdleTime:       600,
		DefaultCommandType:    "scsi",
		DefaultPowerCondition: 0,
		Devices: []service.HDIdleDeviceConfig{
			{
				Name:           "sda",
				IdleTime:       300,
				CommandType:    "scsi",
				PowerCondition: 1,
			},
			{
				Name:           "sdb",
				IdleTime:       900,
				CommandType:    "ata",
				PowerCondition: 0,
			},
			{
				Name:     "sdc",
				IdleTime: 0, // Should use default
			},
		},
	}

	err := suite.service.Start(config)
	suite.NoError(err)
	suite.True(suite.service.IsRunning())

	// Get status to verify configuration
	status, err := suite.service.GetStatus()
	suite.NoError(err)
	suite.NotNil(status)
	suite.True(status.Running)
}
