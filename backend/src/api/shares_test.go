package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ShareHandlerSuite struct {
	suite.Suite
	app              *fxtest.App
	handler          *api.ShareHandler
	mockDirtyService service.DirtyDataServiceInterface
	mockShareService service.ShareServiceInterface
	ctx              context.Context
	cancel           context.CancelFunc
}

func (suite *ShareHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			api.NewShareHandler,
			mock.Mock[service.DirtyDataServiceInterface],
			mock.Mock[service.ShareServiceInterface],
			func() *dto.ContextState {
				return &dto.ContextState{
					ReadOnlyMode:    false,
					Heartbeat:       1,
					DockerInterface: "hassio",
					DockerNet:       "172.30.32.0/23",
				}
			},
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockDirtyService),
		fx.Populate(&suite.mockShareService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *ShareHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	}
	suite.app.RequireStop()
}

func (suite *ShareHandlerSuite) TestCreateShareSuccess() {
	input := dto.SharedResource{Name: "test-share"}
	expectedShare := &dto.SharedResource{Name: "test-share"}

	// Configure mock expectations
	mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(expectedShare, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Post("/share", input)
	suite.Require().Equal(http.StatusCreated, resp.Code)

	// Parse response
	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedShare.Name, result.Name)

	// Verify that SetDirtyShares was called synchronously
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}

type SharedResourceFromFrontend struct {
	OrgName string `json:"org_name,omitempty"`
	dto.SharedResource
}

func (suite *ShareHandlerSuite) TestCreateShareSuccessFull() {
	input := SharedResourceFromFrontend{
		OrgName: "TestOrg",
		SharedResource: dto.SharedResource{
			Name: "UPDATER",
			Users: []dto.User{
				{
					Username: "homeassistant",
					Password: "changeme!",
					IsAdmin:  true,
					RwShares: []string{"addon_configs", "config", "addons", "ssl", "share", "backup", "media", "EFI"},
				},
			},
			RoUsers:     []dto.User{},
			TimeMachine: pointer.Bool(false),
			Usage:       "none",
			VetoFiles:   []string{},
			Disabled:    pointer.Bool(false),
			MountPointData: &dto.MountPointData{
				Path:               "/mnt/Updater",
				PathHash:           "5e9b1c4e4951a06eb81659f8b0835cee0d7e0334",
				Type:               "ADDON",
				FSType:             pointer.String("exfat"),
				Flags:              nil,
				CustomFlags:        nil,
				DeviceId:           "sdb2",
				IsMounted:          true,
				IsToMountAtStartup: pointer.Bool(true),
			},
		},
	}
	expectedShare := &dto.SharedResource{Name: "UPDATER", Users: []dto.User{
		{
			Username: "homeassistant",
			Password: "changeme!",
			IsAdmin:  true,
			RwShares: []string{"addon_configs", "config", "addons", "ssl", "share", "backup", "media", "EFI"},
		},
	}, RoUsers: []dto.User{}, TimeMachine: pointer.Bool(false), Usage: "none", VetoFiles: []string{}, Disabled: pointer.Bool(false), MountPointData: &dto.MountPointData{
		Path:               "/mnt/Updater",
		PathHash:           "5e9b1c4e4951a06eb81659f8b0835cee0d7e0334",
		Type:               "ADDON",
		FSType:             pointer.String("exfat"),
		Flags:              nil,
		CustomFlags:        nil,
		DeviceId:           "sdb2",
		IsMounted:          true,
		IsToMountAtStartup: pointer.Bool(true),
	},
	}

	// Configure mock expectations
	mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(expectedShare, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Post("/share", input)
	suite.Require().Equal(http.StatusCreated, resp.Code)

	// Parse response
	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedShare.Name, result.Name)

	// Verify that SetDirtyShares was called synchronously
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}

func (suite *ShareHandlerSuite) TestCreateShareAlreadyExists() {
	input := dto.SharedResource{Name: "existing-share"}

	// Configure mock expectations
	mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(nil, errors.WithStack(dto.ErrorShareAlreadyExists))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Post("/share", input)
	suite.Require().Equal(http.StatusConflict, resp.Code)
}

