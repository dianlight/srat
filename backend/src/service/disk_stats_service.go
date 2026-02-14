package service

import (
	"context"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/tlog"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/procfs/blockdevice"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// DiskStatsService is a service for collecting disk I/O statistics.
type DiskStatsService interface {
	GetDiskStats() (*dto.DiskHealth, errors.E)
	InvalidateSmartCache(diskId string) // Invalidates SMART enabled cache for a disk (empty string = all)
}

type diskStatsService struct {
	volumeService     VolumeServiceInterface
	blockdevice       *blockdevice.FS
	ctx               context.Context
	lastUpdateTime    time.Time                       // lastUpdateTime is used to track the last time disk stats were updated.
	lastStats         map[string]*blockdevice.IOStats // lastStats stores the last collected disk I/O statistics.
	currentDiskHealth *dto.DiskHealth
	updateMutex       *sync.Mutex
	smartService      SmartServiceInterface
	hdidleService     HDIdleServiceInterface
	readFile          func(string) ([]byte, error)
	sysFsBasePath     string
	smartEnabledCache *cache.Cache // smartEnabledCache tracks SMART enabled/disabled state per disk to avoid unnecessary disk access
}

// NewDiskStatsService creates a new DiskStatsService.
func NewDiskStatsService(lc fx.Lifecycle, VolumeService VolumeServiceInterface, Ctx context.Context, SmartService SmartServiceInterface, HDIdleService HDIdleServiceInterface, EventBus events.EventBusInterface) DiskStatsService {
	var fs blockdevice.FS
	var err error

	// Only try to initialize filesystem if we're not in mock mode
	if os.Getenv("SRAT_MOCK") != "true" {
		fs, err = blockdevice.NewFS("/proc", "/sys")
		if err != nil {
			slog.WarnContext(Ctx, "Failed to create block device filesystem, using mock data", "error", err)
		}
	}

	ds := &diskStatsService{
		volumeService:  VolumeService,
		blockdevice:    &fs,
		ctx:            Ctx,
		lastUpdateTime: time.Now(),
		updateMutex:    &sync.Mutex{},
		lastStats:      make(map[string]*blockdevice.IOStats),
		smartService:   SmartService,
		hdidleService:  HDIdleService,
		readFile:       os.ReadFile,
		sysFsBasePath:  "/sys/fs",
		// Initialize cache with 30 minute default expiration and 10 minute cleanup interval
		smartEnabledCache: cache.New(30*time.Minute, 10*time.Minute),
	}

	// Subscribe to SMART events to populate cache immediately when SMART is enabled/disabled
	// This ensures no disk queries are needed after the state change
	unsubscribeSmart := EventBus.OnSmart(func(ctx context.Context, event events.SmartEvent) errors.E {
		if event.SmartInfo.DiskId != "" {
			// The SmartInfo now contains both Supported (hardware capability) and Enabled (software state)
			// We cache the Enabled state to prevent any disk queries after enable/disable
			enabled := event.SmartInfo.Enabled

			// Set cache with default expiration (30 minutes)
			ds.smartEnabledCache.Set(event.SmartInfo.DiskId, enabled, cache.DefaultExpiration)

			tlog.DebugContext(ctx, "SMART event received, cache populated immediately",
				"disk", event.SmartInfo.DiskId,
				"supported", event.SmartInfo.Supported,
				"enabled", enabled,
				"event_type", event.Type)
		}
		return nil
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := ds.updateDiskStats()
			if err != nil && !errors.Is(err, dto.ErrorNotFound) {
				slog.WarnContext(ctx, "Failed to update disk stats", "error", err)
			}
			wg := Ctx.Value("wg").(*sync.WaitGroup)
			wg.Go(func() { ds.run() })
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if unsubscribeSmart != nil {
				unsubscribeSmart()
			}
			return nil
		},
	})
	return ds
}

func (self *diskStatsService) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Run process closed", "err", self.ctx.Err())
			return errors.WithStack(self.ctx.Err())
		case <-time.After(time.Second * 10):
			err := self.updateDiskStats()
			if err != nil && !errors.Is(err, dto.ErrorNotFound) {
				slog.WarnContext(self.ctx, "Failed to update disk stats", "error", err)
				continue
			}
		}
	}
}

