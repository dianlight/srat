package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/tlog"
	"github.com/patrickmn/go-cache"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

const hwCacheKey = "hardware_info"

// HardwareServiceInterface is the interface other services use.
// It exposes a method that returns a neutral, internal representation
// of hardware info (`hardware.HardwareInfo`) so other packages don't have
// to import the generated `hardware` package.
type HardwareServiceInterface interface {
	GetHardwareInfo() (map[string]dto.Disk, errors.E)
	InvalidateHardwareInfo()
}

type hardwareService struct {
	ctx          context.Context
	haClient     hardware.ClientWithResponsesInterface
	state        *dto.ContextState
	conv         converter.HaHardwareToDtoImpl
	smartService SmartServiceInterface
	cache        *cache.Cache
}

func NewHardwareService(
	lc fx.Lifecycle,
	ctx context.Context,
	state *dto.ContextState,
	haClient hardware.ClientWithResponsesInterface,
	smartServiceInstance SmartServiceInterface,
	eventBus events.EventBusInterface,
) HardwareServiceInterface {
	hs := &hardwareService{
		ctx:          ctx,
		haClient:     haClient,
		conv:         converter.HaHardwareToDtoImpl{},
		smartService: smartServiceInstance,
		state:        state,
		cache:        cache.New(30*time.Minute, 10*time.Minute),
	}
	unsubscribe := eventBus.OnHomeAssistant(func(ctx context.Context, hae events.HomeAssistantEvent) errors.E {
		if hae.Type == events.EventTypes.START {
			hs.InvalidateHardwareInfo()
		}
		return nil
	})
	lc.Append(fx.Hook{

		OnStop: func(ctx context.Context) error {
			tlog.TraceContext(ctx, "HardwareService stopped")
			if unsubscribe != nil {
				unsubscribe()
			}
			return nil
		},
	})
	return hs
}

