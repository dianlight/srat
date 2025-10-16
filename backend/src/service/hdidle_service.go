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

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/tlog"
)

const (
	scsiCommand        = "scsi"
	ataCommand         = "ata"
	defaultIdleTime    = 600 * time.Second
	symlinkResolveOnce = 0
	//symlinkResolveRetry   = 1
	defaultPoolMultiplier = 10
)

// HDIdleServiceInterface provides methods for managing hard disk idle monitoring
type HDIdleServiceInterface interface {
	// Start begins monitoring disk activity and spinning down idle disks
	Start(config *HDIdleConfig) error
	// Stop halts the monitoring process
	Stop() error
	// IsRunning returns true if the service is currently monitoring
	IsRunning() bool
	// GetStatus returns current monitoring status and disk states
	GetStatus() (*HDIdleStatus, error)
}

// HDIdleConfig represents configuration for hd-idle monitoring
type HDIdleConfig struct {
	// Devices to monitor with specific configurations
	Devices []HDIdleDeviceConfig
	// Default idle time in seconds before spinning down (default: 600)
	DefaultIdleTime int
	// Default command type: "scsi" or "ata" (default: "scsi")
	DefaultCommandType string
	// Default power condition (0-15) for SCSI devices (default: 0)
	DefaultPowerCondition uint8
	// Enable debug logging
	Debug bool
	// Log file path (empty = no file logging)
	LogFile string
	// Symlink resolution policy: 0 = resolve once, 1 = retry resolution
	SymlinkPolicy int
	// Ignore spin down detection and force spin down
	IgnoreSpinDownDetection bool
}

// HDIdleDeviceConfig represents per-device configuration
type HDIdleDeviceConfig struct {
	// Device name (e.g., "sda" or "/dev/disk/by-id/...")
	Name string
	// Idle time in seconds before spinning down (0 = use default)
	IdleTime int
	// Command type: "scsi" or "ata" (empty = use default)
	CommandType string
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
	CommandType    string
	PowerCondition uint8
}

// HDIdleService implements HDIdleServiceInterface
type HDIdleService struct {
	apiContext       context.Context
	apiContextCancel context.CancelFunc
	state            *dto.ContextState

	// Internal state
	mu        sync.RWMutex
	running   bool
	stopChan  chan struct{}
	config    *internalConfig
	diskStats []diskState
	lastNow   time.Time
}

type internalConfig struct {
	Devices               []deviceConfig
	DefaultIdle           time.Duration
	DefaultCommandType    string
	DefaultPowerCondition uint8
	Debug                 bool
	LogFile               string
	SymlinkPolicy         int
	IgnoreSpinDown        bool
	SkewTime              time.Duration
	NameMap               map[string]string
}

type deviceConfig struct {
	Name           string
	GivenName      string
	Idle           time.Duration
	CommandType    string
	PowerCondition uint8
}

type diskState struct {
	Name           string
	GivenName      string
	IdleTime       time.Duration
	CommandType    string
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
}

// NewHDIdleService creates a new HDIdleService instance
func NewHDIdleService(in HDIdleServiceParams) HDIdleServiceInterface {
	return &HDIdleService{
		apiContext:       in.ApiContext,
		apiContextCancel: in.ApiContextCancel,
		state:            in.State,
		running:          false,
	}
}

// Start begins monitoring disk activity
func (s *HDIdleService) Start(config *HDIdleConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("HDIdle service is already running")
	}

	// Validate configuration
	if err := s.validateConfig(config); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	// Convert to internal config
	intConfig := s.convertConfig(config)
	s.config = intConfig
	s.diskStats = []diskState{}
	s.stopChan = make(chan struct{})
	s.running = true
	s.lastNow = time.Now()

	tlog.Info("Starting HDIdle service", "config", s.formatConfig())

	// Start monitoring in background
	go s.monitorLoop()

	return nil
}

