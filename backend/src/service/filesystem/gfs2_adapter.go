package filesystem

import (
	"context"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// Gfs2Adapter implements FilesystemAdapter for GFS2 filesystems
type Gfs2Adapter struct {
	baseAdapter
}

// NewGfs2Adapter creates a new Gfs2Adapter instance
func NewGfs2Adapter() FilesystemAdapter {
	return &Gfs2Adapter{
		baseAdapter: newBaseAdapter(
			"gfs2",
			"Global File System 2",
			"gfs2-utils",
			"mkfs.gfs2",
			"fsck.gfs2",
			"", // No label command
			"fsck.gfs2",
			[]dto.FsMagicSignature{
				{Offset: 0x10, Magic: []byte{0x01, 0x16, 0x19, 0x70}},
			},
		),
	}
}

// GetMountFlags returns GFS2-specific mount flags
func (a *Gfs2Adapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "lockproto", Description: "Lock protocol name", NeedsValue: true, ValueDescription: "Protocol name (e.g., lock_dlm, lock_nolock)", ValueValidationRegex: `^lock_[a-z]+$`},
		{Name: "locktable", Description: "Lock table name", NeedsValue: true, ValueDescription: "Table name", ValueValidationRegex: `^[a-zA-Z0-9_:.-]+$`},
		{Name: "hostdata", Description: "Host-specific data", NeedsValue: true, ValueDescription: "Host data string"},
		{Name: "spectator", Description: "Mount as a spectator (read-only)"},
		{Name: "norecovery", Description: "Don't recover the journal"},
		{Name: "quota", Description: "Quota enforcement mode", NeedsValue: true, ValueDescription: "One of: off, account, on", ValueValidationRegex: `^(off|account|on)$`},
		{Name: "data", Description: "Data journaling mode", NeedsValue: true, ValueDescription: "One of: writeback, ordered", ValueValidationRegex: `^(writeback|ordered)$`},
	}
}

// IsSupported checks if GFS2 is supported on the system
func (a *Gfs2Adapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()
	return support, nil
}

// Format formats a device with GFS2 filesystem
func (a *Gfs2Adapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("start", 0, []string{"Starting gfs2 format"})
	}

	args := []string{"-p", "lock_nolock"}

	if options.Force {
		args = append(args, "-O")
	}

	// GFS2 requires a table name
	args = append(args, "-t", "local:gfs2")

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
			progress("failure", 0, []string{"Format failed: mkfs.gfs2 failed with exit code"})
		}
		output := strings.Join(outputLines, "\n")
		return errors.Errorf("mkfs.gfs2 failed with exit code %d: %s", result.ExitCode, output)
	}

	if progress != nil {
		progress("success", 100, []string{"Format completed successfully"})
	}
	return nil
}

// Check runs filesystem check on a GFS2 device
func (a *Gfs2Adapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("start", 0, []string{"Starting gfs2 check"})
	}

	args := []string{}

	if options.AutoFix {
		args = append(args, "-y") // Automatically fix errors
	} else {
		args = append(args, "-n") // No changes, just check
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

	// fsck.gfs2 exit codes:
	// 0 - No errors
	// 1 - File system errors corrected
	// 2 - File system errors corrected, system should be rebooted
	// 4 - File system errors left uncorrected

	switch cmdResult.ExitCode {
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

// GetLabel retrieves the GFS2 filesystem label
func (a *Gfs2Adapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	// GFS2 does not support labels
	return "", errors.Errorf("GFS2 does not support filesystem labels")
}

// SetLabel sets the GFS2 filesystem label
func (a *Gfs2Adapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	defer a.invalidateCommandResultCache()

	// GFS2 does not support labels
	return errors.Errorf("GFS2 does not support filesystem labels")
}

// GetState returns the state of a GFS2 filesystem
func (a *Gfs2Adapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo: make(map[string]interface{}),
	}

	// Run state command in read-only mode to get filesystem state
	output, exitCode, _ := a.runCommandCached(ctx, a.stateCommand, "-n", device)

	// Parse the output to determine filesystem state
	switch exitCode {
	case 0:
		state.IsClean = true
		state.HasErrors = false
		state.StateDescription = "Clean"
	case 1, 2, 4:
		state.IsClean = false
		state.HasErrors = true
		state.StateDescription = "Has errors"
	default:
		state.StateDescription = "Unknown"
	}

	// Check if filesystem is mounted
	state.IsMounted = a.isDeviceMounted(device)

	// Store check output in additional info
	if output != "" {
		state.AdditionalInfo["checkOutput"] = output
	}

	return state, nil
}
