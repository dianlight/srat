package filesystem

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/tlog"
	gocache "github.com/patrickmn/go-cache"
	"github.com/u-root/u-root/pkg/mount"
	"gitlab.com/tozd/go/errors"
)

const (
	commandResultCacheTTL             = 30 * time.Minute
	commandResultCacheCleanupInterval = 10 * time.Minute
)

var commandResultCache = gocache.New(commandResultCacheTTL, commandResultCacheCleanupInterval)

// execCmd interface wraps os/exec.Cmd
type ExecCmd interface {
	CombinedOutput() ([]byte, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

// realExecCmd wraps exec.Cmd to implement ExecCmd interface
type realExecCmd struct {
	cmd *exec.Cmd
}

func (r *realExecCmd) CombinedOutput() ([]byte, error) {
	return r.cmd.CombinedOutput()
}

func (r *realExecCmd) StdoutPipe() (io.ReadCloser, error) {
	return r.cmd.StdoutPipe()
}

func (r *realExecCmd) StderrPipe() (io.ReadCloser, error) {
	return r.cmd.StderrPipe()
}

func (r *realExecCmd) Start() error {
	return r.cmd.Start()
}

func (r *realExecCmd) Wait() error {
	return r.cmd.Wait()
}

type cachedCommandResult struct {
	Output   string
	ExitCode int
	Error    errors.E
}

// CommandResult holds the exit code and error from a command execution
type CommandResult struct {
	ExitCode int
	Error    errors.E
}

// baseAdapter provides common functionality for all filesystem adapters
type baseAdapter struct {
	name          string
	linuxFsModule string
	description   string
	alpinePackage string
	mkfsCommand   string
	fsckCommand   string
	labelCommand  string
	stateCommand  string
	signatures    []dto.FsMagicSignature
	//
	baseTryMountFunc func(source, target, data string, flags uintptr, prepareTarget ...func() error) (*mount.MountPoint, error)
	baseDoMountFunc  func(source, target, fstype, data string, flags uintptr, prepareTarget ...func() error) (*mount.MountPoint, error)
	baseUnmountFunc  func(target string, force, lazy bool) error
	execLookPath     func(string) (string, error)
	execCommand      func(context.Context, string, ...string) ExecCmd
	getFilesystems   func() ([]string, error) // Optional override for osutil.GetFileSystems, used in command availability checks
}

func newBaseAdapter(name, description, alpinePackage, mkfsCommand, fsckCommand, labelCommand, stateCommand string, signatures []dto.FsMagicSignature) baseAdapter {
	return baseAdapter{
		name:             name,
		description:      description,
		alpinePackage:    alpinePackage,
		mkfsCommand:      mkfsCommand,
		fsckCommand:      fsckCommand,
		labelCommand:     labelCommand,
		stateCommand:     stateCommand,
		signatures:       signatures,
		baseTryMountFunc: mount.TryMount,
		baseDoMountFunc:  mount.Mount,
		baseUnmountFunc:  mount.Unmount,
		execLookPath:     exec.LookPath,
		execCommand: func(ctx context.Context, name string, args ...string) ExecCmd {
			return &realExecCmd{cmd: exec.CommandContext(ctx, name, args...)}
		},
		getFilesystems: osutil.GetFileSystems,
	}
}

// commandExists checks if a command is available in the system PATH
func (b *baseAdapter) commandExists(command string) bool {
	_, err := b.execLookPath(command)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false
		}
		tlog.Warn("Error checking command existence", "command", command, "error", err)
		return false
	}
	return true
}

// runCommand executes a command and returns the output
func (b *baseAdapter) runCommand(ctx context.Context, name string, args ...string) (string, int, errors.E) {
	cmd := b.execCommand(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	exitCode := 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		tlog.ErrorContext(ctx, "Error executing command", "command", name, "args", args, "exitCode", exitCode, "Output", string(output))
		return strings.TrimSpace(string(output)), exitCode, errors.WithDetails(err, "Command", name, "Args", strings.Join(args, " "))
	}

	return strings.TrimSpace(string(output)), exitCode, nil
}

// runCommandCached executes a command and caches the result by command name and exact args.
// This limits command execution to at most once per cache TTL for each unique command+args key.
func (b *baseAdapter) runCommandCached(ctx context.Context, name string, args ...string) (string, int, errors.E) {
	cacheKey := b.commandCacheKey(name, args...)

	if cached, ok := commandResultCache.Get(cacheKey); ok {
		if result, castOk := cached.(cachedCommandResult); castOk {
			return result.Output, result.ExitCode, result.Error
		}

		commandResultCache.Delete(cacheKey)
	}

	output, exitCode, err := b.runCommand(ctx, name, args...)
	commandResultCache.Set(cacheKey, cachedCommandResult{
		Output:   output,
		ExitCode: exitCode,
		Error:    err,
	}, commandResultCacheTTL)

	return output, exitCode, err
}

