package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/u-root/u-root/pkg/mount"
	"gitlab.com/tozd/go/errors"
)

// FilesystemServiceInterface defines the methods for managing filesystem types and mount flags.
type FilesystemServiceInterface interface {
	// GetStandardMountFlags returns a list of common, filesystem-agnostic mount flags.
	GetStandardMountFlags() ([]dto.MountFlag, errors.E)

	// GetFilesystemSpecificMountFlags returns a list of mount flags specific to a given filesystem type.
	// Returns an empty list if the filesystem type is not recognized or has no specific flags.
	GetFilesystemSpecificMountFlags(fsType string) ([]dto.MountFlag, errors.E)

	// ResolveLinuxFsModule returns the Linux filesystem module/fstype name for mounting.
	// Falls back to the provided filesystem type when no adapter is found.
	ResolveLinuxFsModule(fsType string) string

	// GetMountFlagsAndData converts a list of MountFlag structs into the syscall flags (uintptr)
	// and the data string (string) for the syscall.Mount function.
	MountFlagsToSyscallFlagAndData(inputFlags []dto.MountFlag) (uintptr, string, errors.E)

	SyscallFlagToMountFlag(syscallFlag uintptr) ([]dto.MountFlag, errors.E)

	SyscallDataToMountFlag(data string) ([]dto.MountFlag, errors.E)

	// FsTypeFromDevice attempts to determine the filesystem type of a block device by reading its magic numbers.
	FsTypeFromDevice(devicePath string) (string, errors.E)

	// GetAdapter returns the filesystem adapter for a given filesystem type.
	// This method should only be used internally by the filesystem service.
	// API handlers should use higher-level methods instead.
	GetAdapter(fsType string) (filesystem.FilesystemAdapter, errors.E)

	// GetSupportedFilesystems returns information about all supported filesystems.
	GetSupportedFilesystems(ctx context.Context) (map[string]dto.FilesystemSupport, errors.E)

	// ListSupportedTypes returns a list of all supported filesystem type names.
	ListSupportedTypes() []string

	// GetSupportAndInfo returns filesystem support information along with name and description.
	// This is the preferred method for API handlers to get filesystem information.
	GetSupportAndInfo(ctx context.Context, fsType string) (*dto.FilesystemInfo, errors.E)

	// FormatPartition formats a device with the specified filesystem type.
	// Returns an error if formatting cannot start, is already in progress, or fails.
	FormatPartition(ctx context.Context, devicePath, fsType string, options dto.FormatOptions) (*dto.CheckResult, errors.E)

	// CheckPartition checks a device's filesystem for errors.
	// Returns an error if check cannot start, is already in progress, or fails.
	CheckPartition(ctx context.Context, devicePath, fsType string, options dto.CheckOptions) (*dto.CheckResult, errors.E)

	// AbortCheckPartition cancels a running filesystem check for the given device path.
	AbortCheckPartition(ctx context.Context, devicePath string) errors.E

	// GetPartitionState returns the state of a partition's filesystem.
	GetPartitionState(ctx context.Context, devicePath, fsType string) (*dto.FilesystemState, errors.E)

	// GetPartitionLabel returns the label of a partition's filesystem.
	GetPartitionLabel(ctx context.Context, devicePath, fsType string) (string, errors.E)

	// SetPartitionLabel sets the label of a partition's filesystem.
	SetPartitionLabel(ctx context.Context, devicePath, fsType, label string) errors.E

	// MountPartition mounts a source to target by delegating mount mechanics to the filesystem adapter.
	MountPartition(ctx context.Context, source, target, fsType, data string, flags uintptr, prepareTarget func() error) (*mount.MountPoint, errors.E)

	// UnmountPartition unmounts a target by delegating unmount mechanics to the filesystem adapter.
	UnmountPartition(ctx context.Context, target, fsType string, force, lazy bool) errors.E

	// CreateBlockDevice creates a loop block device node using mknod.
	CreateBlockDevice(ctx context.Context, device string) errors.E
}

