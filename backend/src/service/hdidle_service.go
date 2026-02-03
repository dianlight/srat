package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/adelolmo/hd-idle/diskstats"
	"github.com/adelolmo/hd-idle/io"
	"github.com/adelolmo/hd-idle/sgio"
	sg "github.com/benmcclelland/sgio"
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
	//sgGetVersionNum = 0x2282
)

// HDIdleServiceInterface provides methods for managing hard disk idle monitoring
type HDIdleServiceInterface interface {
	// Start begins monitoring disk activity and spinning down idle disks
	Start() errors.E
	// Stop halts the monitoring process
	Stop() errors.E
	// IsRunning returns true if the service is currently monitoring
	IsRunning() bool
	GetDeviceStatus(path string) (*dto.HDIdleDeviceStatus, errors.E)
	GetDeviceConfig(path string) (*dto.HDIdleDevice, errors.E)
	SaveDeviceConfig(device dto.HDIdleDevice) errors.E
	// GetProcessStatus returns a ProcessStatus representation of the HDIdle monitoring service.
	// The parentPid is used to indicate this is a subprocess by storing it as a negative value.
	// Convention: Subprocesses have their parent process PID as a negative number.
	GetProcessStatus(parentPid int32) *dto.ProcessStatus
	// CheckDeviceSupport checks if a block device supports HD idle spindown commands.
	// Returns a HDIdleDeviceSupport struct with support information and any error encountered.
	CheckDeviceSupport(blockPath string) (*dto.HDIdleDeviceSupport, errors.E)
	CheckSGSupport(devicePath string) bool
	CheckATASupport(device string) bool
}

// HDIdleService implements HDIdleServiceInterface
type HDIdleService struct {
	db               *gorm.DB
	ctx              context.Context
	apiContextCancel context.CancelFunc
	state            *dto.ContextState
	settingService   SettingServiceInterface
	eventBus         events.EventBusInterface
	mu               sync.RWMutex
	stopChan         chan struct{}
	config           *internalConfig
	diskStats        []*internalDiskState
	lastNow          time.Time
	converter        converter.DtoToDbomConverterImpl
}

type internalDiskState struct {
	dto.HDIdleDeviceStatus
	Reads           uint64            `json:"-"` // Internal: not exposed via API
	Writes          uint64            `json:"-"` // Internal: not exposed via API
	IdleTime        time.Duration     `json:"-"` // Internal: idle threshold for this disk
	InternalCmdType dto.HdidleCommand `json:"-"` // Internal: command type for spindown
	PowerCondition  uint8             `json:"-"` // Internal: power condition for spindown
}

type internalConfig struct {
	Enabled               bool
	Devices               map[string]dto.HDIdleDevice
	DefaultIdle           time.Duration
	DefaultCommandType    dto.HdidleCommand
	DefaultPowerCondition uint8
	IgnoreSpinDown        bool
	SkewTime              time.Duration
	//NameMap               map[string]string
}

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
	ProcessStatus ServerProcessStatus `group:"internal_services"`
}

// NewHDIdleService creates a new HDIdleService instance
func NewHDIdleService(lc fx.Lifecycle, in HDIdleServiceParams) HDIdleServiceOut {

	hdidle_service := &HDIdleService{
		ctx:              in.Ctx,
		apiContextCancel: in.ApiContextCancel,
		state:            in.State,
		db:               in.DB,
		settingService:   in.SettingService,
		eventBus:         in.EventBus,
		converter:        converter.DtoToDbomConverterImpl{},
		config: &internalConfig{
			Enabled:               false,
			Devices:               map[string]dto.HDIdleDevice{},
			DefaultIdle:           time.Duration(0),
			DefaultCommandType:    dto.HdidleCommands.SCSICOMMAND,
			DefaultPowerCondition: uint8(0),
			IgnoreSpinDown:        true,
			SkewTime:              time.Duration(0),
			//NameMap:               map[string]string{},
		},
	}
	hdidle_service.config, _ = hdidle_service.convertConfig()
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
func (s *HDIdleService) Start() errors.E {
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
func (s *HDIdleService) Stop() errors.E {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.IsRunning() {
		// Stopping an already-stopped service is a no-op and not an error
		tlog.DebugContext(s.ctx, "HDIdle service is not running, no action needed")
		return nil
	}

	tlog.DebugContext(s.ctx, "Stopping HDIdle service")
	if s.stopChan != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					tlog.WarnContext(s.ctx, "Panic while closing stop channel", "panic", r)
				}
			}()
			close(s.stopChan)
		}()
	}
	//s.stopChan = nil

	return nil
}

