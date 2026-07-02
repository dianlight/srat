package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// HDIdleHandlerSuite covers the HTTP layer of the HDIdle subsystem after the
// per-disk-only refactor (Phase 1-3):
//   - all routes are now Lab-Mode-gated (403 when off)
//   - public POST /hdidle/start and /hdidle/stop are gone (the service auto-
//     drives from the per-disk records)
//   - PATCH /hdidle/config is gone (was a dead-spec entry)
//   - new POST /disk/{id}/hdidle/ignore-suggestion endpoint
//   - PUT now relies on ResolveDevicePath (404 when the id is not a real disk)
//     and 409s on a non-rotational target without force_enabled
type HDIdleHandlerSuite struct {
	suite.Suite
	app                 *fxtest.App
	handler             *api.HDIdleHandler
	mockHDIdleService   service.HDIdleServiceInterface
	mockHardwareService service.HardwareServiceInterface
	mockSettingService  service.SettingServiceInterface
	ctx                 context.Context
	cancel              context.CancelFunc
}

func TestHDIdleHandlerSuite(t *testing.T) { suite.Run(t, new(HDIdleHandlerSuite)) }

func (suite *HDIdleHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			api.NewHDIdleHandler,
			mock.Mock[service.HDIdleServiceInterface],
			mock.Mock[service.HardwareServiceInterface],
			mock.Mock[service.SettingServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockHDIdleService),
		fx.Populate(&suite.mockHardwareService),
		fx.Populate(&suite.mockSettingService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

// labModeOn registers the "lab mode enabled" stub for tests that require it.
// Called explicitly in every test that exercises a lab-mode-gated endpoint.
// NOT called in TestEndpointsRequireLabMode, which tests the 403 path.
func (suite *HDIdleHandlerSuite) labModeOn() {
	mock.When(suite.mockSettingService.Load()).ThenReturn(&dto.Settings{
		ExperimentalLabMode: true,
	}, nil)
}

func (suite *HDIdleHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// rotationalDisk returns a stub Disk dto with is_rotational=true. The PUT
// 409-non-rotational guard delegates to HardwareService.GetHardwareInfo;
// most tests want a "this is a real HDD" answer, which is what this gives.
func rotationalDisk(diskID string) map[string]dto.Disk {
	t := true
	return map[string]dto.Disk{
		diskID: {Id: &diskID, IsRotational: &t},
	}
}

// =============================================================================
// Lab Mode gate
// =============================================================================

func (suite *HDIdleHandlerSuite) TestEndpointsRequireLabMode() {
	// labModeOn() is intentionally NOT called here.
	// The SetupTest default was removed, so Load() has no registered answer.
	// We register lab mode = off; mockio sticks on this answer for all calls.
	mock.When(suite.mockSettingService.Load()).ThenReturn(&dto.Settings{
		ExperimentalLabMode: false,
	}, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	for _, path := range []string{
		"/disk/sda/hdidle/config",
		"/disk/sda/hdidle/info",
		"/disk/sda/hdidle/support",
	} {
		resp := apiInst.Get(path)
		suite.Equal(http.StatusForbidden, resp.Code, "%s should 403 when lab mode is off", path)
	}

	// Mutating endpoints
	resp := apiInst.Put("/disk/sda/hdidle/config", dto.HDIdleDevice{})
	suite.Equal(http.StatusForbidden, resp.Code, "PUT /config should 403")
	resp = apiInst.Post("/disk/sda/hdidle/ignore-suggestion", struct{}{})
	suite.Equal(http.StatusForbidden, resp.Code, "POST /ignore-suggestion should 403")
}

// =============================================================================
// GET /disk/{disk_id}/hdidle/config
// =============================================================================

func (suite *HDIdleHandlerSuite) TestGetConfigSuccess() {
	suite.labModeOn()
	diskID := "sda"
	expected := &dto.HDIdleDevice{
		HDIdleDeviceSupport: dto.HDIdleDeviceSupport{DevicePath: "/dev/" + diskID},
		IdleTime:            time.Duration(300),
		CommandType:         dto.HdidleCommands.SCSICOMMAND,
		Enabled:             dto.HdidleEnableds.YESENABLED,
	}
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Any[string]())).ThenReturn(expected, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.HDIdleDevice
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("/dev/"+diskID, out.DevicePath)
	suite.Equal(dto.HdidleEnableds.YESENABLED, out.Enabled)
}

func (suite *HDIdleHandlerSuite) TestGetConfigUnknownDiskReturns404() {
	suite.labModeOn()
	mock.When(suite.mockHDIdleService.ResolveDevicePath("unknownid")).
		ThenReturn("", errors.Wrap(dto.ErrorNotFound, "no device"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/unknownid/hdidle/config")
	suite.Equal(http.StatusNotFound, resp.Code)
}

// =============================================================================
// PUT /disk/{disk_id}/hdidle/config
// =============================================================================

func (suite *HDIdleHandlerSuite) TestPutConfigSuccess() {
	suite.labModeOn()
	diskID := "sda"
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHardwareService.GetHardwareInfo()).ThenReturn(rotationalDisk(diskID), nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(nil)

	body := dto.HDIdleDevice{
		IdleTime: time.Duration(600),
		Enabled:  dto.HdidleEnableds.YESENABLED,
	}
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/sda/hdidle/config", body)
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Restart cycle is unconditional now (Stop is idempotent post-Phase-2).
	_ = mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Stop()
	_ = mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Start()
}

func (suite *HDIdleHandlerSuite) TestPutConfigUnknownDiskReturns404() {
	suite.labModeOn()
	mock.When(suite.mockHDIdleService.ResolveDevicePath("ghost")).
		ThenReturn("", errors.Wrap(dto.ErrorNotFound, "no device"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/ghost/hdidle/config", dto.HDIdleDevice{})
	suite.Equal(http.StatusNotFound, resp.Code)
}

func (suite *HDIdleHandlerSuite) TestPutConfigNonRotationalRequiresForce() {
	suite.labModeOn()
	diskID := "ssd0"
	f := false
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHardwareService.GetHardwareInfo()).ThenReturn(map[string]dto.Disk{
		diskID: {Id: &diskID, IsRotational: &f},
	}, nil)

	body := dto.HDIdleDevice{
		Enabled:      dto.HdidleEnableds.YESENABLED,
		ForceEnabled: false,
	}
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/ssd0/hdidle/config", body)
	suite.Equal(http.StatusConflict, resp.Code)

	// Service must not be touched when the conflict fires.
	_ = mock.Verify(suite.mockHDIdleService, matchers.Times(0)).SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())
}

func (suite *HDIdleHandlerSuite) TestPutConfigNonRotationalForcedSucceeds() {
	suite.labModeOn()
	diskID := "ssd0"
	f := false
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHardwareService.GetHardwareInfo()).ThenReturn(map[string]dto.Disk{
		diskID: {Id: &diskID, IsRotational: &f},
	}, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(nil)

	body := dto.HDIdleDevice{
		Enabled:      dto.HdidleEnableds.YESENABLED,
		ForceEnabled: true,
	}
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/ssd0/hdidle/config", body)
	suite.Equal(http.StatusOK, resp.Code)

	_ = mock.Verify(suite.mockHDIdleService, matchers.Times(1)).SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())
}

func (suite *HDIdleHandlerSuite) TestPutConfigDisableUnconditionallyAccepted() {
	suite.labModeOn()
	// Setting enabled=NOENABLED must succeed even on a non-rotational disk
	// without force_enabled — disabling is always allowed.
	diskID := "ssd0"
	f := false
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHardwareService.GetHardwareInfo()).ThenReturn(map[string]dto.Disk{
		diskID: {Id: &diskID, IsRotational: &f},
	}, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(nil)

	body := dto.HDIdleDevice{Enabled: dto.HdidleEnableds.NOENABLED}
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/ssd0/hdidle/config", body)
	suite.Equal(http.StatusOK, resp.Code)
}

// =============================================================================
// GET /disk/{disk_id}/hdidle/info
// =============================================================================

func (suite *HDIdleHandlerSuite) TestGetStatusSuccess() {
	suite.labModeOn()
	diskID := "sda"
	expected := &dto.HDIdleDeviceStatus{Name: diskID, SpunDown: false}
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHDIdleService.GetDeviceStatus(mock.Any[string]())).ThenReturn(expected, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/info")
	suite.Equal(http.StatusOK, resp.Code)
}

// =============================================================================
// GET /disk/{disk_id}/hdidle/support
// =============================================================================

func (suite *HDIdleHandlerSuite) TestCheckSupportSuccess() {
	suite.labModeOn()
	diskID := "sda"
	expected := &dto.HDIdleDeviceSupport{Supported: true, DevicePath: "/dev/" + diskID}
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHDIdleService.CheckDeviceSupport(mock.Any[string]())).ThenReturn(expected, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/support")
	suite.Equal(http.StatusOK, resp.Code)

	var out dto.HDIdleDeviceSupport
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.True(out.Supported)
}

// =============================================================================
// POST /disk/{disk_id}/hdidle/ignore-suggestion
// =============================================================================

func (suite *HDIdleHandlerSuite) TestIgnoreSuggestionPersists() {
	suite.labModeOn()
	diskID := "sda"
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Any[string]())).ThenReturn(&dto.HDIdleDevice{
		HDIdleDeviceSupport: dto.HDIdleDeviceSupport{DevicePath: "/dev/" + diskID},
		Enabled:             dto.HdidleEnableds.NOENABLED,
	}, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/disk/sda/hdidle/ignore-suggestion", struct{}{})
	suite.Equal(http.StatusOK, resp.Code)

	// SaveDeviceConfig must be called exactly once with SuggestionIgnored=true.
	_ = mock.Verify(suite.mockHDIdleService, matchers.Times(1)).SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())
}

func (suite *HDIdleHandlerSuite) TestIgnoreSuggestionPersistsWhenDeviceUnsupported() {
	suite.labModeOn()
	diskID := "sda"
	mock.When(suite.mockHDIdleService.ResolveDevicePath(diskID)).ThenReturn("/dev/"+diskID, nil)
	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Any[string]())).ThenReturn(&dto.HDIdleDevice{
		HDIdleDeviceSupport: dto.HDIdleDeviceSupport{DevicePath: "/dev/" + diskID},
		Enabled:             dto.HdidleEnableds.NOENABLED,
	}, errors.WithStack(dto.ErrorHDIdleNotSupported))
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/disk/sda/hdidle/ignore-suggestion", struct{}{})
	suite.Equal(http.StatusOK, resp.Code)

	// Even when the device is reported as unsupported, the dismissal must
	// still be persisted exactly once.
	_ = mock.Verify(suite.mockHDIdleService, matchers.Times(1)).SaveDeviceConfig(mock.Any[dto.HDIdleDevice]())
}

func (suite *HDIdleHandlerSuite) TestIgnoreSuggestionUnknownDiskReturns404() {
	suite.labModeOn()
	mock.When(suite.mockHDIdleService.ResolveDevicePath("ghost")).
		ThenReturn("", errors.Wrap(dto.ErrorNotFound, "no device"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/disk/ghost/hdidle/ignore-suggestion", struct{}{})
	suite.Equal(http.StatusNotFound, resp.Code)
}
