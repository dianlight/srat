package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/adelolmo/hd-idle/diskstats"
	"github.com/adelolmo/hd-idle/io"
	"github.com/adelolmo/hd-idle/sgio"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tlog"
)

const (
	defaultPoolMultiplier = 10
)

// HDIdleServiceInterface provides methods for managing hard disk idle monitoring
type HDIdleServiceInterface interface {
	// Start begins monitoring disk activity and spinning down idle disks
	Start() errors.E
	// Stop halts the monitoring process
	Stop() errors.E
	// IsRunning returns true if the service is currently monitoring
	IsRunning() bool
	// GetStatus returns current monitoring status and disk states
	GetStatus() (*HDIdleStatus, errors.E)
	GetDeviceStatus(path string) (*HDIdleDiskStatus, errors.E)
	GetDeviceConfig(path string) (*dto.HDIdleDeviceDTO, errors.E)
	SaveDeviceConfig(device dto.HDIdleDeviceDTO) errors.E
	// GetEffectiveConfig returns the effective enabled flag and the list of devices
	// included for monitoring (by GivenName/DevicePath) according to tri-state logic.
	GetEffectiveConfig() HDIdleEffectiveConfig
	// GetProcessStatus returns a ProcessStatus representation of the HDIdle monitoring service.
	// The parentPid is used to indicate this is a subprocess by storing it as a negative value.
	// Convention: Subprocesses have their parent process PID as a negative number.
	GetProcessStatus(parentPid int32) *dto.ProcessStatus
}

// HDIdleDeviceConfig represents per-device configuration
type HDIdleDeviceConfig struct {
	// Device name (e.g., "sda" or "/dev/disk/by-id/...")
	Name string
	// Idle time in seconds before spinning down (0 = use default)
	IdleTime int
	// Command type: "scsi" or "ata" (empty = use default)
	CommandType *dto.HdidleCommand
	// Power condition for SCSI devices (0-15)
	PowerCondition uint8
}

// HDIdleStatus represents the current status of the service
type HDIdleStatus struct {
	Running     bool
	MonitoredAt time.Time
	Disks       []HDIdleDiskStatus
}

// HDIdleDiskStatus represents the status of a monitored disk
type HDIdleDiskStatus struct {
	Name           string
	GivenName      string
	SpunDown       bool
	LastIOAt       time.Time
	SpinDownAt     time.Time
	SpinUpAt       time.Time
	IdleTime       time.Duration
	CommandType    dto.HdidleCommand
	PowerCondition uint8
}

// hDIdleService implements HDIdleServiceInterface
type hDIdleService struct {
	apiContext       context.Context
	apiContextCancel context.CancelFunc
	state            *dto.ContextState
	hdidlerepo       repository.HDIdleDeviceRepositoryInterface
	settingService   SettingServiceInterface

	// Internal state
	mu        sync.RWMutex
	running   bool
	stopChan  chan struct{}
	config    *internalConfig
	diskStats []diskState
	lastNow   time.Time
	converter converter.DtoToDbomConverterImpl
}

// HDIdleEffectiveConfig provides a snapshot of the effective configuration
// used by the HDIdle service after applying global and per-device settings.
type HDIdleEffectiveConfig struct {
	Enabled bool
	Devices []string // list of GivenName (original DevicePath) included for monitoring
}

type internalConfig struct {
	Enabled               bool
	Devices               []deviceConfig
	DefaultIdle           time.Duration
	DefaultCommandType    dto.HdidleCommand
	DefaultPowerCondition uint8
	IgnoreSpinDown        bool
	SkewTime              time.Duration
	NameMap               map[string]string
}

type deviceConfig struct {
	Name           string
	GivenName      string
	Idle           time.Duration
	CommandType    dto.HdidleCommand
	PowerCondition uint8
}

type diskState struct {
	Name           string
	GivenName      string
	IdleTime       time.Duration
	CommandType    dto.HdidleCommand
	PowerCondition uint8
	Reads          uint64
	Writes         uint64
	SpinDownAt     time.Time
	SpinUpAt       time.Time
	LastIOAt       time.Time
	LastSpunDownAt time.Time
	SpunDown       bool
}