// IsRunning returns true if the service is currently monitoring
func (s *HDIdleService) IsRunning() bool {
	return s.stopChan != nil
}

func (s *HDIdleService) GetDeviceStatus(path string) (*dto.HDIdleDeviceStatus, errors.E) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.IsRunning() {
		return nil, nil
	}

	name := s.getRealPathNotSimlink(path)

	if len(s.diskStats) == 0 {
		return nil, nil
	}

	// Find disk state by given name or resolved name
	for _, ds := range s.diskStats {
		tlog.TraceContext(s.ctx, "Checking disk", "name", ds.Name, "searchPath", path, "resolvedName", name)
		if ds.Name == name || ds.Name == strings.TrimPrefix(name, "/dev/") {
			status := ds.HDIdleDeviceStatus
			return &status, nil
		}
	}

	return nil, errors.Errorf("disk %s (%s) not found from %#v", name, path, s.diskStats)
}

// getRealPathNotSimlink resolves symlinks to get the real device name (filename only)
func (s *HDIdleService) getRealPathNotSimlink(path string) string {
	realPath, err := io.RealPath(path)
	if err != nil {
		// If resolution fails, extract filename from original path
		slog.WarnContext(s.ctx, "Failed to resolve real path for device, using original path", "path", path, "error", err)
		return path
	}
	// Return only the filename without the /dev/ prefix
	return realPath
}

