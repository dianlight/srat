package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
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
	shareService      service.ShareServiceInterface
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
			events.NewEventBus,
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
		),
		fx.Populate(&suite.supervisorService),
		fx.Populate(&suite.mountClient),
		fx.Populate(&suite.propertyRepo),
		fx.Populate(&suite.shareService),
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
			events.NewEventBus,
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
		),
		fx.Populate(&suite.supervisorService),
	)
	suite.app.RequireStart()

	// Execute
	mounts, err := suite.supervisorService.NetworkGetAllMounted(context.Background())

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
	mounts, err := suite.supervisorService.NetworkGetAllMounted(context.Background())

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
			events.NewEventBus,
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
		),
		fx.Populate(&suite.supervisorService),
	)
	suite.app.RequireStart()

	// Execute
	err := suite.supervisorService.NetworkUnmountShare(context.Background(), "test-share")

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

	mockGetResponse := &mount.GetMountsResponse{
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
						Name:   pointer.String("test-share"),
						Server: pointer.String("172.30.32.1"),
					},
				},
			},
		},
	}

	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(mockResponse, nil)
	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(mockGetResponse, nil)

	// Execute
	err := suite.supervisorService.NetworkUnmountShare(context.Background(), "test-share")

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

	mockGetResponse := &mount.GetMountsResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
		Body:         []byte(`{"error":"internal server error"}`),
	}

	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(mockResponse, nil)
	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(mockGetResponse, nil)

	// Execute
	err := suite.supervisorService.NetworkUnmountShare(context.Background(), "test-share")

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
	mounts, err := suite.supervisorService.NetworkGetAllMounted(context.Background())

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
	err := suite.supervisorService.NetworkMountShare(context.Background(), testShare)

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
	err := suite.supervisorService.NetworkMountShare(context.Background(), testShare)

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
	err := suite.supervisorService.NetworkMountShare(context.Background(), testShare)

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
	err := suite.supervisorService.NetworkMountShare(context.Background(), testShare)

	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "Error creating mount")
	// Only one create attempt since remove failed
	mock.Verify(suite.mountClient, matchers.Times(1)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}

// TestNetworkMountShare_Issue221_ExactScenario tests the exact scenario described in issue #221
// This test reproduces the specific error message: "Unit was already loaded or has a fragment file"
func (suite *SupervisorServiceSuite) TestNetworkMountShare_Issue221_ExactScenario() {
	// Setup mock responses - simulating the exact issue #221 scenario
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
				Mounts: &[]mount.Mount{}, // Mount doesn't appear in list but systemd unit exists
			},
		},
	}

	// Exact error message from issue #221
	createMountResponse400 := &mount.CreateMountResponse{
		HTTPResponse: &http.Response{StatusCode: 400},
		Body:         []byte(`{"result":"error","message":"Could not mount bind_backup-share due to: Unit mnt-data-supervisor-backup-backup-share.mount was already loaded or has a fragment file."}`),
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
	// First create call returns 400 with exact error from issue #221
	mock.When(suite.mountClient.CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())).
		ThenReturn(createMountResponse400, nil).
		ThenReturn(createMountResponse200, nil) // Second call succeeds
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(removeMountResponse, nil)
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("test-password", nil)

	// Execute with a backup share (as in the issue)
	testShare := dto.SharedResource{
		Name:  "backup-share",
		Usage: "backup",
	}
	err := suite.supervisorService.NetworkMountShare(context.Background(), testShare)

	// Assert - the fix should handle this gracefully
	suite.NoError(err, "Issue #221 fix should handle stale systemd units")
	mock.Verify(suite.mountClient, matchers.Times(2)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}