// FilesystemService implements the FilesystemServiceInterface.
type FilesystemService struct {
	// standardMountFlags holds common, filesystem-agnostic mount flags.
	standardMountFlags []dto.MountFlag
	// fsSpecificMountFlags maps filesystem types to their specific mount flags.
	fsSpecificMountFlags map[string][]dto.MountFlag

	// Precomputed lookup maps for efficiency
	// standardMountFlagsByName maps lowercase standard flag names to their MountFlag struct.
	standardMountFlagsByName map[string]dto.MountFlag
	// allKnownMountFlagsByName maps all lowercase known flag names (standard and specific)
	// to their MountFlag struct, used for description lookups. Standard flags take precedence on conflict,
	// then preferred filesystem types (see preferredMountFlagSources) fill in remaining metadata.
	allKnownMountFlagsByName map[string]dto.MountFlag

	// registry holds the filesystem adapter registry
	registry *filesystem.Registry

	// Context and cancellation for async operations
	ctx        context.Context
	cancelFunc context.CancelFunc

	// WaitGroup for tracking async operations
	wg *sync.WaitGroup

	// Mutex for protecting activeOperations map
	mu sync.Mutex

	// activeOperations tracks which devices currently have operations running
	activeOperations map[string]bool

	// activeOperationInfo tracks operation metadata and cancel functions by device
	activeOperationInfo map[string]filesystemOperation

	// eventBus for emitting filesystem operation events
	eventBus events.EventBusInterface
}

type filesystemOperation struct {
	op     string
	cancel context.CancelFunc
}

// Package-level variables for default configurations.
// These are used to initialize a new FilesystemService.
var (
	// defaultStandardMountFlags is the initial list of common, filesystem-agnostic mount flags.
	defaultStandardMountFlags = []dto.MountFlag{
		{Name: "ro", Description: "Mount read-only"},
		{Name: "rw", Description: "Mount read-write (default)"},
		{Name: "sync", Description: "All I/O to the filesystem should be done synchronously."},
		{Name: "async", Description: "All I/O to the filesystem should be done asynchronously."},
		{Name: "atime", Description: "Do not use noatime feature (default)"},
		{Name: "noatime", Description: "Do not update inode access times on this filesystem."},
		{Name: "diratime", Description: "Update directory inode access times on this filesystem."},
		{Name: "nodiratime", Description: "Do not update directory inode access times on this filesystem."},
		{Name: "dev", Description: "Interpret character or block special devices on the filesystem."},
		{Name: "nodev", Description: "Do not interpret character or block special devices on the filesystem."},
		{Name: "exec", Description: "Permit execution of binaries."},
		{Name: "noexec", Description: "Do not permit execution of binaries."},
		{Name: "suid", Description: "Permit set-user-id or set-group-id bits to take effect."},
		{Name: "nosuid", Description: "Do not permit set-user-id or set-group-id bits to take effect."},
		{Name: "remount", Description: "Attempt to remount an already-mounted filesystem."},
		{Name: "defaults", Description: "Use default options: rw, suid, dev, exec, auto, nouser, async."},
		{Name: "relatime", Description: "Update inode access times relative to modify or change time."},
	}

	// preferredMountFlagSources defines the filesystem types whose mount-flag metadata should
	// be preferred when multiple adapters define the same option (e.g., uid, gid).
	preferredMountFlagSources = []string{"ntfs", "vfat"}

	// syscallFlagMap maps mount flag names (lowercase) to their corresponding syscall constants.
	// Shared with dto.MountFlagsMap() to keep a single source of truth.
	syscallFlagMap = dto.MountFlagsMap()

	// ignoredSyscallFlags are descriptive, represent default states, or are handled by other mechanisms
	// (like the data field of mount) when converting to syscall flags. These will be ignored without warning.
	ignoredSyscallFlags = map[string]bool{
		"rw":       true,
		"async":    true,
		"atime":    true,
		"diratime": true,
		"dev":      true,
		"exec":     true,
		"suid":     true,
		"defaults": true,
		"auto":     true, // mount(8) option, not direct syscall flag
		"nouser":   true, // mount(8) option
		"user":     true, // mount(8) option
		"_netdev":  true, // mount(8) option
		"nofail":   true, // mount(8) option
	}
)

func orderedFilesystemTypesForMountFlags(registry *filesystem.Registry) []string {
	types := registry.ListSupportedTypes()
	if len(types) == 0 {
		return types
	}
	sort.Strings(types)

	ordered := make([]string, 0, len(types))
	seen := make(map[string]bool, len(types))
	available := make(map[string]bool, len(types))
	for _, fsType := range types {
		available[fsType] = true
	}
	for _, preferred := range preferredMountFlagSources {
		if available[preferred] {
			ordered = append(ordered, preferred)
			seen[preferred] = true
		}
	}
	for _, fsType := range types {
		if !seen[fsType] {
			ordered = append(ordered, fsType)
		}
	}

	return ordered
}