func (s *HDIdleService) GetDeviceConfig(path string) (*dto.HDIdleDevice, errors.E) {

	if s.config.Enabled == false {
		return nil, errors.Wrap(dto.ErrorHDIdleNotSupported, "HDIdle service is disabled")
	}

	device, err := gorm.G[dbom.HDIdleDevice](s.db).
		Where(g.HDIdleDevice.DevicePath.Eq(path)).
		First(s.ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Device not in database, return config based on device support check with default
			support, checkErr := s.CheckDeviceSupport(path)
			if checkErr != nil {
				return nil, errors.Wrap(checkErr, "error checking HDIdle support for device")
			}
			if support == nil {
				return nil, errors.New("HDIdle support check returned nil support")
			}
			if !support.Supported {
				return &dto.HDIdleDevice{
					HDIdleDeviceSupport: *support,
				}, errors.Wrap(dto.ErrorHDIdleNotSupported, "HDIdle not supported for this device")
			}
			result := &dto.HDIdleDevice{
				HDIdleDeviceSupport: *support,
				//		DevicePath: support.DevicePath,
				Enabled: dto.HdidleEnableds.YESENABLED,
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
			name := s.getRealPathNotSimlink(path)
			s.config.Devices[name] = *result
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

func (s *HDIdleService) SaveDeviceConfig(device dto.HDIdleDevice) errors.E {
	dbDevice, err := s.converter.HDIdleDeviceDTOToHDIdleDevice(device)
	if err != nil {
		return errors.WithStack(err)
	}

	err = s.db.Save(&dbDevice).Error
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
func (s *HDIdleService) CheckDeviceSupport(blockPath string) (*dto.HDIdleDeviceSupport, errors.E) {
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

	// Check if the blockPath exists and is a block device or symlink
	info, err := os.Lstat(blockPath)
	if err != nil {
		support.ErrorMessage = fmt.Sprintf("failed to stat device path: %v", err)
		return support, nil
	}

	if info.Mode()&os.ModeDevice == 0 && info.Mode()&os.ModeSymlink == 0 {
		support.ErrorMessage = "device path is not a block device or symlink"
		return support, nil
	}

	name := s.getRealPathNotSimlink(blockPath)

	// Try to open the device as an SG device to check basic support
	support.SupportsSCSI = s.CheckSGSupport(blockPath)

	// Device supports SG interface, now check specific command support
	// SCSI devices generally support the START STOP UNIT command

	// Check if the device also supports ATA commands via ATA PASS-THROUGH
	// Most SATA drives connected via AHCI support both
	support.SupportsATA = s.CheckATASupport(name)

	// Set overall support flag
	support.Supported = support.SupportsSCSI // SGSupport is necessary also for ATA commands

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

// CheckSGSupport checks if a device supports the SG (SCSI Generic) interface
// by attempting to open it and verify the SG version
func (s *HDIdleService) CheckSGSupport(devicePath string) bool {
	f, err := sg.OpenScsiDevice(devicePath)
	if err != nil {
		slog.DebugContext(s.ctx, "Failed to open device as SG device", "device", devicePath, "error", err)
		return false
	}
	defer f.Close()
	return true
}

// CheckATASupport checks if a device supports ATA commands via ATA PASS-THROUGH
// by checking device identification in sysfs
func (s *HDIdleService) CheckATASupport(device string) bool {
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
	device = fmt.Sprintf("/dev/%s", deviceName)
	if err := sgio.StopAtaDevice(device, tlog.GetLevel() <= slog.LevelDebug); err != nil {
		if strings.Contains(err.Error(), "INVALID COMMAND OPERATION CODE") {
			return false
		}
		slog.WarnContext(s.ctx, "Error checking ATA device support", "error", err)
		return false
	}

	return true
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
func (s *HDIdleService) GetProcessStatus(parentPid int32) *dto.ProcessStatus {
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
func (s *HDIdleService) convertConfig() (*internalConfig, errors.E) {

	settings, err := s.settingService.Load()
	if err != nil {
		return nil, err
	}
	if settings == nil {
		slog.WarnContext(s.ctx, "Settings are nil while converting HDIdle config, using existing config if any")
		return s.config, nil
	}

	// Global enabled flag from settings (defaults to false when nil)
	globalEnabled := false
	if settings.HDIdleEnabled != nil {
		globalEnabled = *settings.HDIdleEnabled
	}

	intConfig := &internalConfig{
		// Effective enabled: global switch OR at least one per-device forced enabled
		Enabled:               globalEnabled, // may be updated below after loading devices
		Devices:               make(map[string]dto.HDIdleDevice),
		DefaultIdle:           time.Duration((*settings).HDIdleDefaultIdleTime) * time.Second,
		DefaultCommandType:    (*settings).HDIdleDefaultCommandType,
		DefaultPowerCondition: (*settings).HDIdleDefaultPowerCondition,
		IgnoreSpinDown:        (*settings).HDIdleIgnoreSpinDownDetection,
		//NameMap:               make(map[string]string),
	}

	devices, errS := query.HDIdleDeviceQuery[dbom.HDIdleDevice](s.db).All(s.ctx)
	if errS != nil {
		//tlog.Error("Failed to load HDIdle devices from repository", "error", err)
		return nil, errors.Wrap(errS, "failed to load HDIdle devices")
	}

	// Convert devices considering per-device Enabled tri-state
	for _, dev := range devices {
		/*
			deviceRealPath, err := io.RealPath(dev.DevicePath)
			if err != nil {
				deviceRealPath = ""
			}
		*/

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
			devConfig, err := s.converter.HDIdleDeviceToHDIdleDeviceDTO(*dev)
			if err != nil {
				return nil, errors.Wrap(err, "error converting HDIdle device to DTO")
			}
			support, errE := s.CheckDeviceSupport(dev.DevicePath)
			if errE != nil {
				return nil, errors.Wrap(errE, "error checking HDIdle support for device")
			}
			devConfig.HDIdleDeviceSupport = *support
			devConfig.IdleTime = idle
			devConfig.CommandType = *cmdType

			name := s.getRealPathNotSimlink(dev.DevicePath)
			intConfig.Devices[name] = devConfig
			/*
				if deviceRealPath != "" {
					intConfig.NameMap[deviceRealPath] = dev.DevicePath
				}
			*/
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
func (s *HDIdleService) calculatePoolInterval() time.Duration {
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
		if dev.IdleTime == 0 {
			continue
		}
		if dev.IdleTime < interval {
			interval = dev.IdleTime
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
func (s *HDIdleService) observeDiskActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.IsRunning() {
		return
	}

	snapshot := diskstats.Snapshot()
	now := time.Now()

	// Process each disk in the snapshot
	for _, stats := range snapshot {
		s.updateDiskState(stats.Name, stats.Reads, stats.Writes, now)
	}

	s.lastNow = now
}

// updateDiskState updates the state of a disk based on current activity
func (s *HDIdleService) updateDiskState(name string, reads, writes uint64, now time.Time) {

	if dv, ok := s.config.Devices[name]; !ok {
		// Device not configured for HDIdle, skip processing
		slog.DebugContext(s.ctx, "Disk not configured for HDIdle, skipping", "disk", name, "devices", s.config.Devices)
		return
	} else {
		if dv.Enabled == dto.HdidleEnableds.NOENABLED || dv.Supported == false {
			// Device explicitly disabled for HDIdle, skip processing
			return
		}
	}
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

			if idleDuration > ds.IdleTime && timeSinceLastSpunDown > ds.IdleTime {
				// Time to spin down
				if ds.SpunDown && s.config.IgnoreSpinDown {
					tlog.InfoContext(s.ctx, "Spindown (ignoring prior spin down state)", "disk", ds.Name)
				} else {
					tlog.InfoContext(s.ctx, "Spindown", "disk", ds.Name, "type", ds.InternalCmdType.String(), "inactivity", idleDuration.String())
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

	s.eventBus.EmitPower(events.PowerEvent{
		Event: events.Event{
			Type: events.EventTypes.UPDATE,
		},
		PowerStatus: ds.HDIdleDeviceStatus,
	})
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
func (s *HDIdleService) initDiskState(name string, reads, writes uint64, now time.Time) *internalDiskState {
	idle := s.config.DefaultIdle
	command := s.config.DefaultCommandType
	powerCondition := s.config.DefaultPowerCondition

	if devName, ok := s.config.Devices[name]; ok {
		idle = devName.IdleTime
		command = devName.CommandType
		powerCondition = devName.PowerCondition
	}

	return &internalDiskState{
		HDIdleDeviceStatus: dto.HDIdleDeviceStatus{
			Name:     name,
			LastIOAt: now,
			SpinUpAt: now,
			SpunDown: false,
		},
		Writes:          writes,
		Reads:           reads,
		IdleTime:        idle,
		InternalCmdType: command,
		PowerCondition:  powerCondition,
	}
}

// spindownDisk spins down a disk using the appropriate command
func (s *HDIdleService) spindownDisk(device string, command dto.HdidleCommand, powerCondition uint8) errors.E {
	switch command {
	case dto.HdidleCommands.SCSICOMMAND:
		if err := sgio.StartStopScsiDevice(device, powerCondition); err != nil {
			return errors.Errorf("cannot spindown scsi disk %s: %w", device, err)
		}
		return nil
	case dto.HdidleCommands.ATACOMMAND:
		if err := sgio.StopAtaDevice(device, tlog.GetLevel() <= slog.LevelDebug); err != nil {
			return errors.Errorf("cannot spindown ata disk %s: %w", device, err)
		}
		return nil
	default:
		return errors.Errorf("unknown command type: %s", command)
	}
}
