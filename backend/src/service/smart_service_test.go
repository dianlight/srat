package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/dianlight/smartmontools-sdk/bindings/go/v8"
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

// TestGetHealthStatus_TrustsDevicePassedOverCheckHealth is a regression test
// for #1196: when CheckHealth reports failure but the drive-reported
// overall-health (smart_status.passed, the same source as `smartctl -H`) is
// PASSED and no failing attributes exist, the drive must be reported healthy.
func (suite *SmartServiceSuite) TestGetHealthStatus_TrustsDevicePassedOverCheckHealth() {
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			SmartSupport: &smartmontools.SmartSupport{
				Available: true,
				Enabled:   true,
			},
			SmartStatus: &smartmontools.SmartStatus{
				Passed: true,
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
		}, nil)
	mock.When(suite.smartClient.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(false, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	health, err := suite.service.GetHealthStatus(context.Background(), tempFile.Name())

	suite.NoError(err)
	suite.Require().NotNil(health)
	suite.True(health.Passed, "drive-reported PASSED with no failing attributes must stay healthy")
	suite.Equal("healthy", health.OverallStatus)
	suite.Empty(health.FailingAttributes)
}

// TestGetHealthStatus_VerifyClientOverrulesLibFalseAlarm is a regression test
// for #1196: the lib backend omits smart_status and its CheckHealth can false
// alarm on drives `smartctl -H` passes (smartctl -a exit 0). When the exec
// verify client reports healthy with no failing attributes, the drive must be
// reported healthy.
func (suite *SmartServiceSuite) TestGetHealthStatus_VerifyClientOverrulesLibFalseAlarm() {
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	// Lib-backend shape: no SmartStatus, healthy attribute table.
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			SmartSupport: &smartmontools.SmartSupport{
				Available: true,
				Enabled:   true,
			},
			AtaSmartData: &smartmontools.AtaSmartData{
				Table: []smartmontools.SmartAttribute{
					{
						ID:     5,
						Name:   "Reallocated_Sector_Ct",
						Value:  100,
						Worst:  100,
						Thresh: 10,
					},
				},
			},
		}, nil)
	mock.When(suite.smartClient.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(false, nil)

	ctrl := mock.NewMockController(suite.T())
	verifyMock := mock.Mock[smartmontools.SmartClient](ctrl)
	mock.When(verifyMock.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(true, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	suite.service.MockVerifyClient(verifyMock)

	health, err := suite.service.GetHealthStatus(context.Background(), tempFile.Name())

	suite.NoError(err)
	suite.Require().NotNil(health)
	suite.True(health.Passed, "exec verify PASSED with no failing attributes must report healthy")
	suite.Equal("healthy", health.OverallStatus)
	suite.Empty(health.FailingAttributes)
}

// TestGetHealthStatus_VerifyClientConfirmsFailure guards the fallback: when
// the exec verify client agrees the drive is failing, the result stays
// failing.
func (suite *SmartServiceSuite) TestGetHealthStatus_VerifyClientConfirmsFailure() {
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			SmartSupport: &smartmontools.SmartSupport{
				Available: true,
				Enabled:   true,
			},
			AtaSmartData: &smartmontools.AtaSmartData{
				Table: []smartmontools.SmartAttribute{},
			},
		}, nil)
	mock.When(suite.smartClient.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(false, nil)

	ctrl := mock.NewMockController(suite.T())
	verifyMock := mock.Mock[smartmontools.SmartClient](ctrl)
	mock.When(verifyMock.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(false, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	suite.service.MockVerifyClient(verifyMock)

	health, err := suite.service.GetHealthStatus(context.Background(), tempFile.Name())

	suite.NoError(err)
	suite.Require().NotNil(health)
	suite.False(health.Passed)
	suite.Equal("failing", health.OverallStatus)
}

// TestGetHealthStatus_VerifyClientErrorKeepsPrimary ensures a verify failure
// never flips the result: the primary assessment stands.
func (suite *SmartServiceSuite) TestGetHealthStatus_VerifyClientErrorKeepsPrimary() {
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			SmartSupport: &smartmontools.SmartSupport{
				Available: true,
				Enabled:   true,
			},
			AtaSmartData: &smartmontools.AtaSmartData{
				Table: []smartmontools.SmartAttribute{},
			},
		}, nil)
	mock.When(suite.smartClient.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(false, nil)

	ctrl := mock.NewMockController(suite.T())
	verifyMock := mock.Mock[smartmontools.SmartClient](ctrl)
	mock.When(verifyMock.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(false, fmt.Errorf("smartctl not found"))

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	suite.service.MockVerifyClient(verifyMock)

	health, err := suite.service.GetHealthStatus(context.Background(), tempFile.Name())

	suite.NoError(err)
	suite.Require().NotNil(health)
	suite.False(health.Passed)
}

// TestGetTestStatus_VerifyClientGroundTruth is a regression test for #1196:
// when the primary (lib) backend omits self_test.status, the exec verify
// client provides ground truth, including tests started outside this service.
func (suite *SmartServiceSuite) TestGetTestStatus_VerifyClientGroundTruth() {
	canonicalID := "ata-TEST_DEVICE_VERIFY_GROUND_TRUTH"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	// Primary lib-backend shape: SelfTest present, Status nil.
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			AtaSmartData: &smartmontools.AtaSmartData{
				SelfTest: &smartmontools.SelfTest{},
			},
		}, nil)

	ctrl := mock.NewMockController(suite.T())
	verifyMock := mock.Mock[smartmontools.SmartClient](ctrl)
	mock.When(verifyMock.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			AtaSmartData: &smartmontools.AtaSmartData{
				SelfTest: &smartmontools.SelfTest{
					Status: &smartmontools.StatusField{Value: 250, String: "short self-test in progress, 20% remaining"},
				},
			},
		}, nil)
	suite.service.MockVerifyClient(verifyMock)

	status, err := suite.service.GetTestStatus(context.Background(), canonicalID)

	suite.NoError(err)
	suite.Require().NotNil(status)
	suite.True(status.Running, "verify client in-progress status must be reported")
	suite.Equal("short", status.TestType)
	suite.Equal(80, status.PercentComplete)
}

