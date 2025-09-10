package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/tlog"
	"github.com/patrickmn/go-cache"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// HardwareServiceInterface is the interface other services use.
// It exposes a method that returns a neutral, internal representation
// of hardware info (`hardware.HardwareInfo`) so other packages don't have
// to import the generated `hardware` package.
type HardwareServiceInterface interface {
	GetHardwareInfo() ([]dto.Disk, errors.E)
	InvalidateHardwareInfo()
}

type hardwareService struct {
	ctx          context.Context
	haClient     hardware.ClientWithResponsesInterface
	conv         converter.HaHardwareToDtoImpl
	smartService SmartServiceInterface
	cache        *cache.Cache
}

func NewHardwareService(
	lc fx.Lifecycle,
	ctx context.Context,
	haClient hardware.ClientWithResponsesInterface,
	smartServiceInstance SmartServiceInterface,
) HardwareServiceInterface {
	return &hardwareService{
		ctx:          ctx,
		haClient:     haClient,
		conv:         converter.HaHardwareToDtoImpl{},
		smartService: smartServiceInstance,
		cache:        cache.New(30*time.Minute, 10*time.Minute),
	}
}

func (h *hardwareService) GetHardwareInfo() ([]dto.Disk, errors.E) {
	// try cache first
	const hwCacheKey = "hardware_info"
	if h.cache != nil {
		if cached, ok := h.cache.Get(hwCacheKey); ok {
			if disks, castOk := cached.([]dto.Disk); castOk {
				tlog.Debug("Returning hardware info from cache", "drive_count", len(disks))
				return disks, nil
			}
			// unexpected type, invalidate
			tlog.Warn("Invalid type found in hardware info cache, invalidating", "expected", "[]dto.Disk", "actual", fmt.Sprintf("%T", cached))
			h.cache.Delete(hwCacheKey)
		}
	}

	ret := []dto.Disk{}
	hwser, errHw := h.haClient.GetHardwareInfoWithResponse(h.ctx)
	if errHw != nil || hwser == nil {
		if !errors.Is(errHw, dto.ErrorNotFound) {
			return nil, errors.WithDetails(errHw, "message", "failed to get hardware info from HA Supervisor", "hwset", hwser)
		}
		slog.Debug("Hardware info not found, continuing with empty disk list")
	}

	if hwser.StatusCode() != 200 || hwser.JSON200 == nil || hwser.JSON200.Data == nil || hwser.JSON200.Data.Drives == nil {
		errMsg := "Received invalid hardware info response from HA Supervisor"
		slog.Error(errMsg, "status_code", hwser.StatusCode(), "response_body", string(hwser.Body))
		return nil, errors.New(errMsg)
	}

	tlog.Trace("Processing drives from HA Supervisor", "drive_count", len(*hwser.JSON200.Data.Drives))
	for i, drive := range *hwser.JSON200.Data.Drives {
		if drive.Filesystems == nil || len(*drive.Filesystems) == 0 {
			tlog.Debug("Skipping drive with no filesystems", "drive_index", i, "drive_id", drive.Id)
			continue
		}
		var diskDto dto.Disk
		errConvDrive := h.conv.DriveToDisk(drive, &diskDto)
		if errConvDrive != nil {
			tlog.Warn("Error converting drive to disk DTO", "drive_index", i, "drive_id", drive.Id, "err", errConvDrive)
			continue
		}
		if diskDto.Partitions == nil || len(*diskDto.Partitions) == 0 {
			tlog.Debug("Skipping drive DTO with no partitions after conversion", "drive_index", i, "drive_id", drive.Id)
			continue
		}

		// Find corresponding Device entries for Disk and its Partitions
		if hwser.JSON200.Data.Devices != nil {
			for deviceIdx := range *hwser.JSON200.Data.Devices {
				device := &(*hwser.JSON200.Data.Devices)[deviceIdx]
				if device.DevPath == nil || *device.DevPath == "" {
					tlog.Debug("Skipping device with nil or empty name", "drive_index", i, "drive_id", drive.Id, "device_index", deviceIdx)
					continue
				}

				// Match Disk
				if diskDto.LegacyDeviceName != nil && *diskDto.LegacyDeviceName != "" && *device.Name == *diskDto.LegacyDeviceName {
					diskDto.LegacyDevicePath = device.DevPath
					diskDto.DevicePath = device.ById
					smartInfo, errSmart := h.smartService.GetSmartInfo(*diskDto.DevicePath)
					if errSmart != nil {
						if errors.Is(errSmart, dto.ErrorSMARTNotSupported) {
							tlog.Trace("SMART not supported for device", "device", *diskDto.DevicePath, "drive_index", i, "drive_id", drive.Id)
						} else {
							tlog.Warn("Error retrieving SMART info for device", "device", *diskDto.DevicePath, "drive_index", i, "drive_id", drive.Id, "err", errSmart)
						}
					} else if smartInfo != nil {
						diskDto.SmartInfo = smartInfo
					}
					continue
				}

				// Match Partitions
				if diskDto.Partitions != nil {
					for partIdx := range *diskDto.Partitions {
						partition := &(*diskDto.Partitions)[partIdx]
						if partition.LegacyDeviceName == nil || *partition.LegacyDeviceName == "" {
							tlog.Debug("Skipping partition with nil or empty legacy device name", "disk_id", diskDto.Id, "partition_index", partIdx)
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
							partition.System = pointer.Bool(strings.HasPrefix(*partition.Name, "hassos-"))
							break
						}
					}
				}
			}
		}
		ret = append(ret, diskDto)
	}

	return ret, nil
}

// InvalidateHardwareInfo clears the cached hardware info so the next call
// to GetHardwareInfo will fetch fresh data from the HA Supervisor.
func (h *hardwareService) InvalidateHardwareInfo() {
	if h.cache == nil {
		return
	}
	h.cache.Delete("hardware_info")
	tlog.Debug("Invalidated hardware info cache")
}