func (suite *ShareHandlerSuite) TestCreateShareServiceError() {
	input := dto.SharedResource{Name: "error-share"}
	expectedErr := errors.New("database connection failed")

	// Configure mock expectations
	mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(nil, expectedErr)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Post("/share", input)
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *ShareHandlerSuite) TestCreateShareAsyncNotification() {
	input := dto.SharedResource{Name: "test-share"}
	expectedShare := &dto.SharedResource{Name: "test-share"}

	mock.When(suite.mockShareService.CreateShare(mock.Equal(input))).ThenReturn(expectedShare, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Post("/share", input)
	suite.Require().Equal(http.StatusCreated, resp.Code)

	// Parse response
	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedShare.Name, result.Name)

	// Verify that SetDirtyShares was called synchronously
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()

	// Note: NotifyClient is called asynchronously (in a goroutine),
	// so we can't reliably assert it was called in this test without
	// adding synchronization mechanisms or waiting
}

func (suite *ShareHandlerSuite) TestListSharesSuccess() {
	expectedShares := []dto.SharedResource{
		{Name: "share1"},
		{Name: "share2"},
	}

	// Configure mock expectations
	mock.When(suite.mockShareService.ListShares()).ThenReturn(expectedShares, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Get("/shares")
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result []dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedShares, result)
}