func (s *diskStatsService) updateDiskStats() errors.E {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	disks := s.volumeService.GetVolumesData()

	// Check HDIdle service status
	hdidleRunning := false
	if s.hdidleService != nil {
		hdidleRunning = s.hdidleService.IsRunning()
	}

	s.currentDiskHealth = &dto.DiskHealth{
		PerDiskIO: make([]dto.DiskIOStats, 0),
		Global: dto.GlobalDiskStats{
			TotalIOPS:         0,
			TotalReadLatency:  0,
			TotalWriteLatency: 0,
		},
		PerPartitionInfo: make(map[string][]dto.PerPartitionInfo, 0),
		PerDiskInfo:      make(map[string]dto.PerDiskInfo),
		HDIdleRunning:    hdidleRunning,
	}

	if len(disks) != 0 {
		for _, disk := range disks {
			if disk.DevicePath == nil {
				slog.DebugContext(s.ctx, "Skipping disk with nil device", "diskID", *disk.Id)
				continue
			}
			stats, _, err := s.blockdevice.SysBlockDeviceStat(*disk.LegacyDeviceName)

			if err != nil {
				if os.IsNotExist(errors.Unwrap(err)) {
					tlog.TraceContext(s.ctx, "Disk device not found in /proc, skipping", "disk", *disk.LegacyDeviceName)
					continue
				}
				return errors.WithStack(err)
			}
			if s.lastStats[*disk.Id] != nil {

				dstat := dto.DiskIOStats{
					DeviceName:        *disk.LegacyDeviceName,
					DeviceDescription: *disk.Id,
					ReadIOPS:          (float64(stats.ReadIOs) - float64(s.lastStats[*disk.Id].ReadIOs)) / (time.Since(s.lastUpdateTime).Seconds()),
					WriteIOPS:         (float64(stats.WriteIOs) - float64(s.lastStats[*disk.Id].WriteIOs)) / (time.Since(s.lastUpdateTime).Seconds()),
					ReadLatency:       (float64(stats.ReadTicks) - float64(s.lastStats[*disk.Id].ReadTicks)) / (float64(stats.ReadIOs) - float64(s.lastStats[*disk.Id].ReadIOs)),
					WriteLatency:      (float64(stats.WriteTicks) - float64(s.lastStats[*disk.Id].WriteTicks)) / (float64(stats.WriteIOs) - float64(s.lastStats[*disk.Id].WriteIOs)),
				}
				if dstat.ReadIOPS < 0 || math.IsNaN(dstat.ReadIOPS) {
					dstat.ReadIOPS = 0
				}
				if dstat.WriteIOPS < 0 || math.IsNaN(dstat.WriteIOPS) {
					dstat.WriteIOPS = 0
				}
				if dstat.ReadLatency < 0 || math.IsNaN(dstat.ReadLatency) {
					dstat.ReadLatency = 0
				}
				if dstat.WriteLatency < 0 || math.IsNaN(dstat.WriteLatency) {
					dstat.WriteLatency = 0
				}
				s.currentDiskHealth.PerDiskIO = append(s.currentDiskHealth.PerDiskIO, dstat)

				s.currentDiskHealth.Global.TotalIOPS += dstat.ReadIOPS + dstat.WriteIOPS
				if dstat.ReadIOPS+dstat.WriteIOPS > 0 {
					s.currentDiskHealth.Global.TotalReadLatency += dstat.ReadLatency
				}
				if dstat.WriteIOPS > 0 {
					s.currentDiskHealth.Global.TotalWriteLatency += dstat.WriteLatency
				}
				s.lastStats[*disk.Id] = &stats

				// --- Smart data population ---
				// Only query SMART data if SMART is enabled to avoid unnecessary disk access
				if disk.DevicePath != nil && s.isSmartEnabled(*disk.Id) {
					smartStatus, err := s.smartService.GetSmartStatus(s.ctx, *disk.Id)
					if err != nil && !errors.Is(err, dto.ErrorSMARTNotSupported) {
						slog.WarnContext(s.ctx, "Error getting SMART status", "disk", *disk.Id, "err", err)
					} else if smartStatus != nil {
						// Update cache with current enabled state using Set method
						s.smartEnabledCache.Set(*disk.Id, smartStatus.Enabled, cache.DefaultExpiration)
						s.currentDiskHealth.PerDiskIO[len(s.currentDiskHealth.PerDiskIO)-1].SmartData = smartStatus
					}
				}
			} else {
				s.lastStats[*disk.Id] = &stats
			} // --- PerPartitionInfo population ---
			if disk.Partitions != nil {
				for _, part := range *disk.Partitions {

					// Fill PerPartitionInfo
					var fstype, name string
					if part.FsType != nil {
						fstype = *part.FsType
					}
					if part.Name != nil {
						name = *part.Name
					} else if part.LegacyDeviceName != nil {
						name = *part.LegacyDeviceName
					} else if part.Id != nil {
						name = *part.Id
					} else {
						name = "unknown"
					}
					fsckSupported := s.isFsckSupported(fstype)
					fsckNeeded := s.determineFsckNeeded(&part, fstype, fsckSupported)
					// Use partition size if available
					// Get free space using syscall.Statfs
					var totalSpace, freeSpace uint64
					mountPoint := ""
					if part.Size != nil {
						// Prevent integer overflow/underflow converting int -> uint64
						if *part.Size > 0 {
							totalSpace = uint64(*part.Size)
						} else {
							totalSpace = 0
						}
					}
					if part.MountPointData != nil && len(*part.MountPointData) > 0 {
						// Use first mount point if multiple (shouldn't normally happen)
						var mp dto.MountPointData
						for _, m := range *part.MountPointData {
							mp = m
							break
						}
						if mp.Path != "" {
							mountPoint = mp.Path

							var stat syscall.Statfs_t
							if err := syscall.Statfs(mp.Path, &stat); err == nil {
								// Guard against negative block size before converting to uint64
								if stat.Bsize > 0 {
									bsize := uint64(stat.Bsize)
									freeSpace = stat.Bfree * bsize
									totalSpace = stat.Blocks * bsize
								}
							}
						}
					}
					info := dto.PerPartitionInfo{
						Name:          name,
						MountPoint:    mountPoint,
						Device:        *part.Id,
						FSType:        fstype,
						FreeSpace:     freeSpace,
						TotalSpace:    totalSpace,
						FsckNeeded:    fsckNeeded,
						FsckSupported: fsckSupported,
					}
					if s.currentDiskHealth.PerPartitionInfo[*disk.Id] == nil {
						s.currentDiskHealth.PerPartitionInfo[*disk.Id] = make([]dto.PerPartitionInfo, 0)
					}
					s.currentDiskHealth.PerPartitionInfo[*disk.Id] = append(s.currentDiskHealth.PerPartitionInfo[*disk.Id], info)
				}

			}

			// --- PerDiskInfo population (SMART info, health, HDIdle status) ---
			s.populatePerDiskInfo(disk)
		}
	}

	s.lastUpdateTime = time.Now()
	return nil
}

