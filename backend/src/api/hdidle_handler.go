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

func (h *HDIdleHandler) RegisterHDIdleHandler(api huma.API) {
	huma.Get(api, "/disk/{disk_id}/hdidle/info", h.getStatus, huma.OperationTags("disk"))
	huma.Get(api, "/disk/{disk_id}/hdidle/config", h.getConfig, huma.OperationTags("disk"))
	huma.Put(api, "/disk/{disk_id}/hdidle/config", h.putConfig, huma.OperationTags("disk"))
}

type GetHDIdleConfigOutput struct {
	Body dto.HDIdleDeviceDTO
}

func (h *HDIdleHandler) getConfig(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*GetHDIdleConfigOutput, error) {
	config, err := h.hdidleService.GetDeviceConfig(input.DiskID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve HDIdle configuration", err)
	}

	return &GetHDIdleConfigOutput{Body: *config}, nil
}

type PutHDIdleConfigInput struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
	Body   dto.HDIdleDeviceDTO
}

type PutHDIdleConfigOutput struct {
	Body dto.HDIdleDeviceDTO
}

func (h *HDIdleHandler) putConfig(ctx context.Context, input *PutHDIdleConfigInput) (*PutHDIdleConfigOutput, error) {
	config, err := h.hdidleService.GetDeviceConfig(input.DiskID)
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

type DeleteHDIdleConfigOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type GetHDIdleStatusOutput struct {
	Body *service.HDIdleDiskStatus `json:"disks,omitempty"`
}

func (h *HDIdleHandler) getStatus(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID or device path"`
}) (*GetHDIdleStatusOutput, error) {
	status, err := h.hdidleService.GetDeviceStatus(input.DiskID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get HDIdle service status", err)
	}

	output := &GetHDIdleStatusOutput{
		Body: status,
	}
	return output, nil
}
