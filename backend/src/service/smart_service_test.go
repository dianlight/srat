package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/dianlight/smartmontools-go"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	goerrors "gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SmartServiceSuite struct {
	suite.Suite
	service     service.SmartServiceInterface
	smartClient smartmontools.SmartClient
	eventBus    events.EventBusInterface
	app         *fxtest.App
}

func (suite *SmartServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			// Provide EventBus bound to the same context
			func(ctx context.Context) events.EventBusInterface { return events.NewEventBus(ctx) },
			// Provide SmartService via FX
			service.NewSmartService,
			// Provide a mock SmartClient so SmartService receives it via optional param
			mock.Mock[smartmontools.SmartClient],
			mock.Mock[service.BroadcasterServiceInterface],
		),
		fx.Populate(&suite.service),
		fx.Populate(&suite.smartClient),
		fx.Populate(&suite.eventBus),
	)
	suite.app.RequireStart()
}

func (suite *SmartServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *SmartServiceSuite) TestGetSmartInfoDeviceNotExist() {

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact("/dev/nonexistent"))).ThenReturn(nil, fmt.Errorf("SMART Not Supported"))

	// Execute with invalid path
	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "/dev/nonexistent", nil
	})
	info, err := suite.service.GetSmartInfo(context.Background(), "nonexistent")

	// Assert
	suite.Require().Error(err)
	suite.Nil(info)
	suite.True(goerrors.Is(err, dto.ErrorSMARTNotSupported), " expected SMART not supported error %w", err)
	// Verify details
	details := goerrors.Details(err)
	suite.Equal("/dev/nonexistent", details["device"])
	suite.Equal("SMART Not Supported", details["reason"])
}

