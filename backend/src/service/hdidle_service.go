package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

// ataProbeFn is used by CheckATASupport to probe whether a device supports
// ATA PASS-THROUGH. It defaults to sgio.CheckAtaDevice which issues a
// read-only CHECK POWER MODE command (0xE5) instead of sgio.StopAtaDevice
// which issues STANDBY IMMEDIATE (0xE0) and physically spins down the disk.
// The spindownDisk path (spindownDisk) still uses sgio.StopAtaDevice directly
// because an intentional spindown is required there.
var ataProbeFn = sgio.CheckAtaDevice

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
	// ResolveDevicePath converts an API-side disk identifier (raw kernel name
	// like "sda", a by-id symlink like "ata-WDC...", or an already-absolute
	// /dev path) to an absolute device path that exists on this system.
	// Returns dto.ErrorNotFound if no candidate resolves.
	ResolveDevicePath(diskID string) (string, errors.E)
}

// HDIdleService implements HDIdleServiceInterface
type HDIdleService struct {
	db               *gorm.DB
	ctx              context.Context
	apiContextCancel context.CancelFunc
	state            *dto.ContextState
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

	// Cache fields
	LastEmittedSpunDown bool `json:"-"` // Internal: track emitted state to avoid spam
	IsInitialized       bool `json:"-"` // Internal: force first emit
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
	// Lifecycle is fully delegated to Start()/Stop():
	//   - Start() calls convertConfig() once and only launches the monitor
	//     goroutine when at least one device has Enabled ≠ NOENABLED;
	//   - Stop() is idempotent.
	// We avoid eagerly calling convertConfig() at construction time — the
	// service struct's zero-value `config` is good enough until OnStart fires.
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return hdidle_service.Start()
		},
		OnStop: func(ctx context.Context) error {
			if err := hdidle_service.Stop(); err != nil {
				return err
			}
			return nil
		},
	})

	return HDIdleServiceOut{
		HDIdleService: hdidle_service,
		ProcessStatus: hdidle_service,
	}
}

// Start begins monitoring disk activity. It is idempotent: calling Start on
// an already-running service is a no-op (returns nil). The service runs only
// when at least one device row has Enabled ≠ NOENABLED — when no enabled
// devices exist, Start refreshes config but does not launch the monitor
// goroutine.
//
// To restart with a fresh configuration after a DB change, call Stop then
// Start, or use the helper Reconfigure (added in a later phase).
func (s *HDIdleService) Start() errors.E {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotent: already running ⇒ nothing to do. Previously this returned
	// an error, which combined with the stopChan-not-nilled bug to permanently
	// break the service after the first Stop(). Both are now fixed.
	if s.stopChan != nil {
		return nil
	}

	// Refresh internal config from the DB (per-disk-only model).
	var err errors.E
	s.config, err = s.convertConfig()
	if err != nil {
		return err
	}

	// Reset per-disk activity state so each run starts with a clean slate.
	// Without this, a Stop → (DB change) → Start cycle would carry stale
	// SpunDown/LastIOAt values and miss devices removed from the config.
	s.diskStats = nil

	// No enabled devices ⇒ config refreshed but no goroutine launched.
	// The lifecycle hook on PUT /hdidle/config will call Start() again
	// once the user enables a device.
	if !s.config.Enabled {
		return nil
	}

	stop := make(chan struct{})
	s.stopChan = stop
	s.lastNow = time.Now()
	go s.monitorLoop(stop)

	return nil
}

// Stop halts the monitoring process. Idempotent: stopping an already-stopped
// service is a no-op. After Stop returns, the service can be Start()ed again.
func (s *HDIdleService) Stop() errors.E {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopChan == nil {
		tlog.DebugContext(s.ctx, "HDIdle service is not running, no action needed")
		return nil
	}

	tlog.DebugContext(s.ctx, "Stopping HDIdle service")
	close(s.stopChan)
	s.stopChan = nil // critical: allows future Start() calls to succeed.

	return nil
}

// IsRunning returns true if the service is currently monitoring.
//
// Note: callers must not hold s.mu when invoking this function — the read is
// guarded by RLock to remain safe under concurrent Start/Stop.
func (s *HDIdleService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stopChan != nil
}

