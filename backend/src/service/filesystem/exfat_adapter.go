package filesystem

import (
	"context"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// ExfatAdapter implements FilesystemAdapter for exFAT filesystems
type ExfatAdapter struct {
	baseAdapter
}

// NewExfatAdapter creates a new ExfatAdapter instance
func NewExfatAdapter() FilesystemAdapter {
	return &ExfatAdapter{
		baseAdapter: baseAdapter{
			name:          "exfat",
			description:   "Extended File Allocation Table",
			alpinePackage: "exfatprogs",
			mkfsCommand:   "mkfs.exfat",
			fsckCommand:   "fsck.exfat",
			labelCommand:  "exfatlabel",
			stateCommand:  "fsck.exfat",
			signatures: []dto.FsMagicSignature{
				{Offset: 3, Magic: []byte{'E', 'X', 'F', 'A', 'T', ' ', ' ', ' '}},
			},
		},
	}
}

// GetMountFlags returns exFAT-specific mount flags
func (a *ExfatAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "uid", Description: "User ID for files", NeedsValue: true, ValueDescription: "User ID", ValueValidationRegex: `^\d+$`},
		{Name: "gid", Description: "Group ID for files", NeedsValue: true, ValueDescription: "Group ID", ValueValidationRegex: `^\d+$`},
		{Name: "umask", Description: "File mode creation mask", NeedsValue: true, ValueDescription: "Octal mask (e.g., 022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "dmask", Description: "Directory mode creation mask", NeedsValue: true, ValueDescription: "Octal mask (e.g., 022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "fmask", Description: "File mode creation mask (alternative to umask)", NeedsValue: true, ValueDescription: "Octal mask (e.g., 022)", ValueValidationRegex: `^[0-7]{3,4}$`},
		{Name: "iocharset", Description: "Character set for filename conversion", NeedsValue: true, ValueDescription: "Charset name (e.g., utf8)", ValueValidationRegex: `^[a-zA-Z0-9_-]+$`},
	}
}

// IsSupported checks if exFAT is supported on the system
func (a *ExfatAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with exFAT filesystem
func (a *ExfatAdapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	if progress != nil {
		progress("start", 0, []string{"Starting exfat format"})
	}

	args := []string{}

	if options.Label != "" {
		args = append(args, "-n", options.Label)
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
			progress("failure", 0, []string{"Format failed: mkfs.exfat failed with exit code"})
		}
		output := strings.Join(outputLines, "\n")
		return errors.Errorf("mkfs.exfat failed with exit code %d: %s", result.ExitCode, output)
	}

	if progress != nil {
		progress("success", 100, []string{"Format completed successfully"})
	}
	return nil
}

// Check runs filesystem check on an exFAT device
func (a *ExfatAdapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	if progress != nil {
		progress("start", 0, []string{"Starting exfat check"})
	}

	args := []string{}

	if options.AutoFix {
		args = append(args, "-y") // Automatically fix errors
	} else {
		args = append(args, "-n") // No changes, just check
	}

	if options.Verbose {
		args = append(args, "-v") // Verbose output
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

	// fsck.exfat exit codes:
	// 0 - No errors
	// 1 - File system errors corrected
	// 4 - File system errors left uncorrected
	// 8 - Operational error

	switch cmdResult.ExitCode {
	case 0:
		result.Success = true
		result.ErrorsFound = false
		result.ErrorsFixed = false
	case 1:
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

// GetLabel retrieves the exFAT filesystem label
func (a *ExfatAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	output, exitCode, err := runCommand(ctx, a.labelCommand, device)
	if err != nil {
		return "", errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return "", errors.Errorf("exfatlabel failed with exit code %d: %s", exitCode, output)
	}

	// exfatlabel outputs the label directly
	label := strings.TrimSpace(output)
	return label, nil
}

// SetLabel sets the exFAT filesystem label
func (a *ExfatAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	output, exitCode, err := runCommand(ctx, a.labelCommand, device, label)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Label", label)
	}

	if exitCode != 0 {
		return errors.Errorf("exfatlabel failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// GetState returns the state of an exFAT filesystem
func (a *ExfatAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// Run state command in read-only mode to get filesystem state
	output, exitCode, err := runCommandCached(ctx, a.stateCommand, "-n", device)
	if err != nil {
		return state, errors.WithDetails(err, "Device", device)
	}

	// Parse the output to determine filesystem state
	switch exitCode {
	case 0:
		state.IsClean = true
		state.HasErrors = false
		state.StateDescription = "Clean"
	case 1, 4:
		state.IsClean = false
		state.HasErrors = true
		state.StateDescription = "Has errors"
	default:
		state.StateDescription = "Unknown"
	}

	// Check if filesystem is mounted
	outputMount, _, _ := runCommandCached(ctx, "mount")
	state.IsMounted = strings.Contains(outputMount, device)

	// Store check output in additional info
	if output != "" {
		state.AdditionalInfo["checkOutput"] = output
	}

	return state, nil
}