func (h *hardwareService) GetHardwareInfo() (map[string]dto.Disk, errors.E) {
	// try cache first
	if h.cache != nil {
		if cached, ok := h.cache.Get(hwCacheKey); ok {
			if disks, castOk := cached.(map[string]dto.Disk); castOk {
				tlog.DebugContext(h.ctx, "Returning hardware info from cache", "drive_count", len(disks))
				return disks, nil
			}
			// unexpected type, invalidate
			tlog.WarnContext(h.ctx, "Invalid type found in hardware info cache, invalidating", "expected", "map[string]dto.Disk", "actual", fmt.Sprintf("%T", cached))
			h.cache.Delete(hwCacheKey)
		}
	}

	ret := map[string]dto.Disk{}
	if !h.state.HACoreReady {
		slog.DebugContext(h.ctx, "HA Core not ready, cannot get hardware info")
		return ret, nil
	}
	hwser, errHw := h.haClient.GetHardwareInfoWithResponse(h.ctx)
	if errHw != nil || hwser == nil {
		if !errors.Is(errHw, dto.ErrorNotFound) {
			return nil, errors.WithDetails(errHw, "message", "failed to get hardware info from HA Supervisor", "hwset", hwser)
		}
		slog.DebugContext(h.ctx, "Hardware info not found, continuing with empty disk list")
	}

	if hwser.StatusCode() != 200 || hwser.JSON200 == nil || hwser.JSON200.Data == nil || hwser.JSON200.Data.Drives == nil {
		errMsg := "Received invalid hardware info response from HA Supervisor"
		slog.ErrorContext(h.ctx, errMsg, "status_code", hwser.StatusCode(), "response_body", string(hwser.Body))
		return nil, errors.New(errMsg)
	}

	tlog.TraceContext(h.ctx, "Processing drives from HA Supervisor", "drive_count", len(*hwser.JSON200.Data.Drives))
	for i, drive := range *hwser.JSON200.Data.Drives {
		if drive.Filesystems == nil || len(*drive.Filesystems) == 0 {
			tlog.DebugContext(h.ctx, "Skipping drive with no filesystems", "drive_index", i, "drive_id", drive.Id)
			continue
		}
		var diskDto dto.Disk
		errConvDrive := h.conv.DriveToDisk(drive, &diskDto)
		if errConvDrive != nil {
			tlog.WarnContext(h.ctx, "Error converting drive to disk DTO", "drive_index", i, "drive_id", drive.Id, "err", errConvDrive)
			continue
		}
		if diskDto.Partitions == nil || len(*diskDto.Partitions) == 0 {
			tlog.DebugContext(h.ctx, "Skipping drive DTO with no partitions after conversion", "drive_index", i, "drive_id", drive.Id)
			continue
		}

		// Find corresponding Device entries for Disk and its Partitions
		if hwser.JSON200.Data.Devices != nil {
			for deviceIdx := range *hwser.JSON200.Data.Devices {
				device := &(*hwser.JSON200.Data.Devices)[deviceIdx]
				if device.DevPath == nil || *device.DevPath == "" {
					tlog.DebugContext(h.ctx, "Skipping device with nil or empty name", "drive_index", i, "drive_id", drive.Id, "device_index", deviceIdx)
					continue
				}

				// Match Disk
				if diskDto.LegacyDeviceName != nil && *diskDto.LegacyDeviceName != "" && *device.Name == *diskDto.LegacyDeviceName {
					diskDto.LegacyDevicePath = device.DevPath
					diskDto.DevicePath = device.ById
					smartInfo, errSmart := h.smartService.GetSmartInfo(h.ctx, *diskDto.DevicePath)
					if errSmart != nil {
						if errors.Is(errSmart, dto.ErrorSMARTNotSupported) {
							tlog.TraceContext(h.ctx, "SMART not supported for device", "device", *diskDto.DevicePath, "drive_index", i, "drive_id", drive.Id)
							// Set SmartInfo with Supported=false
							diskDto.SmartInfo = &dto.SmartInfo{
								Supported: false,
							}
						} else {
							tlog.WarnContext(h.ctx, "Error retrieving SMART info for device", "device", *diskDto.DevicePath, "drive_index", i, "drive_id", drive.Id, "err", errSmart)
						}
					} else if smartInfo != nil {
						diskDto.SmartInfo = smartInfo
					}
					continue
				} // Match Partitions
				if diskDto.Partitions != nil {
					for pid, part := range *diskDto.Partitions {
						partition := part // copy
						if partition.LegacyDeviceName == nil || *partition.LegacyDeviceName == "" {
							tlog.DebugContext(h.ctx, "Skipping partition with nil or empty legacy device name", "disk_id", diskDto.Id, "partition_id", pid)
							continue
						}
						if *device.Name == *partition.LegacyDeviceName {
							partition.LegacyDevicePath = device.DevPath
							partition.DevicePath = device.ById
							if device.Attributes != nil {
								if device.Attributes.IDFSLABEL != nil {
									partition.Name = device.Attributes.IDFSLABEL
								} else if device.Attributes.IDPARTENTRYNAME != nil {
									partition.Name = device.Attributes.IDPARTENTRYNAME
								}
								if device.Attributes.IDFSTYPE != nil {
									partition.FsType = device.Attributes.IDFSTYPE
								}
							}
							name := ""
							if partition.Name != nil {
								name = *partition.Name
							}
							partition.System = pointer.Bool(strings.HasPrefix(name, "hassos-"))
							// write back into the map
							(*diskDto.Partitions)[pid] = partition
						}
					}
				}
			}
		}
		// Ensure disk has an ID to use as map key
		if diskDto.Id == nil || *diskDto.Id == "" {
			if drive.Id != nil && *drive.Id != "" {
				diskDto.Id = drive.Id
			}
		}
		if diskDto.Id == nil || *diskDto.Id == "" {
			tlog.WarnContext(h.ctx, "Skipping disk with missing ID after conversion", "drive_index", i)
			continue
		}
		ret[*diskDto.Id] = diskDto
	}

	// populate cache
	if h.cache != nil {
		h.cache.SetDefault(hwCacheKey, ret)
	}
	return ret, nil
}

// InvalidateHardwareInfo clears the cached hardware info so the next call
// to GetHardwareInfo will fetch fresh data from the HA Supervisor.
func (h *hardwareService) InvalidateHardwareInfo() {
	if h.cache == nil {
		return
	}
	h.cache.Delete(hwCacheKey)
	tlog.DebugContext(h.ctx, "Invalidated hardware info cache")
}