// NewFilesystemService creates and initializes a new FilesystemService.
func NewFilesystemService(
	ctx context.Context,
	cancelFunc context.CancelFunc,
	eventBus events.EventBusInterface,
) FilesystemServiceInterface {
	// Initialize precomputed maps for efficient lookups
	stdFlagsByName := make(map[string]dto.MountFlag, len(defaultStandardMountFlags))
	for _, f := range defaultStandardMountFlags {
		stdFlagsByName[strings.ToLower(f.Name)] = f
	}

	// Create filesystem adapter registry
	registry := filesystem.NewRegistry()

	// Build fsSpecificMountFlags from adapters using a deterministic order
	orderedTypes := orderedFilesystemTypesForMountFlags(registry)
	fsSpecificMountFlags := make(map[string][]dto.MountFlag, len(orderedTypes))
	for _, fsType := range orderedTypes {
		adapter, err := registry.Get(fsType)
		if err != nil {
			continue
		}
		fsSpecificMountFlags[fsType] = adapter.GetMountFlags()
	}

	// Build allKnownFlagsByName from standard flags and adapter flags
	allKnownFlagsByName := make(map[string]dto.MountFlag, len(defaultStandardMountFlags))
	for k, v := range stdFlagsByName {
		allKnownFlagsByName[k] = v
	}
	for _, fsType := range orderedTypes {
		fsFlags := fsSpecificMountFlags[fsType]
		for _, f := range fsFlags {
			lowerName := strings.ToLower(f.Name)
			// Standard flags take precedence for descriptions if names collide.
			if _, exists := allKnownFlagsByName[lowerName]; !exists {
				allKnownFlagsByName[lowerName] = f
			}
		}
	}

	// Get WaitGroup from context
	wg, ok := ctx.Value("wg").(*sync.WaitGroup)
	if !ok {
		// Fallback to a new WaitGroup if not provided in context
		wg = &sync.WaitGroup{}
	}

	p := &FilesystemService{
		standardMountFlags:   defaultStandardMountFlags,
		fsSpecificMountFlags: fsSpecificMountFlags,

		standardMountFlagsByName: stdFlagsByName,
		allKnownMountFlagsByName: allKnownFlagsByName,

		registry: registry,

		ctx:                 ctx,
		cancelFunc:          cancelFunc,
		wg:                  wg,
		activeOperations:    make(map[string]bool),
		activeOperationInfo: make(map[string]filesystemOperation),
		eventBus:            eventBus,
	}
	return p
}

func (s *FilesystemService) startOperation(devicePath, op string) (context.Context, context.CancelFunc, errors.E) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeOperations[devicePath] {
		return nil, nil, errors.WithDetails(dto.ErrorConflict, "Message", fmt.Sprintf("%s operation already in progress", op), "Device", devicePath)
	}
	opCtx, opCancel := context.WithCancel(s.ctx)
	s.activeOperations[devicePath] = true
	s.activeOperationInfo[devicePath] = filesystemOperation{op: op, cancel: opCancel}
	return opCtx, opCancel, nil
}

func (s *FilesystemService) finishOperation(devicePath string) {
	s.mu.Lock()
	delete(s.activeOperations, devicePath)
	delete(s.activeOperationInfo, devicePath)
	s.mu.Unlock()
}

// GetStandardMountFlags returns the list of standard mount flags.
func (s *FilesystemService) GetStandardMountFlags() ([]dto.MountFlag, errors.E) {
	var filteredFlags []dto.MountFlag
	for _, flag := range s.standardMountFlags {
		// Check if the lowercase version of the flag name is in the ignoredSyscallFlags map
		if _, isIgnored := ignoredSyscallFlags[strings.ToLower(flag.Name)]; !isIgnored {
			filteredFlags = append(filteredFlags, flag)
		}
	}
	return filteredFlags, nil
}

// GetFilesystemSpecificMountFlags returns the list of mount flags specific to the given filesystem type.
func (s *FilesystemService) GetFilesystemSpecificMountFlags(fsType string) ([]dto.MountFlag, errors.E) {
	flags, ok := s.fsSpecificMountFlags[fsType]
	if !ok {
		// Return an empty list if no specific flags are defined for this type
		return []dto.MountFlag{}, nil
	}
	return flags, nil
}

