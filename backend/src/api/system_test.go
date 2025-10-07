package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/shirou/gopsutil/v4/net"
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

func (suite *SystemHandlerSuite) TestGetNICsHandler_ReturnsInterfaces() {
	expected, err := net.InterfacesWithContext(suite.ctx)
	suite.Require().NoError(err)

	resp := suite.testAPI.Get("/nics")
	suite.Equal(http.StatusOK, resp.Code)

	var result []net.InterfaceStat
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.Len(result, len(expected))
}

func (suite *SystemHandlerSuite) TestGetNICsHandler_FiltersVethInterfaces() {
	// Test that veth* interfaces are filtered from the response
	resp := suite.testAPI.Get("/nics")
	suite.Equal(http.StatusOK, resp.Code)

	var result []net.InterfaceStat
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Verify no veth interfaces are in the result
	for _, nic := range result {
		suite.NotContains(nic.Name, "veth", "veth interfaces should be filtered out")
	}
}

func (suite *SystemHandlerSuite) TestGetFSHandler_IncludesFuse3() {
	tempFile, err := os.CreateTemp(suite.T().TempDir(), "filesystems-*")
	suite.Require().NoError(err)
	defer tempFile.Close()

	contents := strings.Join([]string{
		"nodev\tsysfs",
		"nodev\tfuse",
		"\text4",
		"\txfs",
		"nodev\tzfs",
		"nodev\tfuse3",
	}, "\n")
	suite.Require().NoError(os.WriteFile(tempFile.Name(), []byte(contents), 0o644))

	suite.systemHandler.SetFilesystemsPath(tempFile.Name())

	standardFlags := []dto.MountFlag{{Name: "rw"}}
	mock.When(suite.mockFsService.GetStandardMountFlags()).ThenReturn(standardFlags, nil)

	for _, fsType := range []string{"ext4", "xfs", "zfs", "fuse", "fuse3"} {
		mock.When(suite.mockFsService.GetFilesystemSpecificMountFlags(fsType)).ThenReturn([]dto.MountFlag{}, nil)
	}

	resp := suite.testAPI.Get("/filesystems")
	suite.Equal(http.StatusOK, resp.Code)

	var result []dto.FilesystemType
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.Len(result, 5)

	names := make([]string, len(result))
	for i, fsType := range result {
		names[i] = fsType.Name
		suite.Len(fsType.MountFlags, len(standardFlags))
		for idx, flag := range fsType.MountFlags {
			suite.Equal(standardFlags[idx], flag)
		}
	}

	suite.Contains(names, "fuse3")
	suite.Contains(names, "fuse")
	suite.NotContains(names, "sysfs")
}

func (suite *SystemHandlerSuite) TestGetCapabilitiesHandler_Success() {
	// This test checks that the endpoint returns successfully
	// QUIC support detection depends on the actual system state
	resp := suite.testAPI.Get("/capabilities")
	suite.Equal(http.StatusOK, resp.Code)

	var result dto.SystemCapabilities
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// We can't assert the actual value of SupportsQUIC as it depends on the system
	// but we can verify all fields exist and have correct types
	suite.IsType(false, result.SupportsQUIC)
	suite.IsType(false, result.HasKernelModule)
	suite.IsType("", result.SambaVersion)
	suite.IsType(false, result.SambaVersionSufficient)
	// UnsupportedReason is optional

	// If QUIC is not supported, there should be a reason
	if !result.SupportsQUIC {
		suite.NotEmpty(result.UnsupportedReason)
	}
}

func (suite *SystemHandlerSuite) TestRestartHandler_Success() {
	// RestartHandler triggers overseer.Restart() which is a global function
	// In a test, we can't actually restart, but we can verify the handler responds correctly
	resp := suite.testAPI.Put("/restart", struct{}{})

	// The handler returns nil, nil which Huma treats as 204 No Content
	suite.Equal(http.StatusNoContent, resp.Code)
}
