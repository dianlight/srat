package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
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
	GetAdapter(fsType string) (filesystem.FilesystemAdapter, errors.E)

	// GetSupportedFilesystems returns information about all supported filesystems.
	GetSupportedFilesystems(ctx context.Context) (map[string]filesystem.FilesystemSupport, errors.E)

	// ListSupportedTypes returns a list of all supported filesystem type names.
	ListSupportedTypes() []string
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

	// defaultFsSpecificMountFlags maps filesystem types to their specific mount flags.
	defaultFsSpecificMountFlags = map[string][]dto.MountFlag{
		"ntfs": {
			{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true, ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true, ValueDescription: "Group ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "permissions", Description: "Respect NTFS permissions"},
			{Name: "acl", Description: "Enable POSIX Access Control Lists support"},
			{Name: "exec", Description: "Allow executing files (use with caution)"},
		},
		"xfs": {
			{Name: "inode64", Description: "Enable 64-bit inode allocation for large filesystems"},
			{Name: "noquota", Description: "Disable quota enforcement"},
			{Name: "usrquota", Description: "Enable user quota enforcement"},
			{Name: "grpquota", Description: "Enable group quota enforcement"},
			{Name: "prjquota", Description: "Enable project quota enforcement"},
			{Name: "discard", Description: "Enable discard/TRIM support"},
			{Name: "nouuid", Description: "Ignore filesystem UUID to allow mounting duplicates"},
			{Name: "allocsize", Description: "Set preferred allocation size", NeedsValue: true, ValueDescription: "Size in bytes optionally with K, M, or G suffix (e.g., 1G)", ValueValidationRegex: `^[0-9]+([kKmMgG])?$`},
			{Name: "sunit", Description: "Set stripe unit size (in 512-byte blocks)", NeedsValue: true, ValueDescription: "Stripe unit in 512-byte blocks", ValueValidationRegex: `^[0-9]+$`},
			{Name: "swidth", Description: "Set stripe width size (in 512-byte blocks)", NeedsValue: true, ValueDescription: "Stripe width in 512-byte blocks", ValueValidationRegex: `^[0-9]+$`},
			{Name: "logbufs", Description: "Number of log buffers", NeedsValue: true, ValueDescription: "Integer between 2 and 8", ValueValidationRegex: `^[2-8]$`},
			{Name: "logbsize", Description: "Log buffer size in bytes", NeedsValue: true, ValueDescription: "One of: 16384, 32768, 65536, 131072, 262144", ValueValidationRegex: `^(16384|32768|65536|131072|262144)$`},
		},
		"ntfs3": {
			{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true, ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true, ValueDescription: "Group ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "permissions", Description: "Respect NTFS permissions"},
			{Name: "acl", Description: "Enable POSIX Access Control Lists support"},
			{Name: "force", Description: "Force mount even if the volume is marked dirty"},
			{Name: "norecover", Description: "Do not try to recover a dirty volume (default for ntfs3)"},
			{Name: "iocharset", Description: "I/O character set (e.g., utf8)", NeedsValue: true, ValueDescription: "Character set name (e.g., utf8)", ValueValidationRegex: `^[a-zA-Z0-9_-]+$`},
		},
		"zfs": {
			{Name: "zfsutil", Description: "Indicates that the mount is managed by ZFS utilities"},
			{Name: "noauto", Description: "Can be used to prevent automatic mounting by zfs-mount-generator"},
			{Name: "context", Description: "Set SELinux context for all files/directories", NeedsValue: true, ValueDescription: "SELinux context string", ValueValidationRegex: `^[\w:.-]+$`},
			{Name: "fscontext", Description: "Set SELinux context for the filesystem superblock", NeedsValue: true, ValueDescription: "SELinux context string", ValueValidationRegex: `^[\w:.-]+$`},
		},
		"ext2": {
			{Name: "acl", Description: "Enable POSIX Access Control Lists support"},
			{Name: "user_xattr", Description: "Enable user extended attributes"},
			{Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true, ValueDescription: "One of: continue, remount-ro, panic", ValueValidationRegex: `^(continue|remount-ro|panic)$`},
			{Name: "discard", Description: "Enable discard/TRIM support"},
		},
		"ext3": {
			{Name: "data", Description: "Data journaling mode (ordered, writeback, journal)", NeedsValue: true, ValueDescription: "One of: journal, ordered, writeback", ValueValidationRegex: `^(journal|ordered|writeback)$`},
			{Name: "journal_checksum", Description: "Enable journal checksumming"},
			{Name: "journal_async_commit", Description: "Commit data blocks asynchronously"},
			{Name: "acl", Description: "Enable POSIX Access Control Lists support"},
			{Name: "user_xattr", Description: "Enable user extended attributes"},
			{Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true, ValueDescription: "One of: continue, remount-ro, panic", ValueValidationRegex: `^(continue|remount-ro|panic)$`},
			{Name: "discard", Description: "Enable discard/TRIM support"},
			{Name: "barrier", Description: "Enable/disable write barriers (0, 1)", NeedsValue: true, ValueDescription: "0 or 1", ValueValidationRegex: `^[01]$`},
		},
		"vfat": {
			{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true, ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true, ValueDescription: "Group ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "umask", Description: "Set umask (octal) - overrides fmask/dmask", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "iocharset", Description: "I/O character set (e.g., utf8)", NeedsValue: true, ValueDescription: "Character set name (e.g., utf8)", ValueValidationRegex: `^[a-zA-Z0-9_-]+$`},
			{Name: "codepage", Description: "Codepage for short filenames (e.g., 437)", NeedsValue: true, ValueDescription: "Codepage number (e.g., 437)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "shortname", Description: "Shortname case (lower, win95, winnt, mixed)", NeedsValue: true, ValueDescription: "One of: lower, win95, winnt, mixed", ValueValidationRegex: `^(lower|win95|winnt|mixed)$`},
			{Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true, ValueDescription: "One of: continue, remount-ro, panic", ValueValidationRegex: `^(continue|remount-ro|panic)$`},
		},
		"exfat": {
			{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true, ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true, ValueDescription: "Group ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "umask", Description: "Set umask (octal) for files and directories", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "fmask", Description: "Set file permissions mask (octal, overrides umask for files)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "dmask", Description: "Set directory permissions mask (octal, overrides umask for dirs)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "allow_utime", Description: "Allow non-root users to change file timestamps (requires umask/fmask/dmask to allow)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
			{Name: "iocharset", Description: "I/O character set (e.g., utf8)", NeedsValue: true, ValueDescription: "Character set name (e.g., utf8)", ValueValidationRegex: `^[a-zA-Z0-9_-]+$`},
			{Name: "utf8", Description: "Use UTF-8 for filename encoding (often an alias for iocharset=utf8)"},
			{Name: "codepage", Description: "Codepage for filename encoding (e.g., 437)", NeedsValue: true, ValueDescription: "Codepage number (e.g., 437)", ValueValidationRegex: `^[0-9]+$`},
			{Name: "namecase", Description: "Filename case handling (default: asis, or: lower, upper)", NeedsValue: true, ValueDescription: "One of: asis, lower, upper", ValueValidationRegex: `^(asis|lower|upper)$`},
			{Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true, ValueDescription: "One of: continue, remount-ro, panic", ValueValidationRegex: `^(continue|remount-ro|panic)$`},
			{Name: "discard", Description: "Enable discard/TRIM support"},
			{Name: "keep_last_dots", Description: "Keep trailing dots in filenames"},
			{Name: "sys_tz", Description: "Use system timezone for timestamps instead of UTC"},
			{Name: "time_offset", Description: "Time offset in minutes from UTC for timestamps", NeedsValue: true, ValueDescription: "Offset in minutes (e.g., -60 or 120)", ValueValidationRegex: `^[-+]?[0-9]+$`},
		},
		"ext4": {
			{Name: "data", Description: "Data journaling mode (ordered, writeback, journal)", NeedsValue: true, ValueDescription: "One of: journal, ordered, writeback", ValueValidationRegex: `^(journal|ordered|writeback)$`},
			{Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true, ValueDescription: "One of: continue, remount-ro, panic", ValueValidationRegex: `^(continue|remount-ro|panic)$`},
			{Name: "discard", Description: "Enable discard/TRIM support"},
			{Name: "barrier", Description: "Enable/disable write barriers (0, 1)", NeedsValue: true, ValueDescription: "0 or 1", ValueValidationRegex: `^[01]$`},
			{Name: "noauto_da_alloc", Description: "Disable delayed allocation"},
			{Name: "journal_checksum", Description: "Enable journal checksumming"},
			{Name: "journal_async_commit", Description: "Commit data blocks asynchronously"},
		},
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

// Custom error types for FsTypeFromDevice
var (
	ErrorDeviceNotFound    = errors.New("device not found")
	ErrorDeviceAccess      = errors.New("failed to access device")
	ErrorUnknownFilesystem = errors.New("unknown filesystem type")
)

// NewFilesystemService creates and initializes a new FilesystemService.
func NewFilesystemService(ctx context.Context) FilesystemServiceInterface {
	// Initialize precomputed maps for efficient lookups
	stdFlagsByName := make(map[string]dto.MountFlag, len(defaultStandardMountFlags))
	for _, f := range defaultStandardMountFlags {
		stdFlagsByName[strings.ToLower(f.Name)] = f
	}

	allKnownFlagsByName := make(map[string]dto.MountFlag, len(defaultStandardMountFlags)+len(defaultFsSpecificMountFlags)) // Estimate size
	for k, v := range stdFlagsByName {
		allKnownFlagsByName[k] = v
	}
	for _, fsFlags := range defaultFsSpecificMountFlags {
		for _, f := range fsFlags {
			lowerName := strings.ToLower(f.Name)
			// Standard flags take precedence for descriptions if names collide.
			if _, exists := allKnownFlagsByName[lowerName]; !exists {
				allKnownFlagsByName[lowerName] = f
			}
		}
	}

	// Create filesystem adapter registry
	registry := filesystem.NewRegistry()

	// Update fsSpecificMountFlags with adapter mount flags
	updatedFsSpecificMountFlags := make(map[string][]dto.MountFlag)
	for k, v := range defaultFsSpecificMountFlags {
		updatedFsSpecificMountFlags[k] = v
	}

	// Add mount flags from adapters (if not already present)
	for _, adapter := range registry.GetAll() {
		fsType := adapter.GetName()
		if _, exists := updatedFsSpecificMountFlags[fsType]; !exists {
			updatedFsSpecificMountFlags[fsType] = adapter.GetMountFlags()
		}
	}

	p := &FilesystemService{
		standardMountFlags:   defaultStandardMountFlags,
		fsSpecificMountFlags: updatedFsSpecificMountFlags,

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

// fsMagicSignature defines a structure to hold filesystem signature information.
type fsMagicSignature struct {
	fsType string
	offset int64
	magic  []byte
}

// fsSignatures is a list of known filesystem signatures.
// The order can matter if signatures are subsets of others, though distinct offsets help.
var knownFsSignatures = []fsMagicSignature{
	// Filesystems with magic at/near offset 0
	{fsType: "xfs", offset: 0, magic: []byte{'X', 'F', 'S', 'B'}},
	{fsType: "squashfs", offset: 0, magic: []byte{0x68, 0x73, 0x71, 0x73}},              // "hsqs" little-endian
	{fsType: "ntfs", offset: 3, magic: []byte{'N', 'T', 'F', 'S', ' ', ' ', ' ', ' '}},  // "NTFS    "
	{fsType: "exfat", offset: 3, magic: []byte{'E', 'X', 'F', 'A', 'T', ' ', ' ', ' '}}, // "EXFAT   "

	// FAT types
	{fsType: "vfat", offset: 82, magic: []byte{'F', 'A', 'T', '3', '2', ' ', ' ', ' '}}, // FAT32 specific
	{fsType: "vfat", offset: 54, magic: []byte{'F', 'A', 'T', '1', '6', ' ', ' ', ' '}}, // FAT16 specific
	{fsType: "vfat", offset: 54, magic: []byte{'F', 'A', 'T', '1', '2', ' ', ' ', ' '}}, // FAT12 specific

	// Filesystems with magic at larger offsets
	{fsType: "f2fs", offset: 1024, magic: []byte{0x10, 0x20, 0xF5, 0xF2}}, // Little-endian 0xF2F52010
	{fsType: "ext4", offset: 1080, magic: []byte{0x53, 0xEF}},             // ext2/3/4, little-endian 0xEF53

	// ISO9660 - Primary Volume Descriptor
	{fsType: "iso9660", offset: 0x8001, magic: []byte{'C', 'D', '0', '0', '1'}}, // 32769

	// BTRFS
	{fsType: "btrfs", offset: 0x10040, magic: []byte{'_', 'B', 'H', 'R', 'f', 'S', '_', 'M'}}, // 65600
}

const maxDeviceReadLength = 65608 // Max offset (btrfs: 65600) + max magic length (8)

// FsTypeFromDevice attempts to determine the filesystem type of a block device by reading its magic numbers.
func (s *FilesystemService) FsTypeFromDevice(devicePath string) (string, errors.E) {
	file, err := os.Open(devicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.WithDetails(dto.ErrorDeviceNotFound, "Path", devicePath, "Error", err)
		}
		return "", errors.WithDetails(ErrorDeviceAccess, "Path", devicePath, "Operation", "Open", "Error", err)
	}
	defer file.Close()

	buffer := make([]byte, maxDeviceReadLength)
	n, err := file.ReadAt(buffer, 0)
	if err != nil && err != io.EOF {
		// For ReadAt, io.EOF is reported only if no bytes were read.
		// If n > 0 and err == io.EOF, it means a partial read, which is fine.
		// If n == 0 and err == io.EOF, the file is empty or smaller than our read attempt from offset 0.
		return "", errors.WithDetails(ErrorDeviceAccess, "Path", devicePath, "Operation", "ReadAt", "Error", err)
	}

	if n == 0 {
		return "", errors.WithDetails(ErrorUnknownFilesystem, "Path", devicePath, "Reason", "Device is empty or too small")
	}

	// Use the actual number of bytes read for checks
	validBuffer := buffer[:n]

	for _, sig := range knownFsSignatures {
		// Ensure the signature's offset and length are within the bounds of what was read
		sigEndOffset := sig.offset + int64(len(sig.magic))
		if sig.offset < 0 || sigEndOffset > int64(len(validBuffer)) {
			continue // Signature is out of bounds for the data read
		}

		// Compare the magic bytes
		if bytes.Equal(validBuffer[sig.offset:sigEndOffset], sig.magic) {
			slog.Debug("FsTypeFromDevice: Matched signature", "device", devicePath, "fstype", sig.fsType, "offset", sig.offset)
			return sig.fsType, nil
		}
	}

	slog.Debug("FsTypeFromDevice: No known filesystem signature matched", "device", devicePath)
	return "", errors.WithDetails(ErrorUnknownFilesystem, "Path", devicePath)
}

// GetAdapter returns the filesystem adapter for a given filesystem type
func (s *FilesystemService) GetAdapter(fsType string) (filesystem.FilesystemAdapter, errors.E) {
	return s.registry.Get(fsType)
}

// GetSupportedFilesystems returns information about all supported filesystems
func (s *FilesystemService) GetSupportedFilesystems(ctx context.Context) (map[string]filesystem.FilesystemSupport, errors.E) {
	return s.registry.GetSupportedFilesystems(ctx)
}

// ListSupportedTypes returns a list of all supported filesystem type names
func (s *FilesystemService) ListSupportedTypes() []string {
	return s.registry.ListSupportedTypes()
}

