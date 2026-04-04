package filesystem

import (
	"bufio"
	"context"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/tlog"
	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/procfs"
	"github.com/u-root/u-root/pkg/mount"
	"gitlab.com/tozd/go/errors"
)

const (
	commandResultCacheTTL             = 30 * time.Minute
	commandResultCacheCleanupInterval = 10 * time.Minute
)

var (
	commandResultCache     = gocache.New(commandResultCacheTTL, commandResultCacheCleanupInterval)
	defaultCommandRunnerMu sync.RWMutex
	defaultCommandRunner   CommandRunner
)

// CommandRunner abstracts the project-standard command execution service without introducing a package cycle.
type CommandRunner interface {
	Start(ctx context.Context, commandID, label, command string, args ...string) (string, error)
	Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error)
	GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool)
}

// ExecCmd abstracts a started command for adapters and tests.
type ExecCmd interface {
	CombinedOutput() ([]byte, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

type filesystemCommandExecutor interface {
	LookPath(command string) (string, error)
	Command(ctx context.Context, name string, args ...string) ExecCmd
}

type defaultFilesystemCommandExecutor struct {
	runner CommandRunner
}

type commandExecutionError struct {
	exitCode int
	err      error
}

func (e *commandExecutionError) Error() string {
	if e == nil || e.err == nil {
		return "command execution failed"
	}
	return e.err.Error()
}

func (e *commandExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *commandExecutionError) ExitCode() int {
	if e == nil {
		return -1
	}
	return e.exitCode
}

type commandRunnerExecCmd struct {
	ctx    context.Context
	runner CommandRunner
	name   string
	args   []string

	stdoutReader *io.PipeReader
	stdoutWriter *io.PipeWriter
	stderrReader *io.PipeReader
	stderrWriter *io.PipeWriter

	mu            sync.Mutex
	done          chan struct{}
	started       bool
	executionID   string
	nextLineIndex int
	waitErr       error
	pipeCloseOnce sync.Once
}

// SetDefaultCommandRunner wires the shared project command execution service into filesystem adapters.
func SetDefaultCommandRunner(runner CommandRunner) (reset func()) {
	defaultCommandRunnerMu.Lock()
	previous := defaultCommandRunner
	defaultCommandRunner = runner
	defaultCommandRunnerMu.Unlock()

	return func() {
		defaultCommandRunnerMu.Lock()
		defaultCommandRunner = previous
		defaultCommandRunnerMu.Unlock()
	}
}

func getDefaultCommandRunner() CommandRunner {
	defaultCommandRunnerMu.RLock()
	defer defaultCommandRunnerMu.RUnlock()
	return defaultCommandRunner
}

func (d *defaultFilesystemCommandExecutor) resolveRunner() CommandRunner {
	if d != nil && d.runner != nil {
		return d.runner
	}
	if runner := getDefaultCommandRunner(); runner != nil {
		return runner
	}
	return getStandaloneCommandRunner()
}

func (d *defaultFilesystemCommandExecutor) LookPath(command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", errors.New("command is empty")
	}
	if !osutil.CommandExists([]string{command}) {
		return "", errors.New("command not found")
	}
	return command, nil
}

func (d *defaultFilesystemCommandExecutor) Command(ctx context.Context, name string, args ...string) ExecCmd {
	return &commandRunnerExecCmd{
		ctx:    ctx,
		runner: d.resolveRunner(),
		name:   name,
		args:   append([]string(nil), args...),
	}
}

func (c *commandRunnerExecCmd) CombinedOutput() ([]byte, error) {
	if c.runner == nil {
		return nil, errors.New("filesystem command runner is not configured")
	}

	snapshot, err := c.runner.Execute(
		c.ctx,
		"filesystem:"+c.name,
		"Filesystem "+c.name,
		c.name,
		c.args...,
	)
	output := joinCommandOutput(snapshot.Lines)
	if err != nil {
		exitCode := snapshot.ExitCode
		if exitCode == 0 {
			exitCode = -1
		}
		return []byte(output), &commandExecutionError{exitCode: exitCode, err: err}
	}

	return []byte(output), nil
}

func joinCommandOutput(lines []dto.CommandOutputLineSnapshot) string {
	if len(lines) == 0 {
		return ""
	}

	var builder strings.Builder
	for idx, line := range lines {
		if idx > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(line.Line)
	}

	return strings.TrimSpace(builder.String())
}

func (c *commandRunnerExecCmd) StdoutPipe() (io.ReadCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return nil, errors.New("stdout pipe must be requested before start")
	}
	if c.stdoutReader != nil {
		return nil, errors.New("stdout pipe already requested")
	}
	c.stdoutReader, c.stdoutWriter = io.Pipe()
	return c.stdoutReader, nil
}

