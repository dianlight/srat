package filesystem

import (
	"context"
	"strings"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// VfatAdapter implements FilesystemAdapter for vfat (FAT32) filesystems
type VfatAdapter struct {
	baseAdapter
}

// NewVfatAdapter creates a new VfatAdapter instance
func NewVfatAdapter() FilesystemAdapter {
	return &VfatAdapter{
		baseAdapter: baseAdapter{
			name:          "vfat",
			description:   "FAT32 Filesystem",
			alpinePackage: "dosfstools",
			mkfsCommand:   "mkfs.vfat",
			fsckCommand:   "fsck.vfat",
			labelCommand:  "fatlabel",
		},
	}
}

// GetMountFlags returns vfat-specific mount flags
func (a *VfatAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true, ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
		{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true, ValueDescription: "Group ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
		{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "umask", Description: "Set umask (octal) - overrides fmask/dmask", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "iocharset", Description: "I/O character set (e.g., utf8)", NeedsValue: true, ValueDescription: "Character set name (e.g., utf8)", ValueValidationRegex: `^[a-zA-Z0-9_-]+$`},
		{Name: "codepage", Description: "Codepage for short filenames (e.g., 437)", NeedsValue: true, ValueDescription: "Codepage number (e.g., 437)", ValueValidationRegex: `^[0-9]+$`},
		{Name: "shortname", Description: "Shortname case (lower, win95, winnt, mixed)", NeedsValue: true, ValueDescription: "One of: lower, win95, winnt, mixed", ValueValidationRegex: `^(lower|win95|winnt|mixed)$`},
		{Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true, ValueDescription: "One of: continue, remount-ro, panic", ValueValidationRegex: `^(continue|remount-ro|panic)$`},
	}
}

// IsSupported checks if vfat is supported on the system
func (a *VfatAdapter) IsSupported(ctx context.Context) (FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with vfat filesystem
func (a *VfatAdapter) Format(ctx context.Context, device string, options FormatOptions) errors.E {
	args := []string{}

	// FAT32 specific - use -F 32 for FAT32
	args = append(args, "-F", "32")

	if options.Label != "" {
		args = append(args, "-n", options.Label)
	}

	// Add device as the last argument
	args = append(args, device)

	output, exitCode, err := runCommand(ctx, a.mkfsCommand, args...)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Output", output)
	}

	if exitCode != 0 {
		return errors.Errorf("mkfs.vfat failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// Check runs filesystem check on a vfat device
func (a *VfatAdapter) Check(ctx context.Context, device string, options CheckOptions) (CheckResult, errors.E) {
	args := []string{}

	if options.AutoFix {
		args = append(args, "-a") // Automatically repair the filesystem
	} else {
		args = append(args, "-n") // No-op mode, just check
	}

	if options.Verbose {
		args = append(args, "-v") // Verbose output
	}

	args = append(args, device)

	output, exitCode, err := runCommand(ctx, a.fsckCommand, args...)
	
	result := CheckResult{
		ExitCode: exitCode,
		Message:  output,
	}

	// fsck.vfat exit codes:
	// 0 - No errors
	// 1 - Errors corrected
	// 2 - Errors corrected, reboot suggested
	// 4 - Errors left uncorrected
	// 8 - Operational error
	// 16 - Usage error

	switch exitCode {
	case 0:
		result.Success = true
		result.ErrorsFound = false
		result.ErrorsFixed = false
	case 1, 2:
		result.Success = true
		result.ErrorsFound = true
		result.ErrorsFixed = true
	case 4:
		result.Success = false
		result.ErrorsFound = true
		result.ErrorsFixed = false
	default:
		result.Success = false
		result.ErrorsFound = true
		result.ErrorsFixed = false
		if err != nil {
			return result, errors.WithDetails(err, "Device", device, "ExitCode", exitCode)
		}
	}

	return result, nil
}

// GetLabel retrieves the vfat filesystem label
func (a *VfatAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// fatlabel without second argument shows the current label
	output, exitCode, err := runCommand(ctx, a.labelCommand, device)
	if err != nil {
		return "", errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return "", errors.Errorf("fatlabel failed with exit code %d: %s", exitCode, output)
	}

	// The output is just the label
	label := strings.TrimSpace(output)
	return label, nil
}

// SetLabel sets the vfat filesystem label
func (a *VfatAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	// fatlabel device newlabel
	output, exitCode, err := runCommand(ctx, a.labelCommand, device, label)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Label", label)
	}

	if exitCode != 0 {
		return errors.Errorf("fatlabel failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// GetState returns the state of a vfat filesystem
func (a *VfatAdapter) GetState(ctx context.Context, device string) (FilesystemState, errors.E) {
	state := FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// For FAT filesystems, we can run fsck in read-only mode to check state
	output, exitCode, err := runCommand(ctx, a.fsckCommand, "-n", device)
	if err != nil {
		return state, errors.WithDetails(err, "Device", device)
	}

	// Determine state based on fsck exit code
	state.IsClean = exitCode == 0
	state.HasErrors = exitCode != 0

	if exitCode == 0 {
		state.StateDescription = "Clean"
	} else {
		state.StateDescription = "Has errors"
	}

	// Check if filesystem is mounted
	mountOutput, _, _ := runCommand(ctx, "mount")
	state.IsMounted = strings.Contains(mountOutput, device)

	state.AdditionalInfo["fsckOutput"] = output

	return state, nil
}
