package service_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type HDIdleServiceSuite struct {
	suite.Suite
	db             *gorm.DB
	app            *fxtest.App
	service        service.HDIdleServiceInterface
	settingService service.SettingServiceInterface
}

func TestHDIdleServiceSuite(t *testing.T) {
	suite.Run(t, new(HDIdleServiceSuite))
}

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_EmptyPath() {
	support, err := suite.service.CheckDeviceSupport("")
	suite.NoError(err)
	suite.False(support.Supported)
	suite.Equal("device path cannot be empty", support.ErrorMessage)
}

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_InvalidPath() {
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleEnabled: new(true),
	}, nil)

	support, err := suite.service.CheckDeviceSupport("/invalid/path")
	suite.NoError(err)
	suite.False(support.Supported)
	suite.Contains(support.ErrorMessage, "failed to stat device path")
}

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_NotBlockDeviceOrSymlink() {
	tempFile, err := os.CreateTemp("", "testfile")
	suite.NoError(err)
	defer os.Remove(tempFile.Name())

	support, err := suite.service.CheckDeviceSupport(tempFile.Name())
	suite.NoError(err)
	suite.False(support.Supported)
	suite.Equal("device path is not a block device or symlink", support.ErrorMessage)
}

func (suite *HDIdleServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					HACoreReady:  true,
					DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)",
				}
			},
			dbom.NewDB,
			mock.Mock[events.EventBusInterface],
			service.NewHDIdleService,
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[hardware.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.db),
		fx.Populate(&suite.service),
		fx.Populate(&suite.settingService),
	)

	// Default to global disabled to avoid auto-start via OnStart hook.
	// Individual tests will override this as needed.
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleEnabled:                 boolPtr(false),
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: false,
	}, nil)

	suite.app.RequireStart()

	// Service won't auto-start (global disabled by default)
}

// SetupTestWithServiceEnabled provides a setup for tests that need the service to be enabled
func (suite *HDIdleServiceSuite) setupWithServiceEnabled() {
	suite.TearDownTest() // Clean up previous setup if any

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					HACoreReady:  true,
					DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)",
				}
			},
			dbom.NewDB,
			mock.Mock[events.EventBusInterface],
			service.NewHDIdleService,
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[hardware.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.db),
		fx.Populate(&suite.service),
		fx.Populate(&suite.settingService),
	)

	// Enable the service for this setup
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		HDIdleEnabled:                 boolPtr(true),
		HDIdleDefaultIdleTime:         600,
		HDIdleDefaultCommandType:      dto.HdidleCommands.SCSICOMMAND,
		HDIdleDefaultPowerCondition:   0,
		HDIdleIgnoreSpinDownDetection: false,
	}, nil)

	suite.app.RequireStart()
}

func (suite *HDIdleServiceSuite) TearDownTest() {
	if suite.service != nil && suite.service.IsRunning() {
		_ = suite.service.Stop()
	}
	suite.app.RequireStop()
}

func (suite *HDIdleServiceSuite) TestNewHDIdleService() {
	suite.NotNil(suite.service)
	// Service may already be running due to OnStart hook
	// suite.False(suite.service.IsRunning())
}

func (suite *HDIdleServiceSuite) TestStopWhenNotRunning() {
	// Stopping an already-stopped service should be a no-op and not return an error
	_ = suite.service.Stop()
	err := suite.service.Stop()
	suite.NoError(err, "Stop() should not return error when service is not running")
}

func (suite *HDIdleServiceSuite) TestGetDeviceConfig() {
	suite.setupWithServiceEnabled()

	expectedDevice := &dbom.HDIdleDevice{
		DevicePath:     "sda",
		IdleTime:       300,
		CommandType:    &dto.HdidleCommands.SCSICOMMAND,
		PowerCondition: 1,
	}

	suite.NoError(suite.db.Create(expectedDevice).Error)

	config, err := suite.service.GetDeviceConfig("sda")
	suite.NoError(err)
	suite.NotNil(config)
	suite.Equal("sda", config.DevicePath)
	suite.Equal(300*time.Second, config.IdleTime)
}

func (suite *HDIdleServiceSuite) TestGetDeviceUnsupported() {
	suite.setupWithServiceEnabled()

	config, err := suite.service.GetDeviceConfig("nonexistent")
	suite.Require().Error(err)
	suite.ErrorIs(err, dto.ErrorHDIdleNotSupported)
	suite.NotNil(config)
	suite.False(config.Supported)
}

