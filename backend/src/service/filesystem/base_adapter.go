package filesystem

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// baseAdapter provides common functionality for all filesystem adapters
type baseAdapter struct {
	name          string
	description   string
	alpinePackage string
	mkfsCommand   string
	fsckCommand   string
	labelCommand  string
	signatures    []dto.FsMagicSignature
}

// commandExists checks if a command is available in the system PATH
func commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// runCommand executes a command and returns the output
func runCommand(ctx context.Context, name string, args ...string) (string, int, errors.E) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	exitCode := 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, errors.WithDetails(err, "Command", name, "Args", strings.Join(args, " "))
		}
	}

	return strings.TrimSpace(string(output)), exitCode, nil
}

// checkCommandAvailability checks if required commands are available
func (b *baseAdapter) checkCommandAvailability() dto.FilesystemSupport {
	support := dto.FilesystemSupport{
		CanMount:      true, // Most filesystems can be mounted if kernel supports them
		AlpinePackage: b.alpinePackage,
		MissingTools:  []string{},
	}

	if b.mkfsCommand != "" {
		support.CanFormat = commandExists(b.mkfsCommand)
		if !support.CanFormat {
			support.MissingTools = append(support.MissingTools, b.mkfsCommand)
		}
	}

	if b.fsckCommand != "" {
		support.CanCheck = commandExists(b.fsckCommand)
		if !support.CanCheck {
			support.MissingTools = append(support.MissingTools, b.fsckCommand)
		}
	}

	if b.labelCommand != "" {
		support.CanSetLabel = commandExists(b.labelCommand)
		if !support.CanSetLabel {
			support.MissingTools = append(support.MissingTools, b.labelCommand)
		}
	}

	// For now, state checking is not supported by default
	support.CanGetState = false

	return support
}

// GetName returns the filesystem type name
func (b *baseAdapter) GetName() string {
	return b.name
}

// GetDescription returns the filesystem description
func (b *baseAdapter) GetDescription() string {
	return b.description
}

// GetFsSignatureMagic returns the magic number signatures for this filesystem
func (b *baseAdapter) GetFsSignatureMagic() []dto.FsMagicSignature {
	return b.signatures
}

// IsDeviceSupported checks if a device can be mounted with this filesystem
// by examining magic numbers. This is a default implementation that uses
// the magic signature detection system.
func (b *baseAdapter) IsDeviceSupported(ctx context.Context, devicePath string) (bool, errors.E) {
	// Check if device matches any of the adapter's signatures
	return checkDeviceMatchesSignatures(devicePath, b.signatures)
}

// executeCommandWithProgress executes a long-running command with real-time progress monitoring.
// It scans stdout and stderr line-by-line and calls the progress callback with command output.
// Supports context cancellation to gracefully interrupt the running process.
//
// Parameters:
//   - ctx: Context for cancellation support
//   - command: Command name to execute
//   - args: Command arguments
//   - progress: Optional callback for progress updates (receives status, percentual, notes)
//
// Returns:
//   - Combined stdout output as string
//   - Error if command fails or context is cancelled
func (b *baseAdapter) executeCommandWithProgress(
	ctx context.Context,
	command string,
	args []string,
	progress dto.ProgressCallback,
) (string, errors.E) {
	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, command, args...)

	// Setup pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.WithDetails(err, "command", command, "error", "failed to create stdout pipe")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", errors.WithDetails(err, "command", command, "error", "failed to create stderr pipe")
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", errors.WithDetails(err, "command", command, "args", strings.Join(args, " "))
	}

	// Channel to collect output lines
	var outputLines []string
	var errorLines []string
	var mu sync.Mutex

	// WaitGroup to wait for goroutines to finish
	var wg sync.WaitGroup

	// Scan stdout in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			outputLines = append(outputLines, line)
			mu.Unlock()

			// Call progress callback with the output line
			if progress != nil {
				progress("running", 999, []string{line})
			}
		}
	}()

	// Scan stderr in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			errorLines = append(errorLines, line)
			mu.Unlock()

			// Call progress callback with the error line
			if progress != nil {
				progress("running", 999, []string{"ERROR: " + line})
			}
		}
	}()

	// Wait for the command to complete in a separate goroutine
	errChan := make(chan error, 1)
	go func() {
		wg.Wait() // Wait for scanners to finish
		errChan <- cmd.Wait()
	}()

	// Wait for either command completion or context cancellation
	select {
	case <-ctx.Done():
		// Context was cancelled - kill the process
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return "", errors.WithDetails(ctx.Err(), "command", command, "error", "context cancelled")

	case cmdErr := <-errChan:
		// Command completed (successfully or with error)
		mu.Lock()
		output := strings.Join(outputLines, "\n")
		errOutput := strings.Join(errorLines, "\n")
		mu.Unlock()

		if cmdErr != nil {
			// Command failed
			if exitErr, ok := cmdErr.(*exec.ExitError); ok {
				details := errors.WithDetails(
					cmdErr,
					"command", command,
					"args", strings.Join(args, " "),
					"exitCode", exitErr.ExitCode(),
					"stderr", errOutput,
				)
				return output, details
			}
			return output, errors.WithDetails(cmdErr, "command", command, "stderr", errOutput)
		}

		// Command succeeded
		return output, nil
	}
}

// scanOutput is a helper to scan output from a reader and call progress callback
func (b *baseAdapter) scanOutput(reader io.Reader, progress dto.ProgressCallback, prefix string) []string {
	var lines []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if progress != nil {
			if prefix != "" {
				progress("running", 999, []string{prefix + line})
			} else {
				progress("running", 999, []string{line})
			}
		}
	}
	return lines
}