// HDIdleServiceParams defines dependencies for HDIdleService
type HDIdleServiceParams struct {
	fx.In
	ApiContext       context.Context
	ApiContextCancel context.CancelFunc
	State            *dto.ContextState
	Hdidlerepo       repository.HDIdleDeviceRepositoryInterface
	SettingService   SettingServiceInterface
}

// NewHDIdleService creates a new HDIdleService instance
func NewHDIdleService(lc fx.Lifecycle, in HDIdleServiceParams) HDIdleServiceInterface {

	hdidle_service := &hDIdleService{
		apiContext:       in.ApiContext,
		apiContextCancel: in.ApiContextCancel,
		state:            in.State,
		running:          false,
		hdidlerepo:       in.Hdidlerepo,
		settingService:   in.SettingService,
		converter:        converter.DtoToDbomConverterImpl{},
	}

	return hdidle_service
}

// Start begins monitoring disk activity
func (s *hDIdleService) Start() errors.E {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("HDIdle service is already running")
	}

	// Convert to internal config
	intConfig, err := s.convertConfig()
	if err != nil {
		return err
	}
	s.config = intConfig
	s.diskStats = []diskState{}
	s.stopChan = make(chan struct{})
	s.running = true
	s.lastNow = time.Now()

	// Start monitoring in background
	if s.config.Enabled {
		go s.monitorLoop()
	}

	return nil
}

// Stop halts the monitoring process
func (s *hDIdleService) Stop() errors.E {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return errors.New("HDIdle service is not running")
	}

	tlog.Info("Stopping HDIdle service")
	close(s.stopChan)
	s.running = false
	s.diskStats = []diskState{}
	s.config = nil

	return nil
}

// IsRunning returns true if the service is currently monitoring
func (s *hDIdleService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatus returns current monitoring status and disk states
func (s *hDIdleService) GetStatus() (*HDIdleStatus, errors.E) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return &HDIdleStatus{Running: false}, nil
	}

	status := &HDIdleStatus{
		Running:     true,
		MonitoredAt: time.Now(),
		Disks:       make([]HDIdleDiskStatus, len(s.diskStats)),
	}

	for i, ds := range s.diskStats {
		status.Disks[i] = HDIdleDiskStatus{
			Name:           ds.Name,
			GivenName:      ds.GivenName,
			SpunDown:       ds.SpunDown,
			LastIOAt:       ds.LastIOAt,
			SpinDownAt:     ds.SpinDownAt,
			SpinUpAt:       ds.SpinUpAt,
			IdleTime:       ds.IdleTime,
			CommandType:    ds.CommandType,
			PowerCondition: ds.PowerCondition,
		}
	}

	return status, nil
}

func (s *hDIdleService) GetDeviceStatus(path string) (*HDIdleDiskStatus, errors.E) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return &HDIdleDiskStatus{}, nil
	}

	// Find disk state by given name or resolved name
	for _, ds := range s.diskStats {
		if ds.GivenName == path || ds.Name == path {
			return &HDIdleDiskStatus{
				Name:           ds.Name,
				GivenName:      ds.GivenName,
				SpunDown:       ds.SpunDown,
				LastIOAt:       ds.LastIOAt,
				SpinDownAt:     ds.SpinDownAt,
				SpinUpAt:       ds.SpinUpAt,
				IdleTime:       ds.IdleTime,
				CommandType:    ds.CommandType,
				PowerCondition: ds.PowerCondition,
			}, nil
		}
	}

	return &HDIdleDiskStatus{}, errors.Errorf("disk %s not found", path)
}