// ResolveLinuxFsModule returns the Linux filesystem module/fstype name for mounting.
// Falls back to the provided filesystem type when no adapter is found.
func (s *FilesystemService) ResolveLinuxFsModule(fsType string) string {
	if fsType == "" {
		return ""
	}

	adapter, err := s.registry.Get(fsType)
	if err != nil {
		slog.Debug("ResolveLinuxFsModule: adapter not found, using filesystem type", "fsType", fsType, "error", err)
		return fsType
	}

	module := adapter.GetLinuxFsModule()
	if module == "" {
		return fsType
	}

	return module
}

// GetMountFlagsAndData converts a list of MountFlag structs into the syscall flags (uintptr)
// and the data string (string) for the syscall.Mount function.
// It processes flags that correspond to standard mount(2) bitmask options.
// Flags with values (e.g., "uid=<arg>") or flags representing default/permissive states (e.g., "rw", "defaults")
// are typically ignored by this function as they don't directly set a bit in the syscall flags parameter
// or are handled by the absence of restrictive flags.
func (s *FilesystemService) MountFlagsToSyscallFlagAndData(inputFlags []dto.MountFlag) (uintptr, string, errors.E) {
	var syscallFlagValue uintptr = 0
	var dataFlags []string

	for _, mf := range inputFlags {
		rawFlagName := strings.TrimSpace(mf.Name)
		lowerFlagName := strings.ToLower(rawFlagName) // Use lowercase for map lookups

		// ---
		// New Validation Check
		// ---
		if !mf.NeedsValue && mf.FlagValue != "" {
			return 0, "", errors.WithDetails(dto.ErrorInvalidParameter,
				"Flag", mf.Name,
				"Value", mf.FlagValue,
				"Message", "Boolean/switch flag was provided with a value")
		}

		// If a Value is provided in the struct, use it regardless of the Name format
		if mf.FlagValue != "" {
			// Validate the value if a regex is provided
			if mf.NeedsValue && mf.ValueValidationRegex != "" {
				compiledRegex, err := regexp.Compile(mf.ValueValidationRegex)
				if err != nil {
					// This is a configuration error in the predefined regex
					slog.Error("Invalid validation regex configured for flag", "flag", mf.Name, "regex", mf.ValueValidationRegex, "error", err)
					// Potentially return an internal server error, or log and proceed without validation for this flag
					// For now, let's return an error to make it explicit
					return 0, "", errors.WithDetails(err, "Message", "Invalid validation regex", "flag", mf.Name)
				}
				if !compiledRegex.MatchString(mf.FlagValue) {
					return 0, "", errors.WithDetails(dto.ErrorInvalidParameter,
						"Flag", mf.Name, "Value", mf.FlagValue,
						"Message", fmt.Sprintf("Value for flag '%s' does not match expected format. %s", mf.Name, mf.ValueDescription))
				}
			}
			formattedFlag := fmt.Sprintf("%s=%s", rawFlagName, mf.FlagValue)
			slog.Debug("MountFlagsToSyscallFlagAndData: Collecting data flag with explicit value", "flag", formattedFlag)
			dataFlags = append(dataFlags, formattedFlag)
			continue
		}

		if val, ok := syscallFlagMap[lowerFlagName]; ok {
			slog.Debug("MountFlagsToSyscallFlagAndData: Adding syscall flag to bitmask", "flag", rawFlagName, "value", val)
			syscallFlagValue |= val
		} else if ignoredSyscallFlags[lowerFlagName] {
			slog.Debug("MountFlagsToSyscallFlagAndData: Ignoring known descriptive/default flag", "flag", rawFlagName)
		} else if rawFlagName != "" {
			slog.Warn("MountFlagsToSyscallFlagAndData: Unknown or unhandled mount flag for bitmask generation", "flag", mf.Name)
		}
	}

	// Join the collected data flags into a single string
	dataString := strings.Join(dataFlags, ",")

	return syscallFlagValue, dataString, nil
}

