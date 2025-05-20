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

// MountFlag represents a single mount option/flag.
type MountFlag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	NeedsValue  bool   `json:"needsValue,omitempty"` // Add NeedsValue field
	Value       string `json:"value,omitempty"`      // Add Value field for flags like "uid=<arg>"
	// Could add Type (e.g., "standard", "ext4", "nfs") or Category if needed
	// Could add ValueType (e.g., "boolean", "string", "integer") for validation/UI
}

// FilesystemServiceInterface defines the methods for managing filesystem types and mount flags.
type FilesystemServiceInterface interface {
	// GetSupportedFilesystemTypes returns a list of filesystem types explicitly supported or known by the application.
	GetSupportedFilesystemTypes() ([]string, errors.E)

	// GetStandardMountFlags returns a list of common, filesystem-agnostic mount flags.
	GetStandardMountFlags() ([]MountFlag, errors.E)

	// GetFilesystemSpecificMountFlags returns a list of mount flags specific to a given filesystem type.
	// Returns an empty list if the filesystem type is not recognized or has no specific flags.
	GetFilesystemSpecificMountFlags(fsType string) ([]MountFlag, errors.E)

	// GetMountFlagsAndData converts a list of MountFlag structs into the syscall flags (uintptr)
	// and the data string (string) for the syscall.Mount function.
	GetMountFlagsAndData(inputFlags []MountFlag) (uintptr, string, errors.E)
}

// FilesystemService implements the FilesystemServiceInterface.
type FilesystemService struct {
	// ctx context.Context // Currently not needed, but could be added for future async operations

	// Hardcoded lists for now. Could be loaded from config/DB in the future.
	supportedFilesystems []string
	standardMountFlags   []MountFlag
	fsSpecificMountFlags map[string][]MountFlag
}

// NewFilesystemService creates and initializes a new FilesystemService.
func NewFilesystemService(ctx context.Context) FilesystemServiceInterface {
	// Initialize hardcoded data
	supportedFS := []string{
		"auto", // Common option to detect automatically
		"ext4",
		"xfs",
		"btrfs",
		"f2fs",
		"ntfs",  // Older NTFS driver / ntfs-3g (userspace)
		"ntfs3", // Newer kernel NTFS driver
		"vfat",  // FAT32
		"exfat",
		"tmpfs",
		"iso9660", // CD-ROM / DVD images
		"udf",     // DVD / Blu-ray images
		"zfs",     // ZFS filesystem
	}

	standardFlags := []MountFlag{
		{Name: "ro", Description: "Mount read-only"},
		{Name: "rw", Description: "Mount read-write (default)"}, // NeedsValue: false (default state)
		{Name: "sync", Description: "All I/O to the filesystem should be done synchronously."},
		{Name: "async", Description: "All I/O to the filesystem should be done asynchronously."}, // NeedsValue: false (default state)
		{Name: "atime", Description: "Do not use noatime feature (default)"},                     // NeedsValue: false (default state)
		{Name: "noatime", Description: "Do not update inode access times on this filesystem."},
		{Name: "diratime", Description: "Update directory inode access times on this filesystem."},
		{Name: "nodiratime", Description: "Do not update directory inode access times on this filesystem."},
		{Name: "dev", Description: "Interpret character or block special devices on the filesystem."}, // NeedsValue: false (default state)
		{Name: "nodev", Description: "Do not interpret character or block special devices on the filesystem."},
		{Name: "exec", Description: "Permit execution of binaries."}, // NeedsValue: false (default state)
		{Name: "noexec", Description: "Do not permit execution of binaries."},
		{Name: "suid", Description: "Permit set-user-id or set-group-id bits to take effect."}, // NeedsValue: false (default state)
		{Name: "nosuid", Description: "Do not permit set-user-id or set-group-id bits to take effect."},
		{Name: "remount", Description: "Attempt to remount an already-mounted filesystem."},
		{Name: "defaults", Description: "Use default options: rw, suid, dev, exec, auto, nouser, async."}, // NeedsValue: false (composite default)
		// Add more common flags as needed
		// All standard flags are boolean/switch flags, so NeedsValue is false by default or explicitly set.
	}

	fsSpecificFlags := map[string][]MountFlag{
		"ntfs": {
			{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true},
			{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true},
			{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true},
			{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true},
			{Name: "permissions", Description: "Respect NTFS permissions", NeedsValue: false},
			{Name: "acl", Description: "Enable ACL support", NeedsValue: false},
			{Name: "exec", Description: "Allow executing files (use with caution)", NeedsValue: false}, // Note: 'exec' is also a standard flag
			// Add more ntfs specific flags
			// uid, gid, fmask, dmask need values. permissions, acl, exec do not.
		},
		"ntfs3": {
			{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true},
			{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true},
			{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true},
			{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true},
			{Name: "permissions", Description: "Respect NTFS permissions", NeedsValue: false},
			{Name: "acl", Description: "Enable ACL support", NeedsValue: false},
			{Name: "force", Description: "Force mount even if the volume is marked dirty", NeedsValue: false},
			{Name: "norecover", Description: "Do not try to recover a dirty volume (default for ntfs3)", NeedsValue: false},
			{Name: "iocharset", Description: "I/O character set (e.g., utf8)", NeedsValue: true},
			// Add more ntfs3 specific flags
			// uid, gid, fmask, dmask, iocharset need values. permissions, acl, force, norecover do not.
		},
		"zfs": {
			{Name: "zfsutil", Description: "Indicates that the mount is managed by ZFS utilities", NeedsValue: false},
			{Name: "noauto", Description: "Can be used to prevent automatic mounting by zfs-mount-generator", NeedsValue: false},
			{Name: "context", Description: "Set SELinux context for all files/directories", NeedsValue: true},
			{Name: "fscontext", Description: "Set SELinux context for the filesystem superblock", NeedsValue: true},
			// ZFS has many properties managed by `zfs set`, but some can be mount options.
			// zfsutil, noauto do not need values. context, fscontext do.
		},
		// Add more filesystem types with specific flags as needed
	}

	p := &FilesystemService{
		// ctx: ctx,
		supportedFilesystems: supportedFS,
		standardMountFlags:   standardFlags,
		fsSpecificMountFlags: fsSpecificFlags,
	}
	return p
}

