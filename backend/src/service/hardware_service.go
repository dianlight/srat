package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/tlog"
	"github.com/patrickmn/go-cache"
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
	ctx      context.Context
	haClient hardware.ClientWithResponsesInterface
	conv     converter.HaHardwareToDtoImpl
	cache    *cache.Cache
}

func NewHardwareService(
	lc fx.Lifecycle,
	ctx context.Context,
	haClient hardware.ClientWithResponsesInterface,
) HardwareServiceInterface {
	return &hardwareService{
		ctx:      ctx,
		haClient: haClient,
		conv:     converter.HaHardwareToDtoImpl{},
		cache:    cache.New(30*time.Minute, 10*time.Minute),
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
			return nil, errors.WithDetails(errHw, "failed to get hardware info from HA Supervisor", "hwset", hwser)
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

		ret = append(ret, diskDto)
	}

	if hwser.JSON200.Data.Devices != nil {
		tlog.Trace("Processing Devices from HA Supervisor", "device_count", len(*hwser.JSON200.Data.Devices))
		for diskIdx := range ret {
			disk := &ret[diskIdx]
			if disk.Partitions == nil {
				continue
			}
			for partIdx := range *disk.Partitions {
				partition := &(*disk.Partitions)[partIdx]

				if partition.Device == nil || *partition.Device == "" {
					tlog.Debug("Skipping partition with nil or empty device path", "disk_id", disk.Id, "partition_index", partIdx)
					continue
				}
				for deviceIdx := range *hwser.JSON200.Data.Devices {
					device := &(*hwser.JSON200.Data.Devices)[deviceIdx]
					if device.DevPath == nil || *device.DevPath == "" {
						tlog.Debug("Skipping device with nil or empty name", "disk_id", disk.Id, "partition_index", partIdx, "device_index", deviceIdx)
						continue
					}

					if (*device.DevPath == *partition.Device) && device.Attributes != nil {
						if (partition.Name == nil || *partition.Name == "") && device.Attributes.PARTNAME != nil {
							partition.Name = device.Attributes.PARTNAME
							continue
						}

					}
				}
			}
		}
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
