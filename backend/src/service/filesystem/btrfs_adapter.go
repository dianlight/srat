package filesystem

import (
	"context"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// BtrfsAdapter implements FilesystemAdapter for Btrfs filesystems
type BtrfsAdapter struct {
	baseAdapter
}

// NewBtrfsAdapter creates a new BtrfsAdapter instance
func NewBtrfsAdapter() FilesystemAdapter {
	return &BtrfsAdapter{
		baseAdapter: newBaseAdapter(
			"btrfs",
			"BTRFS Filesystem",
			"btrfs-progs",
			"mkfs.btrfs",
			"btrfs",
			"btrfs",
			"btrfs",
			[]dto.FsMagicSignature{
				{Offset: 0x10040, Magic: []byte{'_', 'B', 'H', 'R', 'f', 'S', '_', 'M'}}, // 65600
			},
		),
	}
}

// GetMountFlags returns btrfs-specific mount flags
func (a *BtrfsAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "compress", Description: "Enable compression", NeedsValue: true, ValueDescription: "One of: zlib, lzo, zstd, or none", ValueValidationRegex: `^(zlib|lzo|zstd|none)$`},
		{Name: "compress-force", Description: "Force compression on all files", NeedsValue: true, ValueDescription: "One of: zlib, lzo, zstd", ValueValidationRegex: `^(zlib|lzo|zstd)$`},
		{Name: "autodefrag", Description: "Enable automatic defragmentation"},
		{Name: "discard", Description: "Enable discard/TRIM support"},
		{Name: "ssd", Description: "Enable SSD-specific optimizations"},
		{Name: "nossd", Description: "Disable SSD-specific optimizations"},
		{Name: "space_cache", Description: "Enable space cache", NeedsValue: true, ValueDescription: "One of: v1, v2", ValueValidationRegex: `^(v1|v2)$`},
		{Name: "subvol", Description: "Mount specific subvolume", NeedsValue: true, ValueDescription: "Subvolume path", ValueValidationRegex: `^[a-zA-Z0-9/_-]+$`},
		{Name: "subvolid", Description: "Mount specific subvolume by ID", NeedsValue: true, ValueDescription: "Subvolume ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
	}
}

// IsSupported checks if btrfs is supported on the system
func (a *BtrfsAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with btrfs filesystem
func (a *BtrfsAdapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("start", 0, []string{"Starting btrfs format"})
	}

	args := []string{}

	if options.Force {
		args = append(args, "-f")
	}

	if options.Label != "" {
		args = append(args, "-L", options.Label)
	}

	// Add device as the last argument
	args = append(args, device)

	if progress != nil {
		progress("running", 999, []string{"Progress Status Not Supported"})
	}

	stdoutChan, stderrChan, resultChan := a.executeCommandWithProgress(ctx, a.mkfsCommand, args)

	// Consume output channels
	var outputLines []string
	var errorLines []string
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for line := range stdoutChan {
			outputLines = append(outputLines, line)
			if progress != nil {
				progress("running", 999, []string{line})
			}
		}
	}()

	go func() {
		defer wg.Done()
		for line := range stderrChan {
			errorLines = append(errorLines, line)
			if progress != nil {
				progress("running", 999, []string{"ERROR: " + line})
			}
		}
	}()

	wg.Wait()

	// Wait for command result
	result := <-resultChan
	if result.Error != nil {
		if progress != nil {
			progress("failure", 0, []string{"Format failed: " + result.Error.Error()})
		}
		output := strings.Join(outputLines, "\n")
		return errors.WithDetails(result.Error, "Device", device, "Output", output)
	}

	if result.ExitCode != 0 {
		if progress != nil {
			progress("failure", 0, []string{"Format failed: mkfs.btrfs failed with exit code"})
		}
		output := strings.Join(outputLines, "\n")
		return errors.Errorf("mkfs.btrfs failed with exit code %d: %s", result.ExitCode, output)
	}

	if progress != nil {
		progress("success", 100, []string{"Format completed successfully"})
	}
	return nil
}