// SyscallFlagToMountFlag converts a syscall flag bitmask (uintptr) back into a slice of dto.MountFlag.
func (s *FilesystemService) SyscallFlagToMountFlag(syscallFlag uintptr) ([]dto.MountFlag, errors.E) {
	var result []dto.MountFlag

	// Iterate through the map of known syscall flags.
	// Note: The order of flags in the result slice will depend on map iteration order,
	// which is not guaranteed. If a specific order is needed, consider iterating over a slice.
	for nameInMap, sysVal := range dto.MountFlagsMap() {
		if sysVal == 0 { // Skip zero-value flags if any (e.g. placeholder)
			continue
		}
		if (syscallFlag & sysVal) == sysVal {
			// This flag bit is set.
			mountFlag := dto.MountFlag{
				Name:       nameInMap, // Default to the name from flagMap
				NeedsValue: false,     // Syscall bitmask flags are inherently boolean
			}

			// Try to find more details (like original casing and description)
			if detail, ok := s.standardMountFlagsByName[strings.ToLower(nameInMap)]; ok {
				mountFlag.Name = detail.Name // Use original casing
				mountFlag.Description = detail.Description
			} else {
				slog.Debug("SyscallFlagToMountFlag: No detailed description in standardMountFlags for syscall flag", "flagName", nameInMap)
			}
			result = append(result, mountFlag)
		}
	}
	return result, nil
}

// SyscallDataToMountFlag converts a mount data string (e.g., "uid=1000,gid=1000")
// back into a slice of dto.MountFlag.
func (s *FilesystemService) SyscallDataToMountFlag(data string) ([]dto.MountFlag, errors.E) {
	var result []dto.MountFlag
	if data == "" {
		return result, nil
	}

	options := strings.Split(data, ",")

	for _, opt := range options {
		opt = strings.TrimSpace(opt)
		if opt == "" {
			continue
		}

		parts := strings.SplitN(opt, "=", 2)
		name := strings.TrimSpace(parts[0])
		mountFlag := dto.MountFlag{Name: name}

		if len(parts) == 2 {
			mountFlag.FlagValue = strings.TrimSpace(parts[1])
			mountFlag.NeedsValue = true
		} else {
			mountFlag.NeedsValue = false // Standalone option in data string
		}

		if descFlag, ok := s.allKnownMountFlagsByName[strings.ToLower(name)]; ok {
			mountFlag.Description = descFlag.Description
			// If the flag from allKnownMountFlagsByName indicates it needs a value, copy its description and regex
			mountFlag.ValueDescription = descFlag.ValueDescription
			mountFlag.ValueValidationRegex = descFlag.ValueValidationRegex
		} else {
			slog.Debug("SyscallDataToMountFlag: No description found for data flag", "flagName", name)
		}
		result = append(result, mountFlag)
	}

	return result, nil
}

// FsTypeFromDevice attempts to determine the filesystem type of a block device by reading its magic numbers.
func (s *FilesystemService) FsTypeFromDevice(devicePath string) (string, errors.E) {
	// Get all adapters for detection
	adapters := s.registry.GetAll()

	for _, adapter := range adapters {
		matches, err := adapter.IsDeviceSupported(s.ctx, devicePath)
		if err != nil {
			slog.Warn("Error checking device against adapter signatures", "device", devicePath, "adapter", adapter.GetName(), "error", err)
			continue
		}
		if matches {
			slog.Debug("FsTypeFromDevice: Matched signature with adapter's IsDeviceSupported method", "device", devicePath, "adapter", adapter.GetName())
			return adapter.GetName(), nil
		}
	}

	return "", errors.WithDetails(dto.ErrorUnknownFilesystem, "Path", devicePath)
}

// GetAdapter returns the filesystem adapter for a given filesystem type.
// This method should only be used internally by the filesystem service.
// API handlers should use higher-level methods instead.
func (s *FilesystemService) GetAdapter(fsType string) (filesystem.FilesystemAdapter, errors.E) {
	return s.registry.Get(fsType)
}

// GetSupportedFilesystems returns information about all supported filesystems
func (s *FilesystemService) GetSupportedFilesystems(ctx context.Context) (map[string]dto.FilesystemSupport, errors.E) {
	return s.registry.GetSupportedFilesystems(ctx)
}

// ListSupportedTypes returns a list of all supported filesystem type names
func (s *FilesystemService) ListSupportedTypes() []string {
	return s.registry.ListSupportedTypes()
}

// GetSupportAndInfo returns filesystem support information along with name and description
func (s *FilesystemService) GetSupportAndInfo(ctx context.Context, fsType string) (*dto.FilesystemInfo, errors.E) {
	adapter, err := s.registry.Get(fsType)
	if err != nil {
		return nil, err
	}

	support, err := adapter.IsSupported(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check filesystem support")
	}

	standardFlags, _ := s.GetStandardMountFlags()
	customFlags, _ := s.GetFilesystemSpecificMountFlags(fsType)

	return &dto.FilesystemInfo{
		Name:             adapter.GetName(),
		Type:             fsType,
		Description:      adapter.GetDescription(),
		MountFlags:       standardFlags,
		CustomMountFlags: customFlags,
		Support:          &support,
	}, nil
}

