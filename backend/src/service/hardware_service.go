package service

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/internal/darwinstubs/mount"
	"github.com/dianlight/tlog"
	"github.com/patrickmn/go-cache"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

const hwCacheKey = "hardware_info"

// wholeDiskNameRe matches whole-disk device names without a partition number
// suffix: plain letters ("sda"), NVMe ("nvme0n1"), eMMC ("mmcblk0") and loop
// ("loop0") devices. Partition children like "sda1" never match.
var wholeDiskNameRe = regexp.MustCompile(`^([a-z]+|nvme\d+n\d+|mmcblk\d+|loop\d+)$`)

// partitionCountRecord captures the partition layout of one disk as observed
// in a single hardware snapshot. Records persist across enumerations so
// partition-count jumps can be detected even when the 30-minute hardware cache
// is invalidated between snapshots (issue #990).
type partitionCountRecord struct {
	Serial     string   // drive serial as reported by the Supervisor (may be empty)
	DiskID     string   // canonical disk id used as the result map key
	LegacyName string   // kernel device name of the disk, e.g. "nvme0n1"
	Count      int      // number of partitions in this snapshot
	Partitions []string // sorted legacy device names of the partitions
}

// partitionIncrease describes one anomalous partition-count increase between
// two consecutive published snapshots.
type partitionIncrease struct {
	Key          string
	Serial       string
	DiskID       string
	Previous     int
	Current      int
	NewPartition []string // legacy names that appeared since the previous snapshot
}

// detectPartitionIncreases compares freshly built snapshot records against the
// previously published ones and reports every drive whose partition count grew,
// including which partition names appeared. Drives without a previous record are
// treated as baseline (first sighting); decreases and stable counts are not
// anomalous.
func detectPartitionIncreases(prev, current map[string]partitionCountRecord) map[string]partitionIncrease {
	increases := make(map[string]partitionIncrease)
	for key, rec := range current {
		before, ok := prev[key]
		if !ok || rec.Count <= before.Count {
			continue
		}
		prevParts := make(map[string]struct{}, len(before.Partitions))
		for _, name := range before.Partitions {
			prevParts[name] = struct{}{}
		}
		var added []string
		for _, name := range rec.Partitions {
			if _, seen := prevParts[name]; !seen {
				added = append(added, name)
			}
		}
		slices.Sort(added)
		increases[key] = partitionIncrease{
			Key:          key,
			Serial:       rec.Serial,
			DiskID:       rec.DiskID,
			Previous:     before.Count,
			Current:      rec.Count,
			NewPartition: added,
		}
	}
	return increases
}

// HardwareServiceInterface is the interface other services use.
// It exposes a method that returns a neutral, internal representation
// of hardware info (`hardware.HardwareInfo`) so other packages don't have
// to import the generated `hardware` package.
type HardwareServiceInterface interface {
	GetHardwareInfo() (map[string]dto.Disk, errors.E)
	InvalidateHardwareInfo()
	// Test only
	MockSetFSProbeFunc(f func(string) (string, uintptr, error))
}

