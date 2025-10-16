package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type HDIdleHandler struct {
	repo             repository.HDIdleConfigRepositoryInterface
	hdidleService    service.HDIdleServiceInterface
	converter        converter.DtoToDbomConverter
	broadcastService service.BroadcasterServiceInterface
}

type HDIdleHandlerParams struct {
	fx.In
	Repo             repository.HDIdleConfigRepositoryInterface
	HDIdleService    service.HDIdleServiceInterface
	BroadcastService service.BroadcasterServiceInterface
}

func NewHDIdleHandler(params HDIdleHandlerParams) *HDIdleHandler {
	return &HDIdleHandler{
		repo:             params.Repo,
		hdidleService:    params.HDIdleService,
		converter:        &converter.DtoToDbomConverterImpl{},
		broadcastService: params.BroadcastService,
	}
}

func (h *HDIdleHandler) RegisterHDIdleHandler(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-hdidle-config",
		Method:      http.MethodGet,
		Path:        "/hdidle/config",
		Summary:     "Get HDIdle configuration",
		Description: "Retrieves the current HDIdle disk spin-down configuration",
		Tags:        []string{"HDIdle"},
	}, h.getConfig)

	huma.Register(api, huma.Operation{
		OperationID: "put-hdidle-config",
		Method:      http.MethodPut,
		Path:        "/hdidle/config",
		Summary:     "Update HDIdle configuration",
		Description: "Updates the HDIdle disk spin-down configuration and restarts the service if needed",
		Tags:        []string{"HDIdle"},
	}, h.putConfig)

	huma.Register(api, huma.Operation{
		OperationID: "delete-hdidle-config",
		Method:      http.MethodDelete,
		Path:        "/hdidle/config",
		Summary:     "Delete HDIdle configuration",
		Description: "Deletes the HDIdle configuration and stops the service",
		Tags:        []string{"HDIdle"},
	}, h.deleteConfig)

	huma.Register(api, huma.Operation{
		OperationID: "get-hdidle-status",
		Method:      http.MethodGet,
		Path:        "/hdidle/status",
		Summary:     "Get HDIdle service status",
		Description: "Retrieves the current status of the HDIdle service and monitored disks",
		Tags:        []string{"HDIdle"},
	}, h.getStatus)
}

type GetHDIdleConfigOutput struct {
	Body dto.HDIdleConfigDTO
}

func (h *HDIdleHandler) getConfig(ctx context.Context, input *struct{}) (*GetHDIdleConfigOutput, error) {
	config, err := h.repo.Get()
	if err != nil {
		if errors.Is(err, dto.ErrorNotFound) {
			// Return default configuration if not found
			return &GetHDIdleConfigOutput{
				Body: dto.HDIdleConfigDTO{
					Enabled:                 false,
					DefaultIdleTime:         600,
					DefaultCommandType:      "scsi",
					DefaultPowerCondition:   0,
					Debug:                   false,
					SymlinkPolicy:           0,
					IgnoreSpinDownDetection: false,
					Devices:                 []dto.HDIdleDeviceDTO{},
				},
			}, nil
		}
		return nil, huma.Error500InternalServerError("Failed to retrieve HDIdle configuration", err)
	}

	dtoConfig, convErr := h.converter.HDIdleConfigToHDIdleConfigDTO(*config)
	if convErr != nil {
		return nil, huma.Error500InternalServerError("Failed to convert HDIdle configuration", convErr)
	}

	return &GetHDIdleConfigOutput{Body: dtoConfig}, nil
}

type PutHDIdleConfigInput struct {
	Body dto.HDIdleConfigDTO
}

type PutHDIdleConfigOutput struct {
	Body dto.HDIdleConfigDTO
}