// populatePerDiskInfo populates the PerDiskInfo map with SMART info, health status, and HDIdle status for a disk.
func (s *diskStatsService) populatePerDiskInfo(disk *dto.Disk) {
	if disk == nil || disk.Id == nil {
		return
	}

	diskInfo := dto.PerDiskInfo{
		DeviceId: *disk.Id,
	}

	if disk.DevicePath != nil {
		diskInfo.DevicePath = *disk.DevicePath

		// Only query SMART data if SMART is enabled to avoid unnecessary disk access
		if s.isSmartEnabled(*disk.Id) {
			// Get SMART info (static capabilities)
			smartInfo, err := s.smartService.GetSmartInfo(s.ctx, *disk.Id)
			if err != nil && !errors.Is(err, dto.ErrorSMARTNotSupported) {
				tlog.WarnContext(s.ctx, "Error getting SMART info", "disk", *disk.Id, "err", err)
			} else if smartInfo != nil {
				diskInfo.SmartInfo = smartInfo
			}

			if diskInfo.SmartInfo != nil && diskInfo.SmartInfo.Supported {
				// Get SMART health status
				smartHealth, err := s.smartService.GetHealthStatus(s.ctx, *disk.Id)
				if err != nil && !errors.Is(err, dto.ErrorSMARTNotSupported) {
					tlog.WarnContext(s.ctx, "Error getting SMART health status", "disk", *disk.Id, "err", err)
				} else if smartHealth != nil {
					diskInfo.SmartHealth = smartHealth
				}
			}
		}

		// Get HDIdle status
		if s.hdidleService != nil {
			hdidleStatus := s.getHDIdleDeviceStatus(*disk.DevicePath)
			if hdidleStatus != nil {
				diskInfo.HDIdleStatus = hdidleStatus
			}
		}
	}

	s.currentDiskHealth.PerDiskInfo[*disk.Id] = diskInfo
}

