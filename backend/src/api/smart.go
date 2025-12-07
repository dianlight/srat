package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
)

type SmartHandler struct {
	apiContext    *dto.ContextState
	smartService  service.SmartServiceInterface
	volumeService service.VolumeServiceInterface
	//dirtyService    service.DirtyDataServiceInterface
	broadcasterServ service.BroadcasterServiceInterface
}

func NewSmartHandler(
	smartService service.SmartServiceInterface,
	volumeService service.VolumeServiceInterface,
	apiContext *dto.ContextState,
	//dirtyService service.DirtyDataServiceInterface,
	broadcasterServ service.BroadcasterServiceInterface,
) *SmartHandler {
	return &SmartHandler{
		apiContext:    apiContext,
		smartService:  smartService,
		volumeService: volumeService,
		//dirtyService:    dirtyService,
		broadcasterServ: broadcasterServ,
	}
}

// RegisterSmartHandlers registers the HTTP handlers for SMART-related operations.
// It sets up the following routes:
// - GET /disk/{disk_id}/smart/info: Get SMART information for a disk.
// - GET /disk/{disk_id}/smart/health: Get SMART health status for a disk.
// - GET /disk/{disk_id}/smart/test: Get SMART self-test status for a disk.
// - POST /disk/{disk_id}/smart/test/start: Start a SMART self-test.
// - POST /disk/{disk_id}/smart/test/abort: Abort a running SMART self-test.
// - POST /disk/{disk_id}/smart/enable: Enable SMART on a disk.
// - POST /disk/{disk_id}/smart/disable: Disable SMART on a disk.
//
// Parameters:
// - api: The huma.API instance to register the handlers with.
func (h *SmartHandler) RegisterSmartHandlers(api huma.API) {
	huma.Get(api, "/disk/{disk_id}/smart/info", h.GetSmartInfo, huma.OperationTags("disk"))
	huma.Get(api, "/disk/{disk_id}/smart/status", h.GetSmartStatus, huma.OperationTags("disk"))
	huma.Get(api, "/disk/{disk_id}/smart/health", h.GetSmartHealth, huma.OperationTags("disk"))
	huma.Get(api, "/disk/{disk_id}/smart/test", h.GetSmartTestStatus, huma.OperationTags("disk"))
	huma.Post(api, "/disk/{disk_id}/smart/test/start", h.StartSmartTest, huma.OperationTags("disk"))
	huma.Post(api, "/disk/{disk_id}/smart/test/abort", h.AbortSmartTest, huma.OperationTags("disk"))
	huma.Post(api, "/disk/{disk_id}/smart/enable", h.EnableSmart, huma.OperationTags("disk"))
	huma.Post(api, "/disk/{disk_id}/smart/disable", h.DisableSmart, huma.OperationTags("smart"))
}

// GetSmartInfo retrieves SMART information for a specific disk
func (h *SmartHandler) GetSmartInfo(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*struct{ Body *dto.SmartInfo }, error) {
	devicePath, errE := h.volumeService.GetDevicePathByDeviceID(input.DiskID)
	if errE != nil {
		return nil, huma.Error404NotFound("Disk not found", errors.New("disk not found"))
	}

	smartInfo, errE := h.smartService.GetSmartInfo(ctx, devicePath)
	if errE != nil {
		if errors.Is(errE, dto.ErrorSMARTNotSupported) {
			return nil, huma.Error406NotAcceptable("SMART not supported on this device", errE)
		}
		tlog.ErrorContext(ctx, "Failed to get SMART info", "device", devicePath, "error", errE)
		return nil, huma.Error500InternalServerError("Failed to get SMART info", errE)
	}

	return &struct{ Body *dto.SmartInfo }{Body: smartInfo}, nil
}

