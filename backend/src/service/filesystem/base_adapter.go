package filesystem

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// CommandResult holds the exit code and error from a command execution
type CommandResult struct {
	ExitCode int
	Error    errors.E
}

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
// It returns channels for stdout and stderr output that can be consumed by the caller.
// Supports context cancellation to gracefully interrupt the running process.
//
// Parameters:
//   - ctx: Context for cancellation support
//   - command: Command name to execute
//   - args: Command arguments
//
// Returns:
//   - Channel for stdout lines (closed when command completes)
//   - Channel for stderr lines (closed when command completes)
//   - Channel for command result (receives one CommandResult when command completes, then closes)
func (b *baseAdapter) executeCommandWithProgress(
	ctx context.Context,
	command string,
	args []string,
) (<-chan string, <-chan string, <-chan CommandResult) {
	// Create channels for output and result
	stdoutChan := make(chan string, 100)
	stderrChan := make(chan string, 100)
	resultChan := make(chan CommandResult, 1)

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, command, args...)

	// Setup pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		close(stdoutChan)
		close(stderrChan)
		resultChan <- CommandResult{
			ExitCode: -1,
			Error:    errors.WithDetails(err, "command", command, "error", "failed to create stdout pipe"),
		}
		close(resultChan)
		return stdoutChan, stderrChan, resultChan
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		close(stdoutChan)
		close(stderrChan)
		resultChan <- CommandResult{
			ExitCode: -1,
			Error:    errors.WithDetails(err, "command", command, "error", "failed to create stderr pipe"),
		}
		close(resultChan)
		return stdoutChan, stderrChan, resultChan
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		close(stdoutChan)
		close(stderrChan)
		resultChan <- CommandResult{
			ExitCode: -1,
			Error:    errors.WithDetails(err, "command", command, "args", strings.Join(args, " ")),
		}
		close(resultChan)
		return stdoutChan, stderrChan, resultChan
	}

	// WaitGroup to wait for goroutines to finish
	var wg sync.WaitGroup

	// Scan stdout in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case stdoutChan <- scanner.Text():
			}
		}
	}()

	// Scan stderr in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case stderrChan <- scanner.Text():
			}
		}
	}()

	// Wait for command to complete and send result
	go func() {
		wg.Wait()
		close(stdoutChan)
		close(stderrChan)

		result := CommandResult{ExitCode: 0}
		cmdErr := cmd.Wait()

		select {
		case <-ctx.Done():
			// Context was cancelled
			result.ExitCode = -1
			result.Error = errors.WithDetails(ctx.Err(), "command", command, "error", "context cancelled")
		default:
			if cmdErr != nil {
				if exitErr, ok := cmdErr.(*exec.ExitError); ok {
					result.ExitCode = exitErr.ExitCode()
					result.Error = errors.WithDetails(
						cmdErr,
						"command", command,
						"args", strings.Join(args, " "),
						"exitCode", exitErr.ExitCode(),
					)
				} else {
					result.ExitCode = -1
					result.Error = errors.WithDetails(cmdErr, "command", command)
				}
			}
		}

		resultChan <- result
		close(resultChan)
	}()

	return stdoutChan, stderrChan, resultChan
}
