package service

import (
	"context"
	"log/slog"
	"math"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
	"github.com/prometheus/procfs/blockdevice"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// DiskStatsService is a service for collecting disk I/O statistics.
type DiskStatsService interface {
	GetDiskStats() (*dto.DiskHealth, errors.E)
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
	filesystemService FilesystemServiceInterface
	ioStatFetcher     func(string) (blockdevice.IOStats, error)
}

// NewDiskStatsService creates a new DiskStatsService.
func NewDiskStatsService(lc fx.Lifecycle, VolumeService VolumeServiceInterface, Ctx context.Context, SmartService SmartServiceInterface, HDIdleService HDIdleServiceInterface, FilesystemService FilesystemServiceInterface) DiskStatsService {
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
		volumeService:     VolumeService,
		blockdevice:       &fs,
		ctx:               Ctx,
		lastUpdateTime:    time.Now(),
		updateMutex:       &sync.Mutex{},
		lastStats:         make(map[string]*blockdevice.IOStats),
		smartService:      SmartService,
		hdidleService:     HDIdleService,
		filesystemService: FilesystemService,
	}
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
			stats, err := s.fetchDiskStats(*disk.LegacyDeviceName)

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
				if disk.DevicePath != nil {
					smartStatus, err := s.smartService.GetSmartStatus(s.ctx, *disk.Id)
					if err != nil && !errors.Is(err, dto.ErrorSMARTNotSupported) {
						slog.WarnContext(s.ctx, "Error getting SMART status", "disk", *disk.Id, "err", err)
					} else if smartStatus != nil {
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
					fsckSupported, fsckNeeded := s.getFsckInfo(&part, fstype)
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

func (s *diskStatsService) getFsckInfo(part *dto.Partition, fsType string) (bool, bool) {
	if part == nil || s.filesystemService == nil || fsType == "" {
		return false, false
	}

	info, err := s.filesystemService.GetSupportAndInfo(s.ctx, fsType)
	if err != nil || info == nil || info.Support == nil || !info.Support.CanCheck {
		return false, false
	}

	devicePath := resolvePartitionDevicePath(part)
	if devicePath == "" || !info.Support.CanGetState {
		return true, false
	}

	state, err := s.filesystemService.GetPartitionState(s.ctx, devicePath, fsType)
	if err != nil || state == nil {
		return true, false
	}

	if state.HasErrors || !state.IsClean {
		return true, true
	}

	return true, false
}

func resolvePartitionDevicePath(part *dto.Partition) string {
	if part == nil {
		return ""
	}
	if part.LegacyDevicePath != nil && *part.LegacyDevicePath != "" {
		return *part.LegacyDevicePath
	}
	if part.DevicePath != nil && *part.DevicePath != "" {
		return *part.DevicePath
	}
	if part.LegacyDeviceName != nil && *part.LegacyDeviceName != "" {
		return "/dev/" + *part.LegacyDeviceName
	}
	return ""
}

func (s *diskStatsService) fetchDiskStats(deviceName string) (blockdevice.IOStats, error) {
	if s.ioStatFetcher != nil {
		return s.ioStatFetcher(deviceName)
	}
	if s.blockdevice == nil {
		return blockdevice.IOStats{}, os.ErrInvalid
	}
	stats, _, err := s.blockdevice.SysBlockDeviceStat(deviceName)
	if err != nil {
		return blockdevice.IOStats{}, err
	}
	return stats, nil
}
