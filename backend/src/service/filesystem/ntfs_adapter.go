package filesystem

import (
	"context"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// NtfsAdapter implements FilesystemAdapter for NTFS filesystems
type NtfsAdapter struct {
	baseAdapter
}

// NewNtfsAdapter creates a new NtfsAdapter instance
func NewNtfsAdapter() FilesystemAdapter {
	ret := &NtfsAdapter{
		baseAdapter: newBaseAdapter(
			"ntfs",
			"NTFS Filesystem",
			"ntfs-3g-progs",
			"mkfs.ntfs",
			"ntfsfix",
			"ntfslabel",
			"ntfsfix",
			[]dto.FsMagicSignature{
				{Offset: 3, Magic: []byte{'N', 'T', 'F', 'S', ' ', ' ', ' ', ' '}}, // "NTFS    "
			},
		),
	}
	ret.linuxFsModule = "ntfs3" // Use ntfs3 kernel module for Linux if available, fallback to ntfs-3g user-space driver
	return ret
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
func (a *NtfsAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with ntfs filesystem
func (a *NtfsAdapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("start", 0, []string{"Starting ntfs format"})
	}

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
			progress("failure", 0, []string{"Format failed: mkfs.ntfs failed"})
		}
		output := strings.Join(outputLines, "\n")
		return errors.Errorf("mkfs.ntfs failed with exit code %d: %s", result.ExitCode, output)
	}

	if progress != nil {
		progress("success", 100, []string{"Format completed successfully"})
	}
	return nil
}

// Check runs filesystem check on an ntfs device
func (a *NtfsAdapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("start", 0, []string{"Starting ntfs check"})
	}

	args := []string{}

	// ntfsfix doesn't have the same options as fsck
	// It's primarily for fixing common NTFS inconsistencies

	if !options.AutoFix {
		args = append(args, "-n") // No action, just check
	}

	args = append(args, device)

	if progress != nil {
		progress("running", 999, []string{"Progress Status Not Supported"})
	}

	stdoutChan, stderrChan, resultChan := a.executeCommandWithProgress(ctx, a.fsckCommand, args)

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

	// ntfsfix exit codes:
	// 0 - No errors or errors fixed
	// non-zero - Errors encountered

	switch cmdResult.ExitCode {
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

// GetLabel retrieves the ntfs filesystem label
func (a *NtfsAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// ntfslabel without second argument shows the current label
	output, exitCode, err := a.runCommandCached(ctx, a.labelCommand, device)
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
	defer a.invalidateCommandResultCache()

	// ntfslabel device newlabel
	output, exitCode, err := a.runCommandCached(ctx, a.labelCommand, device, label)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Label", label)
	}

	if exitCode != 0 {
		return errors.Errorf("ntfslabel failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// GetState returns the state of an ntfs filesystem
func (a *NtfsAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]any),
	}

	// check if device is mounted and not run getstate if it is, as ntfsfix doesn't support checking while mounted
	outputMount, _, _ := a.runCommandCached(ctx, "mount")
	isMounted := strings.Contains(outputMount, device)
	state.IsMounted = isMounted

	if isMounted {
		state.IsClean = false
		state.HasErrors = false
		state.StateDescription = "Mounted (state cannot be determined)"
		return state, nil
	}

	// Run state command in check-only mode to determine state
	output, exitCode, err := a.runCommandCached(ctx, a.stateCommand, "-n", device)
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
	mountOutput, _, _ := a.runCommandCached(ctx, "mount")
	state.IsMounted = strings.Contains(mountOutput, device)

	state.AdditionalInfo["ntfsfixOutput"] = output

	return state, nil
}