func (b *baseAdapter) invalidateCommandResultCache() {
	commandResultCache.Flush()
}

func (b *baseAdapter) commandCacheKey(name string, args ...string) string {
	var builder strings.Builder
	builder.Grow(len(name) + len(args)*8)
	appendPart := func(part string) {
		builder.WriteString(strconv.Itoa(len(part)))
		builder.WriteByte(':')
		builder.WriteString(part)
		builder.WriteByte('|')
	}

	appendPart(name)
	for _, arg := range args {
		appendPart(arg)
	}

	return builder.String()
}

// checkCommandAvailability checks if required commands are available
func (b *baseAdapter) checkCommandAvailability() dto.FilesystemSupport {
	filesystem, err := b.getFilesystems()
	if err != nil {
		tlog.Warn("Failed to get filesystems for command availability check", "error", err)
	} else if !slices.Contains(filesystem, b.GetLinuxFsModule()) {
		tlog.Debug("Filesystem module not found in system, marking related commands as unavailable", "filesystem", b.GetLinuxFsModule())
		return dto.FilesystemSupport{
			CanMount:      false,
			AlpinePackage: b.alpinePackage,
			MissingTools:  []string{b.mkfsCommand, b.fsckCommand, b.labelCommand, b.stateCommand},
		}
	}

	support := dto.FilesystemSupport{
		CanMount:      true, // Most filesystems can be mounted if kernel supports them
		AlpinePackage: b.alpinePackage,
		MissingTools:  []string{},
	}

	if b.mkfsCommand != "" {
		support.CanFormat = b.commandExists(b.mkfsCommand)
		if !support.CanFormat {
			support.MissingTools = append(support.MissingTools, b.mkfsCommand)
		}
	}

	if b.fsckCommand != "" {
		support.CanCheck = b.commandExists(b.fsckCommand)
		if !support.CanCheck {
			support.MissingTools = append(support.MissingTools, b.fsckCommand)
		}
	}

	if b.labelCommand != "" {
		support.CanSetLabel = b.commandExists(b.labelCommand)
		if !support.CanSetLabel {
			support.MissingTools = append(support.MissingTools, b.labelCommand)
		}
	}

	if b.stateCommand != "" {
		support.CanGetState = b.commandExists(b.stateCommand)
		if !support.CanGetState {
			support.MissingTools = append(support.MissingTools, b.stateCommand)
		}
	}

	return support
}

// GetName returns the filesystem type name
func (b *baseAdapter) GetName() string {
	return b.name
}

// GetLinuxFsModule returns the Linux filesystem module/fstype name to use for mounting
func (b *baseAdapter) GetLinuxFsModule() string {
	if b.linuxFsModule != "" {
		return b.linuxFsModule
	}
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
	return b.checkDeviceMatchesSignatures(devicePath)
}

// Mount mounts source to target using a generic Linux mount implementation.
func (b *baseAdapter) Mount(
	ctx context.Context,
	source, target, fsType, data string,
	flags uintptr,
	prepareTarget func() error,
) (*mount.MountPoint, errors.E) {
	opts := make([]func() error, 0, 1)
	if prepareTarget != nil {
		opts = append(opts, prepareTarget)
	}

	var (
		mp  *mount.MountPoint
		err error
	)

	if fsType == "" {
		mp, err = b.baseTryMountFunc(source, target, data, flags, opts...)
	} else {
		mp, err = b.baseDoMountFunc(source, target, fsType, data, flags, opts...)
	}
	if err != nil {
		return nil, errors.WithDetails(err,
			"Source", source,
			"Target", target,
			"FSType", fsType,
			"Flags", flags,
			"Data", data,
		)
	}

	return mp, nil
}

// Unmount unmounts target using a generic Linux unmount implementation.
func (b *baseAdapter) Unmount(ctx context.Context, target string, force, lazy bool) errors.E {
	_ = ctx
	if err := b.baseUnmountFunc(target, force, lazy); err != nil {
		return errors.WithDetails(err, "Target", target, "Force", force, "Lazy", lazy)
	}

	return nil
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

	cmd := b.execCommand(ctx, command, args...)

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
	wg.Go(func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case stdoutChan <- scanner.Text():
			}
		}
	})

	// Scan stderr in a goroutine
	wg.Go(func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case stderrChan <- scanner.Text():
			}
		}
	})

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
