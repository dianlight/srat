package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"syscall"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service/filesystem"
	"gitlab.com/tozd/go/errors"
)

// FilesystemServiceInterface defines the methods for managing filesystem types and mount flags.
type FilesystemServiceInterface interface {
	// GetStandardMountFlags returns a list of common, filesystem-agnostic mount flags.
	GetStandardMountFlags() ([]dto.MountFlag, errors.E)

	// GetFilesystemSpecificMountFlags returns a list of mount flags specific to a given filesystem type.
	// Returns an empty list if the filesystem type is not recognized or has no specific flags.
	GetFilesystemSpecificMountFlags(fsType string) ([]dto.MountFlag, errors.E)

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
	GetSupportAndInfo(ctx context.Context, fsType string) (*FilesystemInfo, errors.E)

	// FormatPartition formats a device with the specified filesystem type.
	// Returns an error if formatting cannot start, is already in progress, or fails.
	FormatPartition(ctx context.Context, devicePath, fsType string, options dto.FormatOptions) (*dto.CheckResult, errors.E)

	// CheckPartition checks a device's filesystem for errors.
	// Returns an error if check cannot start, is already in progress, or fails.
	CheckPartition(ctx context.Context, devicePath, fsType string, options dto.CheckOptions) (*dto.CheckResult, errors.E)

	// GetPartitionState returns the state of a partition's filesystem.
	GetPartitionState(ctx context.Context, devicePath, fsType string) (*dto.FilesystemState, errors.E)

	// GetPartitionLabel returns the label of a partition's filesystem.
	GetPartitionLabel(ctx context.Context, devicePath, fsType string) (string, errors.E)

	// SetPartitionLabel sets the label of a partition's filesystem.
	SetPartitionLabel(ctx context.Context, devicePath, fsType, label string) errors.E
}

