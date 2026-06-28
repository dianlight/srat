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
	hdidleService   service.HDIdleServiceInterface
	hardwareService service.HardwareServiceInterface
	settingService  service.SettingServiceInterface
	hdidleService   service.HDIdleServiceInterface
	hardwareService service.HardwareServiceInterface
	settingService  service.SettingServiceInterface
}

type HDIdleHandlerParams struct {
	fx.In
	HDIdleService   service.HDIdleServiceInterface
	HardwareService service.HardwareServiceInterface
	SettingService  service.SettingServiceInterface
}

func NewHDIdleHandler(params HDIdleHandlerParams) *HDIdleHandler {
	return &HDIdleHandler{
		hdidleService:   params.HDIdleService,
		hardwareService: params.HardwareService,
		settingService:  params.SettingService,
		converter:       &converter.DtoToDbomConverterImpl{},
	}
}

// RegisterHDIdleHandler registers the HTTP handlers for HDIdle-related operations.
//
// All routes are gated by Lab Mode (settings.experimental_lab_mode): when
// disabled, every endpoint returns 403 Forbidden. The previous global
// /hdidle/start and /hdidle/stop endpoints have been removed — the service
// is now driven exclusively by per-disk records.
//
// Routes (all require Lab Mode):
//   - GET    /disk/{disk_id}/hdidle/info               — current spin status
//   - GET    /disk/{disk_id}/hdidle/config             — per-disk config
//   - PUT    /disk/{disk_id}/hdidle/config             — update config; 409 on
//     non-rotational without
//     force_enabled=true
//   - GET    /disk/{disk_id}/hdidle/support            — SCSI/ATA capability probe
//   - POST   /disk/{disk_id}/hdidle/ignore-suggestion  — dismiss dashboard badge
func (h *HDIdleHandler) RegisterHDIdleHandler(api huma.API) {
	huma.Get(api, "/disk/{disk_id}/hdidle/info", h.getStatus, huma.OperationTags("disk"))
	huma.Get(api, "/disk/{disk_id}/hdidle/config", h.getConfig, huma.OperationTags("disk"))
	huma.Put(api, "/disk/{disk_id}/hdidle/config", h.putConfig, huma.OperationTags("disk"))
	huma.Get(api, "/disk/{disk_id}/hdidle/support", h.checkSupport, huma.OperationTags("disk"))
	huma.Post(api, "/disk/{disk_id}/hdidle/ignore-suggestion", h.ignoreSuggestion, huma.OperationTags("disk", "volume"))
}

// requireLabMode returns 403 unless settings.experimental_lab_mode is true.
// Called at the top of every public hdidle handler.
func (h *HDIdleHandler) requireLabMode() error {
	settings, err := h.settingService.Load()
	if err != nil {
		return huma.Error500InternalServerError("Failed to read settings", err)
	}
	if settings == nil || !settings.ExperimentalLabMode {
		return huma.Error403Forbidden(
			"HDIdle endpoints require Lab Mode (set experimental_lab_mode=true in settings)",
			dto.ErrorLabModeRequired,
		)
	}
	return nil
}

// findDiskRotational returns the IsRotational tri-state for the given disk_id
// by consulting the hardware service. nil means "unknown" — caller decides
// how to interpret it (we treat unknown the same as non-rotational so users
// must explicitly force-enable on USB enclosures that hide the flag).
func (h *HDIdleHandler) findDiskRotational(diskID string) *bool {
	disks, err := h.hardwareService.GetHardwareInfo()
	if err != nil || disks == nil {
		return nil
	}
	if d, ok := disks[diskID]; ok {
		return d.IsRotational
	}
	return nil
}

// ---------- handlers ----------

type GetHDIdleConfigOutput struct {
	Body dto.HDIdleDevice
}

func (h *HDIdleHandler) getConfig(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID (not the device path)"`
}) (*GetHDIdleConfigOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	devicePath, errR := h.hdidleService.ResolveDevicePath(input.DiskID)
	if errR != nil {
		return nil, huma.Error404NotFound("Disk not found: "+input.DiskID, errR)
	}
	config, err := h.hdidleService.GetDeviceConfig(devicePath)
	if err != nil {
		// ErrorHDIdleNotSupported is an expected outcome (NVMe / unsupported
		// USB bridge), not a 500. The body still carries the support info.
		if config != nil {
			return &GetHDIdleConfigOutput{Body: *config}, nil
		}
		return nil, huma.Error500InternalServerError("Failed to retrieve HDIdle configuration", err)
	}
	if config == nil {
		return nil, huma.Error404NotFound("HDIdle configuration not found for disk: "+input.DiskID, nil)
	}

	return &GetHDIdleConfigOutput{Body: *config}, nil
}

type PutHDIdleConfigInput struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID (not the device path)"`
	Body   dto.HDIdleDevice
}

type PutHDIdleConfigOutput struct {
	Body dto.HDIdleDevice
}

