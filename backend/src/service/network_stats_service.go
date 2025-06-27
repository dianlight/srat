package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tlog"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
	"gitlab.com/tozd/go/errors"
)

// NetworkStatsService is a service for collecting network I/O statistics.
type NetworkStatsService interface {
	GetNetworkStats() (*dto.NetworkStats, error)
}

type networkStatsService struct {
	prop_repo        repository.PropertyRepositoryInterface
	procfs           *procfs.FS
	sysfs            *sysfs.FS // sysfs is used to access system files for network interfaces.
	ctx              context.Context
	lastUpdateTime   time.Time                    // lastUpdateTime is used to track the last time network stats were updated.
	lastStats        map[string]procfs.NetDevLine // lastStats stores the last collected network I/O statistics.
	currentNetHealth *dto.NetworkStats
	updateMutex      *sync.Mutex
}

// NewNetworkStatsService creates a new NetworkStatsService.
func NewNetworkStatsService(Ctx context.Context, prop_repo repository.PropertyRepositoryInterface) NetworkStatsService {
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		tlog.Error("Failed to create procfs filesystem", "error", err)
	}

	sfs, err := sysfs.NewFS("/sys")
	if err != nil {
		tlog.Error("Failed to create sysfs filesystem", "error", err)
	}

	ns := &networkStatsService{
		prop_repo:      prop_repo,
		procfs:         &fs,
		sysfs:          &sfs,
		ctx:            Ctx,
		lastUpdateTime: time.Now(),
		updateMutex:    &sync.Mutex{},
		lastStats:      make(map[string]procfs.NetDevLine),
	}
	Ctx.Value("wg").(*sync.WaitGroup).Add(1)
	go func() {
		defer Ctx.Value("wg").(*sync.WaitGroup).Done()
		ns.run()
	}()
	return ns
}

func (self *networkStatsService) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.Info("Run process closed", "err", self.ctx.Err())
			return errors.WithStack(self.ctx.Err())
		case <-time.After(time.Second * 10):
			err := self.updateNetworkStats()
			if err != nil {
				tlog.Error("Failed to update network stats", "error", err)
				continue
			}
		}
	}
}

func (s *networkStatsService) updateNetworkStats() error {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	stats, err := s.procfs.NetDev()
	if err != nil {
		return err
	}

	nics, err := s.prop_repo.Value("Interfaces", false)
	if err != nil {
		return err
	}

	s.currentNetHealth = &dto.NetworkStats{
		PerNicIO: make([]dto.NicIOStats, 0),
		Global: dto.GlobalNicStats{
			TotalInboundTraffic:  0,
			TotalOutboundTraffic: 0,
		},
	}

	for _, nic := range nics.([]interface{}) {
		if netDev, ok := stats[nic.(string)]; ok {
			if lastNetDev, ok := s.lastStats[nic.(string)]; ok {

				nc, err := s.sysfs.NetClassByIface(nic.(string))
				if err != nil {
					slog.Error("Failed to get sysfs for network interface", "interface", nic, "error", err)
				}
				speed := int64(0)
				if nc != nil && nc.Speed != nil {
					speed = *nc.Speed
				}

				dstat := dto.NicIOStats{
					DeviceName:      nic.(string),
					DeviceMaxSpeed:  speed,
					InboundTraffic:  (float64(netDev.RxBytes) - float64(lastNetDev.RxBytes)) / time.Since(s.lastUpdateTime).Seconds(),
					OutboundTraffic: (float64(netDev.TxBytes) - float64(lastNetDev.TxBytes)) / time.Since(s.lastUpdateTime).Seconds(),
				}
				s.currentNetHealth.PerNicIO = append(s.currentNetHealth.PerNicIO, dstat)
				s.currentNetHealth.Global.TotalInboundTraffic += dstat.InboundTraffic
				s.currentNetHealth.Global.TotalOutboundTraffic += dstat.OutboundTraffic
			}
			s.lastStats[nic.(string)] = netDev
		}
	}
	s.lastUpdateTime = time.Now()
	return nil
}

// GetNetworkStats collects and returns network I/O statistics.
func (s *networkStatsService) GetNetworkStats() (*dto.NetworkStats, error) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if s.currentNetHealth == nil {
		return nil, errors.New("network stats not initialized")
	}
	return s.currentNetHealth, nil
}
