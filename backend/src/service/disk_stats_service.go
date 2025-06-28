package service

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/tlog"
	"github.com/prometheus/procfs/blockdevice"
	"gitlab.com/tozd/go/errors"
)

// DiskStatsService is a service for collecting disk I/O statistics.
type DiskStatsService interface {
	GetDiskStats() (*dto.DiskHealth, error)
}

type diskStatsService struct {
	volumeService     VolumeServiceInterface
	blockdevice       *blockdevice.FS
	ctx               context.Context
	lastUpdateTime    time.Time                       // lastUpdateTime is used to track the last time disk stats were updated.
	lastStats         map[string]*blockdevice.IOStats // lastStats stores the last collected disk I/O statistics.
	currentDiskHealth *dto.DiskHealth
	updateMutex       *sync.Mutex
	smartService      SmartService
}

// NewDiskStatsService creates a new DiskStatsService.
func NewDiskStatsService(VolumeService VolumeServiceInterface, Ctx context.Context, SmartService SmartService) DiskStatsService {
	fs, err := blockdevice.NewFS("/proc", "/sys")
	if err != nil {
		tlog.Error("Failed to create block device filesystem", "error", err)
	}

	ds := &diskStatsService{
		volumeService:  VolumeService,
		blockdevice:    &fs,
		ctx:            Ctx,
		lastUpdateTime: time.Now(),
		updateMutex:    &sync.Mutex{},
		lastStats:      make(map[string]*blockdevice.IOStats),
		smartService:   SmartService,
	}
	Ctx.Value("wg").(*sync.WaitGroup).Add(1)
	go func() {
		defer Ctx.Value("wg").(*sync.WaitGroup).Done()
		ds.run()
	}()
	return ds
}

func (self *diskStatsService) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.Info("Run process closed", "err", self.ctx.Err())
			return errors.WithStack(self.ctx.Err())
		case <-time.After(time.Second * 10):
			err := self.updateDiskStats()
			if err != nil {
				tlog.Error("Failed to update disk stats", "error", err)
				continue
			}
		}
	}
}

func (s *diskStatsService) updateDiskStats() error {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	disks, err := s.volumeService.GetVolumesData()
	if err != nil {
		return err
	}

	s.currentDiskHealth = &dto.DiskHealth{
		PerDiskIO: make([]dto.DiskIOStats, 0),
		Global: dto.GlobalDiskStats{
			TotalIOPS:         0,
			TotalReadLatency:  0,
			TotalWriteLatency: 0,
		},
		PerPartitionInfo: make(map[string][]dto.PerPartitionInfo, 0),
	}

	for _, disk := range *disks {
		stats, _, err := s.blockdevice.SysBlockDeviceStat(*disk.Device)
		if err != nil {
			return err
		}
		if s.lastStats[*disk.Device] != nil {

			dstat := dto.DiskIOStats{
				DeviceName:   *disk.Id,
				ReadIOPS:     (float64(stats.ReadIOs) - float64(s.lastStats[*disk.Device].ReadIOs)) / (time.Since(s.lastUpdateTime).Seconds()),
				WriteIOPS:    (float64(stats.WriteIOs) - float64(s.lastStats[*disk.Device].WriteIOs)) / (time.Since(s.lastUpdateTime).Seconds()),
				ReadLatency:  (float64(stats.ReadTicks) - float64(s.lastStats[*disk.Device].ReadTicks)) / (float64(stats.ReadIOs) - float64(s.lastStats[*disk.Device].ReadIOs)),
				WriteLatency: (float64(stats.WriteTicks) - float64(s.lastStats[*disk.Device].WriteTicks)) / (float64(stats.WriteIOs) - float64(s.lastStats[*disk.Device].WriteIOs)),
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
			s.lastStats[*disk.Device] = &stats
		} else {
			s.lastStats[*disk.Device] = &stats
		}

		// --- Smart data population ---
		smartStats, err := s.smartService.GetSmartInfo(*disk.Device)
		if err != nil {
			tlog.Error("Error getting SMART stats", "disk", *disk.Device, "err", err)
		} else {
			s.currentDiskHealth.PerDiskIO[len(s.currentDiskHealth.PerDiskIO)-1].SmartData = smartStats
		}

		// --- PerPartitionInfo population ---
		if disk.Partitions != nil {
			for _, part := range *disk.Partitions {
				if part.MountPointData != nil {
					for _, mp := range *part.MountPointData {
						// Fill PerPartitionInfo
						var fstype string
						if mp.FSType != nil {
							fstype = *mp.FSType
						}
						// Determine fsck support (simple heuristic)
						fsckSupported := false
						switch fstype {
						case "ext2", "ext3", "ext4", "xfs", "btrfs", "f2fs", "vfat", "exfat", "ntfs":
							fsckSupported = true
						}
						// Heuristic: fsck needed if not mounted, or if fstype supports fsck and not clean (not implemented: always false)
						fsckNeeded := false // TODO: implement real check for dirty/needs fsck
						// Use partition size if available
						var totalSpace, freeSpace uint64
						if part.Size != nil {
							totalSpace = uint64(*part.Size)
						}
						// Try to get free space from MountPointData (not present, so leave 0)
						// Optionally, could use lsblk or statfs here
						info := dto.PerPartitionInfo{
							MountPoint:    mp.Path,
							Device:        mp.Device,
							FSType:        fstype,
							FreeSpace:     freeSpace,
							TotalSpace:    totalSpace,
							FsckNeeded:    fsckNeeded,
							FsckSupported: fsckSupported,
						}
						if s.currentDiskHealth.PerPartitionInfo[*disk.Device] == nil {
							s.currentDiskHealth.PerPartitionInfo[*disk.Device] = make([]dto.PerPartitionInfo, 0)
						}
						s.currentDiskHealth.PerPartitionInfo[*disk.Device] = append(s.currentDiskHealth.PerPartitionInfo[*disk.Device], info)
					}
				}
			}
		}
	}
	s.lastUpdateTime = time.Now()
	return nil
}

// GetDiskStats collects and returns disk I/O statistics.
func (s *diskStatsService) GetDiskStats() (*dto.DiskHealth, error) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if s.currentDiskHealth == nil {
		return nil, errors.New("disk stats not initialized")
	}
	return s.currentDiskHealth, nil
}