func (s *HDIdleService) GetDeviceStatus(path string) (*dto.HDIdleDeviceStatus, errors.E) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read stopChan directly: nested IsRunning() under RLock can deadlock
	// when a concurrent writer is parked between the two RLock calls.
	if s.stopChan == nil {
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

// ResolveDevicePath maps a caller-supplied disk identifier to a /dev path
// that exists on this system. Tried, in order:
//  1. Already an absolute path under /dev — used as-is after stat.
//  2. /dev/disk/by-id/<diskID> — the canonical stable identifier.
//  3. /dev/<diskID> — bare kernel name (e.g. "sda", "nvme0n1").
//
// Returns dto.ErrorNotFound when no candidate exists. The diskID is
// rejected outright if it contains slashes, ".." segments, or NUL — these
// are signs of injection rather than a valid kernel/by-id name.
func (s *HDIdleService) ResolveDevicePath(diskID string) (string, errors.E) {
	if diskID == "" {
		return "", errors.Wrap(dto.ErrorNotFound, "empty disk identifier")
	}

	// Reject path-traversal/injection attempts before any path operation.
	// This guard must come before the /dev/ fast-path: an input like
	// /dev/../proc/self/fd/0 starts with /dev/ but escapes it after cleaning.
	if strings.ContainsAny(diskID, "\\\x00") || strings.Contains(diskID, "..") {
		return "", errors.Wrap(dto.ErrorNotFound, "invalid disk identifier: "+diskID)
	}

	// Allow already-absolute /dev paths, but only after cleaning and re-checking
	// that the canonical path still lives under /dev/ — defends against traversal
	// sequences that survived the string checks above.
	if strings.HasPrefix(diskID, "/dev/") {
		clean := filepath.Clean(diskID)
		if !strings.HasPrefix(clean, "/dev/") {
			return "", errors.Wrap(dto.ErrorNotFound, "invalid disk identifier: "+diskID)
		}
		if _, err := os.Stat(clean); err == nil {
			return clean, nil
		}
		return "", errors.Wrap(dto.ErrorNotFound, "device path does not exist: "+diskID)
	}

	// Reject bare identifiers that still contain slashes (e.g. "disk/by-id/../foo").
	if strings.ContainsAny(diskID, "/") {
		return "", errors.Wrap(dto.ErrorNotFound, "invalid disk identifier: "+diskID)
	}

	candidates := []string{
		"/dev/disk/by-id/" + diskID,
		"/dev/" + diskID,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.Wrap(dto.ErrorNotFound, "no device found for: "+diskID)
}

func (s *HDIdleService) GetDeviceConfig(path string) (*dto.HDIdleDevice, errors.E) {

	// GetDeviceConfig is a configuration-inspection endpoint and remains
	// available regardless of whether the monitor goroutine is running.
	// In the per-disk-only model the monitor only runs when ≥1 disk is
	// enabled — but the UI must still be able to read/write disabled-disk
	// records. The previous "service disabled → 503" guard has been removed.

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
		Kind:      events.PowerEventKindConfig,
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
		tlog.TraceContext(s.ctx, "Failed to open device as SG device", "device", devicePath, "error", err)
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
	if err := ataProbeFn(device, tlog.GetLevel() <= slog.LevelDebug); err != nil {
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

	// Read stopChan once under the held lock instead of calling IsRunning()
	// twice (which would re-acquire the RLock — not guaranteed reentrant).
	running := s.stopChan != nil

	status := &dto.ProcessStatus{
		Pid:           -parentPid, // Negative PID indicates this is a subprocess
		Name:          "powersave-monitor",
		CreateTime:    time.Time{},
		CPUPercent:    0.0,
		MemoryPercent: 0.0,
		OpenFiles:     0,
		Connections:   0,
		Status:        []string{"idle"},
		IsRunning:     running,
	}

	if running {
		status.Connections = len(s.diskStats)
		status.Status = []string{"running"}
	}

	return status
}

// Per-disk default tunables. The global Settings.HDIdle* fields were removed
// when HDIdle moved to a per-disk-only model; these constants replace them
// until Phase 2 introduces per-device override of every field.
const (
	hdidleDefaultIdleSeconds   = 60
	hdidleDefaultPowerCond     = uint8(0)
	hdidleDefaultIgnoreSpinDwn = false
)

// convertConfig converts external config to internal config
func (s *HDIdleService) convertConfig() (*internalConfig, errors.E) {

	intConfig := &internalConfig{
		Enabled:               false,
		Devices:               make(map[string]dto.HDIdleDevice),
		DefaultIdle:           time.Duration(hdidleDefaultIdleSeconds) * time.Second,
		DefaultCommandType:    dto.HdidleCommands.SCSICOMMAND,
		DefaultPowerCondition: hdidleDefaultPowerCond,
		IgnoreSpinDown:        hdidleDefaultIgnoreSpinDwn,
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
			// Skip devices with no path — they cannot be monitored and
			// getRealPathNotSimlink("") panics inside the hd-idle library.
			if dev.DevicePath == "" {
				slog.WarnContext(s.ctx, "HDIdle device has empty DevicePath, skipping", "device", dev)
				continue
			}
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

	// Per-disk-only model: service runs iff at least one device is enabled.
	intConfig.Enabled = len(intConfig.Devices) > 0

	// SkewTime is the threshold for detecting OS suspend/sleep events: when
	// the gap between two ticks exceeds it, updateDiskState resets the disk's
	// idle counters. It must be strictly greater than hdidleActiveInterval (60s)
	// so that normal poll ticks are never misidentified as a suspend/wake event.
	// We use 3× the active interval (180s) as a comfortable margin.
	intConfig.SkewTime = hdidleActiveInterval * 3

	return intConfig, nil
}

// Adaptive polling intervals (goal #4: minimal resource use).
// active: at least one monitored disk is currently spun-up — poll often
//
//	enough to catch the next idle window precisely.
//
// dormant: every monitored disk is spun-down — nothing to do, slow the
//
//	tick by an order of magnitude. The OS still wakes the disk on
//	real I/O; we just stop checking until the disk wakes itself.
const (
	hdidleActiveInterval  = 60 * time.Second
	hdidleDormantInterval = 5 * time.Minute
)

// monitorLoop is the main monitoring loop. The stop channel is passed in
// rather than read from s.stopChan to avoid a race with Stop(): when Stop
// nils s.stopChan, this goroutine still selects on the original channel.
//
// The loop uses a dynamically-reset timer instead of a fixed Ticker so the
// poll cadence can shift between active and dormant intervals based on
// whether any monitored disk is currently spun-up.
func (s *HDIdleService) monitorLoop(stop <-chan struct{}) {
	timer := time.NewTimer(s.nextPollInterval())
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-s.ctx.Done():
			return
		case <-timer.C:
			s.observeDiskActivity()
			// Re-arm with the interval appropriate for the current state.
			timer.Reset(s.nextPollInterval())
		}
	}
}

// nextPollInterval returns the polling cadence for the next tick:
//   - dormant interval (5min) when every monitored disk is spun-down;
//   - active interval (60s) otherwise.
//
// Reads s.diskStats under RLock — must not be called while holding mu.Lock.
func (s *HDIdleService) nextPollInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.diskStats) == 0 {
		// No disks observed yet — stay in active so we discover them quickly.
		return hdidleActiveInterval
	}
	for _, ds := range s.diskStats {
		if !ds.SpunDown {
			return hdidleActiveInterval
		}
	}
	return hdidleDormantInterval
}

