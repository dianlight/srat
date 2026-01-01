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
	"github.com/dianlight/srat/events"
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
	dirtyService     service.DirtyDataServiceInterface
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
			service.NewDirtyDataService,
			events.NewEventBus,
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
		fx.Populate(&suite.dirtyService),
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
	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)
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
				Path: "/mnt/Updater",
				//PathHash:           "5e9b1c4e4951a06eb81659f8b0835cee0d7e0334",
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
		Path: "/mnt/Updater",
		//PathHash:           "5e9b1c4e4951a06eb81659f8b0835cee0d7e0334",
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
	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)
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

	//time.Sleep(5 * time.Second)
	// Verify that SetDirtyShares was called synchronously
	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)

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
				Path: "/mnt/valid-share",
				//PathHash:           "validhash123",
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
				Path: "/mnt/valid-share",
				//PathHash:           "validhash123",
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
	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)
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
	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)
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
	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)
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
	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)
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
				Path: "/mnt/share-mounted-rw-active",
				//PathHash:         "hash1",
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
				Path: "/mnt/share-mounted-rw-inactive",
				//PathHash:         "hash2",
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
				Path: "/mnt/share-mounted-ro-active",
				//PathHash:         "hash3",
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
				Path: "/mnt/share-mounted-ro-inactive",
				//PathHash:         "hash4",
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
			Status: &dto.SharedResourceStatus{
				IsValid: false, // Should be marked as invalid
			},
			MountPointData: &dto.MountPointData{
				Path: "/mnt/share-not-mounted",
				//PathHash:         "hash5",
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
	suite.NotNil(result[0].Status)
	suite.False(result[0].Status.IsValid, "Share should be marked as invalid when volume is not mounted")
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
			Status: &dto.SharedResourceStatus{
				IsValid: false, // Should be marked as invalid
			},
			MountPointData: &dto.MountPointData{ // Volume doesn't exist but has placeholder data
				Path: "/mnt/nonexistent",
				//PathHash:  "hash6",
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
	suite.NotNil(result[0].Status)
	suite.False(result[0].Status.IsValid, "Share should be marked as invalid when volume doesn't exist")
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
		Status: &dto.SharedResourceStatus{
			IsValid: false,
		},
		MountPointData: &dto.MountPointData{
			Path: "/mnt/share-not-mounted",
			//PathHash:         "hash7",
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
	suite.NotNil(result.Status)
	suite.False(result.Status.IsValid)
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
			Path: "/mnt/new-share",
			//PathHash:         "hash8",
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
			Path: "/mnt/new-share",
			//PathHash:         "hash8",
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

	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)
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
			Path: "/mnt/ro-share",
			//PathHash:         "hash9",
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
			Path: "/mnt/ro-share",
			//PathHash:         "hash9",
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

// TestCreateShareWithExFATVolume tests creating a share with an exFAT-formatted volume
func (suite *ShareHandlerSuite) TestCreateShareWithExFATVolume() {
	isWriteSupported := true
	isMounted := true
	isToMountAtStartup := true
	disabled := false
	guestOk := true
	recycleBin := false
	timeMachineMaxSize := "2 TB"
	diskLabel := "Carola"
	timeMachineSupport := dto.TimeMachineSupports.UNSUPPORTED
	input := SharedResourceFromFrontend{
		OrgName: "CAROLA",
		SharedResource: dto.SharedResource{
			Name: "CAROLA",
			Users: []dto.User{
				{
					Username: "homeassistant",
					Password: "changeme!",
					IsAdmin:  true,
				},
			},
			RoUsers:            []dto.User{},
			RecycleBin:         &recycleBin,
			GuestOk:            &guestOk,
			TimeMachineMaxSize: &timeMachineMaxSize,
			Usage:              "media",
			VetoFiles:          []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
			Disabled:           &disabled,
			MountPointData: &dto.MountPointData{
				DiskLabel: &diskLabel,
				DiskSize:  pointer.Uint64(2096874127360),
				Path:      "/mnt/Carola",
				//PathHash:           "0551e312d059cae36f7d0007201d49f5d001f562",
				Root:               "/mnt/Carola",
				Type:               "ADDON",
				FSType:             pointer.String("exfat"),
				Flags:              &dto.MountFlags{{Name: "nodev"}},
				CustomFlags:        nil,
				DeviceId:           "usb-Flash_Disk_3.0_7966051146147389472-0:0-part2",
				IsMounted:          isMounted,
				IsToMountAtStartup: &isToMountAtStartup,
				IsWriteSupported:   &isWriteSupported,
				TimeMachineSupport: &timeMachineSupport,
			},
		},
	}

	expectedShare := &dto.SharedResource{
		Name: "CAROLA",
		Users: []dto.User{
			{
				Username: "homeassistant",
				Password: "changeme!",
				IsAdmin:  true,
			},
		},
		RoUsers:            []dto.User{},
		RecycleBin:         &recycleBin,
		GuestOk:            &guestOk,
		TimeMachineMaxSize: &timeMachineMaxSize,
		Usage:              "media",
		VetoFiles:          []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		Disabled:           &disabled,
		MountPointData: &dto.MountPointData{
			DiskLabel: &diskLabel,
			DiskSize:  pointer.Uint64(2096874127360),
			Path:      "/mnt/Carola",
			//PathHash:           "0551e312d059cae36f7d0007201d49f5d001f562",
			Root:               "/mnt/Carola",
			Type:               "ADDON",
			FSType:             pointer.String("exfat"),
			Flags:              &dto.MountFlags{{Name: "nodev"}},
			CustomFlags:        nil,
			DeviceId:           "usb-Flash_Disk_3.0_7966051146147389472-0:0-part2",
			IsMounted:          isMounted,
			IsToMountAtStartup: &isToMountAtStartup,
			IsWriteSupported:   &isWriteSupported,
			TimeMachineSupport: &timeMachineSupport,
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

	// Assert basic properties
	suite.Equal(expectedShare.Name, result.Name)
	suite.Equal(expectedShare.Usage, result.Usage)
	suite.NotNil(result.GuestOk)
	suite.True(*result.GuestOk)
	suite.NotNil(result.TimeMachineMaxSize)
	suite.Equal("2 TB", *result.TimeMachineMaxSize)
	suite.NotNil(result.RecycleBin)
	suite.False(*result.RecycleBin)
	suite.Equal([]string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"}, result.VetoFiles)

	// Assert mount point data
	suite.NotNil(result.MountPointData)
	suite.NotNil(result.MountPointData.DiskLabel)
	suite.Equal("Carola", *result.MountPointData.DiskLabel)
	suite.NotNil(result.MountPointData.DiskSize)
	suite.Equal(uint64(2096874127360), *result.MountPointData.DiskSize)
	suite.Equal("/mnt/Carola", result.MountPointData.Path)
	suite.NotNil(result.MountPointData.FSType)
	suite.Equal("exfat", *result.MountPointData.FSType)
	suite.Equal("usb-Flash_Disk_3.0_7966051146147389472-0:0-part2", result.MountPointData.DeviceId)
	suite.True(result.MountPointData.IsMounted)
	suite.NotNil(result.MountPointData.IsWriteSupported)
	suite.True(*result.MountPointData.IsWriteSupported)
	suite.NotNil(result.MountPointData.IsToMountAtStartup)
	suite.True(*result.MountPointData.IsToMountAtStartup)
	suite.NotNil(result.MountPointData.TimeMachineSupport)
	suite.Equal(dto.TimeMachineSupports.UNSUPPORTED, *result.MountPointData.TimeMachineSupport)
	suite.NotNil(result.MountPointData.Flags)
	suite.Require().Len(*result.MountPointData.Flags, 1)
	suite.Equal("nodev", (*result.MountPointData.Flags)[0].Name)

	// Assert user properties
	suite.Require().Len(result.Users, 1)
	suite.Equal("homeassistant", result.Users[0].Username)
	suite.True(result.Users[0].IsAdmin)
}

// TestUpdateShareWithExFATVolume tests updating a share with complete exFAT volume data
func (suite *ShareHandlerSuite) TestUpdateShareWithExFATVolume() {
	// Setup test data with complete volume information
	guestOk := true
	recycleBin := false
	timeMachineMaxSize := "2 TB"
	diskLabel := "Carola"
	fstype := "exfat"
	deviceId := "usb-Flash_Disk_3.0_7966051146147389472-0:0-part2"
	isToMountAtStartup := true
	isWriteSupported := true
	timeMachineSupport := dto.TimeMachineSupports.UNSUPPORTED

	vetoFiles := []string{
		"._*",
		".DS_Store",
		"Thumbs.db",
		"icon?",
		".Trashes",
	}

	mountFlags := dto.MountFlags{
		{Name: "nodev"},
	}

	input := dto.SharedResource{
		Name:               "CAROLA",
		Disabled:           pointer.Bool(false),
		GuestOk:            &guestOk,
		RecycleBin:         &recycleBin,
		TimeMachineMaxSize: &timeMachineMaxSize,
		Usage:              dto.UsageAsMedia,
		VetoFiles:          vetoFiles,
		Users: []dto.User{
			{
				Username: "homeassistant",
				Password: "changeme!",
				IsAdmin:  true,
			},
		},
		MountPointData: &dto.MountPointData{
			DiskLabel: &diskLabel,
			DiskSize:  pointer.Uint64(2096874127360),
			Path:      "/mnt/Carola",
			//PathHash:           "0551e312d059cae36f7d0007201d49f5d001f562",
			Root:               "/mnt/Carola",
			Type:               "ADDON",
			FSType:             &fstype,
			Flags:              &mountFlags,
			DeviceId:           deviceId,
			IsMounted:          true,
			IsToMountAtStartup: &isToMountAtStartup,
			IsWriteSupported:   &isWriteSupported,
			TimeMachineSupport: &timeMachineSupport,
		},
	}

	expectedShare := &dto.SharedResource{
		Name:               "CAROLA",
		Disabled:           pointer.Bool(false),
		GuestOk:            &guestOk,
		RecycleBin:         &recycleBin,
		TimeMachineMaxSize: &timeMachineMaxSize,
		Usage:              dto.UsageAsMedia,
		VetoFiles:          vetoFiles,
		Users: []dto.User{
			{
				Username: "homeassistant",
				Password: "changeme!",
				IsAdmin:  true,
			},
		},
		MountPointData: &dto.MountPointData{
			DiskLabel: &diskLabel,
			DiskSize:  pointer.Uint64(2096874127360),
			Path:      "/mnt/Carola",
			//PathHash:           "0551e312d059cae36f7d0007201d49f5d001f562",
			Root:               "/mnt/Carola",
			Type:               "ADDON",
			FSType:             &fstype,
			Flags:              &mountFlags,
			DeviceId:           deviceId,
			IsMounted:          true,
			IsToMountAtStartup: &isToMountAtStartup,
			IsWriteSupported:   &isWriteSupported,
			TimeMachineSupport: &timeMachineSupport,
		},
	}

	// Setup mock expectations
	mock.When(suite.mockShareService.UpdateShare(mock.Equal("CAROLA"), mock.Any[dto.SharedResource]())).
		ThenReturn(expectedShare, nil)

	// Create test API and register handler
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make the request
	resp := api.Put("/share/CAROLA", input)

	// Verify response
	suite.Equal(http.StatusOK, resp.Code)

	// Parse response body
	var result dto.SharedResource
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.NoError(err)

	// Verify basic fields
	suite.Equal("CAROLA", result.Name)
	suite.Equal(dto.UsageAsMedia, result.Usage)
	suite.NotNil(result.GuestOk)
	suite.True(*result.GuestOk)
	suite.NotNil(result.TimeMachineMaxSize)
	suite.Equal("2 TB", *result.TimeMachineMaxSize)
	suite.NotNil(result.RecycleBin)
	suite.False(*result.RecycleBin)

	// Verify veto files
	suite.Len(result.VetoFiles, 5)
	suite.Contains(result.VetoFiles, "._*")
	suite.Contains(result.VetoFiles, ".DS_Store")
	suite.Contains(result.VetoFiles, "Thumbs.db")
	suite.Contains(result.VetoFiles, "icon?")
	suite.Contains(result.VetoFiles, ".Trashes")

	// Verify mount point data with detailed checks
	suite.NotNil(result.MountPointData)
	mpd := result.MountPointData

	suite.NotNil(mpd.DiskLabel)
	suite.Equal("Carola", *mpd.DiskLabel)

	suite.NotNil(mpd.DiskSize)
	suite.Equal(uint64(2096874127360), *mpd.DiskSize)

	suite.Equal("/mnt/Carola", mpd.Path)
	//suite.Equal("0551e312d059cae36f7d0007201d49f5d001f562", mpd.PathHash)

	suite.Equal("/mnt/Carola", mpd.Root)

	suite.Equal("ADDON", mpd.Type)

	suite.NotNil(mpd.FSType)
	suite.Equal("exfat", *mpd.FSType)

	suite.NotNil(mpd.Flags)
	suite.Len(*mpd.Flags, 1)
	suite.Equal("nodev", (*mpd.Flags)[0].Name)

	suite.Equal("usb-Flash_Disk_3.0_7966051146147389472-0:0-part2", mpd.DeviceId)

	suite.True(mpd.IsMounted)

	suite.NotNil(mpd.IsToMountAtStartup)
	suite.True(*mpd.IsToMountAtStartup)

	suite.NotNil(mpd.IsWriteSupported)
	suite.True(*mpd.IsWriteSupported)

	suite.NotNil(mpd.TimeMachineSupport)
	suite.Equal(dto.TimeMachineSupports.UNSUPPORTED, *mpd.TimeMachineSupport)

	// Verify user data
	suite.Len(result.Users, 1)
	suite.Equal("homeassistant", result.Users[0].Username)
	suite.True(result.Users[0].IsAdmin)

	// Verify that SetDirtyShares was called
	//suite.True(suite.dirtyService.GetDirtyDataTracker().Shares)
}

func TestShareHandlerSuite(t *testing.T) {
	suite.Run(t, new(ShareHandlerSuite))
}