func (s *hDIdleService) GetDeviceConfig(path string) (*dto.HDIdleDeviceDTO, errors.E) {
	device, err := s.hdidlerepo.LoadByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.HDIdleDeviceDTO{DevicePath: path}, nil
		}
		return nil, err
	}

	if device == nil {
		return nil, errors.Errorf("device %s not found", path)
	}

	dtoDevice, errN := s.converter.HDIdleDeviceToHDIdleDeviceDTO(*device)
	if errN != nil {
		return nil, errors.WithStack(errN)
	}
	return &dtoDevice, nil
}
func (s *hDIdleService) SaveDeviceConfig(device dto.HDIdleDeviceDTO) errors.E {
	dbDevice, err := s.converter.HDIdleDeviceDTOToHDIdleDevice(device)
	if err != nil {
		return errors.WithStack(err)
	}

	errE := s.hdidlerepo.Update(&dbDevice)
	if errE != nil {
		return errE
	}
	return nil
}

// GetEffectiveConfig returns the effective enabled flag and the list of devices included
// for monitoring. If the service has not been started yet, it computes the configuration
// from current settings and repository to provide a meaningful snapshot.
func (s *hDIdleService) GetEffectiveConfig() HDIdleEffectiveConfig {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if cfg == nil {
		// Compute a temporary config snapshot without mutating internal state
		if snapshot, err := s.convertConfig(); err == nil {
			cfg = snapshot
		}
	}

	ec := HDIdleEffectiveConfig{Enabled: false, Devices: []string{}}
	if cfg == nil {
		return ec
	}
	ec.Enabled = cfg.Enabled
	if len(cfg.Devices) > 0 {
		ec.Devices = make([]string, 0, len(cfg.Devices))
		for _, d := range cfg.Devices {
			// Return GivenName which is the configured DevicePath
			if d.GivenName != "" {
				ec.Devices = append(ec.Devices, d.GivenName)
			} else if d.Name != "" {
				ec.Devices = append(ec.Devices, d.Name)
			}
		}
	}
	return ec
}

// GetProcessStatus returns a ProcessStatus representation of the HDIdle monitoring service.
// This allows the HDIdle monitor to appear as a subprocess in the process metrics.
//
// Parameters:
//   - parentPid: The PID of the parent SRAT process. This is stored as a negative value
//     to indicate that this is a virtual subprocess, not a real OS process.
//
// Convention: Subprocesses use negative PIDs where the absolute value represents the
// parent process PID. For example, if SRAT has PID 1234, the HDIdle subprocess will
// have PID -1234. This convention allows the UI to distinguish between real processes
// and virtual subprocesses/monitoring threads.
//
// Returns:
//   - A ProcessStatus where:
//   - Pid: Negative parent PID (e.g., -1234 if parent is 1234)
//   - Name: "powersave-monitor"
//   - Connections: Number of monitored disks
//   - IsRunning: Whether the monitoring loop is active
//   - Status: ["idle"] when not running, ["running"] when active
func (s *hDIdleService) GetProcessStatus(parentPid int32) *dto.ProcessStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &dto.ProcessStatus{
		Pid:           -parentPid, // Negative PID indicates this is a subprocess
		Name:          "powersave-monitor",
		CreateTime:    time.Time{},
		CPUPercent:    0.0,
		MemoryPercent: 0.0,
		OpenFiles:     0,
		Connections:   0,
		Status:        []string{"idle"},
		IsRunning:     s.running,
	}

	// If running, populate with monitored disk count
	if s.running && s.config != nil {
		status.Connections = len(s.diskStats)
		status.Status = []string{"running"}
	}

	return status
}

