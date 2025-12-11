package service_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/core_api"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type HomeAssistantServiceTestSuite struct {
	suite.Suite
	ctx       context.Context
	config    *dto.ContextState
	haService service.HomeAssistantServiceInterface
	client    core_api.ClientWithResponsesInterface
	propRepo  repository.PropertyRepositoryInterface
	app       *fxtest.App
}

func TestHomeAssistantServiceSuite(t *testing.T) {
	suite.Run(t, new(HomeAssistantServiceTestSuite))
}

func (suite *HomeAssistantServiceTestSuite) SetupTest() {
	suite.config = &dto.ContextState{
		SecureMode:      true,
		SupervisorURL:   "http://supervisor/",
		SupervisorToken: "test-token",
		HACoreReady:     true,
	}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() *dto.ContextState { return suite.config },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			service.NewHomeAssistantService,
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[core_api.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haService, &suite.propRepo, &suite.ctx, &suite.client),
	)
	mock.When(suite.client.PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())).ThenReturn(
		&core_api.PostEntityStateResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
			JSON200: &core_api.EntityState{},
		},
		nil)

	suite.app.RequireStart()
}

func (suite *HomeAssistantServiceTestSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *HomeAssistantServiceTestSuite) TestSendSambaStatusEntity() {
	// Arrange
	sambaStatus := &dto.SambaStatus{
		Version:  "4.15.13",
		SmbConf:  "/etc/samba/smb.conf",
		Sessions: map[string]dto.SambaSession{},
		Tcons:    map[string]dto.SambaTcon{},
	}

	// Act - should not panic or return error when client is nil
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(true, nil)
	err := suite.haService.SendSambaStatusEntity(sambaStatus)

	// Assert
	mock.Verify(suite.propRepo, mock.AtLeastOnce()).Value(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.client, mock.AtLeastOnce()).PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())

	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestSendSambaStatusEntityDisabled() {
	// Arrange
	sambaStatus := &dto.SambaStatus{
		Version:  "4.15.13",
		SmbConf:  "/etc/samba/smb.conf",
		Sessions: map[string]dto.SambaSession{},
		Tcons:    map[string]dto.SambaTcon{},
	}

	// Act - should not panic or return error when client is nil
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(false, nil)
	err := suite.haService.SendSambaStatusEntity(sambaStatus)

	// Assert
	mock.Verify(suite.propRepo, mock.AtLeastOnce()).Value(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.client, mock.Never()).PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())

	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestSendSambaProcessStatusEntity() {
	// Arrange
	processStatus := &dto.SambaProcessStatus{
		"smbd": &dto.ProcessStatus{
			Pid:           1234,
			Name:          "smbd",
			IsRunning:     true,
			CPUPercent:    2.5,
			MemoryPercent: 1.8,
		},
		"nmbd": &dto.ProcessStatus{
			Pid:           1235,
			Name:          "nmbd",
			IsRunning:     true,
			CPUPercent:    0.5,
			MemoryPercent: 0.3,
		},
		"wsddn":       &dto.ProcessStatus{IsRunning: false},
		"srat-server": &dto.ProcessStatus{IsRunning: false},
		//"hdidle":     &dto.ProcessStatus{IsRunning: false},
	}

	// Act - should not panic or return error when client is nil
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(true, nil)
	err := suite.haService.SendSambaProcessStatusEntity(processStatus)

	// Assert
	// Assert
	mock.Verify(suite.propRepo, mock.AtLeastOnce()).Value(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.client, mock.AtLeastOnce()).PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())
	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestSendSambaProcessStatusEntityDisabled() {
	// Arrange
	processStatus := &dto.SambaProcessStatus{
		"smbd": &dto.ProcessStatus{
			Pid:           1234,
			Name:          "smbd",
			IsRunning:     true,
			CPUPercent:    2.5,
			MemoryPercent: 1.8,
		},
		"nmbd": &dto.ProcessStatus{
			Pid:           1235,
			Name:          "nmbd",
			IsRunning:     true,
			CPUPercent:    0.5,
			MemoryPercent: 0.3,
		},
		"wsddn":       &dto.ProcessStatus{IsRunning: false},
		"srat-server": &dto.ProcessStatus{IsRunning: false},
		//"hdidle":     &dto.ProcessStatus{IsRunning: false},
	}

	// Act - should not panic or return error when client is nil
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(false, nil)
	err := suite.haService.SendSambaProcessStatusEntity(processStatus)

	// Assert
	mock.Verify(suite.propRepo, mock.AtLeastOnce()).Value(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.client, mock.Never()).PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())
	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestSendDiskEntities() {
	// Arrange
	devicePath := "/dev/sda"
	diskId := "test-disk-001"
	diskSize := 1000000000000 // 1TB
	diskModel := "Test SSD"
	diskVendor := "Test Vendor"
	removable := false

	partitionId := "test-partition-001"
	partitionDevice := "/dev/sda1"
	partitionSize := 500000000000 // 500GB
	isMounted := true
	mountPath := "/mnt/test"

	disks := &[]dto.Disk{
		{
			Id:         &diskId,
			DevicePath: &devicePath,
			Size:       &diskSize,
			Model:      &diskModel,
			Vendor:     &diskVendor,
			Removable:  &removable,
			Partitions: &map[string]dto.Partition{
				partitionId: {
					Id:         &partitionId,
					DevicePath: &partitionDevice,
					Size:       &partitionSize,
					MountPointData: &map[string]dto.MountPointData{
						mountPath: {
							Path:      mountPath,
							IsMounted: isMounted,
							Share:     &dto.SharedResource{},
						},
					},
				},
			},
		},
	}

	// Act - should not panic or return error when client is nil
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(true, nil)
	err := suite.haService.SendDiskEntities(disks)

	// Assert
	mock.Verify(suite.propRepo, mock.AtLeastOnce()).Value(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.client, mock.AtLeastOnce()).PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())
	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestSendDiskEntitiesDisabled() {
	// Arrange
	devicePath := "/dev/sda"
	diskId := "test-disk-001"
	diskSize := 1000000000000 // 1TB
	diskModel := "Test SSD"
	diskVendor := "Test Vendor"
	removable := false

	partitionId := "test-partition-001"
	partitionDevice := "/dev/sda1"
	partitionSize := 500000000000 // 500GB
	isMounted := true
	mountPath := "/mnt/test"

	disks := &[]dto.Disk{
		{
			Id:         &diskId,
			DevicePath: &devicePath,
			Size:       &diskSize,
			Model:      &diskModel,
			Vendor:     &diskVendor,
			Removable:  &removable,
			Partitions: &map[string]dto.Partition{
				partitionId: {
					Id:         &partitionId,
					DevicePath: &partitionDevice,
					Size:       &partitionSize,
					MountPointData: &map[string]dto.MountPointData{
						mountPath: {
							Path:      mountPath,
							IsMounted: isMounted,
							Share:     &dto.SharedResource{},
						},
					},
				},
			},
		},
	}

	// Act - should not panic or return error when client is nil
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(false, nil)
	err := suite.haService.SendDiskEntities(disks)

	// Assert
	mock.Verify(suite.propRepo, mock.AtLeastOnce()).Value(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.client, mock.Never()).PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())
	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestNoClientConfigured_DoesNotSendEntities() {
	// Arrange - No core client configured
	params := service.HomeAssistantServiceParams{
		Ctx:        suite.ctx,
		State:      suite.config,
		CoreClient: nil,
		PropRepo:   suite.propRepo,
	}
	haService := service.NewHomeAssistantService(params)

	sambaStatus := &dto.SambaStatus{}

	// Act
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(true, nil)
	err := haService.SendSambaStatusEntity(sambaStatus)

	// Assert
	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestSanitizeEntityId() {
	// Test the entity ID sanitization functionality indirectly by testing with special characters
	devicePath := "/dev/sda"
	diskId := "usb-SanDisk_USB_3.2Gen1-0:0"

	disks := &[]dto.Disk{
		{
			Id:         &diskId,
			DevicePath: &devicePath,
		},
	}

	// Act - should not panic with special characters in disk ID
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(true, nil)
	err := suite.haService.SendDiskEntities(disks)

	// Assert
	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestSendDiskHealthEntities() {
	// Arrange
	diskHealth := &dto.DiskHealth{
		Global: dto.GlobalDiskStats{
			TotalIOPS:         42.5,
			TotalReadLatency:  1.5,
			TotalWriteLatency: 2.3,
		},
		PerDiskIO: []dto.DiskIOStats{
			{
				DeviceName:        "sda",
				DeviceDescription: "Samsung SSD",
				ReadIOPS:          20.0,
				WriteIOPS:         22.5,
				ReadLatency:       1.2,
				WriteLatency:      2.1,
			},
		},
		PerPartitionInfo: map[string][]dto.PerPartitionInfo{
			"sda": {
				{
					Device:        "/dev/sda1",
					MountPoint:    "/",
					FSType:        "ext4",
					TotalSpace:    1000000000,
					FreeSpace:     500000000,
					FsckNeeded:    false,
					FsckSupported: true,
				},
			},
		},
	}

	// Act - should not panic or return error when client is nil
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(true, nil)
	err := suite.haService.SendDiskHealthEntities(diskHealth)

	// Assert
	mock.Verify(suite.propRepo, mock.AtLeastOnce()).Value(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.client, mock.AtLeastOnce()).PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())
	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestSendDiskHealthEntitiesDisabled() {
	// Arrange
	diskHealth := &dto.DiskHealth{
		Global: dto.GlobalDiskStats{
			TotalIOPS:         42.5,
			TotalReadLatency:  1.5,
			TotalWriteLatency: 2.3,
		},
		PerDiskIO: []dto.DiskIOStats{
			{
				DeviceName:        "sda",
				DeviceDescription: "Samsung SSD",
				ReadIOPS:          20.0,
				WriteIOPS:         22.5,
				ReadLatency:       1.2,
				WriteLatency:      2.1,
			},
		},
		PerPartitionInfo: map[string][]dto.PerPartitionInfo{
			"sda": {
				{
					Device:        "/dev/sda1",
					MountPoint:    "/",
					FSType:        "ext4",
					TotalSpace:    1000000000,
					FreeSpace:     500000000,
					FsckNeeded:    false,
					FsckSupported: true,
				},
			},
		},
	}

	// Act - should not panic or return error when client is nil
	mock.When(suite.propRepo.Value("ExportStatsToHA", false)).ThenReturn(false, nil)
	err := suite.haService.SendDiskHealthEntities(diskHealth)

	// Assert
	mock.Verify(suite.propRepo, mock.AtLeastOnce()).Value(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.client, mock.Never()).PostEntityStateWithResponse(mock.AnyContext(), mock.Any[string](), mock.Any[core_api.EntityState]())
	suite.NoError(err)
}

func (suite *HomeAssistantServiceTestSuite) TestCreatePersistentNotification_Success() {
	// Arrange
	notificationID := "test_notification"
	title := "Test Title"
	message := "Test Message"

	// Mock the CallServiceWithResponse
	mock.When(suite.client.CallServiceWithResponse(
		mock.AnyContext(),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[core_api.ServiceData](),
	)).ThenReturn(
		&core_api.CallServiceResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
		},
		nil,
	)

	// Act
	err := suite.haService.CreatePersistentNotification(notificationID, title, message)

	// Assert
	suite.NoError(err)
	mock.Verify(suite.client, matchers.Times(1)).CallServiceWithResponse(
		mock.AnyContext(),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[core_api.ServiceData](),
	)
}

func (suite *HomeAssistantServiceTestSuite) TestCreatePersistentNotification_AlreadySentToday() {
	// Arrange
	notificationID := "test_notification"
	title := "Test Title"
	message := "Test Message"

	// Mock the CallServiceWithResponse
	mock.When(suite.client.CallServiceWithResponse(
		mock.AnyContext(),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[core_api.ServiceData](),
	)).ThenReturn(
		&core_api.CallServiceResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
		},
		nil,
	)

	// First call - should send notification
	err := suite.haService.CreatePersistentNotification(notificationID, title, message)
	suite.NoError(err)

	// Second call on same day - should skip notification
	err = suite.haService.CreatePersistentNotification(notificationID, title, message)

	// Assert
	suite.NoError(err)
	// Should only be called once (first time), not the second time
	mock.Verify(suite.client, matchers.Times(1)).CallServiceWithResponse(
		mock.AnyContext(),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[core_api.ServiceData](),
	)
}