func (c *commandRunnerExecCmd) StderrPipe() (io.ReadCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return nil, errors.New("stderr pipe must be requested before start")
	}
	if c.stderrReader != nil {
		return nil, errors.New("stderr pipe already requested")
	}
	c.stderrReader, c.stderrWriter = io.Pipe()
	return c.stderrReader, nil
}

func (c *commandRunnerExecCmd) Start() error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return errors.New("command already started")
	}
	if c.runner == nil {
		c.mu.Unlock()
		return errors.New("filesystem command runner is not configured")
	}
	c.started = true
	c.done = make(chan struct{})
	c.mu.Unlock()

	executionID, err := c.runner.Start(
		c.ctx,
		"filesystem:"+c.name,
		"Filesystem "+c.name,
		c.name,
		c.args...,
	)
	if err != nil {
		c.setWaitErr(&commandExecutionError{exitCode: -1, err: err})
		c.closePipes()
		close(c.done)
		return c.waitErr
	}

	c.mu.Lock()
	c.executionID = executionID
	c.mu.Unlock()

	go c.forwardOutput(executionID)
	return nil
}

func (c *commandRunnerExecCmd) Wait() error {
	c.mu.Lock()
	done := c.done
	started := c.started
	c.mu.Unlock()
	if !started || done == nil {
		return errors.New("command not started")
	}
	<-done

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.waitErr
}

func (c *commandRunnerExecCmd) forwardOutput(executionID string) {
	defer func() {
		c.closePipes()
		close(c.done)
	}()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			c.setWaitErr(&commandExecutionError{exitCode: -1, err: c.ctx.Err()})
			return
		case <-ticker.C:
			snapshot, ok := c.runner.GetSnapshot(executionID)
			if !ok {
				continue
			}

			c.writePendingLines(snapshot)
			if snapshot.Running {
				continue
			}

			if !snapshot.Success {
				errMessage := snapshot.Error
				if errMessage == "" {
					errMessage = "command execution failed"
				}
				c.setWaitErr(&commandExecutionError{exitCode: snapshot.ExitCode, err: errors.New(errMessage)})
			}
			return
		}
	}
}

func (c *commandRunnerExecCmd) writePendingLines(snapshot dto.CommandExecutionSnapshot) {
	c.mu.Lock()
	start := c.nextLineIndex
	if start >= len(snapshot.Lines) {
		c.mu.Unlock()
		return
	}
	pending := append([]dto.CommandOutputLineSnapshot(nil), snapshot.Lines[start:]...)
	c.nextLineIndex = len(snapshot.Lines)
	stdoutWriter := c.stdoutWriter
	stderrWriter := c.stderrWriter
	c.mu.Unlock()

	for _, line := range pending {
		var writer *io.PipeWriter
		switch line.Channel {
		case dto.CommandOutputChannelStdout:
			writer = stdoutWriter
		case dto.CommandOutputChannelStderr:
			writer = stderrWriter
		default:
			continue
		}
		if writer == nil {
			continue
		}
		_, _ = io.WriteString(writer, line.Line+"\n")
	}
}

func (c *commandRunnerExecCmd) closePipes() {
	c.pipeCloseOnce.Do(func() {
		c.mu.Lock()
		stdoutWriter := c.stdoutWriter
		stderrWriter := c.stderrWriter
		c.stdoutWriter = nil
		c.stderrWriter = nil
		c.mu.Unlock()

		if stdoutWriter != nil {
			_ = stdoutWriter.Close()
		}
		if stderrWriter != nil {
			_ = stderrWriter.Close()
		}
	})
}

func (c *commandRunnerExecCmd) setWaitErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.waitErr = err
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
	aliasNames    []string
	description   string
	exportable    bool
	labelRule     string
	// All adapters currently emit indeterminate progress (999) and do not parse numeric progress.
	isFormatReportProgress bool
	isCheckReportProgress  bool
	alpinePackage          string
	mkfsCommand            string
	fsckCommand            string
	labelCommand           string
	stateCommand           string
	signatures             []dto.FsMagicSignature
	//
	baseTryMountFunc func(source, target, data string, flags uintptr, prepareTarget ...func() error) (*mount.MountPoint, error)
	baseDoMountFunc  func(source, target, fstype, data string, flags uintptr, prepareTarget ...func() error) (*mount.MountPoint, error)
	baseUnmountFunc  func(target string, force, lazy bool) error
	commandExecutor  filesystemCommandExecutor
	getFilesystems   func() ([]string, error) // Optional override for osutil.GetFileSystems, used in command availability checks
	isDeviceMountedF func(device string) bool
}