func (suite *HDIdleServiceSuite) TestSaveDeviceConfig() {
	suite.setupWithServiceEnabled()

	device := dto.HDIdleDevice{
		DiskId:         "ssssa",
		IdleTime:       300,
		CommandType:    dto.HdidleCommands.SCSICOMMAND,
		PowerCondition: 1,
	}

	// Note: In a real test, you might want to mock the Update call
	// For now, we'll just test that the method doesn't panic
	err := suite.service.SaveDeviceConfig(device)
	suite.NoError(err)
}

// --- Tri-state enabled behavior tests ---

// --- CheckDeviceSupport tests ---

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_NonExistentDevice() {
	// Test with a device path that doesn't exist
	support, err := suite.service.CheckDeviceSupport("/dev/nonexistent_device_xyz")
	suite.NoError(err)
	suite.NotNil(support)
	suite.False(support.Supported)
	suite.False(support.SupportsSCSI)
	suite.False(support.SupportsATA)
	suite.NotEmpty(support.ErrorMessage)
}

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_RelativePath() {
	// Test with a relative path that doesn't resolve
	support, err := suite.service.CheckDeviceSupport("sda")
	suite.NoError(err)
	suite.NotNil(support)
	// The device path should be returned even if not supported
	suite.NotEmpty(support.DevicePath)
}

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_SymlinkPath() {
	// Test with a symlink-style path (typical for /dev/disk/by-id/)
	// This will fail to resolve but should not panic
	support, err := suite.service.CheckDeviceSupport("/dev/disk/by-id/fake-device-id")
	suite.NoError(err)
	suite.NotNil(support)
	suite.False(support.Supported)
}

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_NullDevice() {
	// Test with /dev/null which exists but is not a block device
	support, err := suite.service.CheckDeviceSupport("/dev/null")
	suite.NoError(err)
	suite.NotNil(support)
	suite.False(support.Supported)
	// /dev/null doesn't support SG interface
	suite.False(support.SupportsSCSI)
	suite.False(support.SupportsATA)
}

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_ReturnsDevicePath() {
	// Verify the function always returns the device path in the result
	testPath := "/dev/some_test_device"
	support, err := suite.service.CheckDeviceSupport(testPath)
	suite.NoError(err)
	suite.NotNil(support)
	// DevicePath should be set (either original or resolved)
	suite.NotEmpty(support.DevicePath)
}

func (suite *HDIdleServiceSuite) TestCheckDeviceSupport_RecommendedCommandNilWhenNotSupported() {
	// When device is not supported, RecommendedCommand should be nil
	support, err := suite.service.CheckDeviceSupport("/dev/nonexistent")
	suite.NoError(err)
	suite.NotNil(support)
	suite.False(support.Supported)
	suite.Nil(support.RecommendedCommand)
}

// Tests for GetProcessStatus
func (suite *HDIdleServiceSuite) TestGetProcessStatus_WhenNotRunning() {
	// When service is not running, should return idle status
	parentPid := int32(12345)
	suite.service.Stop()
	status := suite.service.GetProcessStatus(parentPid)

	suite.NotNil(status)
	suite.Equal(-parentPid, status.Pid, "Subprocess should have negative PID")
	suite.Equal("powersave-monitor", status.Name)
	suite.False(status.IsRunning)
	suite.Equal([]string{"idle"}, status.Status)
	suite.Equal(0, status.Connections)
}

func (suite *HDIdleServiceSuite) TestGetProcessStatus_NegativePidConvention() {
	// Test various parent PIDs to verify the negative PID convention
	testCases := []int32{1, 100, 12345, 99999}

	for _, parentPid := range testCases {
		status := suite.service.GetProcessStatus(parentPid)
		suite.NotNil(status)
		suite.Equal(-parentPid, status.Pid,
			"For parent PID %d, subprocess should have PID %d", parentPid, -parentPid)
	}
}

func (suite *HDIdleServiceSuite) TestGetProcessStatus_ReturnsConsistentName() {
	// GetProcessStatus should always return "powersave-monitor" as name
	status1 := suite.service.GetProcessStatus(1000)
	status2 := suite.service.GetProcessStatus(2000)

	suite.Equal("powersave-monitor", status1.Name)
	suite.Equal("powersave-monitor", status2.Name)
	suite.Equal(status1.Name, status2.Name)
}
