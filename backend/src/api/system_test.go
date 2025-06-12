package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SystemHandlerSuite struct {
	suite.Suite
	systemHandler   *api.SystemHanler
	mockFsService   service.FilesystemServiceInterface
	mockHostService service.HostServiceInterface
	testAPI         humatest.TestAPI
	ctx             context.Context
	cancel          context.CancelFunc
	app             *fxtest.App
}

func TestSystemHandlerSuite(t *testing.T) {
	suite.Run(t, new(SystemHandlerSuite))
}

func (suite *SystemHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			api.NewSystemHanler,
			mock.Mock[service.FilesystemServiceInterface],
			mock.Mock[service.HostServiceInterface],
		),
		fx.Populate(&suite.systemHandler),
		fx.Populate(&suite.mockFsService),
		fx.Populate(&suite.mockHostService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()

	_, testAPI := humatest.New(suite.T())
	suite.systemHandler.RegisterSystemHanler(testAPI)
	suite.testAPI = testAPI
}

func (suite *SystemHandlerSuite) TearDownTest() {
	suite.cancel()
	// suite.ctx.Value("wg").(*sync.WaitGroup).Wait() // If system handler starts goroutines
	suite.app.RequireStop()
}

func (suite *SystemHandlerSuite) TestGetHostnameHandler_Success() {
	expectedHostname := "test-host"
	mock.When(suite.mockHostService.GetHostName()).ThenReturn(expectedHostname, nil).Verify(matchers.Times(1))

	resp := suite.testAPI.Get("/hostname")
	suite.Equal(http.StatusOK, resp.Code)

	var result string
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.Equal(expectedHostname, result)
}

func (suite *SystemHandlerSuite) TestGetHostnameHandler_ServiceError() {
	serviceErr := errors.New("failed to get hostname from service")
	mock.When(suite.mockHostService.GetHostName()).ThenReturn("", serviceErr).Verify(matchers.Times(1))

	resp := suite.testAPI.Get("/hostname")
	// Huma typically maps service errors to 500 by default unless specific error mapping is done.
	suite.Equal(http.StatusInternalServerError, resp.Code)

	// You might want to check the error message in the response if Huma passes it through.
	// For example:
	// var errResp huma.ErrorModel
	// err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	// suite.Require().NoError(err)
	// suite.Contains(errResp.Detail, "failed to get hostname from service")
}

// TODO: Add tests for GetNICsHandler and GetFSHandler if they were not previously tested or if their behavior changes.
// For GetNICsHandler, you'd mock ghw.Network() or use a test fixture.
// For GetFSHandler, you'd mock suite.mockFsService.GetStandardMountFlags() and suite.mockFsService.GetFilesystemSpecificMountFlags().