func (h *HDIdleHandler) putConfig(ctx context.Context, input *PutHDIdleConfigInput) (*PutHDIdleConfigOutput, error) {
	// Convert DTO to DBOM
	dbomConfig, convErr := h.converter.HDIdleConfigDTOToHDIdleConfig(input.Body)
	if convErr != nil {
		return nil, huma.Error400BadRequest("Invalid HDIdle configuration", convErr)
	}

	// Save configuration
	saveErr := h.repo.Save(&dbomConfig)
	if saveErr != nil {
		return nil, huma.Error500InternalServerError("Failed to save HDIdle configuration", saveErr)
	}

	// Restart service if enabled
	if input.Body.Enabled {
		// Stop existing service if running
		if h.hdidleService.IsRunning() {
			if stopErr := h.hdidleService.Stop(); stopErr != nil {
				return nil, huma.Error500InternalServerError("Failed to stop HDIdle service", stopErr)
			}
		}

		// Convert to service config
		serviceConfig := h.convertToServiceConfig(input.Body)

		// Start service
		if startErr := h.hdidleService.Start(&serviceConfig); startErr != nil {
			return nil, huma.Error500InternalServerError("Failed to start HDIdle service", startErr)
		}
	} else {
		// Stop service if it's running and we're disabling
		if h.hdidleService.IsRunning() {
			if stopErr := h.hdidleService.Stop(); stopErr != nil {
				return nil, huma.Error500InternalServerError("Failed to stop HDIdle service", stopErr)
			}
		}
	}

	// Broadcast configuration change
	h.broadcastService.BroadcastMessage(struct {
		Event string `json:"event"`
	}{
		Event: "hdidle_config",
	})

	return &PutHDIdleConfigOutput{Body: input.Body}, nil
}

type DeleteHDIdleConfigOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func (h *HDIdleHandler) deleteConfig(ctx context.Context, input *struct{}) (*DeleteHDIdleConfigOutput, error) {
	// Stop service if running
	if h.hdidleService.IsRunning() {
		if err := h.hdidleService.Stop(); err != nil {
			return nil, huma.Error500InternalServerError("Failed to stop HDIdle service", err)
		}
	}

	// Delete configuration
	if err := h.repo.Delete(); err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete HDIdle configuration", err)
	}

	// Broadcast configuration change
	h.broadcastService.BroadcastMessage(struct {
		Event string `json:"event"`
	}{
		Event: "hdidle_config",
	})

	return &DeleteHDIdleConfigOutput{
		Body: struct {
			Message string `json:"message"`
		}{
			Message: "HDIdle configuration deleted successfully",
		},
	}, nil
}

type GetHDIdleStatusOutput struct {
	Body struct {
		Running     bool                       `json:"running"`
		MonitoredAt string                     `json:"monitored_at,omitempty"`
		Disks       []service.HDIdleDiskStatus `json:"disks,omitempty"`
	}
}

func (h *HDIdleHandler) getStatus(ctx context.Context, input *struct{}) (*GetHDIdleStatusOutput, error) {
	status, err := h.hdidleService.GetStatus()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get HDIdle service status", err)
	}

	output := &GetHDIdleStatusOutput{
		Body: struct {
			Running     bool                       `json:"running"`
			MonitoredAt string                     `json:"monitored_at,omitempty"`
			Disks       []service.HDIdleDiskStatus `json:"disks,omitempty"`
		}{
			Running: status.Running,
			Disks:   status.Disks,
		},
	}

	if status.Running {
		output.Body.MonitoredAt = status.MonitoredAt.Format("2006-01-02T15:04:05Z")
	}

	return output, nil
}

// convertToServiceConfig converts DTO to service config
func (h *HDIdleHandler) convertToServiceConfig(dto dto.HDIdleConfigDTO) service.HDIdleConfig {
	devices := make([]service.HDIdleDeviceConfig, len(dto.Devices))
	for i, dev := range dto.Devices {
		devices[i] = service.HDIdleDeviceConfig{
			Name:           dev.Name,
			IdleTime:       dev.IdleTime,
			CommandType:    dev.CommandType,
			PowerCondition: dev.PowerCondition,
		}
	}

	return service.HDIdleConfig{
		Devices:                 devices,
		DefaultIdleTime:         dto.DefaultIdleTime,
		DefaultCommandType:      dto.DefaultCommandType,
		DefaultPowerCondition:   dto.DefaultPowerCondition,
		Debug:                   dto.Debug,
		LogFile:                 dto.LogFile,
		SymlinkPolicy:           dto.SymlinkPolicy,
		IgnoreSpinDownDetection: dto.IgnoreSpinDownDetection,
	}
}