// convertConfig converts external config to internal config
func (s *hDIdleService) convertConfig() (*internalConfig, errors.E) {

	settings, err := s.settingService.Load()
	if err != nil {
		return nil, err
	}

	// Global enabled flag from settings (defaults to false when nil)
	globalEnabled := false
	if settings.HDIdleEnabled != nil {
		globalEnabled = *settings.HDIdleEnabled
	}

	intConfig := &internalConfig{
		// Effective enabled: global switch OR at least one per-device forced enabled
		Enabled:               globalEnabled, // may be updated below after loading devices
		Devices:               []deviceConfig{},
		DefaultIdle:           time.Duration((*settings).HDIdleDefaultIdleTime) * time.Second,
		DefaultCommandType:    (*settings).HDIdleDefaultCommandType,
		DefaultPowerCondition: (*settings).HDIdleDefaultPowerCondition,
		IgnoreSpinDown:        (*settings).HDIdleIgnoreSpinDownDetection,
		NameMap:               make(map[string]string),
	}

	devices, err := s.hdidlerepo.LoadAll()
	if err != nil {
		tlog.Error("Failed to load HDIdle devices from repository", "error", err)
		return nil, errors.Wrap(err, "failed to load HDIdle devices")
	}

	// Convert devices considering per-device Enabled tri-state
	for _, dev := range devices {
		deviceRealPath, err := io.RealPath(dev.DevicePath)
		if err != nil {
			deviceRealPath = ""
		}

		idle := intConfig.DefaultIdle
		cmdType := &intConfig.DefaultCommandType
		includeDevice := false

		switch dev.Enabled {
		case dto.HdidleEnableds.NOENABLED:
			includeDevice = false
		case dto.HdidleEnableds.YESENABLED:
			includeDevice = true
		case dto.HdidleEnableds.CUSTOMENABLED:
			includeDevice = true
			if dev.IdleTime != 0 {
				idle = time.Duration(dev.IdleTime) * time.Second
			}
			if dev.CommandType != nil {
				cmdType = dev.CommandType
			}
		}

		if includeDevice {
			devConfig := deviceConfig{
				Name:           deviceRealPath,
				GivenName:      dev.DevicePath,
				Idle:           idle,
				CommandType:    *cmdType,
				PowerCondition: dev.PowerCondition,
			}

			intConfig.Devices = append(intConfig.Devices, devConfig)
			if deviceRealPath != "" {
				intConfig.NameMap[deviceRealPath] = dev.DevicePath
			}
		}
	}

	// Calculate skew time and pool interval
	interval := s.calculatePoolInterval(intConfig)
	intConfig.SkewTime = interval * 3

	return intConfig, nil
}

// calculatePoolInterval determines the polling interval
func (s *hDIdleService) calculatePoolInterval(config *internalConfig) time.Duration {
	defaultIdleTime := config.DefaultIdle
	if len(config.Devices) == 0 {
		return defaultIdleTime / defaultPoolMultiplier
	}

	interval := defaultIdleTime
	for _, dev := range config.Devices {
		if dev.Idle == 0 {
			continue
		}
		if dev.Idle < interval {
			interval = dev.Idle
		}
	}

	sleepTime := interval / defaultPoolMultiplier
	if sleepTime == 0 {
		return time.Second
	}
	return sleepTime
}

// monitorLoop is the main monitoring loop
func (s *hDIdleService) monitorLoop() {
	s.mu.RLock()
	if s.config == nil {
		s.mu.RUnlock()
		return
	}
	interval := s.calculatePoolInterval(s.config)
	s.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-s.apiContext.Done():
			return
		case <-ticker.C:
			s.observeDiskActivity()
		}
	}
}

// observeDiskActivity observes disk activity and spins down idle disks
func (s *hDIdleService) observeDiskActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	snapshot := diskstats.Snapshot()
	now := time.Now()

	// Resolve symlinks if needed
	s.resolveSymlinks()

	// Process each disk in the snapshot
	for _, stats := range snapshot {
		s.updateDiskState(stats.Name, stats.Reads, stats.Writes, now)
	}

	s.lastNow = now
}

// resolveSymlinks resolves device symlinks based on policy
func (s *hDIdleService) resolveSymlinks() {

	for i := range s.config.Devices {
		device := &s.config.Devices[i]
		if len(device.Name) == 0 {
			realPath, err := io.RealPath(device.GivenName)
			if err == nil {
				device.Name = realPath
			}
		}
	}
}

