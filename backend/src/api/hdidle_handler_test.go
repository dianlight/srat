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
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type HDIdleHandlerSuite struct {
	suite.Suite
	app               *fxtest.App
	handler           *api.HDIdleHandler
	mockHDIdleService service.HDIdleServiceInterface
	ctx               context.Context
	cancel            context.CancelFunc
}

func TestHDIdleHandlerSuite(t *testing.T) { suite.Run(t, new(HDIdleHandlerSuite)) }

func (suite *HDIdleHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			api.NewHDIdleHandler,
			mock.Mock[service.HDIdleServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockHDIdleService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *HDIdleHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value("wg").(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// =============================================================================
// GET /disk/{disk_id}/hdidle/config Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestGetConfigSuccess() {
	diskID := "sda"
	expectedConfig := &dto.HDIdleDeviceDTO{
		DevicePath:     diskID,
		IdleTime:       300,
		CommandType:    dto.HdidleCommands.SCSICOMMAND,
		PowerCondition: 0,
		Enabled:        dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(expectedConfig, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.HDIdleDeviceDTO
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal(diskID, out.DevicePath)
	suite.Equal(300, out.IdleTime)
	suite.Equal(dto.HdidleEnableds.YESENABLED, out.Enabled)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetDeviceConfig(mock.Substring(diskID))
}

func (suite *HDIdleHandlerSuite) TestGetConfigError() {
	diskID := "sda"

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(nil, errors.New("database error"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetDeviceConfig(mock.Substring(diskID))
}

// =============================================================================
// PUT /disk/{disk_id}/hdidle/config Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestPutConfigSuccess() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath:     diskID,
		IdleTime:       300,
		CommandType:    dto.HdidleCommands.SCSICOMMAND,
		PowerCondition: 0,
		Enabled:        dto.HdidleEnableds.YESENABLED,
	}
	inputConfig := dto.HDIdleDeviceDTO{
		DevicePath:     diskID,
		IdleTime:       600,
		CommandType:    dto.HdidleCommands.ATACOMMAND,
		PowerCondition: 1,
		Enabled:        dto.HdidleEnableds.CUSTOMENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(false)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/sda/hdidle/config", inputConfig)
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.HDIdleDeviceDTO
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal(600, out.IdleTime)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetDeviceConfig(mock.Substring(diskID))
	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())
	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Start()
}

func (suite *HDIdleHandlerSuite) TestPutConfigWithRestartSuccess() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}
	inputConfig := dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   600,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(true)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/sda/hdidle/config", inputConfig)
	suite.Require().Equal(http.StatusOK, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Stop()
	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Start()
}

func (suite *HDIdleHandlerSuite) TestPutConfigDevicePathMismatch() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}
	inputConfig := dto.HDIdleDeviceDTO{
		DevicePath: "sdb", // Mismatched path
		IdleTime:   600,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/sda/hdidle/config", inputConfig)
	suite.Require().Equal(http.StatusBadRequest, resp.Code)
}

func (suite *HDIdleHandlerSuite) TestPutConfigStopError() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}
	inputConfig := dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   600,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(true)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(errors.New("stop failed"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/sda/hdidle/config", inputConfig)
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Stop()
}

func (suite *HDIdleHandlerSuite) TestPutConfigSaveError() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}
	inputConfig := dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   600,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(errors.New("save failed"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/sda/hdidle/config", inputConfig)
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())
}

func (suite *HDIdleHandlerSuite) TestPutConfigStartError() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}
	inputConfig := dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   600,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(false)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(errors.New("start failed"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Put("/disk/sda/hdidle/config", inputConfig)
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Start()
}

// =============================================================================
// GET /disk/{disk_id}/hdidle/info Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestGetStatusSuccess() {
	diskID := "sda"
	expectedStatus := &service.HDIdleDiskStatus{
		Name:           "sda",
		GivenName:      diskID,
		SpunDown:       false,
		LastIOAt:       time.Now(),
		IdleTime:       5 * time.Minute,
		CommandType:    dto.HdidleCommands.SCSICOMMAND,
		PowerCondition: 0,
	}

	mock.When(suite.mockHDIdleService.GetDeviceStatus(mock.Substring(diskID))).ThenReturn(expectedStatus, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/info")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out service.HDIdleDiskStatus
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("sda", out.Name)
	suite.False(out.SpunDown)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetDeviceStatus(mock.Substring(diskID))
}

func (suite *HDIdleHandlerSuite) TestGetStatusError() {
	diskID := "sda"

	mock.When(suite.mockHDIdleService.GetDeviceStatus(mock.Substring(diskID))).ThenReturn(nil, errors.New("disk not found"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/info")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetDeviceStatus(mock.Substring(diskID))
}

// =============================================================================
// GET /hdidle/status Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestGetServiceStatusSuccess() {
	expectedStatus := &service.HDIdleStatus{
		Running:     true,
		MonitoredAt: time.Now(),
		Disks: []service.HDIdleDiskStatus{
			{
				Name:      "sda",
				GivenName: "sda",
				SpunDown:  false,
			},
			{
				Name:      "sdb",
				GivenName: "sdb",
				SpunDown:  true,
			},
		},
	}

	mock.When(suite.mockHDIdleService.GetStatus()).ThenReturn(expectedStatus, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/hdidle/status")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out service.HDIdleStatus
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.True(out.Running)
	suite.Len(out.Disks, 2)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetStatus()
}

func (suite *HDIdleHandlerSuite) TestGetServiceStatusNotRunning() {
	expectedStatus := &service.HDIdleStatus{
		Running: false,
	}

	mock.When(suite.mockHDIdleService.GetStatus()).ThenReturn(expectedStatus, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/hdidle/status")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out service.HDIdleStatus
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.False(out.Running)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetStatus()
}

func (suite *HDIdleHandlerSuite) TestGetServiceStatusError() {
	mock.When(suite.mockHDIdleService.GetStatus()).ThenReturn(nil, errors.New("service error"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/hdidle/status")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetStatus()
}

// =============================================================================
// GET /hdidle/effective-config Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestGetEffectiveConfigEnabled() {
	expectedConfig := service.HDIdleEffectiveConfig{
		Enabled: true,
		Devices: []string{"sda", "sdb"},
	}

	mock.When(suite.mockHDIdleService.GetEffectiveConfig()).ThenReturn(expectedConfig)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/hdidle/effective-config")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out service.HDIdleEffectiveConfig
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.True(out.Enabled)
	suite.Len(out.Devices, 2)
	suite.Contains(out.Devices, "sda")
	suite.Contains(out.Devices, "sdb")

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetEffectiveConfig()
}

func (suite *HDIdleHandlerSuite) TestGetEffectiveConfigDisabled() {
	expectedConfig := service.HDIdleEffectiveConfig{
		Enabled: false,
		Devices: []string{},
	}

	mock.When(suite.mockHDIdleService.GetEffectiveConfig()).ThenReturn(expectedConfig)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/hdidle/effective-config")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out service.HDIdleEffectiveConfig
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.False(out.Enabled)
	suite.Empty(out.Devices)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetEffectiveConfig()
}

// =============================================================================
// POST /hdidle/start Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestStartServiceSuccess() {
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(false)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/hdidle/start", struct{}{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out struct {
		Message string `json:"message"`
		Running bool   `json:"running"`
	}
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("HDIdle service started successfully", out.Message)
	suite.True(out.Running)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).IsRunning()
	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Start()
}

func (suite *HDIdleHandlerSuite) TestStartServiceAlreadyRunning() {
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(true)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/hdidle/start", struct{}{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out struct {
		Message string `json:"message"`
		Running bool   `json:"running"`
	}
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("HDIdle service is already running", out.Message)
	suite.True(out.Running)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).IsRunning()
	mock.Verify(suite.mockHDIdleService, matchers.Times(0)).Start()
}