func (suite *ShareHandlerSuite) TestListSharesWithDisabledShareWithoutMountPoint() {
	disabled := true
	expectedShares := []dto.SharedResource{
		{
			Name: "valid-share",
			Users: []dto.User{
				{
					Username: "testuser",
					Password: "testpass",
					IsAdmin:  true,
				},
			},
			Disabled: pointer.Bool(false),
			MountPointData: &dto.MountPointData{
				Path:               "/mnt/valid-share",
				PathHash:           "validhash123",
				Type:               "ADDON",
				FSType:             pointer.String("ext4"),
				DeviceId:           "sda1",
				IsMounted:          true,
				IsToMountAtStartup: pointer.Bool(true),
			},
		},
		{
			Name: "invalid-share-no-mount",
			Users: []dto.User{
				{
					Username: "testuser2",
					Password: "testpass2",
					IsAdmin:  false,
				},
			},
			Disabled:       &disabled,
			MountPointData: nil, // No mount point data - should be disabled
		},
	}

	// Configure mock expectations
	mock.When(suite.mockShareService.ListShares()).ThenReturn(expectedShares, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Get("/shares")
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result []dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Require().Len(result, 2, "Expected 2 shares in response")

	// Verify first share (valid)
	suite.Equal("valid-share", result[0].Name)
	suite.NotNil(result[0].MountPointData, "Valid share should have mount point data")
	suite.False(*result[0].Disabled, "Valid share should not be disabled")

	// Verify second share (invalid - no mount point)
	suite.Equal("invalid-share-no-mount", result[1].Name)
	suite.Nil(result[1].MountPointData, "Invalid share should have nil mount point data")
	suite.NotNil(result[1].Disabled, "Invalid share should have Disabled field set")
	suite.True(*result[1].Disabled, "Share without mount point data should be disabled")
}

func (suite *ShareHandlerSuite) TestListSharesWithEmptyPathInMountPoint() {
	disabled := true
	expectedShares := []dto.SharedResource{
		{
			Name: "valid-share-with-path",
			Users: []dto.User{
				{
					Username: "testuser",
					Password: "testpass",
					IsAdmin:  true,
				},
			},
			Disabled: pointer.Bool(false),
			MountPointData: &dto.MountPointData{
				Path:               "/mnt/valid-share",
				PathHash:           "validhash123",
				Type:               "ADDON",
				FSType:             pointer.String("ext4"),
				DeviceId:           "sda1",
				IsMounted:          true,
				IsToMountAtStartup: pointer.Bool(true),
			},
		},
		{
			Name: "UPDATER",
			Users: []dto.User{
				{
					Username: "homeassistant",
					Password: "changeme!",
					IsAdmin:  true,
					RwShares: []string{"addon_configs", "config", "addons", "ssl", "share", "backup", "media", "EFI"},
				},
			},
			RoUsers:        []dto.User{},
			TimeMachine:    pointer.Bool(false),
			Usage:          "none",
			VetoFiles:      []string{},
			Disabled:       &disabled,
			MountPointData: nil, // Empty path should result in nil MountPointData
		},
	}

	// Configure mock expectations
	mock.When(suite.mockShareService.ListShares()).ThenReturn(expectedShares, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Get("/shares")
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result []dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Require().Len(result, 2, "Expected 2 shares in response")

	// Verify first share (valid with path)
	suite.Equal("valid-share-with-path", result[0].Name)
	suite.NotNil(result[0].MountPointData, "Valid share should have mount point data")
	suite.Equal("/mnt/valid-share", result[0].MountPointData.Path)
	suite.False(*result[0].Disabled, "Valid share should not be disabled")

	// Verify second share (UPDATER with empty path - should have nil mount_point_data)
	suite.Equal("UPDATER", result[1].Name)
	suite.Nil(result[1].MountPointData, "Share with empty path should have nil mount point data")
	suite.NotNil(result[1].Disabled, "Share with empty path should have Disabled field set")
	suite.True(*result[1].Disabled, "Share with empty path should be disabled")
}

func (suite *ShareHandlerSuite) TestGetShareSuccess() {
	shareName := "test-share"
	expectedShare := &dto.SharedResource{Name: shareName}

	// Configure mock expectations
	mock.When(suite.mockShareService.GetShare(shareName)).ThenReturn(expectedShare, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Get("/share/" + shareName)
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedShare.Name, result.Name)
}

func (suite *ShareHandlerSuite) TestGetShareNotFound() {
	shareName := "nonexistent-share"

	// Configure mock expectations
	mock.When(suite.mockShareService.GetShare(shareName)).ThenReturn(nil, errors.WithStack(dto.ErrorShareNotFound))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Get("/share/" + shareName)
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

func (suite *ShareHandlerSuite) TestDeleteShareSuccess() {
	shareName := "test-share"

	// Configure mock expectations
	mock.When(suite.mockShareService.DeleteShare(shareName)).ThenReturn(nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Delete("/share/" + shareName)
	suite.Require().Equal(http.StatusNoContent, resp.Code)

	// Verify that SetDirtyShares was called
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}

func (suite *ShareHandlerSuite) TestDeleteShareNotFound() {
	shareName := "nonexistent-share"

	// Configure mock expectations
	mock.When(suite.mockShareService.DeleteShare(shareName)).ThenReturn(errors.WithStack(dto.ErrorShareNotFound))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Delete("/share/" + shareName)
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

func (suite *ShareHandlerSuite) TestUpdateShareSuccess() {
	shareName := "test-share"
	input := dto.SharedResource{Name: shareName, Usage: "backup"}
	expectedShare := &dto.SharedResource{Name: shareName, Usage: "backup"}

	// Configure mock expectations
	mock.When(suite.mockShareService.UpdateShare(mock.Equal(shareName), mock.Any[dto.SharedResource]())).ThenReturn(expectedShare, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request - use PUT, not PATCH
	resp := api.Put("/share/"+shareName, input)
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedShare.Name, result.Name)
	suite.Equal(expectedShare.Usage, result.Usage)

	// Verify that SetDirtyShares was called
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}

func (suite *ShareHandlerSuite) TestUpdateShareNotFound() {
	shareName := "nonexistent-share"
	input := dto.SharedResource{Name: shareName}

	// Configure mock expectations
	mock.When(suite.mockShareService.UpdateShare(mock.Equal(shareName), mock.Any[dto.SharedResource]())).ThenReturn(nil, errors.WithStack(dto.ErrorShareNotFound))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request - use PUT, not PATCH
	resp := api.Put("/share/"+shareName, input)
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

func (suite *ShareHandlerSuite) TestDisableShareSuccess() {
	shareName := "test-share"
	disabledShare := &dto.SharedResource{Name: shareName, Disabled: pointer.Bool(true)}

	// Configure mock expectations
	mock.When(suite.mockShareService.DisableShare(shareName)).ThenReturn(disabledShare, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request - use PUT, not POST
	resp := api.Put("/share/"+shareName+"/disable", struct{}{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response - DisableShare returns Body, not empty
	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.Equal(shareName, result.Name)
	suite.NotNil(result.Disabled)
	suite.True(*result.Disabled)

	// Verify that SetDirtyShares was called
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}

func (suite *ShareHandlerSuite) TestDisableShareNotFound() {
	shareName := "nonexistent-share"

	// Configure mock expectations
	mock.When(suite.mockShareService.DisableShare(shareName)).ThenReturn(nil, errors.WithStack(dto.ErrorShareNotFound))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request - use PUT, not POST
	resp := api.Put("/share/"+shareName+"/disable", struct{}{})
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

func (suite *ShareHandlerSuite) TestEnableShareSuccess() {
	shareName := "test-share"
	enabledShare := &dto.SharedResource{Name: shareName, Disabled: pointer.Bool(false)}

	// Configure mock expectations
	mock.When(suite.mockShareService.EnableShare(shareName)).ThenReturn(enabledShare, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request - use PUT, not POST
	resp := api.Put("/share/"+shareName+"/enable", struct{}{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response - EnableShare returns Body, not empty
	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.Equal(shareName, result.Name)
	suite.NotNil(result.Disabled)
	suite.False(*result.Disabled)

	// Verify that SetDirtyShares was called
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}

func (suite *ShareHandlerSuite) TestEnableShareNotFound() {
	shareName := "nonexistent-share"

	// Configure mock expectations
	mock.When(suite.mockShareService.EnableShare(shareName)).ThenReturn(nil, errors.WithStack(dto.ErrorShareNotFound))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request - use PUT, not POST
	resp := api.Put("/share/"+shareName+"/enable", struct{}{})
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

// TestListSharesWithVolumeMountedRW tests shares with mounted volumes that are read-write
func (suite *ShareHandlerSuite) TestListSharesWithVolumeMountedRW() {
	isWriteSupported := true
	isMounted := true
	expectedShares := []dto.SharedResource{
		{
			Name: "share-mounted-rw-active",
			Users: []dto.User{
				{
					Username: "testuser",
					Password: "testpass",
					RwShares: []string{"share-mounted-rw-active"},
				},
			},
			Disabled: pointer.Bool(false),
			MountPointData: &dto.MountPointData{
				Path:             "/mnt/share-mounted-rw-active",
				PathHash:         "hash1",
				Type:             "ADDON",
				FSType:           pointer.String("ext4"),
				DeviceId:         "sda1",
				IsMounted:        true,
				IsWriteSupported: &isWriteSupported,
			},
		},
		{
			Name: "share-mounted-rw-inactive",
			Users: []dto.User{
				{
					Username: "testuser2",
					Password: "testpass2",
					RwShares: []string{"share-mounted-rw-inactive"},
				},
			},
			Disabled: pointer.Bool(true),
			MountPointData: &dto.MountPointData{
				Path:             "/mnt/share-mounted-rw-inactive",
				PathHash:         "hash2",
				Type:             "ADDON",
				FSType:           pointer.String("ext4"),
				DeviceId:         "sda2",
				IsMounted:        isMounted,
				IsWriteSupported: &isWriteSupported,
			},
		},
	}

	mock.When(suite.mockShareService.ListShares()).ThenReturn(expectedShares, nil)

	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	resp := api.Get("/shares")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var result []dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	suite.Require().Len(result, 2)

	// First share: mounted rw, active
	suite.Equal("share-mounted-rw-active", result[0].Name)
	suite.False(*result[0].Disabled)
	suite.True(result[0].MountPointData.IsMounted)
	suite.True(*result[0].MountPointData.IsWriteSupported)
	suite.Require().Len(result[0].Users, 1)
	suite.Contains(result[0].Users[0].RwShares, "share-mounted-rw-active")

	// Second share: mounted rw, inactive (DB says disabled)
	suite.Equal("share-mounted-rw-inactive", result[1].Name)
	suite.True(*result[1].Disabled)
	suite.True(result[1].MountPointData.IsMounted)
	suite.True(*result[1].MountPointData.IsWriteSupported)
}

// TestListSharesWithVolumeMountedRO tests shares with mounted read-only volumes
func (suite *ShareHandlerSuite) TestListSharesWithVolumeMountedRO() {
	isWriteSupported := false
	isMounted := true
	expectedShares := []dto.SharedResource{
		{
			Name: "share-mounted-ro-active",
			Users: []dto.User{
				{
					Username: "testuser",
					Password: "testpass",
					RwShares: []string{}, // Should be empty for RO mount
				},
			},
			RoUsers: []dto.User{
				{
					Username: "testuser",
					Password: "testpass",
				},
			},
			Disabled: pointer.Bool(false),
			MountPointData: &dto.MountPointData{
				Path:             "/mnt/share-mounted-ro-active",
				PathHash:         "hash3",
				Type:             "ADDON",
				FSType:           pointer.String("ext4"),
				DeviceId:         "sda3",
				IsMounted:        isMounted,
				IsWriteSupported: &isWriteSupported,
			},
		},
		{
			Name: "share-mounted-ro-inactive",
			Users: []dto.User{
				{
					Username: "testuser2",
					Password: "testpass2",
					RwShares: []string{}, // Should be empty for RO mount
				},
			},
			RoUsers: []dto.User{
				{
					Username: "testuser2",
					Password: "testpass2",
				},
			},
			Disabled: pointer.Bool(true),
			MountPointData: &dto.MountPointData{
				Path:             "/mnt/share-mounted-ro-inactive",
				PathHash:         "hash4",
				Type:             "ADDON",
				FSType:           pointer.String("ext4"),
				DeviceId:         "sda4",
				IsMounted:        isMounted,
				IsWriteSupported: &isWriteSupported,
			},
		},
	}

	mock.When(suite.mockShareService.ListShares()).ThenReturn(expectedShares, nil)

	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	resp := api.Get("/shares")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var result []dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	suite.Require().Len(result, 2)

	// First share: mounted ro, active
	suite.Equal("share-mounted-ro-active", result[0].Name)
	suite.False(*result[0].Disabled)
	suite.True(result[0].MountPointData.IsMounted)
	suite.False(*result[0].MountPointData.IsWriteSupported)
	suite.Empty(result[0].Users[0].RwShares, "RO mount should have no RW users")
	suite.Require().Len(result[0].RoUsers, 1)

	// Second share: mounted ro, inactive
	suite.Equal("share-mounted-ro-inactive", result[1].Name)
	suite.True(*result[1].Disabled)
	suite.True(result[1].MountPointData.IsMounted)
	suite.False(*result[1].MountPointData.IsWriteSupported)
	suite.Empty(result[1].Users[0].RwShares, "RO mount should have no RW users")
}

// TestListSharesWithVolumeNotMounted tests shares where volume is not mounted
func (suite *ShareHandlerSuite) TestListSharesWithVolumeNotMounted() {
	isWriteSupported := true
	expectedShares := []dto.SharedResource{
		{
			Name: "share-not-mounted-was-active",
			Users: []dto.User{
				{
					Username: "testuser",
					Password: "testpass",
					RwShares: []string{"share-not-mounted-was-active"},
				},
			},
			Disabled: pointer.Bool(true), // Should be disabled if not mounted
			Invalid:  pointer.Bool(true), // Should be marked as invalid/anomaly
			MountPointData: &dto.MountPointData{
				Path:             "/mnt/share-not-mounted",
				PathHash:         "hash5",
				Type:             "ADDON",
				FSType:           pointer.String("ext4"),
				DeviceId:         "sda5",
				IsMounted:        false, // Not mounted
				IsWriteSupported: &isWriteSupported,
			},
		},
	}

	mock.When(suite.mockShareService.ListShares()).ThenReturn(expectedShares, nil)

	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	resp := api.Get("/shares")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var result []dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	suite.Require().Len(result, 1)

	// Share with volume not mounted: should be disabled and marked as anomaly
	suite.Equal("share-not-mounted-was-active", result[0].Name)
	suite.NotNil(result[0].Disabled)
	suite.True(*result[0].Disabled, "Share should be disabled when volume is not mounted")
	suite.NotNil(result[0].Invalid)
	suite.True(*result[0].Invalid, "Share should be marked as invalid/anomaly when volume is not mounted")
	suite.False(result[0].MountPointData.IsMounted)
}

// TestListSharesWithVolumeNotExists tests shares where volume doesn't exist
func (suite *ShareHandlerSuite) TestListSharesWithVolumeNotExists() {
	expectedShares := []dto.SharedResource{
		{
			Name: "share-no-volume",
			Users: []dto.User{
				{
					Username: "testuser",
					Password: "testpass",
					RwShares: []string{"share-no-volume"},
				},
			},
			Disabled: pointer.Bool(true), // Should be disabled
			Invalid:  pointer.Bool(true), // Should be marked as invalid/anomaly
			MountPointData: &dto.MountPointData{ // Volume doesn't exist but has placeholder data
				Path:      "/mnt/nonexistent",
				PathHash:  "hash6",
				Type:      "ADDON",
				IsMounted: false,
				IsInvalid: true, // Mark mount point itself as invalid
			},
		},
	}

	mock.When(suite.mockShareService.ListShares()).ThenReturn(expectedShares, nil)

	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	resp := api.Get("/shares")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var result []dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	suite.Require().Len(result, 1)

	// Share with non-existent volume: should be disabled and marked as anomaly
	suite.Equal("share-no-volume", result[0].Name)
	suite.NotNil(result[0].Disabled)
	suite.True(*result[0].Disabled, "Share should be disabled when volume doesn't exist")
	suite.NotNil(result[0].Invalid)
	suite.True(*result[0].Invalid, "Share should be marked as invalid/anomaly when volume doesn't exist")
	suite.True(result[0].MountPointData.IsInvalid, "Mount point should be marked as invalid")
	suite.False(result[0].MountPointData.IsMounted)
}

// TestGetShareWithVolumeStates tests individual share retrieval with different volume states
func (suite *ShareHandlerSuite) TestGetShareWithVolumeNotMounted() {
	shareName := "share-not-mounted"
	isWriteSupported := true
	expectedShare := &dto.SharedResource{
		Name: shareName,
		Users: []dto.User{
			{
				Username: "testuser",
				Password: "testpass",
				RwShares: []string{shareName},
			},
		},
		Disabled: pointer.Bool(true),
		Invalid:  pointer.Bool(true),
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/share-not-mounted",
			PathHash:         "hash7",
			Type:             "ADDON",
			FSType:           pointer.String("ext4"),
			DeviceId:         "sda7",
			IsMounted:        false,
			IsWriteSupported: &isWriteSupported,
		},
	}

	mock.When(suite.mockShareService.GetShare(shareName)).ThenReturn(expectedShare, nil)

	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	resp := api.Get("/share/" + shareName)
	suite.Require().Equal(http.StatusOK, resp.Code)

	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	suite.Equal(shareName, result.Name)
	suite.NotNil(result.Disabled)
	suite.True(*result.Disabled)
	suite.NotNil(result.Invalid)
	suite.True(*result.Invalid)
	suite.False(result.MountPointData.IsMounted)
}

// TestCreateShareWithVolumeValidation tests share creation with volume validation
func (suite *ShareHandlerSuite) TestCreateShareWithMountedRWVolume() {
	isWriteSupported := true
	input := dto.SharedResource{
		Name: "new-share",
		Users: []dto.User{
			{
				Username: "testuser",
				Password: "testpass",
				RwShares: []string{"new-share"},
			},
		},
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/new-share",
			PathHash:         "hash8",
			Type:             "ADDON",
			FSType:           pointer.String("ext4"),
			DeviceId:         "sda8",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}
	expectedShare := &dto.SharedResource{
		Name:     "new-share",
		Disabled: pointer.Bool(false),
		Users: []dto.User{
			{
				Username: "testuser",
				Password: "testpass",
				RwShares: []string{"new-share"},
			},
		},
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/new-share",
			PathHash:         "hash8",
			Type:             "ADDON",
			FSType:           pointer.String("ext4"),
			DeviceId:         "sda8",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}

	mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(expectedShare, nil)

	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	resp := api.Post("/share", input)
	suite.Require().Equal(http.StatusCreated, resp.Code)

	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	suite.Equal(expectedShare.Name, result.Name)
	suite.False(*result.Disabled)
	suite.True(result.MountPointData.IsMounted)
	suite.True(*result.MountPointData.IsWriteSupported)

	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}

// TestCreateShareWithROVolumeNoRWUsers tests that RO volumes cannot have RW users
func (suite *ShareHandlerSuite) TestCreateShareWithROVolumeHasOnlyROUsers() {
	isWriteSupported := false
	input := dto.SharedResource{
		Name: "ro-share",
		RoUsers: []dto.User{
			{
				Username: "testuser",
				Password: "testpass",
			},
		},
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/ro-share",
			PathHash:         "hash9",
			Type:             "ADDON",
			FSType:           pointer.String("ext4"),
			DeviceId:         "sda9",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}
	expectedShare := &dto.SharedResource{
		Name:     "ro-share",
		Disabled: pointer.Bool(false),
		RoUsers: []dto.User{
			{
				Username: "testuser",
				Password: "testpass",
			},
		},
		Users: []dto.User{}, // No RW users for RO volume
		MountPointData: &dto.MountPointData{
			Path:             "/mnt/ro-share",
			PathHash:         "hash9",
			Type:             "ADDON",
			FSType:           pointer.String("ext4"),
			DeviceId:         "sda9",
			IsMounted:        true,
			IsWriteSupported: &isWriteSupported,
		},
	}

	mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(expectedShare, nil)

	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	resp := api.Post("/share", input)
	suite.Require().Equal(http.StatusCreated, resp.Code)

	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	suite.Equal(expectedShare.Name, result.Name)
	suite.False(*result.Disabled)
	suite.True(result.MountPointData.IsMounted)
	suite.False(*result.MountPointData.IsWriteSupported)
	suite.Empty(result.Users, "RO volume should not have RW users")
	suite.Require().Len(result.RoUsers, 1)
}

func TestShareHandlerSuite(t *testing.T) {
	suite.Run(t, new(ShareHandlerSuite))
}