// TestGetTestStatus_VerifyClientErrorFallsBackToTracker ensures a verify
// failure degrades gracefully to tracked progress instead of an error.
func (suite *SmartServiceSuite) TestGetTestStatus_VerifyClientErrorFallsBackToTracker() {
	canonicalID := "ata-TEST_DEVICE_VERIFY_ERROR_TRACKER"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	mock.When(suite.smartClient.RunSelfTestWithProgress(
		mock.Any[context.Context](),
		mock.Exact(tempFile.Name()),
		mock.Exact("short"),
		mock.Any[smartmontools.ProgressCallback](),
	)).ThenAnswer(matchers.Answer(func(args []any) []any {
		cb := args[3].(smartmontools.ProgressCallback)
		cb(0, "Test started")
		return []any{nil}
	}))
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			AtaSmartData: &smartmontools.AtaSmartData{
				SelfTest: &smartmontools.SelfTest{},
			},
		}, nil)

	ctrl := mock.NewMockController(suite.T())
	verifyMock := mock.Mock[smartmontools.SmartClient](ctrl)
	mock.When(verifyMock.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(nil, fmt.Errorf("smartctl not found"))
	suite.service.MockVerifyClient(verifyMock)

	suite.Require().NoError(suite.service.StartSelfTest(context.Background(), canonicalID, dto.SmartTestTypes.SMARTTESTTYPESHORT))

	status, err := suite.service.GetTestStatus(context.Background(), canonicalID)

	suite.NoError(err)
	suite.Require().NotNil(status)
	suite.True(status.Running, "verify failure must fall back to tracked progress")
}

