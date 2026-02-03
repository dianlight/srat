package filesystem

import (
	"context"
	"strings"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// HfsplusAdapter implements FilesystemAdapter for HFS+ filesystems
type HfsplusAdapter struct {
	baseAdapter
}

// NewHfsplusAdapter creates a new HfsplusAdapter instance
func NewHfsplusAdapter() FilesystemAdapter {
	return &HfsplusAdapter{
		baseAdapter: baseAdapter{
			name:          "hfsplus",
			description:   "Hierarchical File System Plus",
			alpinePackage: "hfsprogs",
			mkfsCommand:   "mkfs.hfsplus",
			fsckCommand:   "fsck.hfsplus",
			labelCommand:  "", // No separate label command
			signatures: []dto.FsMagicSignature{
				{Offset: 0x400, Magic: []byte{0x48, 0x2B}}, // H+
				{Offset: 0x400, Magic: []byte{0x48, 0x58}}, // HX (HFSX variant)
			},
		},
	}
}

// GetMountFlags returns HFS+-specific mount flags
func (a *HfsplusAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "uid", Description: "User ID for files", NeedsValue: true, ValueDescription: "User ID", ValueValidationRegex: `^\d+$`},
		{Name: "gid", Description: "Group ID for files", NeedsValue: true, ValueDescription: "Group ID", ValueValidationRegex: `^\d+$`},
		{Name: "umask", Description: "File mode creation mask", NeedsValue: true, ValueDescription: "Octal mask (e.g., 022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "force", Description: "Force mount even with errors"},
		{Name: "nls", Description: "Character set for filename conversion", NeedsValue: true, ValueDescription: "Charset name (e.g., utf8)", ValueValidationRegex: `^[a-zA-Z0-9_-]+$`},
		{Name: "decompose", Description: "Decompose Unicode filenames"},
	}
}

// IsSupported checks if HFS+ is supported on the system
func (a *HfsplusAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with HFS+ filesystem
func (a *HfsplusAdapter) Format(ctx context.Context, device string, options dto.FormatOptions) errors.E {
	args := []string{}

	if options.Label != "" {
		args = append(args, "-v", options.Label)
	}

	// Add device as the last argument
	args = append(args, device)

	output, exitCode, err := runCommand(ctx, a.mkfsCommand, args...)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Output", output)
	}

	if exitCode != 0 {
		return errors.Errorf("mkfs.hfsplus failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// Check runs filesystem check on an HFS+ device
func (a *HfsplusAdapter) Check(ctx context.Context, device string, options dto.CheckOptions) (dto.CheckResult, errors.E) {
	args := []string{}

	if options.Force {
		args = append(args, "-f") // Force check
	}

	args = append(args, device)

	output, exitCode, err := runCommand(ctx, a.fsckCommand, args...)
	
	result := dto.CheckResult{
		ExitCode: exitCode,
		Message:  output,
	}

	// fsck.hfsplus exit codes:
	// 0 - No errors
	// Other codes indicate errors

	switch exitCode {
	case 0:
		result.Success = true
		result.ErrorsFound = false
		result.ErrorsFixed = false
		if strings.Contains(strings.ToLower(output), "repaired") || strings.Contains(strings.ToLower(output), "fixed") {
			result.ErrorsFound = true
			result.ErrorsFixed = true
		}
	default:
		result.Success = false
		result.ErrorsFound = true
		result.ErrorsFixed = strings.Contains(strings.ToLower(output), "repaired") || strings.Contains(strings.ToLower(output), "fixed")
		if err != nil {
			return result, errors.WithDetails(err, "Device", device, "ExitCode", exitCode)
		}
	}

	return result, nil
}

// GetLabel retrieves the HFS+ filesystem label
func (a *HfsplusAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// HFS+ doesn't have a separate label command
	// Label information would need to be extracted from filesystem superblock
	return "", errors.Errorf("HFS+ label retrieval not supported via command line tools")
}

// SetLabel sets the HFS+ filesystem label
func (a *HfsplusAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	// HFS+ can only set label during format with -v option
	return errors.Errorf("HFS+ does not support changing labels after format. Label must be set during mkfs.hfsplus with -v option")
}

// GetState returns the state of an HFS+ filesystem
func (a *HfsplusAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// Run fsck to get filesystem state
	output, exitCode, _ := runCommand(ctx, a.fsckCommand, device)
	
	// Parse the output to determine filesystem state
	if exitCode == 0 {
		state.IsClean = true
		state.HasErrors = false
		state.StateDescription = "Clean"
	} else {
		state.IsClean = false
		state.HasErrors = true
		state.StateDescription = "Has errors"
	}

	// Check if filesystem is mounted
	outputMount, _, _ := runCommand(ctx, "mount")
	state.IsMounted = strings.Contains(outputMount, device)

	// Store check output in additional info
	if output != "" {
		state.AdditionalInfo["checkOutput"] = output
	}

	return state, nil
}