// TestNetworkMountShare_Update400_NoRetryLogic tests the scenario where an update fails with 400
// With the fix, this should now use retry logic to recover
func (suite *SupervisorServiceSuite) TestNetworkMountShare_Update400_NoRetryLogic() {
	// Setup mock responses - mount exists but update fails with 400
	getMountsResponse := &mount.GetMountsResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"mounts":[{"name":"test-share","server":"172.30.32.1","usage":"media","state":"inactive"}]}}`),
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
						Name:   pointer.String("test-share"),
						Server: pointer.String("172.30.32.1"),
						Usage:  pointer.Any(mount.MountUsage("media")).(*mount.MountUsage),
						State:  pointer.String("inactive"), // Inactive state triggers update
					},
				},
			},
		},
	}

	updateMountResponse400 := &mount.UpdateMountResponse{
		HTTPResponse: &http.Response{StatusCode: 400},
		Body:         []byte(`{"result":"error","message":"Could not update mount due to: Unit was already loaded or has a fragment file."}`),
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
	mock.When(suite.mountClient.UpdateMountWithResponse(mock.Any[context.Context](), mock.Any[string](), mock.Any[mount.Mount]())).ThenReturn(updateMountResponse400, nil)
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(removeMountResponse, nil)
	mock.When(suite.mountClient.CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())).ThenReturn(createMountResponse200, nil)
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("test-password", nil)

	// Execute
	testShare := dto.SharedResource{
		Name:  "test-share",
		Usage: "media",
	}
	err := suite.supervisorService.NetworkMountShare(context.Background(), testShare)

	// Assert - with retry logic, this should now succeed
	suite.NoError(err, "Update path now has retry logic for 400 errors")
	mock.Verify(suite.mountClient, matchers.Times(1)).UpdateMountWithResponse(mock.Any[context.Context](), mock.Any[string](), mock.Any[mount.Mount]())
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
	mock.Verify(suite.mountClient, matchers.Times(1)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
}

// TestNetworkMountShare_Update400_WithRetryLogic tests the retry logic for update operations
// This verifies the fix for the update path to handle stale systemd units (extension of issue #221)
func (suite *SupervisorServiceSuite) TestNetworkMountShare_Update400_WithRetryLogic() {
	// Setup mock responses - mount exists but update fails with 400, then succeeds after remove+recreate
	getMountsResponse := &mount.GetMountsResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"mounts":[{"name":"test-share","server":"172.30.32.1","usage":"share","state":"inactive"}]}}`),
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
						Name:   pointer.String("test-share"),
						Server: pointer.String("172.30.32.1"),
						Usage:  pointer.Any(mount.MountUsage("share")).(*mount.MountUsage),
						State:  pointer.String("inactive"), // Inactive state triggers update
					},
				},
			},
		},
	}

	updateMountResponse400 := &mount.UpdateMountResponse{
		HTTPResponse: &http.Response{StatusCode: 400},
		Body:         []byte(`{"result":"error","message":"Could not update mount due to: Unit was already loaded or has a fragment file."}`),
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
	mock.When(suite.mountClient.UpdateMountWithResponse(mock.Any[context.Context](), mock.Any[string](), mock.Any[mount.Mount]())).ThenReturn(updateMountResponse400, nil)
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(removeMountResponse, nil)
	mock.When(suite.mountClient.CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())).ThenReturn(createMountResponse200, nil)
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("test-password", nil)

	// Execute
	testShare := dto.SharedResource{
		Name:  "test-share",
		Usage: "media",
	}
	err := suite.supervisorService.NetworkMountShare(context.Background(), testShare)

	// Assert - with the fix, it should: attempt update -> get 400 -> remove -> create -> succeed
	suite.NoError(err)
	mock.Verify(suite.mountClient, matchers.Times(1)).UpdateMountWithResponse(mock.Any[context.Context](), mock.Any[string](), mock.Any[mount.Mount]())
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
	mock.Verify(suite.mountClient, matchers.Times(1)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
}

// NetworkUnmountAllShares should unmount all eligible shares (media/share/backup) that are currently mounted
func (suite *SupervisorServiceSuite) TestNetworkUnmountAllShares_Success() {
	// Shares configured in the system
	shares := []dto.SharedResource{
		{Name: "media1", Usage: "media", Status: &dto.SharedResourceStatus{IsValid: true}},
		{Name: "share2", Usage: "share", Status: &dto.SharedResourceStatus{IsValid: true}},
		{Name: "backup3", Usage: "backup", Status: &dto.SharedResourceStatus{IsValid: true}},
	}

	// All three are mounted in HA on our addon IP
	mountedResp := &mount.GetMountsResponse{
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
					{Name: pointer.String("media1"), Server: pointer.String("172.30.32.1")},
					{Name: pointer.String("share2"), Server: pointer.String("172.30.32.1")},
					{Name: pointer.String("backup3"), Server: pointer.String("172.30.32.1")},
				},
			},
		},
	}

	mock.When(suite.shareService.ListShares()).
		ThenReturn(shares, nil).
		ThenReturn([]dto.SharedResource{}, nil) // prevent OnStop double-unmount
	// GetMounts will be called once per NetworkUnmountShare invocation; returning same response is fine
	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(mountedResp, nil)
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(&mount.RemoveMountResponse{HTTPResponse: &http.Response{StatusCode: 200}, Body: []byte(`{"result":"ok"}`)}, nil)

	// Execute
	err := suite.supervisorService.NetworkUnmountAllShares(context.Background())

	// Assert
	suite.NoError(err)
	mock.Verify(suite.mountClient, matchers.Times(3)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}