func (suite *HDIdleHandlerSuite) TestStartServiceError() {
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(false)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(errors.New("failed to start"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/hdidle/start", struct{}{})
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Start()
}

// =============================================================================
// POST /hdidle/stop Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestStopServiceSuccess() {
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(true)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/hdidle/stop", struct{}{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out struct {
		Message string `json:"message"`
		Running bool   `json:"running"`
	}
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("HDIdle service stopped successfully", out.Message)
	suite.False(out.Running)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).IsRunning()
	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Stop()
}

func (suite *HDIdleHandlerSuite) TestStopServiceNotRunning() {
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(false)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/hdidle/stop", struct{}{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out struct {
		Message string `json:"message"`
		Running bool   `json:"running"`
	}
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("HDIdle service is not running", out.Message)
	suite.False(out.Running)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).IsRunning()
	mock.Verify(suite.mockHDIdleService, matchers.Times(0)).Stop()
}

func (suite *HDIdleHandlerSuite) TestStopServiceError() {
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(true)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(errors.New("failed to stop"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Post("/hdidle/stop", struct{}{})
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Stop()
}

// =============================================================================
// DELETE /disk/{disk_id}/hdidle/config Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestDeleteConfigSuccess() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath:     diskID,
		IdleTime:       300,
		CommandType:    dto.HdidleCommands.SCSICOMMAND,
		PowerCondition: 1,
		Enabled:        dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(false)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Delete("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out struct {
		Message string `json:"message"`
	}
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("HDIdle configuration deleted successfully", out.Message)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetDeviceConfig(mock.Substring(diskID))
	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())
}