// FormatPartition formats a device with the specified filesystem type
// This operation executes asynchronously. It returns immediately after starting the operation
// and emits events.FilesystemTaskEvent for start, success, and failure states.
// Returns an error if the operation cannot be started or if another operation is already
// running on the same device.
func (s *FilesystemService) FormatPartition(ctx context.Context, devicePath, fsType string, options dto.FormatOptions) (*dto.CheckResult, errors.E) {
	adapter, err := s.registry.Get(fsType)
	if err != nil {
		return nil, errors.Wrap(err, "unsupported filesystem type")
	}

	// Check if formatting is supported
	support, err := adapter.IsSupported(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check filesystem support")
	}

	if !support.CanFormat {
		msg := fmt.Sprintf("Filesystem %s cannot be formatted on this system", fsType)
		if len(support.MissingTools) > 0 {
			msg += fmt.Sprintf(". Missing tools: %v. Install package: %s",
				support.MissingTools, support.AlpinePackage)
		}
		return nil, errors.New(msg)
	}

	opCtx, _, opErr := s.startOperation(devicePath, "format")
	if opErr != nil {
		return nil, opErr
	}

	// Start async operation
	s.wg.Go(func() {
		defer s.finishOperation(devicePath)

		// Emit start event
		if s.eventBus != nil {
			s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
				Event: events.Event{Type: events.EventTypes.START},
				Task: &dto.FilesystemTask{
					Device:         devicePath,
					Operation:      "format",
					FilesystemType: fsType,
					Status:         "start",
					Message:        fmt.Sprintf("Starting format operation for %s as %s", devicePath, fsType),
				},
			})
		}

		// Log start of operation
		slog.InfoContext(s.ctx, "Starting format operation", "device", devicePath, "fsType", fsType)

		// Create progress callback that emits events
		progressCallback := func(status string, percentual int, notes []string) {
			if s.eventBus != nil {
				message := fmt.Sprintf("Format %s: %s", devicePath, status)
				if len(notes) > 0 {
					message += " - " + strings.Join(notes, ", ")
				}

				var eventType events.EventType
				switch status {
				case "start":
					eventType = events.EventTypes.START
				case "success":
					eventType = events.EventTypes.STOP
				case "failure":
					eventType = events.EventTypes.ERROR
				default:
					eventType = events.EventTypes.START // running state
				}

				s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
					Event: events.Event{Type: eventType},
					Task: &dto.FilesystemTask{
						Device:         devicePath,
						Operation:      "format",
						FilesystemType: fsType,
						Status:         status,
						Message:        message,
						Progress:       percentual,
						Notes:          notes,
					},
				})
			}
		}

		// Perform the format operation with progress callback
		formatErr := adapter.Format(opCtx, devicePath, options, progressCallback)

		if formatErr != nil {
			// Emit failure event
			if s.eventBus != nil {
				s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
					Event: events.Event{Type: events.EventTypes.ERROR},
					Task: &dto.FilesystemTask{
						Device:         devicePath,
						Operation:      "format",
						FilesystemType: fsType,
						Status:         "failure",
						Message:        fmt.Sprintf("Format operation failed for %s", devicePath),
						Error:          formatErr.Error(),
					},
				})
			}
			// Log failure
			slog.ErrorContext(s.ctx, "Format operation failed", "device", devicePath, "fsType", fsType, "error", formatErr)
		} else {
			// Emit success event
			if s.eventBus != nil {
				s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
					Event: events.Event{Type: events.EventTypes.STOP},
					Task: &dto.FilesystemTask{
						Device:         devicePath,
						Operation:      "format",
						FilesystemType: fsType,
						Status:         "success",
						Message:        fmt.Sprintf("Format operation completed successfully for %s", devicePath),
					},
				})
			}
			// Log success
			slog.InfoContext(s.ctx, "Format operation completed successfully", "device", devicePath, "fsType", fsType)
		}
	})

	return &dto.CheckResult{
		Success: true,
		Message: fmt.Sprintf("Format operation started for %s as %s", devicePath, fsType),
	}, nil
}