type hardwareService struct {
	ctx           context.Context
	haClient      hardware.ClientWithResponsesInterface
	state         *dto.ContextState
	conv          converter.HaHardwareToDtoImpl
	smartService  SmartServiceInterface
	hdidleService HDIdleServiceInterface
	cache         *cache.Cache
	// Partition-count anomaly tracking (issue #990): lastPartitionCounts holds
	// the per-drive records of the most recently PUBLISHED snapshot so the next
	// build can warn about unexplained partition-count increases. It survives
	// cache invalidation on purpose; guarded by partitionCountsMu.
	partitionCountsMu   sync.Mutex
	lastPartitionCounts map[string]partitionCountRecord
	// readFile is os.ReadFile by default; tests override it to mock sysfs.
	readFile func(string) ([]byte, error)
	// sysBlockBasePath is "/sys/block" in production; tests override it
	// to point at a temp dir containing fake rotational files.
	sysBlockBasePath string
	// fsProbeFunc detects the filesystem type on a block device (magic scan).
	// It is mount.FSFromBlock by default; tests override it via MockSetFSProbeFunc.
	fsProbeFunc func(string) (string, uintptr, error)
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
		ctx:                 ctx,
		haClient:            haClient,
		conv:                converter.HaHardwareToDtoImpl{},
		smartService:        smartServiceInstance,
		hdidleService:       hdidleServiceInstance,
		state:               state,
		cache:               cache.New(30*time.Minute, 10*time.Minute),
		lastPartitionCounts: make(map[string]partitionCountRecord),
		readFile:            os.ReadFile,
		sysBlockBasePath:    "/sys/block",
		fsProbeFunc:         mount.FSFromBlock,
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
	// Per-drive partition layout of the snapshot being built, used right before
	// the cache write to detect anomalous partition-count jumps (issue #990).
	currentCounts := make(map[string]partitionCountRecord)
	if !h.state.HACoreReady {
		tlog.DebugContext(h.ctx, "HA Core not ready, cannot get hardware info", tlog.WithCaller(0)...)
		return ret, nil
	}
	hwser, errHw := h.haClient.GetHardwareInfoWithResponse(h.ctx)
	if errHw != nil || hwser == nil {
		if errors.Is(errHw, dto.ErrorNotFound) {
			slog.DebugContext(h.ctx, "Hardware info not found, continuing with empty disk list")
			return ret, nil
		}
		return nil, errors.WithDetails(errHw, "message", "failed to get hardware info from HA Supervisor", "hwset", hwser)
	}

	if hwser.StatusCode() != 200 || hwser.JSON200 == nil || hwser.JSON200.Data == nil || hwser.JSON200.Data.Drives == nil {
		errMsg := "Received invalid hardware info response from HA Supervisor"
		slog.ErrorContext(h.ctx, errMsg, "status_code", hwser.StatusCode(), "response_body", string(hwser.Body))
		return nil, errors.New(errMsg)
	}

	// Build map of whole-disk devices by serial for fallback probing of drives without filesystems
	wholeDiskDevicesBySerial := make(map[string]*hardware.Device)
	// Build map of all devices by name so fallback probing can resolve partition
	// children (e.g. "sdc1") into real partition filesystems.
	devicesByName := make(map[string]*hardware.Device)
	// Records the fs probe result for fallback-synthesized whole-disk filesystems,
	// keyed by device by-id path. Used by the main loop to override stale udev
	// ID_FS_TYPE values left behind after a disk's filesystem was removed.
	wholeDiskProbeFstypeByID := make(map[string]string)
	if hwser.JSON200.Data.Devices != nil {
		for deviceIdx := range *hwser.JSON200.Data.Devices {
			device := &(*hwser.JSON200.Data.Devices)[deviceIdx]
			if device.DevPath == nil || *device.DevPath == "" || device.Name == nil || *device.Name == "" {
				continue
			}
			devicesByName[*device.Name] = device
			// Whole-disk devices have names like "sda", "nvme0n1" (no partition number suffix)
			if !wholeDiskNameRe.MatchString(*device.Name) {
				continue
			}
			var serial string
			if device.Attributes != nil && device.Attributes.IDSERIALSHORT != nil {
				serial = *device.Attributes.IDSERIALSHORT
			}
			if serial != "" {
				wholeDiskDevicesBySerial[serial] = device
			}
		}
	}

	// Fallback probe: drives may have partitions without filesystem magic (e.g.
	// sda6 on a system disk) that the Supervisor omits from drive.Filesystems.
	// Synthesize the missing partition entries from the whole-disk device's
	// children; for drives with no filesystems at all, try to detect a
	// whole-disk filesystem instead.
	for driveIdx := range *hwser.JSON200.Data.Drives {
		drive := &(*hwser.JSON200.Data.Drives)[driveIdx]
		if drive.Serial == nil || *drive.Serial == "" {
			continue
		}
		device, ok := wholeDiskDevicesBySerial[*drive.Serial]
		if !ok || device.DevPath == nil || *device.DevPath == "" || device.ById == nil || *device.ById == "" {
			continue
		}
		if drive.Filesystems == nil {
			drive.Filesystems = &[]hardware.Filesystem{}
		}

		// Collect the device paths already reported by the Supervisor so child
		// synthesis does not duplicate filesystems that are already present.
		reportedFsDevices := make(map[string]struct{}, len(*drive.Filesystems))
		for _, fs := range *drive.Filesystems {
			if fs.Device != nil {
				reportedFsDevices[*fs.Device] = struct{}{}
			}
		}

		// When the whole-disk device reports partition children, synthesize one
		// real partition filesystem per child instead of a whole-disk filesystem.
		// This keeps real partitions (e.g. sdc1 on a flash disk) visible while
		// leaving the disk itself as a raw disk with no partitions. Children that
		// already have a reported filesystem (e.g. sda1 on a system disk) are
		// skipped; only the missing ones (e.g. sda6, no filesystem magic) are
		// added so the volume listing matches the real partition table.
		if device.Children != nil && len(*device.Children) > 0 {
			addedChild := false
			for _, childPath := range *device.Children {
				childName := filepath.Base(childPath)
				if childName == "" || childName == "." || childName == string(filepath.Separator) {
					continue
				}
				childDev, ok := devicesByName[childName]
				if !ok || childDev.DevPath == nil || *childDev.DevPath == "" || childDev.ById == nil || *childDev.ById == "" {
					tlog.DebugContext(h.ctx, "Skipping partition child without matching device entry", "drive_id", drive.Id, "child", childName)
					continue
				}
				if _, already := reportedFsDevices[*childDev.DevPath]; already {
					continue
				}
				childFsId := "by-id-" + strings.TrimPrefix(*childDev.ById, "/dev/disk/by-id/")
				childFS := hardware.Filesystem{
					Device:      childDev.DevPath,
					Id:          &childFsId,
					Name:        childDev.Name,
					MountPoints: &[]string{},
				}
				*drive.Filesystems = append(*drive.Filesystems, childFS)
				addedChild = true
			}
			if addedChild {
				tlog.DebugContext(h.ctx, "Synthesized partition filesystems from child devices", "drive_id", drive.Id, "device", *device.DevPath, "children", len(*device.Children))
				continue
			}
		}

		// Whole-disk synthesis only applies when the drive still has no
		// filesystems (none reported and no child produced one).
		if len(*drive.Filesystems) > 0 {
			continue
		}
		fstype, _, _ := h.fsProbeFunc(*device.DevPath)

		// Create a synthetic filesystem for the whole disk even when no readable
		// filesystem magic was found (e.g. an MBR with an unreadable partition):
		// the drive still needs mount/unmount/check/format actions. Use a by-id-
		// prefixed ID so the converter resolves it without hitting /dev/disk/by-uuid/.
		fsId := "by-id-" + strings.TrimPrefix(*device.ById, "/dev/disk/by-id/")
		synthFS := hardware.Filesystem{
			Device:      device.DevPath,
			Id:          &fsId,
			Name:        device.Name,
			MountPoints: &[]string{},
		}
		*drive.Filesystems = append(*drive.Filesystems, synthFS)
		// Remember the probe result so the main loop can avoid trusting stale
		// udev ID_FS_TYPE on a raw whole-disk device (e.g. a previously formatted
		// disk whose filesystem has since been removed).
		wholeDiskProbeFstypeByID[*device.ById] = fstype
		tlog.DebugContext(h.ctx, "Detected whole-disk filesystem via fallback probe", "drive_id", drive.Id, "device", *device.DevPath, "fstype", fstype)
	}

	tlog.DebugContext(h.ctx, "Processing drives from HA Supervisor", "drive_count", len(*hwser.JSON200.Data.Drives))
	for i, drive := range *hwser.JSON200.Data.Drives {
		var diskDto dto.Disk
		errConvDrive := h.conv.DriveToDisk(drive, &diskDto)
		if errConvDrive != nil {
			tlog.WarnContext(h.ctx, "Error converting drive to disk DTO", "drive_index", i, "drive_id", drive.Id, "err", errConvDrive)
			continue
		}

		// Find corresponding Device entries for Disk and its Partitions
		if hwser.JSON200.Data.Devices != nil {
			for deviceIdx := range *hwser.JSON200.Data.Devices {
				device := &(*hwser.JSON200.Data.Devices)[deviceIdx]
				if device.DevPath == nil || *device.DevPath == "" || device.Name == nil || *device.Name == "" || device.ById == nil || *device.ById == "" {
					tlog.DebugContext(h.ctx, "Skipping device with nil or empty DevPath/Name/ById", "drive_index", i, "drive_id", drive.Id, "device_index", deviceIdx)
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

					// Do not continue: a whole-disk filesystem (e.g. superfloppy
					// or an unreadable partition table) produces a partition
					// whose LegacyDeviceName equals the disk name, so it must
					// also run through the partition match below to populate
					// DevicePath/FsType.
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
							}
							// Prefer the fallback probe result for synthesized
							// whole-disk filesystems: udev ID_FS_TYPE can be stale
							// after a disk's filesystem was removed. Real child
							// partitions (not in wholeDiskProbeFstypeByID) keep
							// the existing IDFSTYPE behavior.
							if probeFstype, ok := wholeDiskProbeFstypeByID[*device.ById]; ok {
								if probeFstype != "" {
									partition.FsType = &probeFstype
								}
							} else if device.Attributes != nil && device.Attributes.IDFSTYPE != nil && *device.Attributes.IDFSTYPE != "" {
								partition.FsType = device.Attributes.IDFSTYPE
							} else if device.DevPath != nil && *device.DevPath != "" && h.fsProbeFunc != nil {
								// Issue #1072: Supervisor udev entry reports no
								// ID_FS_TYPE while blkid sees a healthy FS.
								// Probe magic directly so /api/volumes reports
								// the real type and canMount resolves true.
								if probed, _, probeErr := h.fsProbeFunc(*device.DevPath); probeErr == nil && probed != "" {
									partition.FsType = new(probed)
									tlog.DebugContext(h.ctx, "Recovered missing fstype via FS probe", "device", *device.DevPath, "fstype", probed)
								} else if probeErr != nil {
									tlog.DebugContext(h.ctx, "FS probe failed for partition with empty fstype", "device", *device.DevPath, "error", probeErr)
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
		// Issue #1072 follow-up: Supervisor may omit the Devices entry for a
		// partition entirely (observed: sdc1 present in drive.Filesystems with
		// LegacyDevicePath /dev/sdc1 but no matching device, leaving
		// DevicePath nil and FsType nil). Recover via LegacyDevicePath probe
		// and reconstruct DevicePath from the by-id partition Id.
		if diskDto.Partitions != nil {
			for pid, part := range *diskDto.Partitions {
				partition := part
				needsDevicePath := partition.DevicePath == nil || *partition.DevicePath == ""
				needsFsType := partition.FsType == nil || *partition.FsType == ""
				if !needsDevicePath && !needsFsType {
					continue
				}
				if partition.LegacyDeviceName == nil || *partition.LegacyDeviceName == "" {
					continue
				}
				if needsDevicePath && partition.Id != nil && *partition.Id != "" {
					trimmed := strings.TrimSpace(*partition.Id)
					trimmed = strings.TrimPrefix(trimmed, "by-id-")
					if trimmed != "" && !strings.HasPrefix(trimmed, "by-uuid-") {
						reconstructed := "/dev/disk/by-id/" + trimmed
						partition.DevicePath = new(reconstructed)
						tlog.DebugContext(h.ctx, "Reconstructed missing device path from partition id", "partition_id", *partition.Id, "device_path", reconstructed)
					}
				}
				if needsFsType && partition.LegacyDevicePath != nil && *partition.LegacyDevicePath != "" && h.fsProbeFunc != nil {
					if probed, _, probeErr := h.fsProbeFunc(*partition.LegacyDevicePath); probeErr == nil && probed != "" {
						partition.FsType = new(probed)
						tlog.DebugContext(h.ctx, "Recovered missing fstype via legacy device probe", "device", *partition.LegacyDevicePath, "fstype", probed)
					} else if probeErr != nil {
						tlog.DebugContext(h.ctx, "Legacy device probe failed for partition with missing device entry", "device", *partition.LegacyDevicePath, "error", probeErr)
					}
				}
				if partition.DiskId == nil || *partition.DiskId == "" {
					partition.DiskId = diskDto.Id
				}
				(*diskDto.Partitions)[pid] = partition
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

		rec := partitionCountRecord{
			DiskID: *diskDto.Id,
		}
		if drive.Serial != nil {
			rec.Serial = *drive.Serial
		}
		if diskDto.LegacyDeviceName != nil {
			rec.LegacyName = *diskDto.LegacyDeviceName
		}
		if diskDto.Partitions != nil {
			rec.Count = len(*diskDto.Partitions)
			for _, part := range *diskDto.Partitions {
				if part.LegacyDeviceName != nil && *part.LegacyDeviceName != "" {
					rec.Partitions = append(rec.Partitions, *part.LegacyDeviceName)
				}
			}
			slices.Sort(rec.Partitions)
		}
		diffKey := rec.Serial
		if diffKey == "" {
			diffKey = rec.DiskID
		}
		currentCounts[diffKey] = rec
	}

	// Diff partition counts against the previous published snapshot and log any
	// unexplained increase BEFORE publishing, then populate the cache.
	h.recordPartitionCounts(currentCounts)
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

// recordPartitionCounts logs the per-drive serial/partition-count summary of a
// freshly built snapshot and warns when any drive's count increased relative to
// the previously published snapshot. Such increases never originate from SRAT
// itself (no local code partitions disks) and are the fingerprint of the
// phantom-partition defect tracked in issue #990: kernel partition rescans —
// often triggered by side-effecting SG_IO probes — surfacing transient
// partitions through Supervisor enumeration data. Records for drives absent
// from the current snapshot are kept so a reappearing disk is still diffed
// against its own history. Callers must invoke this exactly once per published
// snapshot, immediately before the cache write.
func (h *hardwareService) recordPartitionCounts(current map[string]partitionCountRecord) {
	h.partitionCountsMu.Lock()
	defer h.partitionCountsMu.Unlock()
	if h.lastPartitionCounts == nil {
		h.lastPartitionCounts = make(map[string]partitionCountRecord, len(current))
	}
	for key, rec := range current {
		slog.DebugContext(h.ctx, "Hardware enumeration partition summary",
			"disk_key", key,
			"serial", rec.Serial,
			"disk_id", rec.DiskID,
			"legacy_device_name", rec.LegacyName,
			"partition_count", rec.Count,
		)
	}
	for key, inc := range detectPartitionIncreases(h.lastPartitionCounts, current) {
		slog.WarnContext(h.ctx, "Anomalous partition count increase without local partitioning action",
			"disk_key", key,
			"serial", inc.Serial,
			"disk_id", inc.DiskID,
			"previous_partition_count", inc.Previous,
			"current_partition_count", inc.Current,
			"new_partitions", strings.Join(inc.NewPartition, ","),
		)
	}
	maps.Copy(h.lastPartitionCounts, current)
}

// MockSetFSProbeFunc allows tests to override the filesystem probe used by the
// fallback whole-disk detection (avoids touching real block devices in tests).
func (h *hardwareService) MockSetFSProbeFunc(f func(string) (string, uintptr, error)) {
	h.fsProbeFunc = f
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
