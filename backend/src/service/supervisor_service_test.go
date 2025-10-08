package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SupervisorServiceSuite struct {
	suite.Suite
	supervisorService service.SupervisorServiceInterface
	mountClient       mount.ClientWithResponsesInterface
	propertyRepo      repository.PropertyRepositoryInterface
	app               *fxtest.App
}

func TestSupervisorServiceSuite(t *testing.T) {
	suite.Run(t, new(SupervisorServiceSuite))
}

func (suite *SupervisorServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					HACoreReady:    true,
					SupervisorURL:  "http://supervisor",
					AddonIpAddress: "172.30.32.1",
				}
			},
			service.NewSupervisorService,
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
		),
		fx.Populate(&suite.supervisorService),
		fx.Populate(&suite.mountClient),
		fx.Populate(&suite.propertyRepo),
	)
	suite.app.RequireStart()
}

func (suite *SupervisorServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *SupervisorServiceSuite) TestNetworkGetAllMounted_HACoreNotReady() {
	// Create a new state with HACoreReady = false
	suite.app.RequireStop()
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					HACoreReady: false,
				}
			},
			service.NewSupervisorService,
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
		),
		fx.Populate(&suite.supervisorService),
	)
	suite.app.RequireStart()

	// Execute
	mounts, err := suite.supervisorService.NetworkGetAllMounted()

	// Assert
	suite.Error(err)
	suite.Nil(mounts)
	suite.Contains(err.Error(), "HA Core is not ready")
}

func (suite *SupervisorServiceSuite) TestNetworkGetAllMounted_Success() {
	// Setup mock response
	mockResponse := &mount.GetMountsResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"mounts":[]}}`),
		JSON200: &struct {
			Data *struct {
				DefaultBackupMount *string        `json:"default_backup_mount,omitempty"`
				Mounts             *[]mount.Mount `json:"mounts,omitempty"`
			} `json:"data,omitempty"`
			Result *mount.GetMounts200Result `json:"result,omitempty"`
		}{
			Data: &struct {
				DefaultBackupMount *string        `json:"default_backup_mount,omitempty"`
				Mounts             *[]mount.Mount `json:"mounts,omitempty"`
			}{
				Mounts: &[]mount.Mount{
					{
						Name:   pointer.String("share1"),
						Server: pointer.String("172.30.32.1"),
					},
					{
						Name:   pointer.String("share2"),
						Server: pointer.String("172.30.32.1"),
					},
					{
						Name:   pointer.String("other-share"),
						Server: pointer.String("192.168.1.100"), // Different server
					},
				},
			},
		},
	}

	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	mounts, err := suite.supervisorService.NetworkGetAllMounted()

	// Assert
	suite.NoError(err)
	suite.NotNil(mounts)
	suite.Len(mounts, 2) // Only shares from our addon IP
}

func (suite *SupervisorServiceSuite) TestNetworkUnmountShare_HACoreNotReady() {
	// Create a new state with HACoreReady = false
	suite.app.RequireStop()
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					HACoreReady: false,
				}
			},
			service.NewSupervisorService,
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
		),
		fx.Populate(&suite.supervisorService),
	)
	suite.app.RequireStart()

	// Execute
	err := suite.supervisorService.NetworkUnmountShare("test-share")

	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "HA Core is not ready")
}

func (suite *SupervisorServiceSuite) TestNetworkUnmountShare_Success() {
	// Setup mock response
	mockResponse := &mount.RemoveMountResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok"}`),
	}

	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(mockResponse, nil)

	// Execute
	err := suite.supervisorService.NetworkUnmountShare("test-share")

	// Assert
	suite.NoError(err)
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}

func (suite *SupervisorServiceSuite) TestNetworkUnmountShare_ErrorResponse() {
	// Setup mock response with error
	mockResponse := &mount.RemoveMountResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
		Body:         []byte(`{"error":"internal server error"}`),
	}

	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(mockResponse, nil)

	// Execute
	err := suite.supervisorService.NetworkUnmountShare("test-share")

	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "500")
}

func (suite *SupervisorServiceSuite) TestNetworkGetAllMounted_ErrorFromClient() {
	// Setup mock with client error
	mockResponse := &mount.GetMountsResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
		Body:         []byte(`{"error":"internal server error"}`),
	}

	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(mockResponse, nil)

	// Execute
	mounts, err := suite.supervisorService.NetworkGetAllMounted()

	// Assert
	suite.Error(err)
	suite.Nil(mounts)
}