// TestGetHealthStatus_CheckHealthFailureWithEvidenceStaysFailing guards the
// reconciliation: a false CheckHealth with failing attributes present, or with
// the drive itself reporting FAILED, must still report failing.
func (suite *SmartServiceSuite) TestGetHealthStatus_CheckHealthFailureWithEvidenceStaysFailing() {
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			SmartSupport: &smartmontools.SmartSupport{
				Available: true,
				Enabled:   true,
			},
			SmartStatus: &smartmontools.SmartStatus{
				Passed: true,
			},
			AtaSmartData: &smartmontools.AtaSmartData{
				Table: []smartmontools.SmartAttribute{
					{
						ID:     5, // Reallocated Sectors Count below threshold
						Name:   "Reallocated_Sector_Ct",
						Value:  1,
						Worst:  1,
						Thresh: 10,
					},
				},
			},
		}, nil)
	mock.When(suite.smartClient.CheckHealth(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(false, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	health, err := suite.service.GetHealthStatus(context.Background(), tempFile.Name())

	suite.NoError(err)
	suite.Require().NotNil(health)
	suite.False(health.Passed, "failing attributes must keep the drive failing")
	suite.Equal("failing", health.OverallStatus)
	suite.NotEmpty(health.FailingAttributes)
}

func (suite *SmartServiceSuite) TestGetSmartStatusParsesPackedRawValues() {
	// Regression test for #907: firmware-packed 48-bit raw attribute values must
	// not be exposed as-is. smartctl reports raw strings like
	// "42374h+52m+33.990s" for power-on-hours and "51 (Min/Max -22/57)" for
	// temperature, while the raw integer packs sub-unit data (e.g. hours+msec)
	// and is unusable as-is.
	mockSMARTInfo := &smartmontools.SMARTInfo{
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		Temperature: &smartmontools.Temperature{Current: 51},
		PowerOnTime: &smartmontools.PowerOnTime{Hours: 42374},
		AtaSmartData: &smartmontools.AtaSmartData{
			Table: []smartmontools.SmartAttribute{
				{
					ID:    194, // Temperature_Celsius
					Name:  "Temperature_Celsius",
					Value: 100, // normalized health score, NOT the temperature in °C
					Worst: 57,
					Raw: smartmontools.Raw{
						Value:  1005026082867,
						String: "51 (Min/Max -22/57)",
					},
				},
				{
					ID:    9, // Power_On_Hours_and_Msec
					Name:  "Power_On_Hours_and_Msec",
					Value: 52,
					Worst: 52,
					Raw: smartmontools.Raw{
						Value:  13546283901953414,
						String: "42374h+52m+33.990s",
					},
				},
				{
					ID:    12, // Power_Cycle_Count
					Name:  "Power_Cycle_Count",
					Value: 100,
					Worst: 100,
					Raw: smartmontools.Raw{
						Value:  67,
						String: "67",
					},
				},
			},
		},
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact("/dev/sda"))).ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "/dev/sda", nil
	})

	status, err := suite.service.GetSmartStatus(context.Background(), "sda")

	suite.Require().NoError(err)
	suite.Require().NotNil(status)
	suite.Equal(51, status.Temperature.Value, "temperature must use the normalized °C value, not the packed raw value")
	suite.Equal(42374, status.PowerOnHours.Value, "power-on-hours must be parsed from the raw string")
	suite.Equal(52, status.PowerOnHours.Worst)
	suite.Equal(67, status.PowerCycleCount.Value)
	suite.Equal(100, status.PowerCycleCount.Worst)
}