// Check runs filesystem check on a btrfs device
func (a *BtrfsAdapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("start", 0, []string{"Starting btrfs check"})
	}

	args := []string{"check"}

	if options.AutoFix {
		args = append(args, "--repair")
	} else {
		args = append(args, "--readonly")
	}

	if options.Force {
		args = append(args, "--force")
	}

	args = append(args, device)

	if progress != nil {
		progress("running", 999, []string{"Progress Status Not Supported"})
	}

	stdoutChan, stderrChan, resultChan := a.executeCommandWithProgress(ctx, "btrfs", args)

	// Consume output channels
	var outputLines []string
	var errorLines []string
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for line := range stdoutChan {
			outputLines = append(outputLines, line)
			if progress != nil {
				progress("running", 999, []string{line})
			}
		}
	}()

	go func() {
		defer wg.Done()
		for line := range stderrChan {
			errorLines = append(errorLines, line)
			if progress != nil {
				progress("running", 999, []string{"ERROR: " + line})
			}
		}
	}()

	wg.Wait()

	// Wait for command result
	cmdResult := <-resultChan
	output := strings.Join(outputLines, "\n")

	result := dto.CheckResult{
		ExitCode: cmdResult.ExitCode,
		Message:  output,
	}

	// btrfs check exit codes:
	// 0 - No errors
	// non-zero - Errors found or operational issues

	switch cmdResult.ExitCode {
	case 0:
		result.Success = true
		// Check output to determine if errors were found
		if strings.Contains(strings.ToLower(output), "error") {
			result.ErrorsFound = true
			result.ErrorsFixed = options.AutoFix
		} else {
			result.ErrorsFound = false
			result.ErrorsFixed = false
		}
	default:
		result.Success = false
		result.ErrorsFound = true
		result.ErrorsFixed = false
		if cmdResult.Error != nil {
			if progress != nil {
				progress("failure", 0, []string{"Check failed: " + cmdResult.Error.Error()})
			}
			return result, errors.WithDetails(cmdResult.Error, "Device", device, "ExitCode", cmdResult.ExitCode)
		}
	}

	if result.Success {
		if progress != nil {
			progress("success", 100, []string{"Check completed successfully"})
		}
	} else {
		if progress != nil {
			progress("failure", 0, []string{"Check failed with errors"})
		}
	}

	return result, nil
}

// GetLabel retrieves the btrfs filesystem label
func (a *BtrfsAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// Use 'btrfs filesystem show' to get filesystem information including label
	output, exitCode, err := a.runCommand(ctx, "btrfs", "filesystem", "show", device)
	if err != nil {
		return "", errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return "", errors.Errorf("btrfs filesystem show failed with exit code %d: %s", exitCode, output)
	}

	// Parse the output to find the label
	// Format: Label: 'mylabel'  uuid: ...
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Label:") {
			// Extract label from the line
			if idx := strings.Index(line, "Label:"); idx != -1 {
				labelPart := line[idx+6:] // Skip "Label:"
				labelPart = strings.TrimSpace(labelPart)

				// Label is in single quotes or is 'none'
				if strings.HasPrefix(labelPart, "'") {
					endIdx := strings.Index(labelPart[1:], "'")
					if endIdx != -1 {
						return labelPart[1 : endIdx+1], nil
					}
				} else if strings.HasPrefix(labelPart, "none") {
					return "", nil
				}
			}
		}
	}

	return "", nil
}

// SetLabel sets the btrfs filesystem label
func (a *BtrfsAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	defer a.invalidateCommandResultCache()

	// Use 'btrfs filesystem label' to set the label
	output, exitCode, err := a.runCommand(ctx, "btrfs", "filesystem", "label", device, label)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Label", label)
	}

	if exitCode != 0 {
		return errors.Errorf("btrfs filesystem label failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// GetState returns the state of a btrfs filesystem
func (a *BtrfsAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]any),
	}

	// Use state command device stats to get device statistics
	output, exitCode, err := a.runCommandCached(ctx, a.stateCommand, "device", "stats", device)
	if err == nil && exitCode == 0 {
		state.AdditionalInfo["deviceStats"] = output

		// Check for errors in stats
		if strings.Contains(strings.ToLower(output), "error") {
			state.HasErrors = true
			state.IsClean = false
		} else {
			state.HasErrors = false
			state.IsClean = true
		}
	} else {
		// If we can't get stats, run a readonly check
		checkOutput, checkExitCode, err := a.runCommandCached(ctx, a.stateCommand, "check", "--readonly", device)
		if err != nil {
			return state, errors.WithDetails(err, "Device", device)
		}
		state.IsClean = checkExitCode == 0
		state.HasErrors = checkExitCode != 0
		state.AdditionalInfo["checkOutput"] = checkOutput
	}

	if state.IsClean {
		state.StateDescription = "Clean"
	} else {
		state.StateDescription = "Has errors or inconsistencies"
	}

	// Check if filesystem is mounted
	mountOutput, _, _ := a.runCommandCached(ctx, "mount")
	state.IsMounted = strings.Contains(mountOutput, device)

	return state, nil
}
