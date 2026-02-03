package filesystem

import (
	"context"
	"strings"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// XfsAdapter implements FilesystemAdapter for XFS filesystems
type XfsAdapter struct {
	baseAdapter
}

// NewXfsAdapter creates a new XfsAdapter instance
func NewXfsAdapter() FilesystemAdapter {
	return &XfsAdapter{
		baseAdapter: baseAdapter{
			name:          "xfs",
			description:   "XFS Filesystem",
			alpinePackage: "xfsprogs",
			mkfsCommand:   "mkfs.xfs",
			fsckCommand:   "xfs_repair",
			labelCommand:  "xfs_admin",
			signatures: []dto.FsMagicSignature{
				{Offset: 0, Magic: []byte{'X', 'F', 'S', 'B'}},
			},
		},
	}
}

// GetMountFlags returns xfs-specific mount flags
func (a *XfsAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
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
	}
}

// IsSupported checks if xfs is supported on the system
func (a *XfsAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with xfs filesystem
func (a *XfsAdapter) Format(ctx context.Context, device string, options dto.FormatOptions) errors.E {
	args := []string{}

	if options.Force {
		args = append(args, "-f")
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
		return errors.Errorf("mkfs.xfs failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// Check runs filesystem check on an xfs device
func (a *XfsAdapter) Check(ctx context.Context, device string, options dto.CheckOptions) (dto.CheckResult, errors.E) {
	args := []string{}

	// xfs_repair doesn't support readonly mode directly
	// We check the options and use appropriate flags
	
	if !options.AutoFix {
		args = append(args, "-n") // No modify mode (read-only check)
	}

	if options.Verbose {
		args = append(args, "-v") // Verbose output
	}

	args = append(args, device)

	output, exitCode, err := runCommand(ctx, a.fsckCommand, args...)
	
	result := dto.CheckResult{
		ExitCode: exitCode,
		Message:  output,
	}

	// xfs_repair exit codes:
	// 0 - No errors
	// 1 - Errors found and corrected
	// 2 - Errors found, but not corrected (usually requires unmount)

	switch exitCode {
	case 0:
		result.Success = true
		result.ErrorsFound = false
		result.ErrorsFixed = false
	case 1:
		result.Success = true
		result.ErrorsFound = true
		result.ErrorsFixed = options.AutoFix
	case 2:
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

// GetLabel retrieves the xfs filesystem label
func (a *XfsAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// Use xfs_admin -l to get the label
	output, exitCode, err := runCommand(ctx, a.labelCommand, "-l", device)
	if err != nil {
		return "", errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return "", errors.Errorf("xfs_admin failed with exit code %d: %s", exitCode, output)
	}

	// Parse the output to find the label
	// Format: label = "mylabel"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "label = ") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				label := strings.TrimSpace(parts[1])
				// Remove quotes if present
				label = strings.Trim(label, "\"")
				return label, nil
			}
		}
	}

	return "", nil
}

// SetLabel sets the xfs filesystem label
func (a *XfsAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	// Use xfs_admin -L to set the label
	output, exitCode, err := runCommand(ctx, a.labelCommand, "-L", label, device)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Label", label)
	}

	if exitCode != 0 {
		return errors.Errorf("xfs_admin failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// GetState returns the state of an xfs filesystem
func (a *XfsAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// Run xfs_repair in no-modify mode to check state
	output, exitCode, err := runCommand(ctx, a.fsckCommand, "-n", device)
	if err != nil {
		return state, errors.WithDetails(err, "Device", device)
	}

	// Determine state based on xfs_repair exit code
	state.IsClean = exitCode == 0
	state.HasErrors = exitCode != 0

	if exitCode == 0 {
		state.StateDescription = "Clean"
	} else if exitCode == 1 {
		state.StateDescription = "Has correctable errors"
	} else {
		state.StateDescription = "Has errors"
	}

	// Check if filesystem is mounted
	mountOutput, _, _ := runCommand(ctx, "mount")
	state.IsMounted = strings.Contains(mountOutput, device)

	state.AdditionalInfo["repairOutput"] = output

	return state, nil
}