// CheckPartition checks a device's filesystem for errors
// This operation executes asynchronously. It returns immediately after starting the operation
// and emits events.FilesystemTaskEvent for start, success, and failure states.
// Returns an error if the operation cannot be started or if another operation is already
// running on the same device.
func (s *FilesystemService) CheckPartition(ctx context.Context, devicePath, fsType string, options dto.CheckOptions) (*dto.CheckResult, errors.E) {
	adapter, err := s.registry.Get(fsType)
	if err != nil {
		return nil, errors.Wrap(err, "unsupported filesystem type")
	}

	// Check if checking is supported
	support, err := adapter.IsSupported(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check filesystem support")
	}

	if !support.CanCheck {
		msg := fmt.Sprintf("Filesystem %s cannot be checked on this system", fsType)
		if len(support.MissingTools) > 0 {
			msg += fmt.Sprintf(". Missing tools: %v. Install package: %s",
				support.MissingTools, support.AlpinePackage)
		}
		return nil, errors.New(msg)
	}

	opCtx, _, opErr := s.startOperation(devicePath, "check")
	if opErr != nil {
		return nil, opErr
	}

	// Start async operation
	s.wg.Go(func() {
		defer s.finishOperation(devicePath)

		// Emit start event
		if s.eventBus != nil {
			s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
				Event: events.Event{Type: events.EventTypes.START},
				Task: &dto.FilesystemTask{
					Device:         devicePath,
					Operation:      "check",
					FilesystemType: fsType,
					Status:         "start",
					Message:        fmt.Sprintf("Starting check operation for %s", devicePath),
				},
			})
		}

		// Log start of operation
		slog.InfoContext(s.ctx, "Starting check operation", "device", devicePath, "fsType", fsType)

		// Create progress callback that emits events
		progressCallback := func(status string, percentual int, notes []string) {
			if s.eventBus != nil {
				message := fmt.Sprintf("Check %s: %s", devicePath, status)
				if len(notes) > 0 {
					message += " - " + strings.Join(notes, ", ")
				}

				var eventType events.EventType
				switch status {
				case "start":
					eventType = events.EventTypes.START
				case "success":
					eventType = events.EventTypes.STOP
				case "failure":
					eventType = events.EventTypes.ERROR
				default:
					eventType = events.EventTypes.START // running state
				}

				s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
					Event: events.Event{Type: eventType},
					Task: &dto.FilesystemTask{
						Device:         devicePath,
						Operation:      "check",
						FilesystemType: fsType,
						Status:         status,
						Message:        message,
						Progress:       percentual,
						Notes:          notes,
					},
				})
			}
		}

		// Perform the check operation with progress callback
		result, checkErr := adapter.Check(opCtx, devicePath, options, progressCallback)

		if checkErr != nil {
			if errors.Is(checkErr, context.Canceled) || opCtx.Err() == context.Canceled {
				if s.eventBus != nil {
					s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
						Event: events.Event{Type: events.EventTypes.STOP},
						Task: &dto.FilesystemTask{
							Device:         devicePath,
							Operation:      "check",
							FilesystemType: fsType,
							Status:         "canceled",
							Message:        fmt.Sprintf("Check operation canceled for %s", devicePath),
						},
					})
				}
				slog.InfoContext(s.ctx, "Check operation canceled", "device", devicePath, "fsType", fsType)
				return
			}
			// Emit failure event
			if s.eventBus != nil {
				s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
					Event: events.Event{Type: events.EventTypes.ERROR},
					Task: &dto.FilesystemTask{
						Device:         devicePath,
						Operation:      "check",
						FilesystemType: fsType,
						Status:         "failure",
						Message:        fmt.Sprintf("Check operation failed for %s", devicePath),
						Error:          checkErr.Error(),
					},
				})
			}
			// Log failure
			slog.ErrorContext(s.ctx, "Check operation failed", "device", devicePath, "fsType", fsType, "error", checkErr)
		} else {
			// Emit success event
			if s.eventBus != nil {
				s.eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{
					Event: events.Event{Type: events.EventTypes.STOP},
					Task: &dto.FilesystemTask{
						Device:         devicePath,
						Operation:      "check",
						FilesystemType: fsType,
						Status:         "success",
						Message:        fmt.Sprintf("Check operation completed successfully for %s", devicePath),
					},
				})
			}
			// Log success
			slog.InfoContext(s.ctx, "Check operation completed successfully", "device", devicePath, "fsType", fsType, "result", result)
		}
	})

	return &dto.CheckResult{
		Success: true,
		Message: fmt.Sprintf("Check operation started for %s", devicePath),
	}, nil
}

