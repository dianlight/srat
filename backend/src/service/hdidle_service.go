package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/adelolmo/hd-idle/diskstats"
	"github.com/adelolmo/hd-idle/io"
	"github.com/adelolmo/hd-idle/sgio"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/tlog"

	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dbom/g/query"
)

const (
	defaultPoolMultiplier = 10
	// sgGetVersionNum is the ioctl command to get SG driver version
	// This is the value of SG_GET_VERSION_NUM from the Linux SCSI Generic driver
	sgGetVersionNum = 0x2282
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
	//GetStatus() (*HDIdleStatus, errors.E)
	GetDeviceStatus(path string) (*dto.HDIdleDeviceStatus, errors.E)
	GetDeviceConfig(path string) (*dto.HDIdleDevice, errors.E)
	SaveDeviceConfig(device dto.HDIdleDevice) errors.E
	// GetEffectiveConfig returns the effective enabled flag and the list of devices
	// included for monitoring (by GivenName/DevicePath) according to tri-state logic.
	//GetEffectiveConfig() HDIdleEffectiveConfig
	// GetProcessStatus returns a ProcessStatus representation of the HDIdle monitoring service.
	// The parentPid is used to indicate this is a subprocess by storing it as a negative value.
	// Convention: Subprocesses have their parent process PID as a negative number.
	GetProcessStatus(parentPid int32) *dto.ProcessStatus
	// CheckDeviceSupport checks if a block device supports HD idle spindown commands.
	// Returns a HDIdleDeviceSupport struct with support information and any error encountered.
	CheckDeviceSupport(blockPath string) (*dto.HDIdleDeviceSupport, errors.E)
}

/*
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
*/

// HDIdleStatus represents the current status of the service
/*
type HDIdleStatus struct {
	Running     bool
	MonitoredAt time.Time
	Disks       []dto.HDIdleDiskStatus
}
*/

// hDIdleService implements HDIdleServiceInterface
type hDIdleService struct {
	db               *gorm.DB
	ctx              context.Context
	apiContextCancel context.CancelFunc
	state            *dto.ContextState
	settingService   SettingServiceInterface
	eventBus         events.EventBusInterface

	// Internal state
	mu sync.RWMutex
	//running   bool
	stopChan  chan struct{}
	config    *internalConfig
	diskStats []*internalDiskState
	lastNow   time.Time
	converter converter.DtoToDbomConverterImpl
}

type internalDiskState struct {
	dto.HDIdleDeviceStatus
	Reads           uint64            `json:"-"` // Internal: not exposed via API
	Writes          uint64            `json:"-"` // Internal: not exposed via API
	IdleTime        time.Duration     `json:"-"` // Internal: idle threshold for this disk
	InternalCmdType dto.HdidleCommand `json:"-"` // Internal: command type for spindown
	PowerCondition  uint8             `json:"-"` // Internal: power condition for spindown
}

// HDIdleEffectiveConfig provides a snapshot of the effective configuration
// used by the HDIdle service after applying global and per-device settings.
/*
type HDIdleEffectiveConfig struct {
	Enabled bool
	Devices []string // list of GivenName (original DevicePath) included for monitoring
}
*/
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
	Name string
	//	GivenName      string
	Idle           time.Duration
	CommandType    dto.HdidleCommand
	PowerCondition uint8
}

/*

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
*/
// HDIdleServiceParams defines dependencies for HDIdleService
type HDIdleServiceParams struct {
	fx.In
	DB               *gorm.DB
	Ctx              context.Context
	ApiContextCancel context.CancelFunc
	State            *dto.ContextState
	SettingService   SettingServiceInterface
	EventBus         events.EventBusInterface
}

type HDIdleServiceOut struct {
	fx.Out
	HDIdleService HDIdleServiceInterface
	ProcessStatus SambaServiceProcessStatus `group:"internal_services"`
}

// NewHDIdleService creates a new HDIdleService instance
func NewHDIdleService(lc fx.Lifecycle, in HDIdleServiceParams) HDIdleServiceOut {

	hdidle_service := &hDIdleService{
		ctx:              in.Ctx,
		apiContextCancel: in.ApiContextCancel,
		state:            in.State,
		db:               in.DB,
		settingService:   in.SettingService,
		eventBus:         in.EventBus,
		converter:        converter.DtoToDbomConverterImpl{},
	}
	unsubscribe := make([]func(), 1)
	unsubscribe[0] = in.EventBus.OnSetting(func(ctx context.Context, se events.SettingEvent) errors.E {
		if se.Setting.HDIdleEnabled != nil && *se.Setting.HDIdleEnabled {
			_ = hdidle_service.Stop()
			err := hdidle_service.Start()
			if err != nil {
				tlog.ErrorContext(ctx, "Failed to start HDIdle service after settings change", "error", err)
			}
			return nil
		} else {
			err := hdidle_service.Stop()
			if err != nil {
				tlog.ErrorContext(ctx, "Failed to stop HDIdle service after settings change", "error", err)
			}
			return nil
		}
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			settings, err := in.SettingService.Load()
			if err != nil {
				return err
			}
			if settings.HDIdleEnabled != nil && *settings.HDIdleEnabled {
				return hdidle_service.Start()
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			_ = hdidle_service.Stop()
			for _, unsub := range unsubscribe {
				if unsub != nil {
					unsub()
				}
			}
			return nil
		},
	})

	return HDIdleServiceOut{
		HDIdleService: hdidle_service,
		ProcessStatus: hdidle_service,
	}
}