// Stop halts the monitoring process
func (s *HDIdleService) Stop() error {
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
func (s *HDIdleService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatus returns current monitoring status and disk states
func (s *HDIdleService) GetStatus() (*HDIdleStatus, error) {
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

// validateConfig validates the configuration
func (s *HDIdleService) validateConfig(config *HDIdleConfig) error {
	if config == nil {
		return errors.New("config is nil")
	}

	if config.DefaultCommandType != "" && config.DefaultCommandType != scsiCommand && config.DefaultCommandType != ataCommand {
		return errors.Errorf("invalid default command type: %s (must be 'scsi' or 'ata')", config.DefaultCommandType)
	}

	for _, dev := range config.Devices {
		if dev.Name == "" {
			return errors.New("device name cannot be empty")
		}
		if dev.CommandType != "" && dev.CommandType != scsiCommand && dev.CommandType != ataCommand {
			return errors.Errorf("invalid command type for device %s: %s", dev.Name, dev.CommandType)
		}
	}

	return nil
}

// convertConfig converts external config to internal config
func (s *HDIdleService) convertConfig(config *HDIdleConfig) *internalConfig {
	intConfig := &internalConfig{
		Devices:               []deviceConfig{},
		DefaultIdle:           time.Duration(config.DefaultIdleTime) * time.Second,
		DefaultCommandType:    config.DefaultCommandType,
		DefaultPowerCondition: config.DefaultPowerCondition,
		Debug:                 config.Debug,
		LogFile:               config.LogFile,
		SymlinkPolicy:         config.SymlinkPolicy,
		IgnoreSpinDown:        config.IgnoreSpinDownDetection,
		NameMap:               make(map[string]string),
	}

	// Set defaults if not specified
	if intConfig.DefaultIdle == 0 {
		intConfig.DefaultIdle = defaultIdleTime
	}
	if intConfig.DefaultCommandType == "" {
		intConfig.DefaultCommandType = scsiCommand
	}

	// Convert devices
	for _, dev := range config.Devices {
		deviceRealPath, err := io.RealPath(dev.Name)
		if err != nil {
			deviceRealPath = ""
			if config.Debug {
				tlog.Warn("Unable to resolve symlink", "device", dev.Name, "error", err)
			}
		}

		idle := time.Duration(dev.IdleTime) * time.Second
		if idle == 0 {
			idle = intConfig.DefaultIdle
		}

		cmdType := dev.CommandType
		if cmdType == "" {
			cmdType = intConfig.DefaultCommandType
		}

		devConfig := deviceConfig{
			Name:           deviceRealPath,
			GivenName:      dev.Name,
			Idle:           idle,
			CommandType:    cmdType,
			PowerCondition: dev.PowerCondition,
		}

		intConfig.Devices = append(intConfig.Devices, devConfig)
		if deviceRealPath != "" {
			intConfig.NameMap[deviceRealPath] = dev.Name
		}
	}

	// Calculate skew time and pool interval
	interval := s.calculatePoolInterval(intConfig)
	intConfig.SkewTime = interval * 3

	return intConfig
}

// calculatePoolInterval determines the polling interval
func (s *HDIdleService) calculatePoolInterval(config *internalConfig) time.Duration {
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
func (s *HDIdleService) monitorLoop() {
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
func (s *HDIdleService) observeDiskActivity() {
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
func (s *HDIdleService) resolveSymlinks() {
	if s.config.SymlinkPolicy == symlinkResolveOnce {
		return
	}

	for i := range s.config.Devices {
		device := &s.config.Devices[i]
		if len(device.Name) == 0 {
			realPath, err := io.RealPath(device.GivenName)
			if err == nil {
				device.Name = realPath
				s.logToFile(fmt.Sprintf("symlink %s resolved to %s", device.GivenName, realPath))
			} else if s.config.Debug {
				tlog.Warn("Cannot resolve symlink", "device", device.GivenName, "error", err)
			}
		}
	}
}

// updateDiskState updates the state of a disk based on current activity
func (s *HDIdleService) updateDiskState(name string, reads, writes uint64, now time.Time) {
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
		s.logSpinupAfterSleep(ds.Name)
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
			s.logSpinup(ds, now, givenName)
			ds.SpinUpAt = now
		}

		ds.Reads = reads
		ds.Writes = writes
		ds.LastIOAt = now
		ds.SpunDown = false
	}

	// Debug logging
	if s.config.Debug {
		idleDuration := now.Sub(ds.LastIOAt)
		tlog.Debug("Disk state",
			"disk", ds.Name,
			"command", ds.CommandType,
			"spunDown", ds.SpunDown,
			"reads", ds.Reads,
			"writes", ds.Writes,
			"idleTime", ds.IdleTime.Seconds(),
			"idleDuration", idleDuration.Seconds(),
		)
	}
}

// findDiskStateIndex finds the index of a disk in the state array
func (s *HDIdleService) findDiskStateIndex(diskName string) int {
	for i, stats := range s.diskStats {
		if stats.Name == diskName {
			return i
		}
	}
	return -1
}

// initDiskState initializes a new disk state
func (s *HDIdleService) initDiskState(name string, reads, writes uint64, now time.Time) diskState {
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
func (s *HDIdleService) resolveDeviceGivenName(name string) string {
	if givenName, ok := s.config.NameMap[name]; ok {
		return givenName
	}
	return name
}

// spindownDisk spins down a disk using the appropriate command
func (s *HDIdleService) spindownDisk(device, command string, powerCondition uint8) error {
	switch command {
	case scsiCommand:
		if err := sgio.StartStopScsiDevice(device, powerCondition); err != nil {
			return errors.Errorf("cannot spindown scsi disk %s: %w", device, err)
		}
		return nil
	case ataCommand:
		if err := sgio.StopAtaDevice(device, s.config.Debug); err != nil {
			return errors.Errorf("cannot spindown ata disk %s: %w", device, err)
		}
		return nil
	default:
		return errors.Errorf("unknown command type: %s", command)
	}
}

// logSpinup logs a disk spinup event
func (s *HDIdleService) logSpinup(ds *diskState, now time.Time, givenName string) {
	text := fmt.Sprintf("date: %s, time: %s, disk: %s, running: %d, stopped: %d",
		now.Format("2006-01-02"), now.Format("15:04:05"), givenName,
		int(ds.SpinDownAt.Sub(ds.SpinUpAt).Seconds()), int(now.Sub(ds.SpinDownAt).Seconds()))
	s.logToFile(text)
}

// logSpinupAfterSleep logs a disk spinup after system sleep
func (s *HDIdleService) logSpinupAfterSleep(name string) {
	now := time.Now()
	text := fmt.Sprintf("date: %s, time: %s, disk: %s, assuming disk spun up after long sleep",
		now.Format("2006-01-02"), now.Format("15:04:05"), name)
	s.logToFile(text)
}

// logToFile writes a message to the log file
func (s *HDIdleService) logToFile(text string) {
	// Implementation intentionally minimal - use tlog for logging
	// This is kept for compatibility with hd-idle behavior
	if s.config.LogFile != "" {
		tlog.Debug("HDIdle log", "message", text)
	}
}

// formatConfig formats the config for logging
func (s *HDIdleService) formatConfig() string {
	if s.config == nil {
		return "nil"
	}
	return fmt.Sprintf("symlinkPolicy=%d, defaultIdle=%vs, defaultCommand=%s, defaultPowerCondition=%v, debug=%t, devices=%d, ignoreSpinDown=%t",
		s.config.SymlinkPolicy, s.config.DefaultIdle.Seconds(), s.config.DefaultCommandType,
		s.config.DefaultPowerCondition, s.config.Debug, len(s.config.Devices), s.config.IgnoreSpinDown)
}
