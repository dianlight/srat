package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/dianlight/smartmontools-go"
	"github.com/dianlight/srat/dto"
	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/suite"
	goerrors "gitlab.com/tozd/go/errors"
)

// mockSmartClient implements smartmontools.SmartClient for testing
type mockSmartClient struct {
	getSMARTInfoFunc          func(devicePath string) (*smartmontools.SMARTInfo, error)
	checkHealthFunc           func(devicePath string) (bool, error)
	runSelfTestFunc           func(devicePath string, testType string) error
	abortSelfTestFunc         func(devicePath string) error
	enableSMARTFunc           func(devicePath string) error
	disableSMARTFunc          func(devicePath string) error
	isSMARTSupportedFunc      func(devicePath string) (*smartmontools.SmartSupport, error)
	scanDevicesFunc           func() ([]smartmontools.Device, error)
	getDeviceInfoFunc         func(devicePath string) (map[string]any, error)
	getAvailableSelfTestsFunc func(devicePath string) (*smartmontools.SelfTestInfo, error)
	runSelfTestWithProgress   func(devicePath string, testType string, cb smartmontools.ProgressCallback) error
}

// All interface methods accept a context. Tests ignore the context value.
func (m *mockSmartClient) GetSMARTInfo(_ context.Context, devicePath string) (*smartmontools.SMARTInfo, error) {
	if m.getSMARTInfoFunc != nil {
		return m.getSMARTInfoFunc(devicePath)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) CheckHealth(_ context.Context, devicePath string) (bool, error) {
	if m.checkHealthFunc != nil {
		return m.checkHealthFunc(devicePath)
	}
	return false, errors.New("not implemented")
}

func (m *mockSmartClient) RunSelfTest(_ context.Context, devicePath string, testType string) error {
	if m.runSelfTestFunc != nil {
		return m.runSelfTestFunc(devicePath, testType)
	}
	return errors.New("not implemented")
}

func (m *mockSmartClient) AbortSelfTest(_ context.Context, devicePath string) error {
	if m.abortSelfTestFunc != nil {
		return m.abortSelfTestFunc(devicePath)
	}
	return errors.New("not implemented")
}

func (m *mockSmartClient) EnableSMART(_ context.Context, devicePath string) error {
	if m.enableSMARTFunc != nil {
		return m.enableSMARTFunc(devicePath)
	}
	return errors.New("not implemented")
}

func (m *mockSmartClient) DisableSMART(_ context.Context, devicePath string) error {
	if m.disableSMARTFunc != nil {
		return m.disableSMARTFunc(devicePath)
	}
	return errors.New("not implemented")
}

func (m *mockSmartClient) IsSMARTSupported(_ context.Context, devicePath string) (*smartmontools.SmartSupport, error) {
	if m.isSMARTSupportedFunc != nil {
		return m.isSMARTSupportedFunc(devicePath)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) ScanDevices(_ context.Context) ([]smartmontools.Device, error) {
	if m.scanDevicesFunc != nil {
		return m.scanDevicesFunc()
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) GetDeviceInfo(_ context.Context, devicePath string) (map[string]any, error) {
	if m.getDeviceInfoFunc != nil {
		return m.getDeviceInfoFunc(devicePath)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) GetAvailableSelfTests(_ context.Context, devicePath string) (*smartmontools.SelfTestInfo, error) {
	if m.getAvailableSelfTestsFunc != nil {
		return m.getAvailableSelfTestsFunc(devicePath)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) RunSelfTestWithProgress(_ context.Context, devicePath string, testType string, cb smartmontools.ProgressCallback) error {
	if m.runSelfTestWithProgress != nil {
		return m.runSelfTestWithProgress(devicePath, testType, cb)
	}
	return errors.New("not implemented")
}

type SmartServiceSuite struct {
	suite.Suite
	service    SmartServiceInterface
	mockClient *mockSmartClient
}

func (suite *SmartServiceSuite) SetupTest() {
	suite.mockClient = &mockSmartClient{}
	suite.service = NewSmartServiceWithClient(suite.mockClient)
}

func (suite *SmartServiceSuite) TearDownTest() {
}

func (suite *SmartServiceSuite) TestGetSmartInfoCacheHit() {
	// Setup: Manually set cache
	expectedInfo := &dto.SmartInfo{DiskType: "SATA"}
	cacheKey := smartCacheKeyPrefix + "/dev/sda" + "_info"
	suite.service.(*smartService).cache.Set(cacheKey, expectedInfo, gocache.DefaultExpiration)

	// Execute
	info, err := suite.service.GetSmartInfo(context.Background(), "/dev/sda")

	// Assert
	suite.NoError(err)
	suite.Equal(expectedInfo, info)
}

func (suite *SmartServiceSuite) TestGetSmartInfoDeviceNotExist() {
	// Execute with invalid path
	info, err := suite.service.GetSmartInfo(context.Background(), "/dev/nonexistent")

	// Assert
	suite.Error(err)
	suite.Nil(info)
	suite.True(goerrors.Is(err, dto.ErrorSMARTNotSupported))
	// Verify details
	details := goerrors.Details(err)
	suite.Equal("/dev/nonexistent", details["device"])
	suite.Equal("does not exist", details["reason"])
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

	suite.mockClient.getSMARTInfoFunc = func(devicePath string) (*smartmontools.SMARTInfo, error) {
		if devicePath == tempFile.Name() {
			return mockSMARTInfo, nil
		}
		return nil, errors.New("device not found")
	}

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

	suite.mockClient.getSMARTInfoFunc = func(devicePath string) (*smartmontools.SMARTInfo, error) {
		if devicePath == tempFile.Name() {
			return mockSMARTInfo, nil
		}
		return nil, errors.New("device not found")
	}

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

	suite.mockClient.getSMARTInfoFunc = func(devicePath string) (*smartmontools.SMARTInfo, error) {
		if devicePath == tempFile.Name() {
			return mockSMARTInfo, nil
		}
		return nil, errors.New("device not found")
	}

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
	os.Chmod(tempFile.Name(), 0000)
	defer os.Chmod(tempFile.Name(), 0644) // Restore for cleanup

	// Execute
	info, err := suite.service.GetSmartInfo(context.Background(), tempFile.Name())

	// Assert
	suite.Error(err)
	suite.Nil(info)
	suite.True(goerrors.Is(err, dto.ErrorSMARTNotSupported))
	details := goerrors.Details(err)
	suite.NotNil(details, "Error should have details")
	reasonVal, ok := details["reason"]
	suite.True(ok, "Error details should contain 'reason' key")
	reason, ok := reasonVal.(string)
	suite.True(ok, "Error reason should be a string")
	suite.Contains(reason, "not readable",
		"Expected reason to contain 'not readable', got: %s", reason)
}

func TestSmartServiceSuite(t *testing.T) {
	suite.Run(t, new(SmartServiceSuite))
}

func (suite *SmartServiceSuite) TestGetHealthStatusDeviceNotExist() {
	// Execute with non-existent device
	health, err := suite.service.GetHealthStatus(context.Background(), "/dev/nonexistent")

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

	suite.mockClient.getSMARTInfoFunc = func(devicePath string) (*smartmontools.SMARTInfo, error) {
		if devicePath == tempFile.Name() {
			return mockSMARTInfo, nil
		}
		return nil, errors.New("device not found")
	}

	suite.mockClient.checkHealthFunc = func(devicePath string) (bool, error) {
		if devicePath == tempFile.Name() {
			return true, nil
		}
		return false, errors.New("device not found")
	}

	// Execute
	health, err := suite.service.GetHealthStatus(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
	suite.NotNil(health)
	suite.True(health.Passed)
	suite.Equal("healthy", health.OverallStatus)
}

func (suite *SmartServiceSuite) TestStartSelfTestInvalidType() {
	err := suite.service.StartSelfTest(context.Background(), "/dev/sda", dto.SmartTestType("invalid"))

	suite.Error(err)
	suite.True(goerrors.Is(err, dto.ErrorInvalidParameter))
}

func (suite *SmartServiceSuite) TestStartSelfTestDeviceNotExist() {
	err := suite.service.StartSelfTest(context.Background(), "/dev/nonexistent", dto.SmartTestTypeShort)

	suite.Error(err)
}

func (suite *SmartServiceSuite) TestStartSelfTestSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.mockClient.runSelfTestFunc = func(devicePath string, testType string) error {
		if devicePath == tempFile.Name() && testType == "short" {
			return nil
		}
		return errors.New("unexpected parameters")
	}

	// Execute
	err := suite.service.StartSelfTest(context.Background(), tempFile.Name(), dto.SmartTestTypeShort)

	// Assert
	suite.NoError(err)
}

func (suite *SmartServiceSuite) TestEnableDisableSMARTDeviceNotExist() {
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

	suite.mockClient.enableSMARTFunc = func(devicePath string) error {
		if devicePath == tempFile.Name() {
			return nil
		}
		return errors.New("unexpected device")
	}

	suite.mockClient.isSMARTSupportedFunc = func(devicePath string) (*smartmontools.SmartSupport, error) {
		if devicePath == tempFile.Name() {
			return &smartmontools.SmartSupport{
				Available: true,
				Enabled:   true,
			}, nil
		}
		return nil, errors.New("unexpected device")
	}

	// Execute
	err := suite.service.EnableSMART(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
}

func (suite *SmartServiceSuite) TestDisableSMARTSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.mockClient.disableSMARTFunc = func(devicePath string) error {
		if devicePath == tempFile.Name() {
			return nil
		}
		return errors.New("unexpected device")
	}

	suite.mockClient.isSMARTSupportedFunc = func(devicePath string) (*smartmontools.SmartSupport, error) {
		if devicePath == tempFile.Name() {
			return &smartmontools.SmartSupport{
				Available: true,
				Enabled:   false,
			}, nil
		}
		return nil, errors.New("unexpected device")
	}

	// Execute
	err := suite.service.DisableSMART(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
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

	suite.mockClient.getSMARTInfoFunc = func(devicePath string) (*smartmontools.SMARTInfo, error) {
		if devicePath == tempFile.Name() {
			return mockSMARTInfo, nil
		}
		return nil, errors.New("device not found")
	}

	// Execute
	status, err := suite.service.GetTestStatus(context.Background(), tempFile.Name())

	// Assert
	suite.NoError(err)
	suite.NotNil(status)
	suite.Equal("short test completed without error", status.Status)
	suite.Equal("short", status.TestType)
}

func (suite *SmartServiceSuite) TestAbortSelfTestDeviceNotExist() {
	err := suite.service.AbortSelfTest(context.Background(), "/dev/nonexistent")

	suite.Error(err)
}

func (suite *SmartServiceSuite) TestAbortSelfTestSuccess() {
	// Create a temporary file
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())

	suite.mockClient.abortSelfTestFunc = func(devicePath string) error {
		if devicePath == tempFile.Name() {
			return nil
		}
		return errors.New("unexpected device")
	}

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
