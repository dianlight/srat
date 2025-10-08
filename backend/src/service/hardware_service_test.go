package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type HardwareServiceSuite struct {
	suite.Suite
	hardwareService service.HardwareServiceInterface
	haClient        hardware.ClientWithResponsesInterface
	smartService    service.SmartServiceInterface
	app             *fxtest.App
}

func TestHardwareServiceSuite(t *testing.T) {
	suite.Run(t, new(HardwareServiceSuite))
}

func (suite *HardwareServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			service.NewHardwareService,
			mock.Mock[hardware.ClientWithResponsesInterface],
			mock.Mock[service.SmartServiceInterface],
		),
		fx.Populate(&suite.hardwareService),
		fx.Populate(&suite.haClient),
		fx.Populate(&suite.smartService),
	)
	suite.app.RequireStart()
}

func (suite *HardwareServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_Success() {
	// Setup mock response
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id: pointer.String("drive1"),
						Filesystems: &[]hardware.Filesystem{
							{
								Device: pointer.String("/dev/sda1"),
								Name:   pointer.String("filesystem1"),
							},
						},
					},
				},
				Devices: &[]hardware.Device{},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	mock.Verify(suite.haClient, matchers.Times(1)).GetHardwareInfoWithResponse(mock.Any[context.Context]())
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_EmptyDrives() {
	// Setup mock response with no drives
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives:  &[]hardware.Drive{},
				Devices: &[]hardware.Device{},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Empty(disks)
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_ErrorResponse() {
	// Setup mock response with error
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
		Body:         []byte(`{"error":"internal server error"}`),
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.Error(err)
	suite.Nil(disks)
}

func (suite *HardwareServiceSuite) TestInvalidateHardwareInfo() {
	// Setup: First call to populate cache
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives:  &[]hardware.Drive{},
				Devices: &[]hardware.Device{},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// First call - should hit the API
	_, err := suite.hardwareService.GetHardwareInfo()
	suite.NoError(err)

	// Invalidate cache
	suite.hardwareService.InvalidateHardwareInfo()

	// Second call - should hit the API again after cache invalidation
	_, err = suite.hardwareService.GetHardwareInfo()
	suite.NoError(err)

	// Verify API was called twice (not cached after invalidation)
	mock.Verify(suite.haClient, matchers.Times(2)).GetHardwareInfoWithResponse(mock.Any[context.Context]())
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_ClientError() {
	// Setup mock with an error from the client
	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(nil, context.DeadlineExceeded)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.Error(err)
	suite.Nil(disks)
}

func (suite *HardwareServiceSuite) TestGetHardwareInfo_SkipsDrivesWithoutFilesystems() {
	// Setup mock response with drives that have no filesystems
	mockResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"drives":[]}}`),
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{
					{
						Id:          pointer.String("drive1"),
						Filesystems: nil, // No filesystems
					},
					{
						Id:          pointer.String("drive2"),
						Filesystems: &[]hardware.Filesystem{}, // Empty filesystems
					},
				},
				Devices: &[]hardware.Device{},
			},
		},
	}

	mock.When(suite.haClient.GetHardwareInfoWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	disks, err := suite.hardwareService.GetHardwareInfo()

	// Assert
	suite.NoError(err)
	suite.NotNil(disks)
	suite.Empty(disks) // Should skip drives without filesystems
}