func (suite *SmartServiceSuite) TestGetSmartStatusIgnoresPackedRawString() {
	// Regression test for #907 (lib/direct backend): the lib backend renders the
	// raw 48-bit value as a bare decimal string ("39393440081287") instead of the
	// smartctl-formatted "42374h+52m+33.990s". Such implausible values must not
	// override the properly parsed top-level smartctl fields.
	mockSMARTInfo := &smartmontools.SMARTInfo{
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		Temperature:     &smartmontools.Temperature{Current: 51},
		PowerOnTime:     &smartmontools.PowerOnTime{Hours: 42374},
		PowerCycleCount: 67,
		AtaSmartData: &smartmontools.AtaSmartData{
			Table: []smartmontools.SmartAttribute{
				{
					ID:    194, // Temperature_Celsius
					Name:  "Temperature_Celsius",
					Value: 100, // normalized health score, NOT the temperature in °C
					Worst: 57,
					Raw: smartmontools.Raw{
						Value:  1005026082867,
						String: "39393440081287",
					},
				},
				{
					ID:    9, // Power_On_Hours_and_Msec
					Name:  "Power_On_Hours_and_Msec",
					Value: 52,
					Worst: 52,
					Raw: smartmontools.Raw{
						Value:  39393440081287,
						String: "39393440081287",
					},
				},
			},
		},
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact("/dev/sda"))).ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "/dev/sda", nil
	})

	status, err := suite.service.GetSmartStatus(context.Background(), "sda")

	suite.Require().NoError(err)
	suite.Require().NotNil(status)
	suite.Equal(51, status.Temperature.Value, "temperature must keep the converter's top-level °C value, not the attr health score or the packed raw string")
	suite.Equal(42374, status.PowerOnHours.Value, "packed raw string must not override smartctl power_on_time.hours")
}

func (suite *SmartServiceSuite) TestGetSmartStatusDecodesPackedPowerOnTime() {
	// Regression test for #907 (lib/direct backend): the lib backend emits
	// power_on_time.hours as a 64-bit packed value where the low 32 bits hold
	// the hours and the high 32 bits carry a sub-hour counter. Observed live:
	// 0x9b8a0000a587 → 42375h. The packed value must be decoded, never shown.
	mockSMARTInfo := &smartmontools.SMARTInfo{
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		Temperature:     &smartmontools.Temperature{Current: 51},
		PowerOnTime:     &smartmontools.PowerOnTime{Hours: 0x9b8a0000a587}, // 42375 packed
		PowerCycleCount: 67,
		// Note: the lib JSON has no ata_smart_attributes table, so the ATA
		// branch in GetSmartStatus is a no-op and the converter value flows
		// straight into PowerOnHours.
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact("/dev/sda"))).ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "/dev/sda", nil
	})

	status, err := suite.service.GetSmartStatus(context.Background(), "sda")

	suite.Require().NoError(err)
	suite.Require().NotNil(status)
	suite.Equal(42375, status.PowerOnHours.Value, "packed 64-bit power_on_time.hours must be decoded to the low-32-bit hours")
	suite.Equal(51, status.Temperature.Value)
	suite.Equal(67, status.PowerCycleCount.Value)
}

func (suite *SmartServiceSuite) TestGetSmartStatusZeroesImplausiblePowerOnTime() {
	// A packed power_on_time.hours whose low 32 bits are not a plausible hour
	// count must be zeroed, never surfaced (lib backend edge case).
	mockSMARTInfo := &smartmontools.SMARTInfo{
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		PowerOnTime: &smartmontools.PowerOnTime{Hours: 0x123400000000}, // low 32 bits = 0
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact("/dev/sda"))).ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "/dev/sda", nil
	})

	status, err := suite.service.GetSmartStatus(context.Background(), "sda")

	suite.Require().NoError(err)
	suite.Require().NotNil(status)
	suite.Equal(0, status.PowerOnHours.Value, "implausible packed power_on_time.hours must be zeroed, not exposed")
}

func (suite *SmartServiceSuite) TestGetSmartStatusTemperatureFallsBackToRawString() {
	// When the top-level smartctl `temperature` object is missing, the
	// temperature must be parsed from the leading integer of the raw string
	// ("51 (Min/Max -22/57)") instead of the normalized attr health score.
	mockSMARTInfo := &smartmontools.SMARTInfo{
		SmartSupport: &smartmontools.SmartSupport{
			Available: true,
			Enabled:   true,
		},
		// No Temperature object: converter leaves Temperature.Value at 0.
		AtaSmartData: &smartmontools.AtaSmartData{
			Table: []smartmontools.SmartAttribute{
				{
					ID:    194, // Temperature_Celsius
					Name:  "Temperature_Celsius",
					Value: 100, // normalized health score, NOT the temperature
					Worst: 57,
					Raw: smartmontools.Raw{
						Value:  1005026082867,
						String: "51 (Min/Max -22/57)",
					},
				},
			},
		},
	}

	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact("/dev/sda"))).ThenReturn(mockSMARTInfo, nil)

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return "/dev/sda", nil
	})

	status, err := suite.service.GetSmartStatus(context.Background(), "sda")

	suite.Require().NoError(err)
	suite.Require().NotNil(status)
	suite.Equal(51, status.Temperature.Value, "temperature must fall back to the raw-string °C when the top-level temperature is missing")
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