// Start begins monitoring disk activity
func (s *hDIdleService) Start() errors.E {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.IsRunning() {
		return errors.New("HDIdle service is already running")
	}

	// Convert to internal config
	var err errors.E
	s.config, err = s.convertConfig()
	if err != nil {
		return err
	}

	// Start monitoring in background
	if s.config.Enabled {
		s.stopChan = make(chan struct{})
		s.lastNow = time.Now()
		go func() {

			//s.running = true
			defer s.Stop()
			s.monitorLoop()
		}()
	}

	return nil
}

// Stop halts the monitoring process
func (s *hDIdleService) Stop() errors.E {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.IsRunning() {
		return errors.New("HDIdle service is not running")
	}

	tlog.DebugContext(s.ctx, "Stopping HDIdle service")
	if s.stopChan != nil {
		close(s.stopChan)
	}
	//s.stopChan = nil

	return nil
}

// IsRunning returns true if the service is currently monitoring
func (s *hDIdleService) IsRunning() bool {
	return s.stopChan != nil
}

/*
// GetStatus returns current monitoring status and disk states
func (s *hDIdleService) GetStatus() (*HDIdleStatus, errors.E) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.IsRunning() {
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

*/

func (s *hDIdleService) GetDeviceStatus(path string) (*dto.HDIdleDeviceStatus, errors.E) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.IsRunning() {
		return &dto.HDIdleDeviceStatus{
			Supported: false,
			Enabled:   false,
		}, nil
	}

	name := s.resolveDeviceNameFromPath(path)

	if len(s.diskStats) == 0 {
		return &dto.HDIdleDeviceStatus{
			Supported: false,
			Enabled:   false,
		}, nil
	}

	// Find disk state by given name or resolved name
	for _, ds := range s.diskStats {
		tlog.TraceContext(s.ctx, "Checking disk", "name", ds.Name, "searchPath", path, "resolvedName", name)
		if ds.Name == name {
			status := ds.HDIdleDeviceStatus
			return &status, nil
		}
	}

	return nil, errors.Errorf("disk %s (%s) not found from %#v", name, path, s.diskStats)
}

// resolveDeviceNameFromPath resolves symlinks to get the real device name (filename only)
func (s *hDIdleService) resolveDeviceNameFromPath(path string) string {
	realPath, err := io.RealPath(path)
	if err != nil {
		// If resolution fails, extract filename from original path
		return strings.TrimPrefix(path, "/dev/")
	}
	// Return only the filename without the /dev/ prefix
	return strings.TrimPrefix(realPath, "/dev/")
}

func (s *hDIdleService) GetDeviceConfig(path string) (*dto.HDIdleDevice, errors.E) {

	//device, err := g.HDIdleDeviceQuery[dbom.HDIdleDevice](s.db).LoadByPath(s.ctx, path)
	device, err := gorm.G[dbom.HDIdleDevice](s.db).
		//	Where("device_path = ?", path).
		Where(g.HDIdleDevice.DevicePath.Eq(path)).
		First(s.ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Device not in database, return config based on device support check with default
			support, checkErr := s.CheckDeviceSupport(path)
			if checkErr != nil {
				return nil, errors.Wrap(checkErr, "error checking HDIdle support for device")
			}
			if !support.Supported {
				return nil, errors.Wrap(dto.ErrorHDIdleNotSupported, "HDIdle not supported for this device")
			}
			result := &dto.HDIdleDevice{
				Supported:  true,
				DevicePath: support.DevicePath,
				Enabled:    dto.HdidleEnableds.YESENABLED,
			}
			if support.RecommendedCommand != nil {
				result.CommandType = *support.RecommendedCommand
			} else {
				result.CommandType = s.config.DefaultCommandType
			}

			result.IdleTime = s.config.DefaultIdle
			result.PowerCondition = s.config.DefaultPowerCondition
			/*
				errE := s.createDeviceConfig(*result)
				if errE != nil {
					return nil, errors.Wrap(errE, "error saving default HDIdle config for device")
				}
			*/
			return result, nil
		}
		return nil, errors.WithStack(err)
	}

	dtoDevice, errN := s.converter.HDIdleDeviceToHDIdleDeviceDTO(device)
	if errN != nil {
		return nil, errors.WithStack(errN)
	}
	return &dtoDevice, nil
}

