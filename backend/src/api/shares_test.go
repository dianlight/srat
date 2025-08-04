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
				Device:             "sdb2",
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
		Device:             "sdb2",
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
	mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(nil, dto.ErrorShareAlreadyExists)

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
	mock.When(suite.mockShareService.GetShare(shareName)).ThenReturn(nil, dto.ErrorShareNotFound)

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
	mock.When(suite.mockShareService.DeleteShare(shareName)).ThenReturn(dto.ErrorShareNotFound)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterShareHandler(api)

	// Make HTTP request
	resp := api.Delete("/share/" + shareName)
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

func TestShareHandlerSuite(t *testing.T) {
	suite.Run(t, new(ShareHandlerSuite))
}
