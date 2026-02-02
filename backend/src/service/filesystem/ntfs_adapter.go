package filesystem

import (
	"context"
	"strings"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// NtfsAdapter implements FilesystemAdapter for NTFS filesystems
type NtfsAdapter struct {
	baseAdapter
}

// NewNtfsAdapter creates a new NtfsAdapter instance
func NewNtfsAdapter() FilesystemAdapter {
	return &NtfsAdapter{
		baseAdapter: baseAdapter{
			name:          "ntfs",
			description:   "NTFS Filesystem",
			alpinePackage: "ntfs-3g-progs",
			mkfsCommand:   "mkfs.ntfs",
			fsckCommand:   "ntfsfix",
			labelCommand:  "ntfslabel",
		},
	}
}

// GetMountFlags returns ntfs-specific mount flags
func (a *NtfsAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "uid", Description: "Set owner of all files to user ID", NeedsValue: true, ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
		{Name: "gid", Description: "Set group of all files to group ID", NeedsValue: true, ValueDescription: "Group ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
		{Name: "fmask", Description: "Set file permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "dmask", Description: "Set directory permissions mask (octal)", NeedsValue: true, ValueDescription: "Octal permission mask (e.g., 0022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "permissions", Description: "Respect NTFS permissions"},
		{Name: "acl", Description: "Enable POSIX Access Control Lists support"},
		{Name: "exec", Description: "Allow executing files (use with caution)"},
	}
}

// IsSupported checks if ntfs is supported on the system
func (a *NtfsAdapter) IsSupported(ctx context.Context) (FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with ntfs filesystem
func (a *NtfsAdapter) Format(ctx context.Context, device string, options FormatOptions) errors.E {
	args := []string{}

	// Quick format
	args = append(args, "-Q")

	if options.Force {
		args = append(args, "-F")
	}

	if options.Label != "" {
		args = append(args, "-L", options.Label)
	}

	// Add device as the last argument
	args = append(args, device)

	output, exitCode, err := runCommand(ctx, a.mkfsCommand, args...)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Output", output)
	}

	if exitCode != 0 {
		return errors.Errorf("mkfs.ntfs failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// Check runs filesystem check on an ntfs device
func (a *NtfsAdapter) Check(ctx context.Context, device string, options CheckOptions) (CheckResult, errors.E) {
	args := []string{}

	// ntfsfix doesn't have the same options as fsck
	// It's primarily for fixing common NTFS inconsistencies
	
	if !options.AutoFix {
		args = append(args, "-n") // No action, just check
	}

	args = append(args, device)

	output, exitCode, err := runCommand(ctx, a.fsckCommand, args...)
	
	result := CheckResult{
		ExitCode: exitCode,
		Message:  output,
	}

	// ntfsfix exit codes:
	// 0 - No errors or errors fixed
	// non-zero - Errors encountered

	switch exitCode {
	case 0:
		result.Success = true
		// Check output to determine if errors were found and fixed
		if strings.Contains(strings.ToLower(output), "repaired") || 
		   strings.Contains(strings.ToLower(output), "fixed") {
			result.ErrorsFound = true
			result.ErrorsFixed = true
		} else {
			result.ErrorsFound = false
			result.ErrorsFixed = false
		}
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

// GetLabel retrieves the ntfs filesystem label
func (a *NtfsAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// ntfslabel without second argument shows the current label
	output, exitCode, err := runCommand(ctx, a.labelCommand, device)
	if err != nil {
		return "", errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return "", errors.Errorf("ntfslabel failed with exit code %d: %s", exitCode, output)
	}

	// The output is just the label
	label := strings.TrimSpace(output)
	return label, nil
}

// SetLabel sets the ntfs filesystem label
func (a *NtfsAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	// ntfslabel device newlabel
	output, exitCode, err := runCommand(ctx, a.labelCommand, device, label)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Label", label)
	}

	if exitCode != 0 {
		return errors.Errorf("ntfslabel failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// GetState returns the state of an ntfs filesystem
func (a *NtfsAdapter) GetState(ctx context.Context, device string) (FilesystemState, errors.E) {
	state := FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// Run ntfsfix in check-only mode to determine state
	output, exitCode, err := runCommand(ctx, a.fsckCommand, "-n", device)
	if err != nil {
		return state, errors.WithDetails(err, "Device", device)
	}

	// Determine state based on ntfsfix exit code and output
	state.IsClean = exitCode == 0 && !strings.Contains(strings.ToLower(output), "error")
	state.HasErrors = exitCode != 0 || strings.Contains(strings.ToLower(output), "error")

	if state.IsClean {
		state.StateDescription = "Clean"
	} else {
		state.StateDescription = "Has errors or inconsistencies"
	}

	// Check if filesystem is mounted
	mountOutput, _, _ := runCommand(ctx, "mount")
	state.IsMounted = strings.Contains(mountOutput, device)

	state.AdditionalInfo["ntfsfixOutput"] = output

	return state, nil
}