func (suite *HDIdleHandlerSuite) TestDeleteConfigWithRestartSuccess() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(true)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Delete("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusOK, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Stop()
	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Start()
}

func (suite *HDIdleHandlerSuite) TestDeleteConfigGetError() {
	diskID := "sda"

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(nil, errors.New("not found"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Delete("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).GetDeviceConfig(mock.Substring(diskID))
}

func (suite *HDIdleHandlerSuite) TestDeleteConfigSaveError() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(errors.New("save failed"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Delete("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())
}

func (suite *HDIdleHandlerSuite) TestDeleteConfigStopError() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(true)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(errors.New("stop failed"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Delete("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Stop()
}

func (suite *HDIdleHandlerSuite) TestDeleteConfigStartError() {
	diskID := "sda"
	existingConfig := &dto.HDIdleDeviceDTO{
		DevicePath: diskID,
		IdleTime:   300,
		Enabled:    dto.HdidleEnableds.YESENABLED,
	}

	mock.When(suite.mockHDIdleService.GetDeviceConfig(mock.Substring(diskID))).ThenReturn(existingConfig, nil)
	mock.When(suite.mockHDIdleService.SaveDeviceConfig(mock.Any[dto.HDIdleDeviceDTO]())).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.IsRunning()).ThenReturn(true)
	mock.When(suite.mockHDIdleService.Stop()).ThenReturn(nil)
	mock.When(suite.mockHDIdleService.Start()).ThenReturn(errors.New("start failed"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Delete("/disk/sda/hdidle/config")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).Start()
}

// =============================================================================
// GET /disk/{disk_id}/hdidle/support Tests
// =============================================================================

func (suite *HDIdleHandlerSuite) TestCheckSupportSuccess() {
	diskID := "sda"
	recommendedCmd := dto.HdidleCommands.ATACOMMAND
	expectedSupport := &service.HDIdleDeviceSupport{
		Supported:          true,
		SupportsSCSI:       true,
		SupportsATA:        true,
		RecommendedCommand: &recommendedCmd,
		DevicePath:         "/dev/sda",
		ErrorMessage:       "",
	}

	mock.When(suite.mockHDIdleService.CheckDeviceSupport(mock.Substring(diskID))).ThenReturn(expectedSupport, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/support")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out service.HDIdleDeviceSupport
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.True(out.Supported)
	suite.True(out.SupportsSCSI)
	suite.True(out.SupportsATA)
	suite.Equal("/dev/sda", out.DevicePath)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).CheckDeviceSupport(mock.Substring(diskID))
}

func (suite *HDIdleHandlerSuite) TestCheckSupportNotSupported() {
	diskID := "sda"
	expectedSupport := &service.HDIdleDeviceSupport{
		Supported:    false,
		SupportsSCSI: false,
		SupportsATA:  false,
		DevicePath:   "/dev/sda",
		ErrorMessage: "device does not support SG interface",
	}

	mock.When(suite.mockHDIdleService.CheckDeviceSupport(mock.Substring(diskID))).ThenReturn(expectedSupport, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/support")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out service.HDIdleDeviceSupport
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.False(out.Supported)
	suite.Equal("device does not support SG interface", out.ErrorMessage)

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).CheckDeviceSupport(mock.Substring(diskID))
}

func (suite *HDIdleHandlerSuite) TestCheckSupportError() {
	diskID := "sda"

	mock.When(suite.mockHDIdleService.CheckDeviceSupport(mock.Substring(diskID))).ThenReturn(nil, errors.New("device error"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterHDIdleHandler(apiInst)

	resp := apiInst.Get("/disk/sda/hdidle/support")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code, "Returned body: %s", resp.Body.String())

	mock.Verify(suite.mockHDIdleService, matchers.Times(1)).CheckDeviceSupport(mock.Substring(diskID))
}
