package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/tlog"
	"github.com/patrickmn/go-cache"
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
	ctx           context.Context
	haClient      hardware.ClientWithResponsesInterface
	state         *dto.ContextState
	conv          converter.HaHardwareToDtoImpl
	smartService  SmartServiceInterface
	hdidleService HDIdleServiceInterface
	cache         *cache.Cache
	// readFile is os.ReadFile by default; tests override it to mock sysfs.
	readFile func(string) ([]byte, error)
	// sysBlockBasePath is "/sys/block" in production; tests override it
	// to point at a temp dir containing fake rotational files.
	sysBlockBasePath string
}

func NewHardwareService(
	lc fx.Lifecycle,
	ctx context.Context,
	state *dto.ContextState,
	haClient hardware.ClientWithResponsesInterface,
	smartServiceInstance SmartServiceInterface,
	hdidleServiceInstance HDIdleServiceInterface,
	eventBus events.EventBusInterface,
) HardwareServiceInterface {
	hs := &hardwareService{
		ctx:              ctx,
		haClient:         haClient,
		conv:             converter.HaHardwareToDtoImpl{},
		smartService:     smartServiceInstance,
		hdidleService:    hdidleServiceInstance,
		state:            state,
		cache:            cache.New(30*time.Minute, 10*time.Minute),
		readFile:         os.ReadFile,
		sysBlockBasePath: "/sys/block",
	}
	unsubscribe := eventBus.OnHomeAssistant(func(ctx context.Context, hae events.HomeAssistantEvent) errors.E {
		if hae.Type == events.EventTypes.START {
			tlog.DebugContext(ctx, "Home Assistant started event received, invalidating hardware info cache")
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
		tlog.DebugContext(h.ctx, "HA Core not ready, cannot get hardware info", tlog.WithCaller(0)...)
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

	tlog.DebugContext(h.ctx, "Processing drives from HA Supervisor", "drive_count", len(*hwser.JSON200.Data.Drives))
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
					Id := strings.TrimPrefix(*device.ById, "/dev/disk/by-id/")
					diskDto.Id = &Id
					smartInfo, errSmart := h.smartService.GetSmartInfo(h.ctx, *diskDto.Id)
					if errSmart != nil {
						if errors.Is(errSmart, dto.ErrorSMARTNotSupported) {
							tlog.TraceContext(h.ctx, "SMART not supported for device", "device", *diskDto.Id, "drive_index", i, "drive_id", drive.Id)
							// Set SmartInfo with Supported=false
							diskDto.SmartInfo = &dto.SmartInfo{
								Supported: false,
							}
						} else {
							tlog.WarnContext(h.ctx, "Error retrieving SMART info for device", "device", *diskDto.Id, "drive_index", i, "drive_id", drive.Id, "err", errSmart)
						}
					} else if smartInfo != nil {
						diskDto.SmartInfo = smartInfo
					}
					hdidleDevice, errHDidle := h.hdidleService.GetDeviceConfig(*diskDto.DevicePath)
					if errHDidle != nil {
						if errors.Is(errHDidle, dto.ErrorHDIdleNotSupported) {
							tlog.TraceContext(h.ctx, "HDIdle not supported for device", "device", *diskDto.DevicePath, "drive_index", i, "drive_id", drive.Id)
							diskDto.HDIdleDevice = hdidleDevice
						} else {
							tlog.WarnContext(h.ctx, "Error retrieving HDIdle config for device", "device", *diskDto.DevicePath, "drive_index", i, "drive_id", drive.Id, "err", errHDidle)
						}
					} else if hdidleDevice != nil {
						diskDto.HDIdleDevice = hdidleDevice
					}

					// Detect rotational medium (HDD vs SSD/NVMe). Used by the
					// dashboard to decide whether to suggest HDIdle and by the
					// per-disk card to warn before force-enabling on an SSD.
					if diskDto.LegacyDeviceName != nil {
						diskDto.IsRotational = h.detectRotational(*diskDto.LegacyDeviceName, diskDto.SmartInfo)

						// USB bridge passthrough of the sysfs rotational flag is
						// unreliable: many enclosures report rotational=1 for
						// flash drives. When SMART is unavailable (typical for
						// USB flash drives) or reports no rotation rate, demote
						// to unknown instead of trusting the sysfs flag alone.
						if diskDto.IsRotational != nil && *diskDto.IsRotational {
							if diskDto.ConnectionBus != nil && strings.EqualFold(*diskDto.ConnectionBus, "usb") {
								if diskDto.SmartInfo == nil || !diskDto.SmartInfo.Supported || diskDto.SmartInfo.RotationRate == 0 {
									diskDto.IsRotational = nil
									tlog.DebugContext(h.ctx, "USB device with unreliable rotational flag – demoting to unknown",
										"disk_id", *diskDto.Id, "legacy_device_name", *diskDto.LegacyDeviceName)
								}
							}
						}
					}

					continue
				}
				// Match Partitions
				if diskDto.Partitions != nil {
					for pid, part := range *diskDto.Partitions {
						partition := part // copy
						partition.DiskId = diskDto.Id
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
							if partition.Name != nil {
								partition.System = new(strings.HasPrefix(*partition.Name, "hassos-"))
							} else {
								partition.System = new(false)
							}
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
		tlog.TraceContext(h.ctx, "Adding disk DTO to result map", "disk_id", *diskDto.Id)
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
	tlog.TraceContext(h.ctx, "Invalidated hardware info cache")
}

// detectRotational reports whether a block device is a rotational HDD.
// Tri-state return:
//   - *true  → rotational (HDD)
//   - *false → non-rotational (SSD/NVMe)
//   - nil    → unknown (sysfs missing AND SMART unavailable)
//
// Strategy: read /sys/block/<dev>/queue/rotational first (kernel-authoritative,
// 1=HDD / 0=SSD); if that file is missing or unparseable, fall back to
// smartInfo.RotationRate when SMART is supported (>0=HDD, 0=SSD).
//
// devName must be the bare kernel name (e.g. "sda", "nvme0n1") with no slashes
// or path traversal — we sanitize defensively before joining.
func (h *hardwareService) detectRotational(devName string, smartInfo *dto.SmartInfo) *bool {
	// Defensive sanitization: reject empty, anything with separators or "..".
	if devName == "" || strings.ContainsAny(devName, "/\\") || strings.Contains(devName, "..") {
		return rotationalFromSmart(smartInfo)
	}

	path := filepath.Join(h.sysBlockBasePath, devName, "queue", "rotational")
	data, err := h.readFile(path)
	if err == nil {
		switch strings.TrimSpace(string(data)) {
		case "1":
			t := true
			return &t
		case "0":
			f := false
			return &f
		}
		// File exists but content is unexpected — fall through to SMART.
	}

	return rotationalFromSmart(smartInfo)
}

// rotationalFromSmart derives rotational state from a SmartInfo payload.
// Only trustworthy when SMART is reported supported by the device — when
// Supported=false, RotationRate=0 means "unknown" rather than "SSD".
func rotationalFromSmart(smartInfo *dto.SmartInfo) *bool {
	if smartInfo == nil || !smartInfo.Supported {
		return nil
	}
	if smartInfo.RotationRate > 0 {
		t := true
		return &t
	}
	f := false
	return &f
}