func (h *HDIdleHandler) putConfig(ctx context.Context, input *PutHDIdleConfigInput) (*PutHDIdleConfigOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	devicePath, errR := h.hdidleService.ResolveDevicePath(input.DiskID)
	if errR != nil {
		return nil, huma.Error404NotFound("Disk not found: "+input.DiskID, errR)
	}

	// Goal #6: enabling on a non-rotational disk requires explicit force.
	// is_rotational is nil for unknown (USB enclosures hiding the flag) — we
	// treat unknown as non-rotational so the user has to opt in via the
	// confirm dialog. Accepting NOENABLED unconditionally means the user can
	// always disable a previously-force-enabled disk.
	enabling := input.Body.Enabled != dto.HdidleEnableds.NOENABLED
	if enabling && !input.Body.ForceEnabled {
		rot := h.findDiskRotational(input.DiskID)
		if rot == nil || !*rot {
			return nil, huma.Error409Conflict(
				"Cannot enable HDIdle on a non-rotational (SSD/NVMe) disk without force_enabled=true",
				dto.ErrorHDIdleNonRotational,
			)
		}
	}

	// Stamp the resolved path onto the body so the caller does not have to
	// echo it back. Anything they sent in DevicePath is overwritten —
	// it was a fragile contract anyway (server resolves symlinks, client
	// often guesses).
	input.Body.HDIdleDeviceSupport.DevicePath = devicePath

	if err := h.hdidleService.SaveDeviceConfig(input.Body); err != nil {
		return nil, huma.Error500InternalServerError("Failed to save HDIdle device configuration", err)
	}

	// Restart the monitor so it picks up the new per-disk config. Start()
	// is now idempotent and safe to call after a no-op Stop().
	if stopErr := h.hdidleService.Stop(); stopErr != nil {
		return nil, huma.Error500InternalServerError("Failed to stop HDIdle service", stopErr)
	}
	if err := h.hdidleService.Start(); err != nil {
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
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	devicePath, errR := h.hdidleService.ResolveDevicePath(input.DiskID)
	if errR != nil {
		return nil, huma.Error404NotFound("Disk not found: "+input.DiskID, errR)
	}
	status, err := h.hdidleService.GetDeviceStatus(devicePath)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get HDIdle service status", err)
	}

	if status == nil {
		return nil, huma.Error404NotFound("HDIdle status not found for the specified disk", nil)
	}

	return &GetHDIdleStatusOutput{Body: status}, nil
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
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	devicePath, errR := h.hdidleService.ResolveDevicePath(input.DiskID)
	if errR != nil {
		return nil, huma.Error404NotFound("Disk not found: "+input.DiskID, errR)
	}
	support, err := h.hdidleService.CheckDeviceSupport(devicePath)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to check HDIdle device support", err)
	}

	return &GetHDIdleSupportOutput{Body: support}, nil
}

// IgnoreSuggestionOutput carries the updated device config so callers can
// refresh their state without a second round-trip. Using a non-empty Body
// causes huma to emit 200 OK (an empty struct would yield 204 No Content).
type IgnoreSuggestionOutput struct {
	Body dto.HDIdleDevice
}

// ignoreSuggestion sets HDIdleDevice.SuggestionIgnored=true so the dashboard
// stops showing the "Enable HDIdle" badge for this disk. Idempotent: calling
// it on a row that does not yet exist creates a no-op row whose only purpose
// is to remember the dismissal.
func (h *HDIdleHandler) ignoreSuggestion(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" required:"true" doc:"The disk ID"`
}) (*IgnoreSuggestionOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	devicePath, errR := h.hdidleService.ResolveDevicePath(input.DiskID)
	if errR != nil {
		return nil, huma.Error404NotFound("Disk not found: "+input.DiskID, errR)
	}

	// Build a minimal config row with only SuggestionIgnored=true; the rest
	// stays at zero-value/default. SaveDeviceConfig handles upsert.
	cfg, err := h.hdidleService.GetDeviceConfig(devicePath)
	if err != nil {
		if cfg == nil {
			cfg = &dto.HDIdleDevice{
				DiskId:  input.DiskID,
				Enabled: dto.HdidleEnableds.NOENABLED,
			}
			cfg.HDIdleDeviceSupport.DevicePath = devicePath
		} else {
			cfg.SuggestionIgnored = true
			return &IgnoreSuggestionOutput{Body: *cfg}, nil
		}
	}
	cfg.SuggestionIgnored = true
	cfg.SuggestionIgnored = true
	if err := h.hdidleService.SaveDeviceConfig(*cfg); err != nil {
		// Even if the device is not yet known, we still want to persist the
		// dismissal — synthesize a minimal record.
		cfg = &dto.HDIdleDevice{
			DiskId:  input.DiskID,
			Enabled: dto.HdidleEnableds.NOENABLED,
		}
		cfg.HDIdleDeviceSupport.DevicePath = devicePath
	}
	cfg.SuggestionIgnored = true
	if err := h.hdidleService.SaveDeviceConfig(*cfg); err != nil {
		return nil, huma.Error500InternalServerError("Failed to persist suggestion dismissal", err)
	}

	return &IgnoreSuggestionOutput{Body: *cfg}, nil
}
