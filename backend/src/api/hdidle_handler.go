package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"go.uber.org/fx"
)

type HDIdleHandler struct {
	hdidleService service.HDIdleServiceInterface
	converter     converter.DtoToDbomConverter
}

type HDIdleHandlerParams struct {
	fx.In
	HDIdleService service.HDIdleServiceInterface
}

func NewHDIdleHandler(params HDIdleHandlerParams) *HDIdleHandler {
	return &HDIdleHandler{
		hdidleService: params.HDIdleService,
		converter:     &converter.DtoToDbomConverterImpl{},
	}
}

// RegisterHDIdleHandler registers the HTTP handlers for HDIdle-related operations.
// It sets up the following routes:
// - POST /hdidle/start: Start the HDIdle monitoring service.
// - POST /hdidle/stop: Stop the HDIdle monitoring service.
// - GET /disk/{disk_id}/hdidle/info: Get HDIdle status for a specific disk.
// - GET /disk/{disk_id}/hdidle/config: Get HDIdle configuration for a specific disk.
// - PUT /disk/{disk_id}/hdidle/config: Update HDIdle configuration for a specific disk.
// - GET /disk/{disk_id}/hdidle/support: Check if a disk supports HDIdle spindown commands.
//
// Parameters:
// - api: The huma.API instance to register the handlers with.
func (h *HDIdleHandler) RegisterHDIdleHandler(api huma.API) {
	huma.Post(api, "/hdidle/start", h.startService, huma.OperationTags("hdidle"))
	huma.Post(api, "/hdidle/stop", h.stopService, huma.OperationTags("hdidle"))

	// Per-disk HDIdle endpoints
	huma.Get(api, "/disk/{disk_id}/hdidle/info", h.getStatus, huma.OperationTags("disk"))
	huma.Get(api, "/disk/{disk_id}/hdidle/config", h.getConfig, huma.OperationTags("disk"))
	huma.Put(api, "/disk/{disk_id}/hdidle/config", h.putConfig, huma.OperationTags("disk"))
	huma.Get(api, "/disk/{disk_id}/hdidle/support", h.checkSupport, huma.OperationTags("disk"))
}

type GetHDIdleConfigOutput struct {
	Body dto.HDIdleDevice
}

func (h *HDIdleHandler) getConfig(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID (not the device path)"`
}) (*GetHDIdleConfigOutput, error) {
	// disk_id represents the stable disk identifier. Convert it to the device path.
	devicePath := "/dev/disk/by-id/" + input.DiskID
	config, err := h.hdidleService.GetDeviceConfig(devicePath)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve HDIdle configuration", err)
	}

	return &GetHDIdleConfigOutput{Body: *config}, nil
}

type PutHDIdleConfigInput struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
	Body   dto.HDIdleDevice
}

type PutHDIdleConfigOutput struct {
	Body dto.HDIdleDevice
}

func (h *HDIdleHandler) putConfig(ctx context.Context, input *PutHDIdleConfigInput) (*PutHDIdleConfigOutput, error) {
	// disk_id represents the stable disk identifier. Convert it to the device path.
	devicePath := "/dev/disk/by-id/" + input.DiskID
	config, err := h.hdidleService.GetDeviceConfig(devicePath)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve HDIdle configuration", err)
	}

	if input.Body.DevicePath != config.DevicePath {
		return nil, huma.Error400BadRequest("Device path in body does not match path in URL", nil)
	}

	err = h.hdidleService.SaveDeviceConfig(input.Body)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to save HDIdle device configuration", err)
	}
	if h.hdidleService.IsRunning() {
		if stopErr := h.hdidleService.Stop(); stopErr != nil {
			return nil, huma.Error500InternalServerError("Failed to stop HDIdle service", stopErr)
		}
	}

	err = h.hdidleService.Start()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to start HDIdle service", err)
	}

	return &PutHDIdleConfigOutput{Body: input.Body}, nil
}

type GetHDIdleStatusOutput struct {
	Body *dto.HDIdleDeviceStatus `json:"disks,omitempty"`
}

func (h *HDIdleHandler) getStatus(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID (not the device path)"`
}) (*GetHDIdleStatusOutput, error) {
	// disk_id represents the stable disk identifier. Convert it to the device path.
	devicePath := "/dev/disk/by-id/" + input.DiskID
	status, err := h.hdidleService.GetDeviceStatus(devicePath)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get HDIdle service status", err)
	}

	if status == nil {
		return nil, huma.Error404NotFound("HDIdle status not found for the specified disk", nil)
	}

	output := &GetHDIdleStatusOutput{
		Body: status,
	}
	return output, nil
}

// StartHDIdleServiceOutput represents the response for starting the HDIdle service.
type StartHDIdleServiceOutput struct {
	Body struct {
		Message string `json:"message"`
		Running bool   `json:"running"`
	}
}

// startService starts the HDIdle monitoring service.
func (h *HDIdleHandler) startService(ctx context.Context, input *struct{}) (*StartHDIdleServiceOutput, error) {
	if h.hdidleService.IsRunning() {
		return &StartHDIdleServiceOutput{
			Body: struct {
				Message string `json:"message"`
				Running bool   `json:"running"`
			}{
				Message: "HDIdle service is already running",
				Running: true,
			},
		}, nil
	}

	err := h.hdidleService.Start()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to start HDIdle service", err)
	}

	return &StartHDIdleServiceOutput{
		Body: struct {
			Message string `json:"message"`
			Running bool   `json:"running"`
		}{
			Message: "HDIdle service started successfully",
			Running: true,
		},
	}, nil
}

// StopHDIdleServiceOutput represents the response for stopping the HDIdle service.
type StopHDIdleServiceOutput struct {
	Body struct {
		Message string `json:"message"`
		Running bool   `json:"running"`
	}
}

// stopService stops the HDIdle monitoring service.
func (h *HDIdleHandler) stopService(ctx context.Context, input *struct{}) (*StopHDIdleServiceOutput, error) {
	if !h.hdidleService.IsRunning() {
		return &StopHDIdleServiceOutput{
			Body: struct {
				Message string `json:"message"`
				Running bool   `json:"running"`
			}{
				Message: "HDIdle service is not running",
				Running: false,
			},
		}, nil
	}

	err := h.hdidleService.Stop()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to stop HDIdle service", err)
	}

	return &StopHDIdleServiceOutput{
		Body: struct {
			Message string `json:"message"`
			Running bool   `json:"running"`
		}{
			Message: "HDIdle service stopped successfully",
			Running: false,
		},
	}, nil
}

// GetHDIdleSupportOutput represents the response for checking HDIdle device support.
type GetHDIdleSupportOutput struct {
	Body *dto.HDIdleDeviceSupport
}

// checkSupport checks if a specific disk supports HDIdle spindown commands.
// This verifies whether the device supports SCSI and/or ATA spindown commands
// and returns a recommended command type.
func (h *HDIdleHandler) checkSupport(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID (not the device path)"`
}) (*GetHDIdleSupportOutput, error) {
	// disk_id represents the stable disk identifier. Convert it to the device path.
	devicePath := "/dev/disk/by-id/" + input.DiskID
	support, err := h.hdidleService.CheckDeviceSupport(devicePath)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to check HDIdle device support", err)
	}

	return &GetHDIdleSupportOutput{Body: support}, nil
}