// getHDIdleDeviceStatus retrieves the HDIdle status for a specific device.
func (s *diskStatsService) getHDIdleDeviceStatus(devicePath string) *dto.HDIdleDeviceStatus {
	if s.hdidleService == nil {
		return nil
	}

	// Get device-specific status if HDIdle is running
	if s.hdidleService.IsRunning() {
		deviceStatus, err := s.hdidleService.GetDeviceStatus(devicePath)
		if err != nil {
			tlog.DebugContext(s.ctx, "Error getting HDIdle device status", "device", devicePath, "err", err)
		} else if deviceStatus != nil {
			return deviceStatus
		}
	}

	return nil
}

// GetDiskStats collects and returns disk I/O statistics.
func (s *diskStatsService) GetDiskStats() (*dto.DiskHealth, errors.E) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if s.currentDiskHealth == nil {
		return nil, errors.New("disk stats not initialized")
	}
	return s.currentDiskHealth, nil
}

func (s *diskStatsService) isFsckSupported(fstype string) bool {
	switch strings.ToLower(fstype) {
	case "ext2", "ext3", "ext4", "xfs", "btrfs", "f2fs", "vfat", "exfat", "ntfs", "ntfs3":
		return true
	default:
		return false
	}
}

func (s *diskStatsService) determineFsckNeeded(part *dto.Partition, fstype string, fsckSupported bool) bool {
	if part == nil || !fsckSupported {
		return false
	}

	mounted, hasMountInfo := partitionMountState(part)
	if hasMountInfo && !mounted {
		return true
	}

	if partitionHasDirtyIndicators(part) {
		return true
	}

	if s.hasPendingFsState(part, fstype) {
		return true
	}

	return false
}

func partitionMountState(part *dto.Partition) (isMounted bool, hasInfo bool) {
	checkMounts := func(mounts *map[string]dto.MountPointData) {
		if mounts == nil {
			return
		}
		if len(*mounts) > 0 {
			hasInfo = true
		}
		for _, mp := range *mounts {
			if mp.IsMounted {
				isMounted = true
			}
			if mp.Path != "" || mp.IsToMountAtStartup != nil {
				hasInfo = true
			}
		}
	}

	checkMounts(part.MountPointData)
	checkMounts(part.HostMountPointData)

	return isMounted, hasInfo
}

func partitionHasDirtyIndicators(part *dto.Partition) bool {
	hasIndicator := func(mounts *map[string]dto.MountPointData) bool {
		if mounts == nil {
			return false
		}
		for _, mp := range *mounts {
			if mp.InvalidError != nil && containsDirtyKeyword(*mp.InvalidError) {
				return true
			}
			if mp.Warnings != nil && containsDirtyKeyword(*mp.Warnings) {
				return true
			}
		}
		return false
	}

	return hasIndicator(part.MountPointData) || hasIndicator(part.HostMountPointData)
}

