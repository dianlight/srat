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
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SmartHandlerSuite struct {
	suite.Suite
	app           *fxtest.App
	handler       *api.SmartHandler
	mockSmartSvc  service.SmartServiceInterface
	mockVolumeSvc service.VolumeServiceInterface
	mockDirtySvc  service.DirtyDataServiceInterface
	mockBroadSvc  service.BroadcasterServiceInterface
	ctx           context.Context
	cancel        context.CancelFunc
}

func TestSmartHandlerSuite(t *testing.T) { suite.Run(t, new(SmartHandlerSuite)) }

func (suite *SmartHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			api.NewSmartHandler,
			mock.Mock[service.SmartServiceInterface],
			mock.Mock[service.VolumeServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
			mock.Mock[service.BroadcasterServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockSmartSvc),
		fx.Populate(&suite.mockVolumeSvc),
		fx.Populate(&suite.mockDirtySvc),
		fx.Populate(&suite.mockBroadSvc),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *SmartHandlerSuite) TearDownTest() {
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

func (suite *SmartHandlerSuite) TestGetSmartInfoSuccess() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}
	smartInfo := &dto.SmartInfo{
		DiskType: "SATA",
		Temperature: dto.SmartTempValue{
			Value: 45,
			Min:   30,
			Max:   50,
		},
		PowerOnHours: dto.SmartRangeValue{
			Value: 1000,
		},
		PowerCycleCount: dto.SmartRangeValue{
			Value: 100,
		},
	}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)
	mock.When(suite.mockSmartSvc.GetSmartInfo(devicePath)).ThenReturn(smartInfo, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Get("/disk/sda/smart/info")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.SmartInfo
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("SATA", out.DiskType)
	suite.Equal(45, out.Temperature.Value)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
	mock.Verify(suite.mockSmartSvc, matchers.Times(1)).GetSmartInfo(devicePath)
}

func (suite *SmartHandlerSuite) TestGetSmartInfoNotSupported() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)
	mock.When(suite.mockSmartSvc.GetSmartInfo(devicePath)).ThenReturn(nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Get("/disk/sda/smart/info")
	suite.Require().Equal(http.StatusNotAcceptable, resp.Code)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
	mock.Verify(suite.mockSmartSvc, matchers.Times(1)).GetSmartInfo(devicePath)
}

func (suite *SmartHandlerSuite) TestGetSmartHealthSuccess() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}
	health := &dto.SmartHealthStatus{
		Passed:            true,
		OverallStatus:     "healthy",
		FailingAttributes: []string{},
	}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)
	mock.When(suite.mockSmartSvc.GetHealthStatus(devicePath)).ThenReturn(health, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Get("/disk/sda/smart/health")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.SmartHealthStatus
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.True(out.Passed)
	suite.Equal("healthy", out.OverallStatus)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
	mock.Verify(suite.mockSmartSvc, matchers.Times(1)).GetHealthStatus(devicePath)
}

func (suite *SmartHandlerSuite) TestGetSmartTestStatusSuccess() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}
	testStatus := &dto.SmartTestStatus{
		Status:          "idle",
		TestType:        "",
		PercentComplete: 0,
	}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)
	mock.When(suite.mockSmartSvc.GetTestStatus(devicePath)).ThenReturn(testStatus, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Get("/disk/sda/smart/test")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.SmartTestStatus
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal("idle", out.Status)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
	mock.Verify(suite.mockSmartSvc, matchers.Times(1)).GetTestStatus(devicePath)
}

func (suite *SmartHandlerSuite) TestStartSmartTestSuccess() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)
	mock.When(suite.mockSmartSvc.StartSelfTest(devicePath, dto.SmartTestTypeShort)).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Post("/disk/sda/smart/test/start", map[string]any{
		"test_type": "short",
	})
	suite.Require().Equal(http.StatusOK, resp.Code)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
	mock.Verify(suite.mockSmartSvc, matchers.Times(1)).StartSelfTest(devicePath, dto.SmartTestTypeShort)
	mock.Verify(suite.mockDirtySvc, matchers.Times(1)).SetDirtyVolumes()
}

func (suite *SmartHandlerSuite) TestAbortSmartTestSuccess() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)
	mock.When(suite.mockSmartSvc.AbortSelfTest(devicePath)).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Post("/disk/sda/smart/test/abort", map[string]any{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
	mock.Verify(suite.mockSmartSvc, matchers.Times(1)).AbortSelfTest(devicePath)
	mock.Verify(suite.mockDirtySvc, matchers.Times(1)).SetDirtyVolumes()
}

func (suite *SmartHandlerSuite) TestEnableSmartSuccess() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)
	mock.When(suite.mockSmartSvc.EnableSMART(devicePath)).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Post("/disk/sda/smart/enable", map[string]any{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
	mock.Verify(suite.mockSmartSvc, matchers.Times(1)).EnableSMART(devicePath)
	mock.Verify(suite.mockDirtySvc, matchers.Times(1)).SetDirtyVolumes()
}

func (suite *SmartHandlerSuite) TestDisableSmartSuccess() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)
	mock.When(suite.mockSmartSvc.DisableSMART(devicePath)).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Post("/disk/sda/smart/disable", map[string]any{})
	suite.Require().Equal(http.StatusOK, resp.Code)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
	mock.Verify(suite.mockSmartSvc, matchers.Times(1)).DisableSMART(devicePath)
	mock.Verify(suite.mockDirtySvc, matchers.Times(1)).SetDirtyVolumes()
}

func (suite *SmartHandlerSuite) TestDiskNotFound() {
	disks := &[]dto.Disk{}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Get("/disk/unknown/smart/info")
	suite.Require().Equal(http.StatusNotFound, resp.Code)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
}

func (suite *SmartHandlerSuite) TestReadOnlyModeRejectsStartTest() {
	diskID := "sda"
	devicePath := "/dev/sda"
	disks := &[]dto.Disk{
		{
			Id:         &diskID,
			DevicePath: &devicePath,
		},
	}

	// Create handler with read-only mode enabled
	readOnlyHandler := api.NewSmartHandler(
		suite.mockSmartSvc,
		suite.mockVolumeSvc,
		&dto.ContextState{ReadOnlyMode: true},
		suite.mockDirtySvc,
		suite.mockBroadSvc,
	)

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks, nil)

	_, apiInst := humatest.New(suite.T())
	readOnlyHandler.RegisterSmartHandlers(apiInst)

	resp := apiInst.Post("/disk/sda/smart/test/start", map[string]any{
		"test_type": "short",
	})
	suite.Require().Equal(http.StatusForbidden, resp.Code)
}