func (s *hDIdleService) SaveDeviceConfig(device dto.HDIdleDevice) errors.E {
	dbDevice, err := s.converter.HDIdleDeviceDTOToHDIdleDevice(device)
	if err != nil {
		return errors.WithStack(err)
	}

	err = s.db.Debug().Save(&dbDevice).Error
	if err != nil {
		return errors.WithStack(err)
	}

	s.eventBus.EmitPower(events.PowerEvent{
		Event: events.Event{
			Type: events.EventTypes.UPDATE,
		},
		PowerInfo: device,
	})

	return nil
}

// CheckDeviceSupport checks if a block device supports HD idle spindown commands.
// It verifies that the device can be opened as an SG (SCSI Generic) device and
// determines which spindown command types are supported (SCSI and/or ATA).
//
// Parameters:
//   - blockPath: The path to the block device (e.g., "/dev/sda" or "/dev/disk/by-id/...")
//
// Returns:
//   - HDIdleDeviceSupport: Contains support information including:
//   - Supported: true if any spindown command is supported
//   - SupportsSCSI: true if SCSI START STOP UNIT command is supported
//   - SupportsATA: true if ATA STANDBY command is supported
//   - RecommendedCommand: the recommended command type for this device
//   - DevicePath: the resolved real path of the device
//   - ErrorMessage: error details if the device is not supported
func (s *hDIdleService) CheckDeviceSupport(blockPath string) (*dto.HDIdleDeviceSupport, errors.E) {
	support := &dto.HDIdleDeviceSupport{
		Supported:    false,
		SupportsSCSI: false,
		SupportsATA:  false,
		DevicePath:   blockPath,
	}

	// Handle empty path
	if blockPath == "" {
		support.ErrorMessage = "device path cannot be empty"
		return support, nil
	}

	name := s.resolveDeviceNameFromPath(blockPath)

	// Try to open the device as an SG device to check basic support
	if !s.checkSGSupport("/dev/" + name) {
		support.ErrorMessage = "device does not support SG (SCSI Generic) interface"
		return support, nil
	}

	// Device supports SG interface, now check specific command support
	// SCSI devices generally support the START STOP UNIT command
	support.SupportsSCSI = true

	// Check if the device also supports ATA commands via ATA PASS-THROUGH
	// Most SATA drives connected via AHCI support both
	support.SupportsATA = s.checkATASupport(name)

	// Set overall support flag
	support.Supported = support.SupportsSCSI || support.SupportsATA

	// Determine recommended command type
	if support.Supported {
		if support.SupportsATA {
			// ATA is generally preferred for SATA drives
			cmd := dto.HdidleCommands.ATACOMMAND
			support.RecommendedCommand = &cmd
		} else if support.SupportsSCSI {
			cmd := dto.HdidleCommands.SCSICOMMAND
			support.RecommendedCommand = &cmd
		}
	}

	return support, nil
}

// checkSGSupport checks if a device supports the SG (SCSI Generic) interface
// by attempting to open it and verify the SG version
func (s *hDIdleService) checkSGSupport(devicePath string) bool {
	f, err := os.OpenFile(devicePath, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer f.Close()

	// Check SG version using ioctl
	var version uint32
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		sgGetVersionNum,
		uintptr(unsafe.Pointer(&version)),
	)
	if errno != 0 || version < 30000 {
		return false
	}

	return true
}

// checkATASupport checks if a device supports ATA commands via ATA PASS-THROUGH
// by checking device identification in sysfs
func (s *hDIdleService) checkATASupport(device string) bool {
	// Extract device name from path (e.g., "sda" from "/dev/sda")
	deviceName := strings.TrimPrefix(device, "/dev/")

	// Check if the device has ATA characteristics in sysfs
	sysPath := fmt.Sprintf("/sys/block/%s/device/vendor", deviceName)
	if _, err := os.Stat(sysPath); err != nil {
		// Also check for partition parent
		if len(deviceName) > 0 {
			// Try parent device (e.g., sda1 -> sda)
			parentName := strings.TrimRight(deviceName, "0123456789")
			sysPath = fmt.Sprintf("/sys/block/%s/device/vendor", parentName)
			if _, err := os.Stat(sysPath); err != nil {
				return false
			}
		} else {
			return false
		}
	}

	// If the device exists in sysfs and has basic block device properties,
	// it likely supports ATA commands. Most modern SATA drives support both
	// SCSI (via SAT - SCSI/ATA Translation) and native ATA commands.
	return true
}