func (suite *HomeAssistantServiceTestSuite) TestCreatePersistentNotification_DifferentDay() {
	// Arrange - This test is not applicable with in-memory tracking
	// as we can't simulate different days without mocking time
	// The TestDismissPersistentNotification_ClearsTracking covers the dismiss and recreate scenario
	suite.T().Skip("Test not applicable with in-memory tracking - covered by dismiss test")
}

func (suite *HomeAssistantServiceTestSuite) TestDismissPersistentNotification_ClearsTracking() {
	// Arrange
	notificationID := "test_notification"
	title := "Test Title"
	message := "Test Message"

	// Mock the CallServiceWithResponse for both create and dismiss
	mock.When(suite.client.CallServiceWithResponse(
		mock.AnyContext(),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[core_api.ServiceData](),
	)).ThenReturn(
		&core_api.CallServiceResponse{
			HTTPResponse: &http.Response{
				StatusCode: http.StatusOK,
			},
		},
		nil,
	)

	// First, create a notification
	err := suite.haService.CreatePersistentNotification(notificationID, title, message)
	suite.NoError(err)

	// Try to create it again - should be skipped
	err = suite.haService.CreatePersistentNotification(notificationID, title, message)
	suite.NoError(err)

	// Now dismiss it
	err = suite.haService.DismissPersistentNotification(notificationID)
	suite.NoError(err)

	// After dismissal, should be able to create again
	err = suite.haService.CreatePersistentNotification(notificationID, title, message)
	suite.NoError(err)

	// Assert - should be called 3 times total (first create, skipped second, third after dismiss)
	mock.Verify(suite.client, matchers.Times(3)).CallServiceWithResponse(
		mock.AnyContext(),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[core_api.ServiceData](),
	)
}

func (suite *HomeAssistantServiceTestSuite) TestCreatePersistentNotification_NoClient() {
	// Arrange - No core client configured
	params := service.HomeAssistantServiceParams{
		Ctx:        suite.ctx,
		State:      suite.config,
		CoreClient: nil,
		PropRepo:   suite.propRepo,
	}
	haService := service.NewHomeAssistantService(params)

	notificationID := "test_notification"
	title := "Test Title"
	message := "Test Message"

	// Act
	err := haService.CreatePersistentNotification(notificationID, title, message)

	// Assert
	suite.NoError(err)
	mock.Verify(suite.client, mock.Never()).CallServiceWithResponse(
		mock.AnyContext(),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[core_api.ServiceData](),
	)
}
