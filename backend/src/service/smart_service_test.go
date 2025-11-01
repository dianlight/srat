package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/dianlight/smartmontools-go"
	"github.com/dianlight/srat/dto"
	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/suite"
	goerrors "gitlab.com/tozd/go/errors"
)

// mockSmartClient implements smartmontools.SmartClient for testing
type mockSmartClient struct {
	getSMARTInfoFunc      func(devicePath string) (*smartmontools.SMARTInfo, error)
	checkHealthFunc       func(devicePath string) (bool, error)
	runSelfTestFunc       func(devicePath string, testType string) error
	abortSelfTestFunc     func(devicePath string) error
	enableSMARTFunc       func(devicePath string) error
	disableSMARTFunc      func(devicePath string) error
	isSMARTSupportedFunc  func(devicePath string) (*smartmontools.SMARTSupportInfo, error)
	scanDevicesFunc       func() ([]smartmontools.Device, error)
	getDeviceInfoFunc     func(devicePath string) (map[string]interface{}, error)
	getAvailableSelfTestsFunc func(devicePath string) (*smartmontools.SelfTestInfo, error)
}

func (m *mockSmartClient) GetSMARTInfo(devicePath string) (*smartmontools.SMARTInfo, error) {
	if m.getSMARTInfoFunc != nil {
		return m.getSMARTInfoFunc(devicePath)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) CheckHealth(devicePath string) (bool, error) {
	if m.checkHealthFunc != nil {
		return m.checkHealthFunc(devicePath)
	}
	return false, errors.New("not implemented")
}

func (m *mockSmartClient) RunSelfTest(devicePath string, testType string) error {
	if m.runSelfTestFunc != nil {
		return m.runSelfTestFunc(devicePath, testType)
	}
	return errors.New("not implemented")
}

func (m *mockSmartClient) AbortSelfTest(devicePath string) error {
	if m.abortSelfTestFunc != nil {
		return m.abortSelfTestFunc(devicePath)
	}
	return errors.New("not implemented")
}

func (m *mockSmartClient) EnableSMART(devicePath string) error {
	if m.enableSMARTFunc != nil {
		return m.enableSMARTFunc(devicePath)
	}
	return errors.New("not implemented")
}

func (m *mockSmartClient) DisableSMART(devicePath string) error {
	if m.disableSMARTFunc != nil {
		return m.disableSMARTFunc(devicePath)
	}
	return errors.New("not implemented")
}

func (m *mockSmartClient) IsSMARTSupported(devicePath string) (*smartmontools.SMARTSupportInfo, error) {
	if m.isSMARTSupportedFunc != nil {
		return m.isSMARTSupportedFunc(devicePath)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) ScanDevices() ([]smartmontools.Device, error) {
	if m.scanDevicesFunc != nil {
		return m.scanDevicesFunc()
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) GetDeviceInfo(devicePath string) (map[string]interface{}, error) {
	if m.getDeviceInfoFunc != nil {
		return m.getDeviceInfoFunc(devicePath)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) GetAvailableSelfTests(devicePath string) (*smartmontools.SelfTestInfo, error) {
	if m.getAvailableSelfTestsFunc != nil {
		return m.getAvailableSelfTestsFunc(devicePath)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSmartClient) RunSelfTestWithProgress(ctx context.Context, devicePath string, testType string, callback smartmontools.ProgressCallback) error {
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
	cacheKey := smartCacheKeyPrefix + "/dev/sda"
	suite.service.(*smartService).cache.Set(cacheKey, expectedInfo, gocache.DefaultExpiration)

	// Execute
	info, err := suite.service.GetSmartInfo("/dev/sda")

	// Assert
	suite.NoError(err)
	suite.Equal(expectedInfo, info)
}

func (suite *SmartServiceSuite) TestGetSmartInfoDeviceNotExist() {
	// Execute with invalid path
	info, err := suite.service.GetSmartInfo("/dev/nonexistent")

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
	info, err := suite.service.GetSmartInfo(tempFile.Name())
	
	// Assert
	suite.NoError(err)
	suite.NotNil(info)
	suite.Equal("SATA", info.DiskType)
	suite.True(info.Enabled)
	suite.Equal(35, info.Temperature.Value)
	suite.Equal(1000, info.PowerOnHours.Value)
	suite.Equal(50, info.PowerCycleCount.Value)
}

func (suite *SmartServiceSuite) TestGetSmartInfoDeviceNotReadable() {
	// Create a temp file and remove read permission
	tempFile, _ := os.CreateTemp("", "testdevice")
	defer os.Remove(tempFile.Name())
	os.Chmod(tempFile.Name(), 0000)
	defer os.Chmod(tempFile.Name(), 0644) // Restore for cleanup

	// Execute
	info, err := suite.service.GetSmartInfo(tempFile.Name())

	// Assert
	suite.Error(err)
	suite.Nil(info)
	suite.True(goerrors.Is(err, dto.ErrorSMARTNotSupported))
	details := goerrors.Details(err)
	reason := details["reason"].(string)
	suite.True(strings.Contains(reason, "not readable"),
		"Expected reason to contain 'not readable', got: %s", reason)
}

func TestSmartServiceSuite(t *testing.T) {
	suite.Run(t, new(SmartServiceSuite))
}

func (suite *SmartServiceSuite) TestGetHealthStatusDeviceNotExist() {
	// Execute with non-existent device
	health, err := suite.service.GetHealthStatus("/dev/nonexistent")

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
	health, err := suite.service.GetHealthStatus(tempFile.Name())
	
	// Assert
	suite.NoError(err)
	suite.NotNil(health)
	suite.True(health.Passed)
	suite.Equal("healthy", health.OverallStatus)
}

func (suite *SmartServiceSuite) TestStartSelfTestInvalidType() {
	err := suite.service.StartSelfTest("/dev/sda", dto.SmartTestType("invalid"))

	suite.Error(err)
	suite.True(goerrors.Is(err, dto.ErrorInvalidParameter))
}

func (suite *SmartServiceSuite) TestStartSelfTestDeviceNotExist() {
	err := suite.service.StartSelfTest("/dev/nonexistent", dto.SmartTestTypeShort)

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
	err := suite.service.StartSelfTest(tempFile.Name(), dto.SmartTestTypeShort)
	
	// Assert
	suite.NoError(err)
}

func (suite *SmartServiceSuite) TestEnableDisableSMARTDeviceNotExist() {
	// Test EnableSMART
	err := suite.service.EnableSMART("/dev/nonexistent")
	suite.Error(err)

	// Test DisableSMART
	err = suite.service.DisableSMART("/dev/nonexistent")
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
	
	suite.mockClient.isSMARTSupportedFunc = func(devicePath string) (*smartmontools.SMARTSupportInfo, error) {
		if devicePath == tempFile.Name() {
			return &smartmontools.SMARTSupportInfo{
				Supported: true,
				Enabled:   true,
			}, nil
		}
		return nil, errors.New("unexpected device")
	}
	
	// Execute
	err := suite.service.EnableSMART(tempFile.Name())
	
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
	
	suite.mockClient.isSMARTSupportedFunc = func(devicePath string) (*smartmontools.SMARTSupportInfo, error) {
		if devicePath == tempFile.Name() {
			return &smartmontools.SMARTSupportInfo{
				Supported: true,
				Enabled:   false,
			}, nil
		}
		return nil, errors.New("unexpected device")
	}
	
	// Execute
	err := suite.service.DisableSMART(tempFile.Name())
	
	// Assert
	suite.NoError(err)
}

func (suite *SmartServiceSuite) TestGetTestStatusDeviceNotExist() {
	status, err := suite.service.GetTestStatus("/dev/nonexistent")

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
				Status: "short test completed without error",
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
	status, err := suite.service.GetTestStatus(tempFile.Name())
	
	// Assert
	suite.NoError(err)
	suite.NotNil(status)
	suite.Equal("short test completed without error", status.Status)
	suite.Equal("short", status.TestType)
}

func (suite *SmartServiceSuite) TestAbortSelfTestDeviceNotExist() {
	err := suite.service.AbortSelfTest("/dev/nonexistent")

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
	err := suite.service.AbortSelfTest(tempFile.Name())
	
	// Assert
	suite.NoError(err)
}