/*
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
	if len(cfg.Devices) > 0 && ec.Enabled {
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
*/
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
		IsRunning:     s.IsRunning(),
	}

	// If running, populate with monitored disk count
	if s.IsRunning() {
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

	devices, errS := query.HDIdleDeviceQuery[dbom.HDIdleDevice](s.db).All(s.ctx)
	if errS != nil {
		//tlog.Error("Failed to load HDIdle devices from repository", "error", err)
		return nil, errors.Wrap(errS, "failed to load HDIdle devices")
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
				Name: deviceRealPath,
				//				GivenName:      dev.DevicePath,
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

	// Update effective enabled state: global switch OR at least one device explicitly enabled
	if !globalEnabled && len(intConfig.Devices) > 0 {
		// When global is off but devices were explicitly enabled, enable service for those devices
		intConfig.Enabled = true
	}

	// Calculate skew time and pool interval
	interval := s.calculatePoolInterval()
	intConfig.SkewTime = interval * 3

	return intConfig, nil
}

// calculatePoolInterval determines the polling interval
func (s *hDIdleService) calculatePoolInterval() time.Duration {
	if s.config == nil {
		slog.WarnContext(s.ctx, "Null config while calculating pool interval, using default 10s")
		return time.Second * 10
	}
	defaultIdleTime := s.config.DefaultIdle
	if len(s.config.Devices) == 0 {
		return defaultIdleTime / defaultPoolMultiplier
	}

	interval := defaultIdleTime
	for _, dev := range s.config.Devices {
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
	interval := s.calculatePoolInterval()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-s.ctx.Done():
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

	if !s.IsRunning() {
		return
	}

	snapshot := diskstats.Snapshot()
	now := time.Now()

	// Resolve symlinks if needed
	//s.resolveSymlinks()

	// Process each disk in the snapshot
	for _, stats := range snapshot {
		s.updateDiskState(stats.Name, stats.Reads, stats.Writes, now)
	}

	s.lastNow = now
}

/*
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
*/

// updateDiskState updates the state of a disk based on current activity
func (s *hDIdleService) updateDiskState(name string, reads, writes uint64, now time.Time) {
	// Find existing disk state
	dsi := s.findDiskStateIndex(name)
	if dsi < 0 {
		// New disk, initialize it
		s.diskStats = append(s.diskStats, s.initDiskState(name, reads, writes, now))
		return
	}

	ds := s.diskStats[dsi]

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
			timeSinceLastSpunDown := now.Sub(ds.SpinDownAt)

			if ds.IdleTimeMillis != 0 && idleDuration.Milliseconds() > ds.IdleTimeMillis && timeSinceLastSpunDown.Milliseconds() > ds.IdleTimeMillis {
				// Time to spin down
				//givenName := s.resolveDeviceGivenName(ds.Name)
				if ds.SpunDown && s.config.IgnoreSpinDown {
					tlog.InfoContext(s.ctx, "Spindown (ignoring prior spin down state)", "disk", ds.Name)
				} else {
					tlog.InfoContext(s.ctx, "Spindown", "disk", ds.Name)
				}

				device := fmt.Sprintf("/dev/%s", ds.Name)
				if err := s.spindownDisk(device, ds.InternalCmdType, ds.PowerCondition); err != nil {
					tlog.ErrorContext(s.ctx, "Failed to spindown disk", "disk", ds.Name, "error", err)
				}

				//		ds.LastSpunDownAt = now
				ds.SpinDownAt = now
				ds.SpunDown = true
			}
		}
	} else {
		// Disk had activity
		if ds.SpunDown {
			// Disk just spun up
			//givenName := s.resolveDeviceGivenName(ds.Name)
			tlog.InfoContext(s.ctx, "Spinup", "disk", ds.Name)
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
func (s *hDIdleService) initDiskState(name string, reads, writes uint64, now time.Time) *internalDiskState {
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

	return &internalDiskState{
		HDIdleDeviceStatus: dto.HDIdleDeviceStatus{
			Name: name,
			//GivenName:      s.resolveDeviceGivenName(name),
			LastIOAt:       now,
			SpinUpAt:       now,
			SpunDown:       false,
			IdleTimeMillis: idle.Milliseconds(),
			CommandType:    command.String(),
			Supported:      true,
			Enabled:        true,
		},
		Writes:          writes,
		Reads:           reads,
		IdleTime:        idle,
		InternalCmdType: command,
		PowerCondition:  powerCondition,
	}
}

/*
// resolveDeviceGivenName resolves the given name for a device
func (s *hDIdleService) resolveDeviceGivenName(name string) string {
	if givenName, ok := s.config.NameMap[name]; ok {
		return givenName
	}
	return name
}
*/

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
