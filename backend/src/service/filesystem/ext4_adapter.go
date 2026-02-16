package filesystem

import (
	"context"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// Ext4Adapter implements FilesystemAdapter for ext4 filesystems
type Ext4Adapter struct {
	baseAdapter
}

// NewExt4Adapter creates a new Ext4Adapter instance
func NewExt4Adapter() FilesystemAdapter {
	return &Ext4Adapter{
		baseAdapter: baseAdapter{
			name:          "ext4",
			description:   "EXT4 Filesystem",
			alpinePackage: "e2fsprogs",
			mkfsCommand:   "mkfs.ext4",
			fsckCommand:   "fsck.ext4",
			labelCommand:  "tune2fs",
			stateCommand:  "tune2fs",
			signatures: []dto.FsMagicSignature{
				{Offset: 1080, Magic: []byte{0x53, 0xEF}}, // ext2/3/4, little-endian 0xEF53
			},
		},
	}
}

// GetMountFlags returns ext4-specific mount flags
func (a *Ext4Adapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "data", Description: "Data journaling mode (ordered, writeback, journal)", NeedsValue: true, ValueDescription: "One of: journal, ordered, writeback", ValueValidationRegex: `^(journal|ordered|writeback)$`},
		{Name: "errors", Description: "Behavior on error (remount-ro, continue, panic)", NeedsValue: true, ValueDescription: "One of: continue, remount-ro, panic", ValueValidationRegex: `^(continue|remount-ro|panic)$`},
		{Name: "discard", Description: "Enable discard/TRIM support"},
		{Name: "barrier", Description: "Enable/disable write barriers (0, 1)", NeedsValue: true, ValueDescription: "0 or 1", ValueValidationRegex: `^[01]$`},
		{Name: "noauto_da_alloc", Description: "Disable delayed allocation"},
		{Name: "journal_checksum", Description: "Enable journal checksumming"},
		{Name: "journal_async_commit", Description: "Commit data blocks asynchronously"},
	}
}

// IsSupported checks if ext4 is supported on the system
func (a *Ext4Adapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with ext4 filesystem
func (a *Ext4Adapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	// Report start
	if progress != nil {
		progress("start", 0, []string{"Starting ext4 format"})
	}

	args := []string{}

	if options.Force {
		args = append(args, "-F")
	}

	if options.Label != "" {
		args = append(args, "-L", options.Label)
	}

	// Add device as the last argument
	args = append(args, device)

	// Report running (progress not supported for mkfs.ext4)
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
			progress("failure", 0, []string{"mkfs.ext4 failed with exit code"})
		}
		output := strings.Join(outputLines, "\n")
		return errors.Errorf("mkfs.ext4 failed with exit code %d: %s", result.ExitCode, output)
	}

	// Report success
	if progress != nil {
		progress("success", 100, []string{"Format completed successfully"})
	}

	return nil
}

// Check runs filesystem check on an ext4 device
func (a *Ext4Adapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	// Report start
	if progress != nil {
		progress("start", 0, []string{"Starting ext4 filesystem check"})
	}

	args := []string{}

	if options.AutoFix {
		args = append(args, "-y") // Automatically fix errors
	} else {
		args = append(args, "-n") // No changes, just check
	}

	if options.Force {
		args = append(args, "-f") // Force check even if clean
	}

	if options.Verbose {
		args = append(args, "-v") // Verbose output
	}

	args = append(args, device)

	// Report running (progress not supported for fsck.ext4)
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

	// fsck.ext4 exit codes:
	// 0 - No errors
	// 1 - File system errors corrected
	// 2 - File system errors corrected, system should be rebooted
	// 4 - File system errors left uncorrected
	// 8 - Operational error
	// 16 - Usage or syntax error
	// 32 - Fsck canceled by user request
	// 128 - Shared library error

	switch cmdResult.ExitCode {
	case 0:
		result.Success = true
		result.ErrorsFound = false
		result.ErrorsFixed = false
		if progress != nil {
			progress("success", 100, []string{"Check completed: no errors found"})
		}
	case 1, 2:
		result.Success = true
		result.ErrorsFound = true
		result.ErrorsFixed = true
		if progress != nil {
			progress("success", 100, []string{"Check completed: errors corrected"})
		}
	case 4:
		result.Success = false
		result.ErrorsFound = true
		result.ErrorsFixed = false
		if progress != nil {
			progress("failure", 0, []string{"Check failed: errors left uncorrected"})
		}
	default:
		result.Success = false
		result.ErrorsFound = true
		result.ErrorsFixed = false
		if progress != nil {
			progress("failure", 0, []string{"Check failed with exit code"})
		}
		if cmdResult.Error != nil {
			return result, errors.WithDetails(cmdResult.Error, "Device", device, "ExitCode", cmdResult.ExitCode)
		}
	}

	return result, nil
}

// GetLabel retrieves the ext4 filesystem label
func (a *Ext4Adapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// Use tune2fs -l to get filesystem information
	output, exitCode, err := runCommand(ctx, a.labelCommand, "-l", device)
	if err != nil {
		return "", errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return "", errors.Errorf("tune2fs failed with exit code %d: %s", exitCode, output)
	}

	// Parse the output to find the label
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Filesystem volume name:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				label := strings.TrimSpace(parts[1])
				if label == "<none>" {
					return "", nil
				}
				return label, nil
			}
		}
	}

	return "", nil
}

// SetLabel sets the ext4 filesystem label
func (a *Ext4Adapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	// Use tune2fs -L to set the label
	output, exitCode, err := runCommand(ctx, a.labelCommand, "-L", label, device)
	if err != nil {
		return errors.WithDetails(err, "Device", device, "Label", label)
	}

	if exitCode != 0 {
		return errors.Errorf("tune2fs failed with exit code %d: %s", exitCode, output)
	}

	return nil
}

// GetState returns the state of an ext4 filesystem
func (a *Ext4Adapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// Use tune2fs -l to get filesystem state information
	output, exitCode, err := runCommand(ctx, a.labelCommand, "-l", device)
	if err != nil {
		return state, errors.WithDetails(err, "Device", device)
	}

	if exitCode != 0 {
		return state, errors.Errorf("tune2fs failed with exit code %d: %s", exitCode, output)
	}

	// Parse the output to determine filesystem state
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Filesystem state:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				stateStr := strings.TrimSpace(parts[1])
				state.IsClean = strings.Contains(strings.ToLower(stateStr), "clean")
				state.HasErrors = strings.Contains(strings.ToLower(stateStr), "error")
				state.StateDescription = stateStr
			}
		} else if strings.HasPrefix(line, "Mount count:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				state.AdditionalInfo["mountCount"] = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Maximum mount count:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				state.AdditionalInfo["maxMountCount"] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Check if filesystem is mounted
	// This is a simple check - in a production system, you'd want to check /proc/mounts
	output, _, _ = runCommand(ctx, "mount")
	state.IsMounted = strings.Contains(output, device)

	if state.StateDescription == "" {
		state.StateDescription = "Unknown"
	}

	return state, nil
}