// GetSmartInfo retrieves SMART information for a specific disk
func (h *SmartHandler) GetSmartStatus(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*struct{ Body *dto.SmartStatus }, error) {
	devicePath, errE := h.volumeService.GetDevicePathByDeviceID(input.DiskID)
	if errE != nil {
		return nil, huma.Error404NotFound("Disk not found", errors.New("disk not found"))
	}

	smartStatus, errE := h.smartService.GetSmartStatus(ctx, devicePath)
	if errE != nil {
		if errors.Is(errE, dto.ErrorSMARTNotSupported) {
			return nil, huma.Error406NotAcceptable("SMART not supported on this device", errE)
		}
		tlog.ErrorContext(ctx, "Failed to get SMART status", "device", devicePath, "error", errE)
		return nil, huma.Error500InternalServerError("Failed to get SMART status", errE)
	}

	return &struct{ Body *dto.SmartStatus }{Body: smartStatus}, nil
}

// GetSmartHealth retrieves SMART health status for a specific disk
func (h *SmartHandler) GetSmartHealth(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*struct{ Body *dto.SmartHealthStatus }, error) {
	// Get disk info to find device path

	devicePath, errE := h.volumeService.GetDevicePathByDeviceID(input.DiskID)
	if errE != nil {
		return nil, huma.Error404NotFound("Disk not found", errors.New("disk not found"))
	}

	healthStatus, errE := h.smartService.GetHealthStatus(ctx, devicePath)
	if errE != nil {
		if errors.Is(errE, dto.ErrorSMARTNotSupported) {
			return nil, huma.Error406NotAcceptable("SMART not supported on this device", errE)
		}
		tlog.ErrorContext(ctx, "Failed to get SMART health status", "device", devicePath, "error", errE)
		return nil, huma.Error500InternalServerError("Failed to get SMART health status", errE)
	}

	return &struct{ Body *dto.SmartHealthStatus }{Body: healthStatus}, nil
}

// GetSmartTestStatus retrieves the status of a SMART self-test for a specific disk
func (h *SmartHandler) GetSmartTestStatus(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*struct{ Body *dto.SmartTestStatus }, error) {

	devicePath, errE := h.volumeService.GetDevicePathByDeviceID(input.DiskID)
	if errE != nil {
		return nil, huma.Error404NotFound("Disk not found", errors.New("disk not found"))
	}

	testStatus, errE := h.smartService.GetTestStatus(ctx, devicePath)
	if errE != nil {
		if errors.Is(errE, dto.ErrorSMARTNotSupported) {
			return nil, huma.Error406NotAcceptable("SMART not supported on this device", errE)
		}
		tlog.ErrorContext(ctx, "Failed to get SMART test status", "device", devicePath, "error", errE)
		return nil, huma.Error500InternalServerError("Failed to get SMART test status", errE)
	}

	return &struct{ Body *dto.SmartTestStatus }{Body: testStatus}, nil
}

// StartSmartTest starts a SMART self-test on a specific disk
func (h *SmartHandler) StartSmartTest(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
	Body   struct {
		TestType dto.SmartTestType `json:"test_type" required:"true" doc:"Type of test: short, long, or conveyance"`
	}
}) (*struct{ Body string }, error) {
	// Check read-only mode
	if h.apiContext.ReadOnlyMode {
		return nil, huma.Error403Forbidden("Read-only mode enabled", errors.New("read-only mode"))
	}

	devicePath, errE := h.volumeService.GetDevicePathByDeviceID(input.DiskID)
	if errE != nil {
		return nil, huma.Error404NotFound("Disk not found", errors.New("disk not found"))
	}

	// Start the test (progress callback support pending upstream library capability)
	errE = h.smartService.StartSelfTest(ctx, devicePath, input.Body.TestType)
	if errE != nil {
		if errors.Is(errE, dto.ErrorSMARTNotSupported) {
			return nil, huma.Error406NotAcceptable("SMART not supported on this device", errE)
		}
		if errors.Is(errE, dto.ErrorSMARTTestInProgress) {
			return nil, huma.Error422UnprocessableEntity("SMART test already in progress", errE)
		}
		tlog.ErrorContext(ctx, "Failed to start SMART test", "device", devicePath, "test_type", input.Body.TestType, "error", errE)
		return nil, huma.Error500InternalServerError("Failed to start SMART test", errE)
	}

	return &struct{ Body string }{Body: fmt.Sprintf("SMART %s test started on disk %s", input.Body.TestType, input.DiskID)}, nil
}

