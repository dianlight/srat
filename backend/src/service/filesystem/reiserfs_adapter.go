package filesystem

import (
	"context"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// ReiserfsAdapter implements FilesystemAdapter for ReiserFS filesystems
type ReiserfsAdapter struct {
	baseAdapter
}

// NewReiserfsAdapter creates a new ReiserfsAdapter instance
func NewReiserfsAdapter() FilesystemAdapter {
	return &ReiserfsAdapter{
		baseAdapter: baseAdapter{
			name:          "reiserfs",
			description:   "Reiser File System",
			alpinePackage: "reiserfsprogs",
			mkfsCommand:   "mkfs.reiserfs",
			fsckCommand:   "fsck.reiserfs",
			labelCommand:  "reiserfstune",
			signatures: []dto.FsMagicSignature{
				{Offset: 0x10034, Magic: []byte{'R', 'e', 'I', 's', 'E', 'r', 'F', 's'}},      // ReiserFS v3.5
				{Offset: 0x10034, Magic: []byte{'R', 'e', 'I', 's', 'E', 'r', '2', 'F', 's'}}, // ReiserFS v3.6
				{Offset: 0x10034, Magic: []byte{'R', 'e', 'I', 's', 'E', 'r', '3', 'F', 's'}}, // ReiserFS v3.6 with journal
			},
		},
	}
}

// GetMountFlags returns ReiserFS-specific mount flags
func (a *ReiserfsAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "conv", Description: "Convert old format to new"},
		{Name: "hash", Description: "Hash function to use", NeedsValue: true, ValueDescription: "One of: rupasov, tea, r5, detect", ValueValidationRegex: `^(rupasov|tea|r5|detect)$`},
		{Name: "hashed_relocation", Description: "Use hashed relocation"},
		{Name: "no_unhashed_relocation", Description: "Disable unhashed relocation"},
		{Name: "noborder", Description: "Disable border allocator"},
		{Name: "nolog", Description: "Disable journaling"},
		{Name: "notail", Description: "Disable tail packing"},
		{Name: "replayonly", Description: "Replay journal only"},
		{Name: "resize", Description: "Resize filesystem", NeedsValue: true, ValueDescription: "New size in blocks", ValueValidationRegex: `^\d+$`},
	}
}

// IsSupported checks if ReiserFS is supported on the system
func (a *ReiserfsAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with ReiserFS filesystem
func (a *ReiserfsAdapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	if progress != nil {
		progress("start", 0, []string{"Starting reiserfs format"})
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
			progress("failure", 0, []string{"Format failed: mkfs.reiserfs failed with exit code"})
		}
		output := strings.Join(outputLines, "\n")
		return errors.Errorf("mkfs.reiserfs failed with exit code %d: %s", result.ExitCode, output)
	}

	if progress != nil {
		progress("success", 100, []string{"Format completed successfully"})
	}
	return nil
}

// Check runs filesystem check on a ReiserFS device
func (a *ReiserfsAdapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	if progress != nil {
		progress("start", 0, []string{"Starting reiserfs check"})
	}

	args := []string{}

	if options.AutoFix {
		args = append(args, "--fix-fixable") // Automatically fix errors
	} else {
		args = append(args, "--check") // No changes, just check
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

	// fsck.reiserfs exit codes:
	// 0 - No errors
	// 1 - File system errors corrected
	// 2 - File system errors corrected, system should be rebooted
	// 4 - File system errors left uncorrected
	// 6 - Errors were found but not fixed
	// 8 - Operational error

	switch cmdResult.ExitCode {
	case 0:
		result.Success = true
		result.ErrorsFound = false
		result.ErrorsFixed = false
	case 1, 2:
		result.Success = true
		result.ErrorsFound = true
		result.ErrorsFixed = true
	case 4, 6:
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

// GetLabel retrieves the ReiserFS filesystem label
func (a *ReiserfsAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// Use reiserfstune to get filesystem information
	output, exitCode, err := runCommand(ctx, a.labelCommand, device)
	if err != nil {
		return "", errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return "", errors.Errorf("reiserfstune failed with exit code %d: %s", exitCode, output)
	}

	// Parse the output to find the label
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "LABEL:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				label := strings.TrimSpace(parts[1])
				return label, nil
			}
		}
	}

	return "", nil
}

// SetLabel sets the ReiserFS filesystem label
func (a *ReiserfsAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	// Use reiserfstune -l to set the label
	output, exitCode, err := runCommand(ctx, a.labelCommand, "-l", label, device)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Label", label)
	}

	if exitCode != 0 {
		return errors.Errorf("reiserfstune failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// GetState returns the state of a ReiserFS filesystem
func (a *ReiserfsAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// Run a read-only check to get filesystem state
	output, exitCode, _ := runCommand(ctx, a.fsckCommand, "--check", device)

	// Parse the output to determine filesystem state
	if exitCode == 0 {
		state.IsClean = true
		state.HasErrors = false
		state.StateDescription = "Clean"
	} else if exitCode == 1 || exitCode == 2 || exitCode == 4 || exitCode == 6 {
		state.IsClean = false
		state.HasErrors = true
		state.StateDescription = "Has errors"
	} else {
		state.StateDescription = "Unknown"
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