// NetworkUnmountAllShares should skip disabled and invalid shares
func (suite *SupervisorServiceSuite) TestNetworkUnmountAllShares_SkipsDisabledAndInvalid() {
	disabled := pointer.Bool(true)
	shares := []dto.SharedResource{
		{Name: "disabled-media", Usage: "media", Disabled: disabled, Status: &dto.SharedResourceStatus{IsValid: true}},
		{Name: "invalid-share", Usage: "share", Status: &dto.SharedResourceStatus{IsValid: false}},
		{Name: "ok-backup", Usage: "backup", Status: &dto.SharedResourceStatus{IsValid: true}},
		{Name: "none-usage", Usage: "none", Status: &dto.SharedResourceStatus{IsValid: true}},
	}

	mountedResp := &mount.GetMountsResponse{
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
					{Name: pointer.String("ok-backup"), Server: pointer.String("172.30.32.1")},
					{Name: pointer.String("disabled-media"), Server: pointer.String("172.30.32.1")},
					{Name: pointer.String("invalid-share"), Server: pointer.String("172.30.32.1")},
				},
			},
		},
	}

	mock.When(suite.shareService.ListShares()).
		ThenReturn(shares, nil).
		ThenReturn([]dto.SharedResource{}, nil) // prevent OnStop double-unmount
	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(mountedResp, nil)
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(&mount.RemoveMountResponse{HTTPResponse: &http.Response{StatusCode: 200}, Body: []byte(`{"result":"ok"}`)}, nil)

	// Execute
	err := suite.supervisorService.NetworkUnmountAllShares(context.Background())

	// Assert: ok-backup and invalid-share (usage=share) should be unmounted
	suite.NoError(err)
	mock.Verify(suite.mountClient, matchers.Times(2)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}

// NetworkMountAllShares should do nothing when HA Core is not ready
func (suite *SupervisorServiceSuite) TestNetworkMountAllShares_HACoreNotReady() {
	// Recreate app with HACoreReady=false
	suite.app.RequireStop()
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			func() *dto.ContextState {
				return &dto.ContextState{HACoreReady: false, SupervisorURL: "http://supervisor", AddonIpAddress: "172.30.32.1"}
			},
			service.NewSupervisorService,
			events.NewEventBus,
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
		),
		fx.Populate(&suite.supervisorService),
		fx.Populate(&suite.mountClient),
		fx.Populate(&suite.propertyRepo),
		fx.Populate(&suite.shareService),
	)
	suite.app.RequireStart()

	// Execute
	err := suite.supervisorService.NetworkMountAllShares(context.Background())

	// Assert: no mount attempts
	suite.NoError(err)
	mock.Verify(suite.mountClient, matchers.Times(0)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
	mock.Verify(suite.mountClient, matchers.Times(0)).UpdateMountWithResponse(mock.Any[context.Context](), mock.Any[string](), mock.Any[mount.Mount]())
}

