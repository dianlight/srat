package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
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
func NewNetworkStatsService(lc fx.Lifecycle, Ctx context.Context, prop_repo repository.PropertyRepositoryInterface) NetworkStatsService {
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		slog.Error("Failed to create procfs filesystem", "error", err)
	}

	sfs, err := sysfs.NewFS("/sys")
	if err != nil {
		slog.Error("Failed to create sysfs filesystem", "error", err)
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
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			Ctx.Value("wg").(*sync.WaitGroup).Add(1)
			err := ns.updateNetworkStats()
			if err != nil {
				slog.Error("Failed to update network stats", "error", err)
			}
			go func() {
				defer Ctx.Value("wg").(*sync.WaitGroup).Done()
				ns.run()
			}()
			return nil
		},
	})
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
				slog.Error("Failed to update network stats", "error", err)
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
	var nicSlice []interface{}

	BindAllInterfaces, err := s.prop_repo.Value("BindAllInterfaces", false)
	if err != nil {
		return err
	}
	bindAll, ok := BindAllInterfaces.(bool)
	if !ok {
		slog.Warn("BindAllInterfaces property from DB is not of expected type bool", "type", fmt.Sprintf("%T", BindAllInterfaces))
		s.lastUpdateTime = time.Now()
		return nil
	}
	if bindAll {
		for nicName := range stats {
			if nicName == "lo" {
				continue
			}
			nicSlice = append(nicSlice, nicName)
		}
	} else {
		nics, err := s.prop_repo.Value("Interfaces", false)
		if err != nil {
			return err
		}
		var ok bool
		nicSlice, ok = nics.([]interface{})
		if !ok {
			if nics != nil {
				slog.Warn("Interfaces property from DB is not of expected type []interface{}", "type", fmt.Sprintf("%T", nics))
			} else {
				slog.Debug("Interfaces property from DB is nil")
			}
			s.lastUpdateTime = time.Now()
			return nil
		}
	}

	s.currentNetHealth = &dto.NetworkStats{
		PerNicIO: make([]dto.NicIOStats, 0),
		Global: dto.GlobalNicStats{
			TotalInboundTraffic:  0,
			TotalOutboundTraffic: 0,
		},
	}

	for _, nic := range nicSlice {
		nicName, ok := nic.(string)
		if !ok {
			slog.Warn("Skipping non-string value in interfaces list", "value", nic)
			continue
		}

		if netDev, ok := stats[nicName]; ok {
			if lastNetDev, ok := s.lastStats[nicName]; ok {

				nc, err := s.sysfs.NetClassByIface(nicName)
				if err != nil {
					slog.Error("Failed to get sysfs for network interface", "interface", nicName, "error", err)
				}
				speed := int64(0)
				if nc != nil && nc.Speed != nil {
					speed = *nc.Speed
				}

				dstat := dto.NicIOStats{
					DeviceName:      nicName,
					DeviceMaxSpeed:  speed,
					InboundTraffic:  (float64(netDev.RxBytes) - float64(lastNetDev.RxBytes)) / time.Since(s.lastUpdateTime).Seconds(),
					OutboundTraffic: (float64(netDev.TxBytes) - float64(lastNetDev.TxBytes)) / time.Since(s.lastUpdateTime).Seconds(),
				}
				s.currentNetHealth.PerNicIO = append(s.currentNetHealth.PerNicIO, dstat)
				s.currentNetHealth.Global.TotalInboundTraffic += dstat.InboundTraffic
				s.currentNetHealth.Global.TotalOutboundTraffic += dstat.OutboundTraffic
			}
			s.lastStats[nicName] = netDev
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