func (suite *SmartServiceSuite) TestDisableSMARTSuccessWhenInfoReadFailsAfterDisable() {
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	mock.When(suite.smartClient.DisableSMART(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil)
	mock.When(suite.smartClient.IsSMARTSupported(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(&smartmontools.SmartSupport{Available: true, Enabled: false}, nil)
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil, fmt.Errorf("SMART is disabled"))

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	var captured events.SmartEvent
	var emitted bool
	unsubscribe := suite.eventBus.OnSmart(func(_ context.Context, se events.SmartEvent) goerrors.E {
		captured = se
		emitted = true
		return nil
	})
	defer unsubscribe()

	err := suite.service.DisableSMART(suite.T().Context(), tempFile.Name())

	suite.NoError(err)
	suite.True(emitted, "expected a minimal disabled-state SmartEvent")
	suite.Equal(tempFile.Name(), captured.SmartInfo.DiskId)
	suite.False(captured.SmartInfo.Enabled)
	suite.True(captured.SmartInfo.Supported)
}

func (suite *SmartServiceSuite) TestDisableSMARTPreservesModelDetailsWhenInfoReadFailsAfterDisable() {
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})

	fullInfo := &smartmontools.SMARTInfo{
		ModelFamily:  "TestFamily",
		ModelName:    "TestModel",
		SerialNumber: "SN123",
		Firmware:     "FW1",
		SmartSupport: &smartmontools.SmartSupport{Available: true, Enabled: true},
	}
	calls := 0
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenAnswer(func(args []any) []any {
			calls++
			if calls == 1 {
				return []any{fullInfo, nil}
			}
			return []any{nil, fmt.Errorf("SMART is disabled")}
		})

	_, err := suite.service.GetSmartInfo(suite.T().Context(), tempFile.Name())
	suite.Require().NoError(err)

	mock.When(suite.smartClient.DisableSMART(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil)
	mock.When(suite.smartClient.IsSMARTSupported(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(&smartmontools.SmartSupport{Available: true, Enabled: false}, nil)

	var captured events.SmartEvent
	var emitted bool
	unsubscribe := suite.eventBus.OnSmart(func(_ context.Context, se events.SmartEvent) goerrors.E {
		captured = se
		emitted = true
		return nil
	})
	defer unsubscribe()

	err = suite.service.DisableSMART(suite.T().Context(), tempFile.Name())

	suite.NoError(err)
	suite.True(emitted, "expected a disabled-state SmartEvent")
	suite.Equal(tempFile.Name(), captured.SmartInfo.DiskId)
	suite.False(captured.SmartInfo.Enabled)
	suite.True(captured.SmartInfo.Supported)
	suite.Equal("TestFamily", captured.SmartInfo.ModelFamily)
	suite.Equal("TestModel", captured.SmartInfo.ModelName)
	suite.Equal("SN123", captured.SmartInfo.SerialNumber)
	suite.Equal("FW1", captured.SmartInfo.Firmware)
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

// TestStartSelfTest_CompletionEventEmitted verifies that StartSelfTest drives
// completion purely from SDK progress callbacks: the initial callback emits
// Running=true and the final callback emits Running=false with the canonical
// deviceId, so the frontend receives a definitive completion signal (#1196).
func (suite *SmartServiceSuite) TestStartSelfTest_CompletionEventEmitted() {
	canonicalID := "ata-TEST_DEVICE_COMPLETION"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	mock.When(suite.smartClient.RunSelfTestWithProgress(
		mock.Any[context.Context](),
		mock.Exact(tempFile.Name()),
		mock.Exact("short"),
		mock.Any[smartmontools.ProgressCallback](),
	)).ThenAnswer(matchers.Answer(func(args []any) []any {
		cb := args[3].(smartmontools.ProgressCallback)
		cb(0, "Test started")
		cb(100, "completed without error")
		return []any{nil}
	}))

	var capturedEvents []events.SmartEvent
	unsubscribe := suite.eventBus.OnSmart(func(_ context.Context, se events.SmartEvent) goerrors.E {
		capturedEvents = append(capturedEvents, se)
		return nil
	})
	defer unsubscribe()

	err := suite.service.StartSelfTest(context.Background(), canonicalID, dto.SmartTestTypes.SMARTTESTTYPESHORT)

	suite.NoError(err)
	suite.Require().NotEmpty(capturedEvents, "StartSelfTest must emit SmartEvents from progress callbacks")

	// The first event must report the test as running
	suite.True(capturedEvents[0].SmartTestStatus.Running,
		"first SmartEvent from StartSelfTest must have Running=true")
	suite.Equal(canonicalID, capturedEvents[0].SmartTestStatus.DiskId,
		"progress event must carry the canonical deviceId, not the raw device path")

	// The last event must be the completion event
	last := capturedEvents[len(capturedEvents)-1]
	suite.False(last.SmartTestStatus.Running,
		"last SmartEvent from StartSelfTest must have Running=false")
	suite.Equal(canonicalID, last.SmartTestStatus.DiskId,
		"completion event must carry the canonical deviceId, not the raw device path")
	suite.Equal(100, last.SmartTestStatus.PercentComplete,
		"completion event must have PercentComplete=100")
}

// TestStartSelfTest_NoPrematureCompletionEvent is a regression test for #1196:
// StartSelfTest must not emit a Running=false completion event synchronously.
// When the SDK invokes no progress callbacks, no SmartEvent may be emitted.
func (suite *SmartServiceSuite) TestStartSelfTest_NoPrematureCompletionEvent() {
	canonicalID := "ata-TEST_DEVICE_NO_PREMATURE"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	mock.When(suite.smartClient.RunSelfTestWithProgress(
		mock.Any[context.Context](),
		mock.Exact(tempFile.Name()),
		mock.Exact("short"),
		mock.Any[smartmontools.ProgressCallback](),
	)).ThenReturn(nil)

	var capturedEvents []events.SmartEvent
	unsubscribe := suite.eventBus.OnSmart(func(_ context.Context, se events.SmartEvent) goerrors.E {
		capturedEvents = append(capturedEvents, se)
		return nil
	})
	defer unsubscribe()

	err := suite.service.StartSelfTest(context.Background(), canonicalID, dto.SmartTestTypes.SMARTTESTTYPESHORT)

	suite.NoError(err)
	suite.Empty(capturedEvents,
		"StartSelfTest must not emit a completion event before any progress callback fires")
}

// TestStartSelfTest_AlreadyInProgress is a regression test for #1196: starting
// a second test while the tracker reports a running test must fail with
// ErrorSMARTTestInProgress (mapped to 422 by the API layer) without touching
// the drive again.
func (suite *SmartServiceSuite) TestStartSelfTest_AlreadyInProgress() {
	canonicalID := "ata-TEST_DEVICE_IN_PROGRESS"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	mock.When(suite.smartClient.RunSelfTestWithProgress(
		mock.Any[context.Context](),
		mock.Exact(tempFile.Name()),
		mock.Exact("short"),
		mock.Any[smartmontools.ProgressCallback](),
	)).ThenAnswer(matchers.Answer(func(args []any) []any {
		cb := args[3].(smartmontools.ProgressCallback)
		cb(0, "Test started")
		return []any{nil}
	}))

	suite.Require().NoError(suite.service.StartSelfTest(context.Background(), canonicalID, dto.SmartTestTypes.SMARTTESTTYPESHORT))

	err := suite.service.StartSelfTest(context.Background(), canonicalID, dto.SmartTestTypes.SMARTTESTTYPESHORT)

	suite.Error(err)
	suite.True(goerrors.Is(err, dto.ErrorSMARTTestInProgress),
		"second StartSelfTest while a test runs must return ErrorSMARTTestInProgress, got %v", err)
	mock.Verify(suite.smartClient, matchers.Times(1)).RunSelfTestWithProgress(
		mock.Any[context.Context](),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[smartmontools.ProgressCallback](),
	)
}

// TestGetTestStatus_TrackerFallbackWhenDriveBlind is a regression test for
// #1196: on backends that omit self_test.status (lib/direct), GetTestStatus
// must report the running test tracked from StartSelfTest callbacks instead
// of unknown/unknown.
func (suite *SmartServiceSuite) TestGetTestStatus_TrackerFallbackWhenDriveBlind() {
	canonicalID := "ata-TEST_DEVICE_TRACKER_FALLBACK"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	mock.When(suite.smartClient.RunSelfTestWithProgress(
		mock.Any[context.Context](),
		mock.Exact(tempFile.Name()),
		mock.Exact("short"),
		mock.Any[smartmontools.ProgressCallback](),
	)).ThenAnswer(matchers.Answer(func(args []any) []any {
		cb := args[3].(smartmontools.ProgressCallback)
		cb(0, "Test started")
		return []any{nil}
	}))
	// Lib/direct backend shape: SelfTest present (capabilities) but Status nil.
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			AtaSmartData: &smartmontools.AtaSmartData{
				SelfTest: &smartmontools.SelfTest{
					PollingMinutes: &smartmontools.PollingMinutes{Short: 1, Extended: 48, Conveyance: 2},
				},
			},
		}, nil)

	suite.Require().NoError(suite.service.StartSelfTest(context.Background(), canonicalID, dto.SmartTestTypes.SMARTTESTTYPESHORT))

	status, err := suite.service.GetTestStatus(context.Background(), canonicalID)

	suite.NoError(err)
	suite.Require().NotNil(status)
	suite.True(status.Running, "GetTestStatus must report Running=true from the tracker while the test runs")
	suite.Equal("short", status.TestType)
	suite.Equal(canonicalID, status.DiskId)
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

// TestAbortSelfTest_MarksTrackerAborted is a regression test for #1196: after
// aborting a tracked running test, GetTestStatus must report Running=false
// (instead of falling back to unknown) while the backend omits drive status.
func (suite *SmartServiceSuite) TestAbortSelfTest_MarksTrackerAborted() {
	canonicalID := "ata-TEST_DEVICE_ABORT_TRACKER"
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.service.MockDeviceToDevice(func(deviceId string) (string, error) {
		return tempFile.Name(), nil
	})
	mock.When(suite.smartClient.RunSelfTestWithProgress(
		mock.Any[context.Context](),
		mock.Exact(tempFile.Name()),
		mock.Exact("short"),
		mock.Any[smartmontools.ProgressCallback](),
	)).ThenAnswer(matchers.Answer(func(args []any) []any {
		cb := args[3].(smartmontools.ProgressCallback)
		cb(0, "Test started")
		return []any{nil}
	}))
	mock.When(suite.smartClient.AbortSelfTest(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).ThenReturn(nil)
	mock.When(suite.smartClient.GetSMARTInfo(mock.Any[context.Context](), mock.Exact(tempFile.Name()))).
		ThenReturn(&smartmontools.SMARTInfo{
			AtaSmartData: &smartmontools.AtaSmartData{
				SelfTest: &smartmontools.SelfTest{},
			},
		}, nil)

	suite.Require().NoError(suite.service.StartSelfTest(context.Background(), canonicalID, dto.SmartTestTypes.SMARTTESTTYPESHORT))
	suite.Require().NoError(suite.service.AbortSelfTest(context.Background(), canonicalID))

	status, err := suite.service.GetTestStatus(context.Background(), canonicalID)

	suite.NoError(err)
	suite.Require().NotNil(status)
	suite.False(status.Running, "aborted test must report Running=false")
	suite.Contains(status.Status, "abort")
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