// updateDiskState updates the state of a disk based on current activity
func (s *hDIdleService) updateDiskState(name string, reads, writes uint64, now time.Time) {
	// Find existing disk state
	dsi := s.findDiskStateIndex(name)
	if dsi < 0 {
		// New disk, initialize it
		s.diskStats = append(s.diskStats, s.initDiskState(name, reads, writes, now))
		return
	}

	ds := &s.diskStats[dsi]

	// Check for suspend/sleep events (long interval)
	intervalDuration := now.Unix() - s.lastNow.Unix()
	if intervalDuration > s.config.SkewTime.Milliseconds()/1000 {
		// Reset after sleep
		ds.SpinUpAt = now
		ds.LastIOAt = now
		ds.SpunDown = false
	}

	// Check if disk had activity
	if ds.Writes == writes && ds.Reads == reads {
		// No activity
		if !ds.SpunDown || s.config.IgnoreSpinDown {
			idleDuration := now.Sub(ds.LastIOAt)
			timeSinceLastSpunDown := now.Sub(ds.LastSpunDownAt)

			if ds.IdleTime != 0 && idleDuration > ds.IdleTime && timeSinceLastSpunDown > ds.IdleTime {
				// Time to spin down
				givenName := s.resolveDeviceGivenName(ds.Name)
				if ds.SpunDown && s.config.IgnoreSpinDown {
					tlog.Info("Spindown (ignoring prior spin down state)", "disk", givenName)
				} else {
					tlog.Info("Spindown", "disk", givenName)
				}

				device := fmt.Sprintf("/dev/%s", ds.Name)
				if err := s.spindownDisk(device, ds.CommandType, ds.PowerCondition); err != nil {
					tlog.Error("Failed to spindown disk", "disk", givenName, "error", err)
				}

				ds.LastSpunDownAt = now
				ds.SpinDownAt = now
				ds.SpunDown = true
			}
		}
	} else {
		// Disk had activity
		if ds.SpunDown {
			// Disk just spun up
			givenName := s.resolveDeviceGivenName(ds.Name)
			tlog.Info("Spinup", "disk", givenName)
			ds.SpinUpAt = now
		}

		ds.Reads = reads
		ds.Writes = writes
		ds.LastIOAt = now
		ds.SpunDown = false
	}
}

// findDiskStateIndex finds the index of a disk in the state array
func (s *hDIdleService) findDiskStateIndex(diskName string) int {
	for i, stats := range s.diskStats {
		if stats.Name == diskName {
			return i
		}
	}
	return -1
}

// initDiskState initializes a new disk state
func (s *hDIdleService) initDiskState(name string, reads, writes uint64, now time.Time) diskState {
	idle := s.config.DefaultIdle
	command := s.config.DefaultCommandType
	powerCondition := s.config.DefaultPowerCondition

	// Check if there's a specific config for this device
	for _, dev := range s.config.Devices {
		if dev.Name == name {
			idle = dev.Idle
			command = dev.CommandType
			powerCondition = dev.PowerCondition
			break
		}
	}

	return diskState{
		Name:           name,
		GivenName:      s.resolveDeviceGivenName(name),
		LastIOAt:       now,
		SpinUpAt:       now,
		SpunDown:       false,
		Writes:         writes,
		Reads:          reads,
		IdleTime:       idle,
		CommandType:    command,
		PowerCondition: powerCondition,
	}
}

// resolveDeviceGivenName resolves the given name for a device
func (s *hDIdleService) resolveDeviceGivenName(name string) string {
	if givenName, ok := s.config.NameMap[name]; ok {
		return givenName
	}
	return name
}

// spindownDisk spins down a disk using the appropriate command
func (s *hDIdleService) spindownDisk(device string, command dto.HdidleCommand, powerCondition uint8) errors.E {
	switch command {
	case dto.HdidleCommands.SCSICOMMAND:
		if err := sgio.StartStopScsiDevice(device, powerCondition); err != nil {
			return errors.Errorf("cannot spindown scsi disk %s: %w", device, err)
		}
		return nil
	case dto.HdidleCommands.ATACOMMAND:
		if err := sgio.StopAtaDevice(device, false); err != nil {
			return errors.Errorf("cannot spindown ata disk %s: %w", device, err)
		}
		return nil
	default:
		return errors.Errorf("unknown command type: %s", command)
	}
}
