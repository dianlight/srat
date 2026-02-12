package service

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// NetworkStatsService is a service for collecting network I/O statistics.
type NetworkStatsService interface {
	GetNetworkStats() (*dto.NetworkStats, errors.E)
}

type networkStatsService struct {
	//prop_repo        repository.PropertyRepositoryInterface
	procfs           *procfs.FS
	sysfs            *sysfs.FS // sysfs is used to access system files for network interfaces.
	ctx              context.Context
	lastUpdateTime   time.Time                    // lastUpdateTime is used to track the last time network stats were updated.
	lastStats        map[string]procfs.NetDevLine // lastStats stores the last collected network I/O statistics.
	currentNetHealth *dto.NetworkStats
	updateMutex      *sync.Mutex
	settingService   SettingServiceInterface
}

// NewNetworkStatsService creates a new NetworkStatsService.
func NewNetworkStatsService(lc fx.Lifecycle,
	Ctx context.Context,
	settingService SettingServiceInterface,
	// prop_repo repository.PropertyRepositoryInterface,
) NetworkStatsService {
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		slog.ErrorContext(Ctx, "Failed to create procfs filesystem", "error", err)
	}

	sfs, err := sysfs.NewFS("/sys")
	if err != nil {
		slog.ErrorContext(Ctx, "Failed to create sysfs filesystem", "error", err)
	}

	ns := &networkStatsService{
		//prop_repo:      prop_repo,
		procfs:         &fs,
		sysfs:          &sfs,
		ctx:            Ctx,
		settingService: settingService,
		lastUpdateTime: time.Now(),
		updateMutex:    &sync.Mutex{},
		lastStats:      make(map[string]procfs.NetDevLine),
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := ns.updateNetworkStats()
			if err != nil {
				// Ignore NotFound error, log others
				if errors.Is(err, dto.ErrorNotFound) {
					// ignore
				} else {
					slog.ErrorContext(ctx, "Failed to update network stats", "error", err)
				}
			}
			Ctx.Value("wg").(*sync.WaitGroup).Go(func() {
				ns.run()
			})
			return nil
		},
	})
	return ns
}

func (self *networkStatsService) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Run process closed", "err", self.ctx.Err())
			return errors.WithStack(self.ctx.Err())
		case <-time.After(time.Second * 10):
			err := self.updateNetworkStats()
			if err != nil {
				slog.ErrorContext(self.ctx, "Failed to update network stats", "error", err)
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
	var nicSlice []string

	setting, err := s.settingService.Load()
	if setting == nil || err != nil {
		slog.WarnContext(s.ctx, "Errore getting setting", "error", err, "setting_nil", setting == nil)
		s.lastUpdateTime = time.Now()
		return nil
	}
	if setting.BindAllInterfaces {
		for nicName := range stats {
			if nicName == "lo" {
				continue
			}
			// Skip virtual ethernet (veth) interfaces used by containers
			if strings.HasPrefix(nicName, "veth") {
				continue
			}
			nicSlice = append(nicSlice, nicName)
		}
	} else {
		nicSlice = setting.Interfaces
	}

	s.currentNetHealth = &dto.NetworkStats{
		PerNicIO: make([]dto.NicIOStats, 0),
		Global: dto.GlobalNicStats{
			TotalInboundTraffic:  0,
			TotalOutboundTraffic: 0,
		},
	}

	for _, nicName := range nicSlice {

		// Skip virtual ethernet (veth) interfaces used by containers
		if strings.HasPrefix(nicName, "veth") {
			slog.DebugContext(s.ctx, "Skipping veth interface", "interface", nicName)
			continue
		}

		if netDev, ok := stats[nicName]; ok {
			if lastNetDev, ok := s.lastStats[nicName]; ok {

				nc, err := s.sysfs.NetClassByIface(nicName)
				if err != nil {
					// Interface may have been removed between /proc/net/dev read and sysfs access
					// This is common with virtual interfaces (veth). Skip silently.
					slog.DebugContext(s.ctx, "Network interface no longer accessible, skipping", "interface", nicName, "error", err)
					delete(s.lastStats, nicName) // Clean up old stats for removed interface
					continue
				}
				speed := int64(0)
				if nc != nil && nc.Speed != nil {
					speed = *nc.Speed
				}

				// Get IP and netmask
				var ip, netmask string
				if iface, err := net.InterfaceByName(nicName); err == nil {
					if addrs, err := iface.Addrs(); err == nil {
						for _, addr := range addrs {
							if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
								if ipnet.IP.To4() != nil {
									ip = ipnet.IP.String()
									netmask = net.IP(ipnet.Mask).String()
									break // Take the first IPv4 address
								}
							}
						}
					}
				}

				dstat := dto.NicIOStats{
					DeviceName:      nicName,
					DeviceMaxSpeed:  speed,
					InboundTraffic:  (float64(netDev.RxBytes) - float64(lastNetDev.RxBytes)) / time.Since(s.lastUpdateTime).Seconds(),
					OutboundTraffic: (float64(netDev.TxBytes) - float64(lastNetDev.TxBytes)) / time.Since(s.lastUpdateTime).Seconds(),
					IP:              ip,
					Netmask:         netmask,
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
func (s *networkStatsService) GetNetworkStats() (*dto.NetworkStats, errors.E) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if s.currentNetHealth == nil {
		return nil, errors.New("network stats not initialized")
	}
	return s.currentNetHealth, nil
}