func (s *FilesystemService) AbortCheckPartition(ctx context.Context, devicePath string) errors.E {
	s.mu.Lock()
	opInfo, ok := s.activeOperationInfo[devicePath]
	if !ok || !s.activeOperations[devicePath] {
		s.mu.Unlock()
		return errors.WithDetails(dto.ErrorNotFound, "Message", "check operation not running", "Device", devicePath)
	}
	if opInfo.op != "check" {
		s.mu.Unlock()
		return errors.WithDetails(dto.ErrorConflict, "Message", "operation running is not a check", "Device", devicePath, "Operation", opInfo.op)
	}
	cancel := opInfo.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// GetPartitionState returns the state of a partition's filesystem
func (s *FilesystemService) GetPartitionState(ctx context.Context, devicePath, fsType string) (*dto.FilesystemState, errors.E) {
	adapter, err := s.registry.Get(fsType)
	if err != nil {
		return nil, errors.Wrap(dto.ErrorUnsupportedFilesystem, "unsupported filesystem type")
	}

	state, stateErr := adapter.GetState(ctx, devicePath)
	if stateErr != nil {
		return nil, errors.Wrap(stateErr, "failed to get partition state")
	}

	return &state, nil
}

// GetPartitionLabel returns the label of a partition's filesystem
func (s *FilesystemService) GetPartitionLabel(ctx context.Context, devicePath, fsType string) (string, errors.E) {
	adapter, err := s.registry.Get(fsType)
	if err != nil {
		return "", errors.Wrap(dto.ErrorUnsupportedFilesystem, "unsupported filesystem type")
	}

	label, labelErr := adapter.GetLabel(ctx, devicePath)
	if labelErr != nil {
		return "", errors.Wrap(labelErr, "failed to get partition label")
	}

	return label, nil
}

// SetPartitionLabel sets the label of a partition's filesystem
func (s *FilesystemService) SetPartitionLabel(ctx context.Context, devicePath, fsType, label string) errors.E {
	adapter, err := s.registry.Get(fsType)
	if err != nil {
		return errors.Wrap(dto.ErrorUnsupportedFilesystem, "unsupported filesystem type")
	}

	// Check if setting label is supported
	support, err := adapter.IsSupported(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check filesystem support")
	}

	if !support.CanSetLabel {
		msg := fmt.Sprintf("Filesystem %s cannot set labels on this system", fsType)
		if len(support.MissingTools) > 0 {
			msg += fmt.Sprintf(". Missing tools: %v. Install package: %s",
				support.MissingTools, support.AlpinePackage)
		}
		return errors.New(msg)
	}

	setErr := adapter.SetLabel(ctx, devicePath, label)
	if setErr != nil {
		return errors.Wrap(setErr, "failed to set partition label")
	}

	return nil
}

func (s *FilesystemService) adapterForMountOps(fsType string) (filesystem.FilesystemAdapter, string) {
	if strings.TrimSpace(fsType) == "" {
		return filesystem.NewExt4Adapter(), ""
	}

	adapter, err := s.registry.Get(fsType)
	if err != nil {
		return filesystem.NewExt4Adapter(), fsType
	}

	mountFsType := adapter.GetLinuxFsModule()
	if mountFsType == "" {
		mountFsType = fsType
	}

	return adapter, mountFsType
}

// MountPartition mounts a source to target by delegating mount mechanics to the filesystem adapter.
func (s *FilesystemService) MountPartition(
	ctx context.Context,
	source, target, fsType, data string,
	flags uintptr,
	prepareTarget func() error,
) (*mount.MountPoint, errors.E) {
	adapter, mountFsType := s.adapterForMountOps(fsType)
	return adapter.Mount(ctx, source, target, mountFsType, data, flags, prepareTarget)
}

// UnmountPartition unmounts a target by delegating unmount mechanics to the filesystem adapter.
func (s *FilesystemService) UnmountPartition(ctx context.Context, target, fsType string, force, lazy bool) errors.E {
	adapter, _ := s.adapterForMountOps(fsType)
	return adapter.Unmount(ctx, target, force, lazy)
}

// CreateBlockDevice creates a loop block device node using mknod.
// Returns nil if the device already exists.
func (s *FilesystemService) CreateBlockDevice(ctx context.Context, device string) errors.E {
	if err := osutil.CreateBlockDevice(ctx, device); err != nil {
		return errors.Wrap(err, "failed to create block device")
	}
	return nil
}