// AbortSmartTest aborts a running SMART self-test on a specific disk
func (h *SmartHandler) AbortSmartTest(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*struct{ Body string }, error) {
	// Check read-only mode
	if h.apiContext.ReadOnlyMode {
		return nil, huma.Error403Forbidden("Read-only mode enabled", errors.New("read-only mode"))
	}

	devicePath, errE := h.volumeService.GetDevicePathByDeviceID(input.DiskID)
	if errE != nil {
		return nil, huma.Error404NotFound("Disk not found", errors.New("disk not found"))
	}

	// Abort the test
	errE = h.smartService.AbortSelfTest(ctx, devicePath)
	if errE != nil {
		if errors.Is(errE, dto.ErrorSMARTNotSupported) {
			return nil, huma.Error406NotAcceptable("SMART not supported on this device", errE)
		}
		tlog.ErrorContext(ctx, "Failed to abort SMART test", "device", devicePath, "error", errE)
		return nil, huma.Error500InternalServerError("Failed to abort SMART test", errE)
	}

	return &struct{ Body string }{Body: fmt.Sprintf("SMART test aborted on disk %s", input.DiskID)}, nil
}

// EnableSmart enables SMART on a specific disk
func (h *SmartHandler) EnableSmart(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*struct{ Body string }, error) {
	// Check read-only mode
	if h.apiContext.ReadOnlyMode {
		return nil, huma.Error403Forbidden("Read-only mode enabled", errors.New("read-only mode"))
	}

	// Get disk info to find device path

	devicePath, errE := h.volumeService.GetDevicePathByDeviceID(input.DiskID)
	if errE != nil {
		return nil, huma.Error404NotFound("Disk not found", errors.New("disk not found"))
	}

	// Enable SMART
	errE = h.smartService.EnableSMART(ctx, devicePath)
	if errE != nil {
		if errors.Is(errE, dto.ErrorSMARTNotSupported) {
			return nil, huma.Error406NotAcceptable("SMART not supported on this device", errE)
		}
		tlog.ErrorContext(ctx, "Failed to enable SMART", "device", devicePath, "error", errE)
		return nil, huma.Error500InternalServerError("Failed to enable SMART", errE)
	}

	return &struct{ Body string }{Body: fmt.Sprintf("SMART enabled on disk %s", input.DiskID)}, nil
}

// DisableSmart disables SMART on a specific disk
func (h *SmartHandler) DisableSmart(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*struct{ Body string }, error) {
	// Check read-only mode
	if h.apiContext.ReadOnlyMode {
		return nil, huma.Error403Forbidden("Read-only mode enabled", errors.New("read-only mode"))
	}

	devicePath, errE := h.volumeService.GetDevicePathByDeviceID(input.DiskID)
	if errE != nil {
		return nil, huma.Error404NotFound("Disk not found", errors.New("disk not found"))
	}

	// Disable SMART
	errE = h.smartService.DisableSMART(ctx, devicePath)
	if errE != nil {
		if errors.Is(errE, dto.ErrorSMARTNotSupported) {
			return nil, huma.Error406NotAcceptable("SMART not supported on this device", errE)
		}
		tlog.ErrorContext(ctx, "Failed to disable SMART", "device", devicePath, "error", errE)
		return nil, huma.Error500InternalServerError("Failed to disable SMART", errE)
	}

	return &struct{ Body string }{Body: fmt.Sprintf("SMART disabled on disk %s", input.DiskID)}, nil
}