// FilesystemInfo combines filesystem type information with capability details
type FilesystemInfo struct {
	// Name is the filesystem type name
	Name string

	// Type is the filesystem type (same as name for consistency)
	Type string

	// Description provides a human-readable description of the filesystem
	Description string

	// MountFlags contains standard mount flags
	MountFlags []dto.MountFlag

	// CustomMountFlags contains filesystem-specific mount flags
	CustomMountFlags []dto.MountFlag

	// Support contains filesystem capability information
	Support *dto.FilesystemSupport
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
	// to their MountFlag struct, used for description lookups. Standard flags take precedence on conflict.
	allKnownMountFlagsByName map[string]dto.MountFlag

	// registry holds the filesystem adapter registry
	registry *filesystem.Registry
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

	// syscallFlagMap maps mount flag names (lowercase) to their corresponding syscall constants.
	// This map includes flags that SET a bit. Flags like "rw" or "async"
	// represent the ABSENCE of a restrictive bit and are handled by not setting MS_RDONLY or MS_SYNCHRONOUS.
	// "defaults" is also handled by the base state (0) and subsequent overrides.
	syscallFlagMap = map[string]uintptr{
		"ro":          syscall.MS_RDONLY,
		"nosuid":      syscall.MS_NOSUID,
		"nodev":       syscall.MS_NODEV,
		"noexec":      syscall.MS_NOEXEC,
		"sync":        syscall.MS_SYNCHRONOUS,
		"remount":     syscall.MS_REMOUNT,
		"mand":        syscall.MS_MANDLOCK,
		"dirsync":     syscall.MS_DIRSYNC,
		"noatime":     syscall.MS_NOATIME,
		"nodiratime":  syscall.MS_NODIRATIME,
		"bind":        syscall.MS_BIND,
		"rec":         syscall.MS_REC, // Used with MS_BIND for recursive bind mounts (rbind)
		"silent":      syscall.MS_SILENT,
		"posixacl":    syscall.MS_POSIXACL,
		"acl":         syscall.MS_POSIXACL, // Common alias for posixacl
		"unbindable":  syscall.MS_UNBINDABLE,
		"private":     syscall.MS_PRIVATE,
		"slave":       syscall.MS_SLAVE,
		"shared":      syscall.MS_SHARED,
		"relatime":    syscall.MS_RELATIME,
		"strictatime": syscall.MS_STRICTATIME,
		// "lazytime":    syscall.MS_LAZYTIME, // Not universally available, explicitly not mapped
	}

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

// NewFilesystemService creates and initializes a new FilesystemService.
func NewFilesystemService(ctx context.Context) FilesystemServiceInterface {
	// Initialize precomputed maps for efficient lookups
	stdFlagsByName := make(map[string]dto.MountFlag, len(defaultStandardMountFlags))
	for _, f := range defaultStandardMountFlags {
		stdFlagsByName[strings.ToLower(f.Name)] = f
	}

	// Create filesystem adapter registry
	registry := filesystem.NewRegistry()

	// Build fsSpecificMountFlags from adapters
	fsSpecificMountFlags := make(map[string][]dto.MountFlag)
	for _, adapter := range registry.GetAll() {
		fsType := adapter.GetName()
		fsSpecificMountFlags[fsType] = adapter.GetMountFlags()
	}

	// Build allKnownFlagsByName from standard flags and adapter flags
	allKnownFlagsByName := make(map[string]dto.MountFlag, len(defaultStandardMountFlags))
	for k, v := range stdFlagsByName {
		allKnownFlagsByName[k] = v
	}
	for _, fsFlags := range fsSpecificMountFlags {
		for _, f := range fsFlags {
			lowerName := strings.ToLower(f.Name)
			// Standard flags take precedence for descriptions if names collide.
			if _, exists := allKnownFlagsByName[lowerName]; !exists {
				allKnownFlagsByName[lowerName] = f
			}
		}
	}

	p := &FilesystemService{
		standardMountFlags:   defaultStandardMountFlags,
		fsSpecificMountFlags: fsSpecificMountFlags,

		standardMountFlagsByName: stdFlagsByName,
		allKnownMountFlagsByName: allKnownFlagsByName,

		registry: registry,
	}
	return p
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
	
	fsType, err := filesystem.DetectFilesystemType(devicePath, adapters)
	if err != nil {
		// Map the error to the appropriate DTO error type for backward compatibility
		if errors.Is(err, filesystem.ErrorDeviceNotFound) {
			return "", errors.WithDetails(dto.ErrorDeviceNotFound, "Path", devicePath)
		}
		return "", err
	}
	
	if fsType == "" {
		return "", errors.WithDetails(filesystem.ErrorUnknownFilesystem, "Path", devicePath)
	}
	
	slog.Debug("FsTypeFromDevice: Matched signature", "device", devicePath, "fstype", fsType)
	return fsType, nil
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
func (s *FilesystemService) GetSupportAndInfo(ctx context.Context, fsType string) (*FilesystemInfo, errors.E) {
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

	return &FilesystemInfo{
		Name:             adapter.GetName(),
		Type:             fsType,
		Description:      adapter.GetDescription(),
		MountFlags:       standardFlags,
		CustomMountFlags: customFlags,
		Support:          &support,
	}, nil
}

// FormatPartition formats a device with the specified filesystem type
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

	// Format the partition
	formatErr := adapter.Format(ctx, devicePath, options)
	if formatErr != nil {
		return nil, errors.Wrap(formatErr, "failed to format partition")
	}

	return &dto.CheckResult{
		Success:  true,
		Message:  fmt.Sprintf("Successfully formatted %s as %s", devicePath, fsType),
		ExitCode: 0,
	}, nil
}

// CheckPartition checks a device's filesystem for errors
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

	// Check the filesystem
	result, checkErr := adapter.Check(ctx, devicePath, options)
	if checkErr != nil {
		return nil, errors.Wrap(checkErr, "failed to check partition")
	}

	return &result, nil
}

// GetPartitionState returns the state of a partition's filesystem
func (s *FilesystemService) GetPartitionState(ctx context.Context, devicePath, fsType string) (*dto.FilesystemState, errors.E) {
	adapter, err := s.registry.Get(fsType)
	if err != nil {
		return nil, errors.Wrap(err, "unsupported filesystem type")
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
		return "", errors.Wrap(err, "unsupported filesystem type")
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
		return errors.Wrap(err, "unsupported filesystem type")
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

