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

func (suite *SupervisorServiceSuite) TestNetworkMountShare_CreateSuccess() {
	// Setup mock responses
	getMountsResponse := &mount.GetMountsResponse{
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
				Mounts: &[]mount.Mount{},
			},
		},
	}

	createMountResponse := &mount.CreateMountResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok"}`),
	}

	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(getMountsResponse, nil)
	mock.When(suite.mountClient.CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())).ThenReturn(createMountResponse, nil)
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("test-password", nil)

	// Execute
	testShare := dto.SharedResource{
		Name:  "test-share",
		Usage: "media",
	}
	err := suite.supervisorService.NetworkMountShare(testShare)

	// Assert
	suite.NoError(err)
	mock.Verify(suite.mountClient, matchers.Times(1)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
}

func (suite *SupervisorServiceSuite) TestNetworkMountShare_Create400WithRetrySuccess() {
	// Setup mock responses - first create fails with 400, then remove succeeds, then create succeeds
	getMountsResponse := &mount.GetMountsResponse{
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
				Mounts: &[]mount.Mount{},
			},
		},
	}

	createMountResponse400 := &mount.CreateMountResponse{
		HTTPResponse: &http.Response{StatusCode: 400},
		Body:         []byte(`{"result":"error","message":"Could not mount bind_test-share due to: Unit mnt-data-supervisor-media-test-share.mount was already loaded or has a fragment file."}`),
	}

	removeMountResponse := &mount.RemoveMountResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok"}`),
	}

	createMountResponse200 := &mount.CreateMountResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok"}`),
	}

	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(getMountsResponse, nil)
	// First create call returns 400
	mock.When(suite.mountClient.CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())).
		ThenReturn(createMountResponse400, nil).
		ThenReturn(createMountResponse200, nil) // Second call succeeds
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(removeMountResponse, nil)
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("test-password", nil)

	// Execute
	testShare := dto.SharedResource{
		Name:  "test-share",
		Usage: "media",
	}
	err := suite.supervisorService.NetworkMountShare(testShare)

	// Assert
	suite.NoError(err)
	mock.Verify(suite.mountClient, matchers.Times(2)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}

func (suite *SupervisorServiceSuite) TestNetworkMountShare_Create400WithRetryFail() {
	// Setup mock responses - create fails with 400, remove succeeds, but retry also fails
	getMountsResponse := &mount.GetMountsResponse{
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
				Mounts: &[]mount.Mount{},
			},
		},
	}

	createMountResponse400 := &mount.CreateMountResponse{
		HTTPResponse: &http.Response{StatusCode: 400},
		Body:         []byte(`{"result":"error","message":"Could not mount"}`),
	}

	removeMountResponse := &mount.RemoveMountResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok"}`),
	}

	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(getMountsResponse, nil)
	mock.When(suite.mountClient.CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())).ThenReturn(createMountResponse400, nil)
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(removeMountResponse, nil)
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("test-password", nil)

	// Execute
	testShare := dto.SharedResource{
		Name:  "test-share",
		Usage: "media",
	}
	err := suite.supervisorService.NetworkMountShare(testShare)

	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "after removing stale mount")
	mock.Verify(suite.mountClient, matchers.Times(2)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}

func (suite *SupervisorServiceSuite) TestNetworkMountShare_Create400WithRemoveFail() {
	// Setup mock responses - create fails with 400, remove also fails
	getMountsResponse := &mount.GetMountsResponse{
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
				Mounts: &[]mount.Mount{},
			},
		},
	}

	createMountResponse400 := &mount.CreateMountResponse{
		HTTPResponse: &http.Response{StatusCode: 400},
		Body:         []byte(`{"result":"error","message":"Could not mount"}`),
	}

	removeMountResponse500 := &mount.RemoveMountResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
		Body:         []byte(`{"result":"error"}`),
	}

	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(getMountsResponse, nil)
	mock.When(suite.mountClient.CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())).ThenReturn(createMountResponse400, nil)
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(removeMountResponse500, nil)
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("test-password", nil)

	// Execute
	testShare := dto.SharedResource{
		Name:  "test-share",
		Usage: "media",
	}
	err := suite.supervisorService.NetworkMountShare(testShare)

	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "Error creating mount")
	// Only one create attempt since remove failed
	mock.Verify(suite.mountClient, matchers.Times(1)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}
