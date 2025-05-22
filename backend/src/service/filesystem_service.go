package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"syscall"

	"github.com/dianlight/srat/dto"
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
	}

	// defaultFsSpecificMountFlags maps filesystem types to their specific mount flags.
	defaultFsSpecificMountFlags = map[string][]dto.MountFlag{
		"ntfs": {
			{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true},
			{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true},
			{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true},
			{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true},
			{Name: "permissions", Description: "Respect NTFS permissions"},
			{Name: "acl", Description: "Enable ACL support"},
			{Name: "exec", Description: "Allow executing files (use with caution)"},
		},
		"ntfs3": {
			{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true},
			{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true},
			{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true},
			{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true},
			{Name: "permissions", Description: "Respect NTFS permissions"},
			{Name: "acl", Description: "Enable ACL support"},
			{Name: "force", Description: "Force mount even if the volume is marked dirty"},
			{Name: "norecover", Description: "Do not try to recover a dirty volume (default for ntfs3)"},
			{Name: "iocharset", Description: "I/O character set (e.g., utf8)", NeedsValue: true},
		},
		"zfs": {
			{Name: "zfsutil", Description: "Indicates that the mount is managed by ZFS utilities"},
			{Name: "noauto", Description: "Can be used to prevent automatic mounting by zfs-mount-generator"},
			{Name: "context", Description: "Set SELinux context for all files/directories", NeedsValue: true},
			{Name: "fscontext", Description: "Set SELinux context for the filesystem superblock", NeedsValue: true},
		},
		"ext2":  {{Name: "acl", Description: "Enable POSIX Access Control Lists support"}, {Name: "user_xattr", Description: "Enable user extended attributes"}, {Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true}, {Name: "discard", Description: "Enable discard/TRIM support"}},
		"ext3":  {{Name: "data", Description: "Data journaling mode (ordered, writeback, journal)", NeedsValue: true}, {Name: "journal_checksum", Description: "Enable journal checksumming"}, {Name: "journal_async_commit", Description: "Commit data blocks asynchronously"}, {Name: "acl", Description: "Enable POSIX Access Control Lists support"}, {Name: "user_xattr", Description: "Enable user extended attributes"}, {Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true}, {Name: "discard", Description: "Enable discard/TRIM support"}, {Name: "barrier", Description: "Enable/disable write barriers (0, 1)", NeedsValue: true}},
		"vfat":  {{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true}, {Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true}, {Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true}, {Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true}, {Name: "umask", Description: "Set umask (octal) - overrides fmask/dmask", NeedsValue: true}, {Name: "iocharset", Description: "I/O character set (e.g., utf8)", NeedsValue: true}, {Name: "codepage", Description: "Codepage for short filenames (e.g., 437)", NeedsValue: true}, {Name: "shortname", Description: "Shortname case (lower, win95, mixed)", NeedsValue: true}, {Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true}},
		"exfat": {{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true}, {Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true}, {Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true}, {Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true}, {Name: "umask", Description: "Set umask (octal) - overrides fmask/dmask", NeedsValue: true}},
		"ext4":  {{Name: "data", Description: "Data journaling mode (ordered, writeback, journal)", NeedsValue: true}, {Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true}, {Name: "discard", Description: "Enable discard/TRIM support"}, {Name: "nodiscard", Description: "Disable discard/TRIM support (default)"}, {Name: "barrier", Description: "Enable/disable write barriers (0, 1)", NeedsValue: true}, {Name: "auto_da_alloc", Description: "Enable delayed allocation (default)"}, {Name: "noauto_da_alloc", Description: "Disable delayed allocation"}, {Name: "journal_checksum", Description: "Enable journal checksumming"}, {Name: "nojournal_checksum", Description: "Disable journal checksumming (default)"}, {Name: "journal_async_commit", Description: "Commit data blocks asynchronously"}},
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

	p := &FilesystemService{
		standardMountFlags:   defaultStandardMountFlags,
		fsSpecificMountFlags: defaultFsSpecificMountFlags,

		standardMountFlagsByName: stdFlagsByName,
		allKnownMountFlagsByName: allKnownFlagsByName,
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

		// --- New Validation Check ---
		if !mf.NeedsValue && mf.FlagValue != "" {
			return 0, "", errors.WithDetails(dto.ErrorInvalidParameter,
				"Flag", mf.Name,
				"Value", mf.FlagValue,
				"Message", "Boolean/switch flag was provided with a value")
		}

		// If a Value is provided in the struct, use it regardless of the Name format
		if mf.FlagValue != "" {
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
			// Note: descFlag.NeedsValue could also be consulted here if it's considered more authoritative
			// than the presence of "=" in parsing. For now, parsing dictates NeedsValue.
		} else {
			slog.Debug("SyscallDataToMountFlag: No description found for data flag", "flagName", name)
		}
		result = append(result, mountFlag)
	}

	return result, nil
}