func containsDirtyKeyword(value string) bool {
	lower := strings.ToLower(value)
	keywords := []string{"fsck", "dirty", "recover", "inconsist", "corrupt"}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func (s *diskStatsService) hasPendingFsState(part *dto.Partition, fstype string) bool {
	device := normalizeDeviceName(part)
	if device == "" {
		return false
	}

	switch strings.ToLower(fstype) {
	case "ext2", "ext3", "ext4":
		if s.checkSysfsState("ext4", device, []string{"needs_recovery", "error", "not clean"}) {
			return true
		}
		base := filepath.Join(s.sysFsBasePath, "ext4", device)
		if s.checkSysfsBool(filepath.Join(base, "needs_recovery")) {
			return true
		}
		if s.checkSysfsNonZero(filepath.Join(base, "errors_count")) {
			return true
		}
	case "xfs":
		if s.checkSysfsState("xfs", device, []string{"dirty", "recover"}) {
			return true
		}
		base := filepath.Join(s.sysFsBasePath, "xfs", device)
		if s.checkSysfsNonZero(filepath.Join(base, "errors")) {
			return true
		}
	case "f2fs":
		if s.checkSysfsState("f2fs", device, []string{"dirty", "corrupt", "invalid"}) {
			return true
		}
	}

	return false
}

func (s *diskStatsService) checkSysfsState(fsDir, device string, keywords []string) bool {
	statePath := filepath.Join(s.sysFsBasePath, fsDir, device, "state")
	content, err := s.readTrimmed(statePath)
	if err != nil {
		return false
	}
	lower := strings.ToLower(content)
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func (s *diskStatsService) checkSysfsBool(path string) bool {
	content, err := s.readTrimmed(path)
	if err != nil {
		return false
	}
	if content == "" {
		return false
	}
	switch strings.ToLower(content) {
	case "0", "false", "clean":
		return false
	default:
		return true
	}
}

func (s *diskStatsService) checkSysfsNonZero(path string) bool {
	content, err := s.readTrimmed(path)
	if err != nil {
		return false
	}
	if content == "" {
		return false
	}
	if val, err := strconv.ParseUint(content, 10, 64); err == nil {
		return val > 0
	}
	return true
}

func (s *diskStatsService) readTrimmed(path string) (string, error) {
	if s.readFile == nil {
		return "", os.ErrInvalid
	}
	data, err := s.readFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func normalizeDeviceName(part *dto.Partition) string {
	if part == nil {
		return ""
	}
	if part.LegacyDeviceName != nil && *part.LegacyDeviceName != "" {
		return sanitizeDeviceName(*part.LegacyDeviceName)
	}
	if part.LegacyDevicePath != nil && *part.LegacyDevicePath != "" {
		return sanitizeDeviceName(filepath.Base(*part.LegacyDevicePath))
	}
	if part.DevicePath != nil && *part.DevicePath != "" {
		return sanitizeDeviceName(filepath.Base(*part.DevicePath))
	}
	return ""
}

// isSmartEnabled checks if SMART is enabled for a disk.
// It uses a cache to avoid querying the disk unnecessarily.
// Cache is populated from SMART events when SMART is enabled/disabled via API.
// On cache miss, it queries SMART info once and caches the result.
func (s *diskStatsService) isSmartEnabled(diskId string) bool {
	// Check cache first using go-cache
	if cachedValue, found := s.smartEnabledCache.Get(diskId); found {
		if enabled, ok := cachedValue.(bool); ok {
			tlog.TraceContext(s.ctx, "SMART cache hit", "disk", diskId, "enabled", enabled)
			return enabled
		}
	}

	// Cache miss - query once to populate cache
	// This should rarely happen because cache is populated from SMART events
	tlog.DebugContext(s.ctx, "SMART cache miss, querying disk", "disk", diskId)

	smartInfo, err := s.smartService.GetSmartInfo(s.ctx, diskId)
	if err != nil {
		// SMART not supported or error - cache as disabled to avoid future queries
		s.smartEnabledCache.Set(diskId, false, cache.DefaultExpiration)
		tlog.WarnContext(s.ctx, "SMART query failed, caching as disabled", "disk", diskId, "err", err)
		return false
	}

	if smartInfo == nil || !smartInfo.Supported {
		// SMART not supported - cache as disabled
		s.smartEnabledCache.Set(diskId, false, cache.DefaultExpiration)
		tlog.DebugContext(s.ctx, "SMART not supported, caching as disabled", "disk", diskId)
		return false
	}

	// Use the Enabled field from SmartInfo (now populated by converter)
	enabled := smartInfo.Enabled
	s.smartEnabledCache.Set(diskId, enabled, cache.DefaultExpiration)
	tlog.DebugContext(s.ctx, "SMART state cached from query", "disk", diskId, "enabled", enabled)
	return enabled
}

// InvalidateSmartCache clears the SMART enabled cache for a specific disk or all disks.
// This method is exposed in the interface but typically not needed since cache is populated from events.
func (s *diskStatsService) InvalidateSmartCache(diskId string) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if diskId == "" {
		// Clear entire cache using Flush
		s.smartEnabledCache.Flush()
		tlog.DebugContext(s.ctx, "SMART cache flushed (all disks)")
	} else {
		// Clear specific disk using Delete
		s.smartEnabledCache.Delete(diskId)
		tlog.DebugContext(s.ctx, "SMART cache entry deleted", "disk", diskId)
	}
}

func sanitizeDeviceName(name string) string {
	trimmed := strings.TrimSpace(name)
	trimmed = strings.TrimPrefix(trimmed, "/dev/")
	trimmed = strings.ReplaceAll(trimmed, "/", "!")
	return trimmed
}