// GetSupportedFilesystemTypes returns the list of supported filesystem types.
func (s *FilesystemService) GetSupportedFilesystemTypes() ([]string, errors.E) {
	// In a real-world scenario, this might dynamically check /proc/filesystems
	// or filter a predefined list based on system capabilities.
	// For now, return the hardcoded list.
	return s.supportedFilesystems, nil
}

// GetStandardMountFlags returns the list of standard mount flags.
func (s *FilesystemService) GetStandardMountFlags() ([]MountFlag, errors.E) {
	return s.standardMountFlags, nil
}

// GetFilesystemSpecificMountFlags returns the list of mount flags specific to the given filesystem type.
func (s *FilesystemService) GetFilesystemSpecificMountFlags(fsType string) ([]MountFlag, errors.E) {
	flags, ok := s.fsSpecificMountFlags[fsType]
	if !ok {
		// Return an empty list if no specific flags are defined for this type
		return []MountFlag{}, nil
	}
	return flags, nil
}

// GetMountFlagsAndData converts a list of MountFlag structs into the syscall flags (uintptr)
// and the data string (string) for the syscall.Mount function.
// It processes flags that correspond to standard mount(2) bitmask options.
// Flags with values (e.g., "uid=<arg>") or flags representing default/permissive states (e.g., "rw", "defaults")
// are typically ignored by this function as they don't directly set a bit in the syscall flags parameter
// or are handled by the absence of restrictive flags.
func (s *FilesystemService) GetMountFlagsAndData(inputFlags []MountFlag) (uintptr, string, errors.E) {
	var syscallFlagValue uintptr = 0
	var dataFlags []string

	// Map of mount flag names (lowercase) to their corresponding syscall constants.
	// This map includes flags that SET a bit. Flags like "rw" or "async"
	// represent the ABSENCE of a restrictive bit and are handled by not setting MS_RDONLY or MS_SYNCHRONOUS.
	// "defaults" is also handled by the base state (0) and subsequent overrides.
	flagMap := map[string]uintptr{
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
		// "lazytime":    syscall.MS_LAZYTIME, // Ignore LAZYTIME
	}

	// Flags that are descriptive, represent default states, or are handled by other mechanisms (like the data field of mount).
	// These will be ignored without warning.
	ignoredFlags := map[string]bool{
		"rw": true, "async": true, "atime": true, "diratime": true,
		"dev": true, "exec": true, "suid": true, "defaults": true,
		"auto":    true, // mount(8) option, not direct syscall flag
		"nouser":  true, // mount(8) option
		"user":    true, // mount(8) option
		"_netdev": true, // mount(8) option
		"nofail":  true, // mount(8) option
	}

	for _, mf := range inputFlags {
		rawFlagName := strings.TrimSpace(mf.Name)
		lowerFlagName := strings.ToLower(rawFlagName) // Use lowercase for map lookups

		// --- New Validation Check ---
		if !mf.NeedsValue && mf.Value != "" {
			return 0, "", errors.WithDetails(dto.ErrorInvalidParameter,
				"Flag", mf.Name,
				"Value", mf.Value,
				"Message", "Boolean/switch flag was provided with a value")
		}

		// If a Value is provided in the struct, use it regardless of the Name format
		if mf.Value != "" {
			formattedFlag := fmt.Sprintf("%s=%s", rawFlagName, mf.Value)
			slog.Debug("GetMountFlagsAndData: Collecting data flag with explicit value", "flag", formattedFlag)
			dataFlags = append(dataFlags, formattedFlag)
			continue
		}

		if val, ok := flagMap[lowerFlagName]; ok {
			//lowerFlagName := strings.ToLower(rawFlagName) // Ensure lowercase for map lookup
			slog.Debug("GetMountFlagsAndData: Adding syscall flag to bitmask", "flag", rawFlagName, "value", val)
			syscallFlagValue |= val
		} else if ignoredFlags[lowerFlagName] {
			slog.Debug("GetMountFlagsAndData: Ignoring known descriptive/default flag", "flag", rawFlagName)
		} else if rawFlagName != "" {
			slog.Warn("GetSyscallFlags: Unknown or unhandled mount flag for bitmask generation", "flag", mf.Name)
		}
	}

	// Join the collected data flags into a single string
	dataString := strings.Join(dataFlags, ",")

	return syscallFlagValue, dataString, nil
}