func (suite *SmartServiceSuite) TestGetSmartInfoSuccess() {
	// Create a temporary file to simulate device existence
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	// Mock smartctl response
	mockSMARTInfo := &smartmontools.SMARTInfo{
		Device: smartmontools.Device{
			Name: tempFile.Name(),
			Type: "sat",
		},
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		Temperature: &smartmontools.Temperature{
			Current: 35,
		},
		PowerOnTime: &smartmontools.PowerOnTime{
			Hours: 1000,
		},
		PowerCycleCount: 50,
		AtaSmartData: &smartmontools.AtaSmartData{
			Table: []smartmontools.SmartAttribute{
				{
					ID:     194, // Temperature
					Name:   "Temperature_Celsius",
					Value:  35,
					Worst:  30,
					Thresh: 0,
					Raw: smartmontools.Raw{
						Value: 35,
					},
				},
				{
					ID:     9, // Power On Hours
					Name:   "Power_On_Hours",
					Value:  100,
					Worst:  100,
					Thresh: 0,
					Raw: smartmontools.Raw{
						Value: 1000,
					},
				},
				{
					ID:     12, // Power Cycle Count
					Name:   "Power_Cycle_Count",
					Value:  100,
					Worst:  100,
					Thresh: 0,
					Raw: smartmontools.Raw{
						Value: 50,
					},
				},
			},
		},
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	// Execute
	info, err := suite.service.GetSmartInfo(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
	suite.NotNil(info)
	suite.Equal("SATA", info.DiskType)
	suite.True(info.Supported)
	// Dynamic fields like Enabled, Temperature, PowerOnHours, PowerCycleCount are now in SmartStatus
}

func (suite *SmartServiceSuite) TestGetSmartInfoWithRotationRate() {
	// Create a temporary file to simulate device existence
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	rpm := 7200
	// Mock smartctl response with rotation rate
	mockSMARTInfo := &smartmontools.SMARTInfo{
		Device: smartmontools.Device{
			Name: tempFile.Name(),
			Type: "sat",
		},
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		RotationRate: &rpm,
		Temperature: &smartmontools.Temperature{
			Current: 40,
		},
		PowerOnTime: &smartmontools.PowerOnTime{
			Hours: 5000,
		},
		PowerCycleCount: 100,
		AtaSmartData: &smartmontools.AtaSmartData{
			Table: []smartmontools.SmartAttribute{},
		},
	}

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	// Execute
	info, err := suite.service.GetSmartInfo(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
	suite.NotNil(info)
	suite.Equal("SATA", info.DiskType)
	suite.Equal(7200, info.RotationRate, "RPM should be populated when > 0")
	suite.True(info.Supported)
}

func (suite *SmartServiceSuite) TestGetSmartInfoWithZeroRotationRate() {
	// Create a temporary file to simulate device existence
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	rpm := 0 // SSD
	// Mock smartctl response with zero rotation rate
	mockSMARTInfo := &smartmontools.SMARTInfo{
		Device: smartmontools.Device{
			Name: tempFile.Name(),
			Type: "sat",
		},
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		RotationRate: &rpm,
		Temperature: &smartmontools.Temperature{
			Current: 30,
		},
		PowerOnTime: &smartmontools.PowerOnTime{
			Hours: 2000,
		},
		PowerCycleCount: 75,
		AtaSmartData: &smartmontools.AtaSmartData{
			Table: []smartmontools.SmartAttribute{},
		},
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	// Execute
	info, err := suite.service.GetSmartInfo(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
	suite.NotNil(info)
	suite.Equal("SATA", info.DiskType)
	suite.Equal(0, info.RotationRate, "RPM should not be populated when = 0 (SSD)")
	suite.True(info.Supported)
}

func (suite *SmartServiceSuite) TestGetSmartInfoDeviceNotReadable() {
	// Skip this test if running as root (uid 0) since permission checks don't work
	if os.Getuid() == 0 {
		suite.T().Skip("Skipping permission test when running as root")
	}

	// Create a temp file and remove read permission
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())
	suite.Require().NoError(os.Chmod(tempFile.Name(), 0000))
	defer func() {
		_ = os.Chmod(tempFile.Name(), 0644) // Restore for cleanup
	}()

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	// Mock smartClient to return error for unreadable device
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(nil, fmt.Errorf("device not readable: permission denied"))

	// Execute
	info, err := suite.service.GetSmartInfo(context.Background(), "tempFile.Name()")

	// Assert
	suite.Error(err)
	suite.Nil(info)
	// The error should be a generic error since it's a permission issue
	suite.Contains(err.Error(), "failed to get SMART info", "Error should indicate SMART info retrieval failure")
}

func TestSmartServiceSuite(t *testing.T) {
	suite.Run(t, new(SmartServiceSuite))
}

func (suite *SmartServiceSuite) TestGetHealthStatusDeviceNotExist() {

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "", fmt.Errorf("device not exists")
	})

	// Execute with non-existent device
	health, err := suite.service.GetHealthStatus(context.Background(), "nonexistent")

	// Expect error since device doesn't exist
	suite.Error(err)
	suite.Nil(health)
}

func (suite *SmartServiceSuite) TestGetHealthStatusSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	// Mock GetSMARTInfo
	mockSMARTInfo := &smartmontools.SMARTInfo{
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		AtaSmartData: &smartmontools.AtaSmartData{
			Table: []smartmontools.SmartAttribute{
				{
					ID:     5, // Reallocated Sectors Count
					Name:   "Reallocated_Sector_Ct",
					Value:  100,
					Worst:  100,
					Thresh: 10,
				},
			},
		},
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(mockSMARTInfo, nil)
	mock.When(suite.smartClient.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(true, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	// Execute
	health, err := suite.service.GetHealthStatus(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
	suite.NotNil(health)
	suite.True(health.Passed)
	suite.Equal("healthy", health.OverallStatus)
}

func (suite *SmartServiceSuite) TestStartSelfTestInvalidType() {
	stype, err := dto.ParseSmartTestType("invalid type")
	suite.Require().NoError(err, " expected no error for invalid test type:%v err:%w", stype, err)
	suite.Require().NotNil(stype)
	suite.Require().False(stype.IsValid(), " expected unknown test type for invalid input")
	err = suite.service.StartSelfTest(context.Background(), "/dev/sda", stype)

	suite.Error(err)
	suite.True(goerrors.Is(err, dto.ErrorInvalidParameter), " expected invalid parameter error %w", err)
}

func (suite *SmartServiceSuite) TestStartSelfTestDeviceNotExist() {
	err := suite.service.StartSelfTest(context.Background(), "/dev/nonexistent", dto.SmartTestTypes.SMARTTESTTYPESHORT)

	suite.Error(err)
}

func (suite *SmartServiceSuite) TestStartSelfTestSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.RunSelfTest(mock.Any[context.Context](), mock.Exact(tempFile.Name()), mock.Exact("short"))).ThenReturn(nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	// Execute
	err := suite.service.StartSelfTest(context.Background(), tempFile.Name(), dto.SmartTestTypes.SMARTTESTTYPESHORT)

	// Assert
	suite.NoError(err)
}

func (suite *SmartServiceSuite) TestEnableDisableSMARTDeviceNotExist() {

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "", fmt.Errorf("not found")
	})
	// Test EnableSMART
	err := suite.service.EnableSMART(context.Background(), "/dev/nonexistent")
	suite.Error(err)

	// Test DisableSMART
	err = suite.service.DisableSMART(context.Background(), "/dev/nonexistent")
	suite.Error(err)
}

func (suite *SmartServiceSuite) TestEnableSMARTSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.EnableSMART(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil)
	mock.When(suite.smartClient.IsSMARTSupported(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(&smartmontools.SmartSupport{Available: true, Enabled: true}, nil)
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{SmartSupport: &smartmontools.SmartSupport{Available: true, Enabled: true}}, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	// Execute
	err := suite.service.EnableSMART(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
}

func (suite *SmartServiceSuite) TestDisableSMARTSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.DisableSMART(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil)
	mock.When(suite.smartClient.IsSMARTSupported(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(&smartmontools.SmartSupport{Available: true, Enabled: false}, nil)
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(&smartmontools.SMARTInfo{SmartSupport: &smartmontools.SmartSupport{Available: true, Enabled: false}}, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	// Execute
	err := suite.service.DisableSMART(suite.T().Context(), tempFile.Name())

	// Assert
	suite.NoError(err)
}

// TestEnableSMART_EventDiskIdMatchesDeviceId verifies that the SmartEvent emitted by
// EnableSMART carries the canonical deviceId (as passed to the function), not the raw
// device path returned by the device-to-device mapper.
func (suite *SmartServiceSuite) TestEnableSMART_EventDiskIdMatchesDeviceId() {
	canonicalID := "ata-TEST_DEVICE_12345"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.EnableSMART(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil)
	mock.When(suite.smartClient.IsSMARTSupported(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(&smartmontools.SmartSupport{Available: true, Enabled: true}, nil)
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{SmartSupport: &smartmontools.SmartSupport{Available: true, Enabled: true}}, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	var capturedDiskId string
	unsubscribe := suite.eventBus.OnSmart(func(_ context.Context, se events.SmartEvent) goerrors.E {
		capturedDiskId = se.SmartInfo.DiskId
		return nil
	})
	defer unsubscribe()

	err := suite.service.EnableSMART(context.Background(), canonicalID)

	suite.NoError(err)
	suite.Equal(canonicalID, capturedDiskId,
		"EmitSmart must use the canonical deviceId, not the raw device path")
}

// TestDisableSMART_EventDiskIdMatchesDeviceId verifies that the SmartEvent emitted by
// DisableSMART carries the canonical deviceId (as passed to the function), not the raw
// device path returned by the device-to-device mapper.
func (suite *SmartServiceSuite) TestDisableSMART_EventDiskIdMatchesDeviceId() {
	canonicalID := "ata-TEST_DEVICE_67890"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.DisableSMART(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil)
	mock.When(suite.smartClient.IsSMARTSupported(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(&smartmontools.SmartSupport{Available: true, Enabled: false}, nil)
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{SmartSupport: &smartmontools.SmartSupport{Available: true, Enabled: false}}, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	var capturedDiskId string
	unsubscribe := suite.eventBus.OnSmart(func(_ context.Context, se events.SmartEvent) goerrors.E {
		capturedDiskId = se.SmartInfo.DiskId
		return nil
	})
	defer unsubscribe()

	err := suite.service.DisableSMART(context.Background(), canonicalID)

	suite.NoError(err)
	suite.Equal(canonicalID, capturedDiskId,
		"EmitSmart must use the canonical deviceId, not the raw device path")
}

func (suite *SmartServiceSuite) TestGetSmartInfo_DiskIdMatchesDeviceId() {
	canonicalID := "ata-TEST_DEVICE_GETINFO"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{SmartSupport: &smartmontools.SmartSupport{Available: true, Enabled: true}}, nil)

	result, err := suite.service.GetSmartInfo(context.Background(), canonicalID)

	suite.NoError(err)
	suite.Require().NotNil(result)
	suite.Equal(canonicalID, result.DiskId,
		"GetSmartInfo must return the canonical deviceId, not the raw device path")
}

func (suite *SmartServiceSuite) TestGetTestStatusDeviceNotExist() {

	status, err := suite.service.GetTestStatus(context.Background(), "/dev/nonexistent")

	suite.Error(err)
	suite.Nil(status)
}

func (suite *SmartServiceSuite) TestGetTestStatusSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	// Mock GetSMARTInfo with self-test status
	mockSMARTInfo := &smartmontools.SMARTInfo{
		AtaSmartData: &smartmontools.AtaSmartData{
			SelfTest: &smartmontools.SelfTest{
				Status: &smartmontools.StatusField{String: "short test completed without error"},
			},
		},
	}

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(mockSMARTInfo, nil)

	// Execute
	status, err := suite.service.GetTestStatus(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
	suite.NotNil(status)
	suite.Equal("short test completed without error", status.Status)
	suite.Equal("short", status.TestType)
	suite.False(status.Running)
}

func (suite *SmartServiceSuite) TestGetTestStatusInProgress() {
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mockSMARTInfo := &smartmontools.SMARTInfo{
		AtaSmartData: &smartmontools.AtaSmartData{
			SelfTest: &smartmontools.SelfTest{
				Status: &smartmontools.StatusField{String: "in progress, 30% remaining"},
			},
		},
	}

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(mockSMARTInfo, nil)

	status, err := suite.service.GetTestStatus(context.Background(), tempFile.Name())

	suite.NoError(err)
	suite.Require().NotNil(status)
	suite.True(status.Running, "GetTestStatus must set Running=true when test is in progress")
	suite.Equal(70, status.PercentComplete, "PercentComplete must be 100 - remaining")
}

// TestStartSelfTest_CompletionEventEmitted verifies that StartSelfTest emits a
// final SmartEvent with Running=false and the canonical deviceId after the test
// completes, so the frontend always receives a definitive completion signal.
func (suite *SmartServiceSuite) TestStartSelfTest_CompletionEventEmitted() {
	canonicalID := "ata-TEST_DEVICE_COMPLETION"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	// RunSelfTestWithProgress is not explicitly mocked — returns nil with no
	// progress callbacks invoked. The only event emitted is the completion one.

	var capturedEvents []events.SmartEvent
	unsubscribe := suite.eventBus.OnSmart(func(_ context.Context, se events.SmartEvent) goerrors.E {
		capturedEvents = append(capturedEvents, se)
		return nil
	})
	defer unsubscribe()

	err := suite.service.StartSelfTest(context.Background(), canonicalID, dto.SmartTestTypes.SMARTTESTTYPESHORT)

	suite.NoError(err)
	suite.Require().NotEmpty(capturedEvents, "StartSelfTest must emit at least one SmartEvent")

	// The last event must be the completion event
	last := capturedEvents[len(capturedEvents)-1]
	suite.False(last.SmartTestStatus.Running,
		"last SmartEvent from StartSelfTest must have Running=false")
	suite.Equal(canonicalID, last.SmartTestStatus.DiskId,
		"completion event must carry the canonical deviceId, not the raw device path")
	suite.Equal(100, last.SmartTestStatus.PercentComplete,
		"completion event must have PercentComplete=100")
}

func (suite *SmartServiceSuite) TestAbortSelfTestDeviceNotExist() {
	err := suite.service.AbortSelfTest(context.Background(), "/dev/nonexistent")

	suite.Error(err)
}

func (suite *SmartServiceSuite) TestAbortSelfTestSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	mock.When(suite.smartClient.AbortSelfTest(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil)

	// Execute
	err := suite.service.AbortSelfTest(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
}

// TestUserCapacityParsing tests that both legacy (int64) and new (object) user_capacity formats
// from smartctl can be properly parsed
func (suite *SmartServiceSuite) TestUserCapacityParsing() {
	testCases := []struct {
		name          string
		jsonFile      string
		expectedBytes int64
	}{
		{
			name:          "Legacy format (smartctl < 7.3)",
			jsonFile:      "../../test/data/smartctl-legacy-user-capacity.json",
			expectedBytes: 240057409536,
		},
		{
			name:          "New format (smartctl >= 7.3)",
			jsonFile:      "../../test/data/smartctl-7.3-user-capacity.json",
			expectedBytes: 240057409536,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Read test JSON file
			data, err := os.ReadFile(tc.jsonFile)
			suite.NoError(err)

			// Parse JSON using a compatibility struct that supports both legacy (number)
			// and new (object with bytes field) formats for user_capacity
			type compat struct {
				UserCapacity any `json:"user_capacity"`
			}
			var s compat
			err = json.Unmarshal(data, &s)
			suite.NoError(err)

			// Extract bytes accounting for both representations
			var bytes int64
			switch v := s.UserCapacity.(type) {
			case float64:
				bytes = int64(v)
			case map[string]any:
				if b, ok := v["bytes"].(float64); ok {
					bytes = int64(b)
				}
			default:
				bytes = 0
			}

			suite.Equal(tc.expectedBytes, bytes)
		})
	}
}