// observeDiskActivity observes disk activity and spins down idle disks
func (s *HDIdleService) observeDiskActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read stopChan directly under the held write lock — calling IsRunning()
	// here would deadlock because IsRunning() takes an RLock on the same mutex.
	if s.stopChan == nil {
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
		tlog.TraceContext(s.ctx, "Disk not configured for HDIdle, skipping", "disk", name, "devices", s.config.Devices)
		return
	} else {
		if dv.Enabled == dto.HdidleEnableds.NOENABLED || !dv.Supported {
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

	stateChanged := !ds.IsInitialized

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

				ds.SpinDownAt = now
				ds.SpunDown = true
			}
		}
	} else {
		// Disk had activity
		if ds.SpunDown {
			// Disk just spun up
			tlog.InfoContext(s.ctx, "Spinup", "disk", ds.Name)
			ds.SpinUpAt = now
		}

		ds.Reads = reads
		ds.Writes = writes
		ds.LastIOAt = now
		ds.SpunDown = false
	}

	// Always trigger state update on transition
	if ds.SpunDown != ds.LastEmittedSpunDown {
		stateChanged = true
	}

	if stateChanged {
		ds.LastEmittedSpunDown = ds.SpunDown
		ds.IsInitialized = true

		s.eventBus.EmitPower(events.PowerEvent{
			Event: events.Event{
				Type: events.EventTypes.UPDATE,
			},
			Kind:        events.PowerEventKindStatus,
			PowerStatus: ds.HDIdleDeviceStatus,
		})
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