// NetworkMountAllShares should skip when AddonIpAddress is empty
func (suite *SupervisorServiceSuite) TestNetworkMountAllShares_AddonIpEmpty() {
	// Recreate app with empty AddonIpAddress
	suite.app.RequireStop()
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			func() *dto.ContextState {
				return &dto.ContextState{HACoreReady: true, SupervisorURL: "http://supervisor", AddonIpAddress: ""}
			},
			service.NewSupervisorService,
			events.NewEventBus,
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
		),
		fx.Populate(&suite.supervisorService),
		fx.Populate(&suite.mountClient),
		fx.Populate(&suite.propertyRepo),
		fx.Populate(&suite.shareService),
	)
	suite.app.RequireStart()

	// Prepare shares (should be ignored due to empty AddonIp)
	mock.When(suite.shareService.ListShares()).ThenReturn([]dto.SharedResource{
		{Name: "media1", Usage: "media", Status: &dto.SharedResourceStatus{IsValid: true}},
	}, nil)
	// Ensure teardown does not panic when OnStop triggers unmount all: return empty mounts
	emptyResp := &mount.GetMountsResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"mounts":[]}}`),
		JSON200: &struct {
			Data *struct {
				DefaultBackupMount *string        `json:"default_backup_mount,omitempty"`
				Mounts             *[]mount.Mount `json:"mounts,omitempty"`
			} `json:"data,omitempty"`
			Result *mount.GetMounts200Result `json:"result,omitempty"`
		}{Data: &struct {
			DefaultBackupMount *string        `json:"default_backup_mount,omitempty"`
			Mounts             *[]mount.Mount `json:"mounts,omitempty"`
		}{Mounts: &[]mount.Mount{}}},
	}
	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).ThenReturn(emptyResp, nil)

	// Execute
	err := suite.supervisorService.NetworkMountAllShares(context.Background())

	// Assert
	suite.NoError(err)
	mock.Verify(suite.mountClient, matchers.Times(0)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
}

// NetworkMountAllShares mounts eligible shares and unmounts lost mounts
func (suite *SupervisorServiceSuite) TestNetworkMountAllShares_MountsEligibleAndUnmountsLost() {
	shares := []dto.SharedResource{
		{Name: "media1", Usage: "media", Status: &dto.SharedResourceStatus{IsValid: true}},
		{Name: "share2", Usage: "share", Status: &dto.SharedResourceStatus{IsValid: true}},
		{Name: "backup3", Usage: "backup", Status: &dto.SharedResourceStatus{IsValid: true}},
		{Name: "internalX", Usage: "internal", Status: &dto.SharedResourceStatus{IsValid: true}}, // ignored
	}

	// Sequence of GetMounts responses:
	// - First 3 calls (one per share) return empty list
	// - 4th call (networkUnmountLostShares) returns a list including an orphan share
	// - 5th call (NetworkUnmountShare) returns same list to confirm presence
	emptyResp := &mount.GetMountsResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"mounts":[]}}`),
		JSON200: &struct {
			Data *struct {
				DefaultBackupMount *string        `json:"default_backup_mount,omitempty"`
				Mounts             *[]mount.Mount `json:"mounts,omitempty"`
			} `json:"data,omitempty"`
			Result *mount.GetMounts200Result `json:"result,omitempty"`
		}{Data: &struct {
			DefaultBackupMount *string        `json:"default_backup_mount,omitempty"`
			Mounts             *[]mount.Mount `json:"mounts,omitempty"`
		}{Mounts: &[]mount.Mount{}}},
	}

	lostResp := &mount.GetMountsResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		Body:         []byte(`{"result":"ok","data":{"mounts":[{"name":"orphan-share","server":"172.30.32.1"},{"name":"external","server":"192.168.1.1"}]}}`),
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
					{Name: pointer.String("orphan-share"), Server: pointer.String("172.30.32.1")},
					{Name: pointer.String("external"), Server: pointer.String("192.168.1.1")},
				},
			},
		},
	}

	mock.When(suite.shareService.ListShares()).
		ThenReturn(shares, nil).
		ThenReturn([]dto.SharedResource{}, nil) // prevent OnStop extra actions
	mock.When(suite.propertyRepo.Value(mock.Any[string](), mock.Any[bool]())).ThenReturn("test-password", nil)
	mock.When(suite.mountClient.GetMountsWithResponse(mock.Any[context.Context]())).
		ThenReturn(emptyResp, nil).
		ThenReturn(emptyResp, nil).
		ThenReturn(emptyResp, nil).
		ThenReturn(lostResp, nil).
		ThenReturn(lostResp, nil)

	mock.When(suite.mountClient.CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())).ThenReturn(&mount.CreateMountResponse{HTTPResponse: &http.Response{StatusCode: 200}, Body: []byte(`{"result":"ok"}`)}, nil)
	mock.When(suite.mountClient.RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())).ThenReturn(&mount.RemoveMountResponse{HTTPResponse: &http.Response{StatusCode: 200}, Body: []byte(`{"result":"ok"}`)}, nil)

	// Execute
	err := suite.supervisorService.NetworkMountAllShares(context.Background())

	// Assert
	suite.NoError(err)
	// 3 eligible shares should trigger create
	mock.Verify(suite.mountClient, matchers.Times(3)).CreateMountWithResponse(mock.Any[context.Context](), mock.Any[mount.Mount]())
	// One orphan share should be unmounted
	mock.Verify(suite.mountClient, matchers.Times(1)).RemoveMountWithResponse(mock.Any[context.Context](), mock.Any[string]())
}
