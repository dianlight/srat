package filesystem

import (
	"context"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// F2fsAdapter implements FilesystemAdapter for F2FS filesystems
type F2fsAdapter struct {
	baseAdapter
}

// NewF2fsAdapter creates a new F2fsAdapter instance
func NewF2fsAdapter() FilesystemAdapter {
	return &F2fsAdapter{
		baseAdapter: newBaseAdapter(
			"f2fs",
			"Flash-Friendly File System",
			"f2fs-tools",
			"mkfs.f2fs",
			"fsck.f2fs",
			"", // No separate label command
			"fsck.f2fs",
			[]dto.FsMagicSignature{
				{Offset: 0x400, Magic: []byte{0x10, 0x20, 0xF5, 0xF2}},
			},
		),
	}
}

// GetMountFlags returns F2FS-specific mount flags
func (a *F2fsAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "background_gc", Description: "Background garbage collection mode", NeedsValue: true, ValueDescription: "One of: on, off, sync", ValueValidationRegex: `^(on|off|sync)$`},
		{Name: "disable_roll_forward", Description: "Disable roll-forward recovery"},
		{Name: "discard", Description: "Enable discard/TRIM support"},
		{Name: "no_heap", Description: "Disable heap-style segment allocation"},
		{Name: "nouser_xattr", Description: "Disable user extended attributes"},
		{Name: "noacl", Description: "Disable POSIX Access Control Lists"},
		{Name: "active_logs", Description: "Number of active logs", NeedsValue: true, ValueDescription: "2, 4, or 6", ValueValidationRegex: `^[246]$`},
		{Name: "inline_data", Description: "Enable inline data"},
		{Name: "inline_dentry", Description: "Enable inline directory entries"},
	}
}

// IsSupported checks if F2FS is supported on the system
func (a *F2fsAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with F2FS filesystem
func (a *F2fsAdapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("start", 0, []string{"Starting f2fs format"})
	}

	args := []string{}

	if options.Force {
		args = append(args, "-f")
	}

	if options.Label != "" {
		args = append(args, "-l", options.Label)
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
			progress("failure", 0, []string{"Format failed: mkfs.f2fs failed with exit code"})
		}
		output := strings.Join(outputLines, "\n")
		return errors.Errorf("mkfs.f2fs failed with exit code %d: %s", result.ExitCode, output)
	}

	if progress != nil {
		progress("success", 100, []string{"Format completed successfully"})
	}
	return nil
}

// Check runs filesystem check on an F2FS device
func (a *F2fsAdapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("start", 0, []string{"Starting f2fs check"})
	}

	args := []string{}

	if options.AutoFix {
		args = append(args, "-a") // Automatically fix errors
	}

	if options.Force {
		args = append(args, "-f") // Force check
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

	// fsck.f2fs exit codes:
	// 0 - No errors
	// Negative values or other codes indicate errors

	switch cmdResult.ExitCode {
	case 0:
		result.Success = true
		result.ErrorsFound = false
		result.ErrorsFixed = false
		if strings.Contains(strings.ToLower(output), "fixed") {
			result.ErrorsFound = true
			result.ErrorsFixed = true
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

// GetLabel retrieves the F2FS filesystem label
func (a *F2fsAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// F2FS doesn't have a separate label command
	// Use fsck.f2fs to get info
	output, exitCode, err := a.runCommand(ctx, a.fsckCommand, "-d", "1", device)
	if err != nil {
		return "", errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return "", errors.Errorf("fsck.f2fs failed with exit code %d: %s", exitCode, output)
	}

	// Parse the output to find the label
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "volume_name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				label := strings.TrimSpace(parts[1])
				return label, nil
			}
		}
	}

	return "", nil
}

// SetLabel sets the F2FS filesystem label
func (a *F2fsAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	defer a.invalidateCommandResultCache()

	// F2FS can only set label during format
	return errors.Errorf("F2FS does not support changing labels after format. Label must be set during mkfs.f2fs with -l option")
}

// GetState returns the state of an F2FS filesystem
func (a *F2fsAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// Run state command to get filesystem state
	output, exitCode, _ := a.runCommandCached(ctx, a.stateCommand, device)

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
	state.IsMounted = a.isDeviceMounted(device)

	// Store check output in additional info
	if output != "" {
		state.AdditionalInfo["checkOutput"] = output
	}

	return state, nil
}