func newBaseAdapter(
	name, description string,
	exportable bool,
	alpinePackage, mkfsCommand, fsckCommand, labelCommand, stateCommand, labelRule string,
	signatures []dto.FsMagicSignature,
	aliasNames ...string,
) baseAdapter {
	return baseAdapter{
		name:                   name,
		aliasNames:             slices.Clone(aliasNames),
		description:            description,
		exportable:             exportable,
		labelRule:              labelRule,
		isFormatReportProgress: false,
		isCheckReportProgress:  false,
		alpinePackage:          alpinePackage,
		mkfsCommand:            mkfsCommand,
		fsckCommand:            fsckCommand,
		labelCommand:           labelCommand,
		stateCommand:           stateCommand,
		signatures:             signatures,
		baseTryMountFunc:       mount.TryMount,
		baseDoMountFunc:        mount.Mount,
		baseUnmountFunc:        mount.Unmount,
		commandExecutor:        &defaultFilesystemCommandExecutor{},
		getFilesystems:         osutil.GetFileSystems,
	}
}

func (b *baseAdapter) SetCommandRunner(runner CommandRunner) (reset func()) {
	original := b.commandExecutor
	b.commandExecutor = &defaultFilesystemCommandExecutor{runner: runner}
	return func() {
		b.commandExecutor = original
	}
}

// commandExists checks if a command is available in the system PATH
func (b *baseAdapter) commandExists(command string) bool {
	_, err := b.commandExecutor.LookPath(command)
	if err != nil {
		if strings.Contains(err.Error(), "command not found") {
			return false
		}
		tlog.Warn("Error checking command existence", "command", command, "error", err)
		return false
	}
	return true
}

// runCommand executes a command and returns the output
func (b *baseAdapter) runCommand(ctx context.Context, name string, args ...string) (string, int, errors.E) {
	cmd := b.commandExecutor.Command(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	exitCode := 0

	if err != nil {
		exitCode = -1
		if exitErr, ok := err.(interface{ ExitCode() int }); ok {
			exitCode = exitErr.ExitCode()
		}
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "executable file not found") ||
			strings.Contains(err.Error(), "no such file or directory") ||
			strings.Contains(err.Error(), "cannot find the file") ||
			strings.Contains(err.Error(), "command not found") ||
			strings.Contains(err.Error(), "not configured") {
			tlog.ErrorContext(ctx, "Error executing command", "command", name, "args", args, "exitCode", exitCode, "Output", string(output), "error", err)
			return strings.TrimSpace(string(output)), exitCode, errors.WithDetails(err, "Command", name, "Args", strings.Join(args, " "), "error", "command not found")
		}
		return strings.TrimSpace(string(output)), exitCode, nil
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

func (b *baseAdapter) isDeviceMounted(device string) bool {
	if b.isDeviceMountedF != nil {
		return b.isDeviceMountedF(device)
	}

	mounts, err := procfs.GetMounts()
	if err != nil {
		return false
	}

	for _, mount := range mounts {
		if mount.Source == device {
			return true
		}
	}

	return false
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
			CanMount:               false,
			IsExportable:           b.exportable,
			IsFormatReportProgress: b.isFormatReportProgress,
			IsCheckReportProgress:  b.isCheckReportProgress,
			LabelRule:              b.labelRule,
			AlpinePackage:          b.alpinePackage,
			MissingTools:           []string{b.mkfsCommand, b.fsckCommand, b.labelCommand, b.stateCommand},
		}
	}

	support := dto.FilesystemSupport{
		CanMount:               true, // Most filesystems can be mounted if kernel supports them
		IsExportable:           b.exportable,
		IsFormatReportProgress: b.isFormatReportProgress,
		IsCheckReportProgress:  b.isCheckReportProgress,
		LabelRule:              b.labelRule,
		AlpinePackage:          b.alpinePackage,
		MissingTools:           []string{},
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

// IsExportable returns whether the filesystem can be exported via Linux NFS.
func (b *baseAdapter) IsExportable(ctx context.Context) bool {
	_ = ctx

	return b.exportable
}

// GetLinuxFsModule returns the Linux filesystem module/fstype name to use for mounting
func (b *baseAdapter) GetLinuxFsModule() string {
	if b.linuxFsModule != "" {
		return b.linuxFsModule
	}
	return b.name
}

// GetAliasNames returns other filesystem names that should resolve to this adapter.
func (b *baseAdapter) GetAliasNames() []string {
	aliases := slices.Clone(b.aliasNames)
	linuxModule := b.GetLinuxFsModule()
	if linuxModule != "" && linuxModule != b.name && !slices.Contains(aliases, linuxModule) {
		aliases = append(aliases, linuxModule)
	}

	return aliases
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

	cmd := b.commandExecutor.Command(ctx, command, args...)

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
				if exitErr, ok := cmdErr.(interface{ ExitCode() int }); ok {
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
