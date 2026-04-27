package filesystem

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/commandexec"
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
	defaultCommandRunner   commandexec.Executor = commandexec.NewCommandExecutor(nil)
)

// SetDefaultCommandRunner wires the shared project command execution service into filesystem adapters.
func SetDefaultCommandRunner(runner commandexec.Executor) (reset func()) {
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

func getDefaultCommandRunner() commandexec.Executor {
	defaultCommandRunnerMu.RLock()
	runner := defaultCommandRunner
	defaultCommandRunnerMu.RUnlock()
	if runner != nil {
		return runner
	}

	defaultCommandRunnerMu.Lock()
	defer defaultCommandRunnerMu.Unlock()
	if defaultCommandRunner == nil {
		defaultCommandRunner = commandexec.NewCommandExecutor(nil)
	}
	return defaultCommandRunner
}

func (b *baseAdapter) resolveCommandExecutor() commandexec.Executor {
	if b != nil && b.commandExecutor != nil {
		return b.commandExecutor
	}
	return getDefaultCommandRunner()
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
	commandExecutor  commandexec.Executor
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
		getFilesystems:         osutil.GetFileSystems,
	}
}

// SetCommandRunner temporarily overrides the shared command executor used by this adapter.
// It returns a reset function, which is mainly used by tests.
func (b *baseAdapter) SetCommandRunner(runner commandexec.Executor) (reset func()) {
	original := b.commandExecutor
	b.commandExecutor = runner
	return func() {
		b.commandExecutor = original
	}
}

// commandExists checks if a command is available in the system PATH
func (b *baseAdapter) commandExists(command string) bool {
	executor := b.resolveCommandExecutor()
	_, err := executor.LookPath(command)
	if err != nil {
		if strings.Contains(err.Error(), "command not found") {
			return false
		}
		tlog.Warn("Error checking command existence", "command", command, "error", err)
		return false
	}
	return true
}

// runCommand executes a command and returns the output.
func (b *baseAdapter) runCommand(ctx context.Context, name string, args ...string) (string, int, errors.E) {
	return b.runCommandMode(ctx, false, name, args...)
}

func (b *baseAdapter) runCommandMode(ctx context.Context, quiet bool, name string, args ...string) (string, int, errors.E) {
	executor := b.resolveCommandExecutor()
	var (
		snapshot dto.CommandExecutionSnapshot
		err      error
	)
	if quiet {
		snapshot, err = executor.ExecuteQuiet(ctx, "filesystem:"+name, "Filesystem "+name, name, args...)
	} else {
		snapshot, err = executor.Execute(ctx, "filesystem:"+name, "Filesystem "+name, name, args...)
	}
	output := commandexec.JoinChannelOutput(snapshot.Lines)
	exitCode := snapshot.ExitCode

	if err != nil {
		if exitCode == 0 {
			exitCode = -1
		}
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "executable file not found") ||
			strings.Contains(err.Error(), "no such file or directory") ||
			strings.Contains(err.Error(), "cannot find the file") ||
			strings.Contains(err.Error(), "command not found") ||
			strings.Contains(err.Error(), "not configured") {
			tlog.ErrorContext(ctx, "Error executing command", "command", name, "args", args, "exitCode", exitCode, "Output", output, "error", err)
			return output, exitCode, errors.WithDetails(err, "Command", name, "Args", strings.Join(args, " "), "error", "command not found")
		}
		return output, exitCode, nil
	}

	return output, exitCode, nil
}

// runCommandCached executes a command and caches the result by command name and exact args.
// This limits command execution to at most once per cache TTL for each unique command+args key.
func (b *baseAdapter) runCommandCached(ctx context.Context, name string, args ...string) (string, int, errors.E) {
	return b.runCommandCachedMode(ctx, false, name, args...)
}

// runCommandCachedQuiet executes a cached command without emitting command lifecycle events.
func (b *baseAdapter) runCommandCachedQuiet(ctx context.Context, name string, args ...string) (string, int, errors.E) {
	return b.runCommandCachedMode(ctx, true, name, args...)
}

func (b *baseAdapter) runCommandCachedMode(ctx context.Context, quiet bool, name string, args ...string) (string, int, errors.E) {
	cacheKey := b.commandCacheKey(name, args...)

	if cached, ok := commandResultCache.Get(cacheKey); ok {
		if result, castOk := cached.(cachedCommandResult); castOk {
			return result.Output, result.ExitCode, result.Error
		}

		commandResultCache.Delete(cacheKey)
	}

	output, exitCode, err := b.runCommandMode(ctx, quiet, name, args...)
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
	stdoutChan := make(chan string, 100)
	stderrChan := make(chan string, 100)
	resultChan := make(chan CommandResult, 1)

	executor := b.resolveCommandExecutor()
	executionID, err := executor.Start(ctx, "filesystem:"+command, "Filesystem "+command, command, args...)
	if err != nil {
		close(stdoutChan)
		close(stderrChan)
		resultChan <- CommandResult{
			ExitCode: -1,
			Error:    errors.WithDetails(err, "command", command, "args", strings.Join(args, " ")),
		}
		close(resultChan)
		return stdoutChan, stderrChan, resultChan
	}

	go func() {
		defer close(stdoutChan)
		defer close(stderrChan)
		defer close(resultChan)

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		stdoutLines := make([]string, 0)
		stderrLines := make([]string, 0)
		nextLineIndex := 0
		for {
			select {
			case <-ctx.Done():
				resultChan <- CommandResult{
					ExitCode: -1,
					Error:    errors.WithDetails(ctx.Err(), "command", command, "error", "context cancelled"),
				}
				return
			case <-ticker.C:
				snapshot, ok := executor.GetSnapshot(executionID)
				if !ok {
					continue
				}
				if nextLineIndex > len(snapshot.Lines) {
					nextLineIndex = len(snapshot.Lines)
				}

				for _, line := range snapshot.Lines[nextLineIndex:] {
					var target chan string
					switch line.Channel {
					case dto.CommandOutputChannelStdout:
						stdoutLines = append(stdoutLines, line.Line)
						target = stdoutChan
					case dto.CommandOutputChannelStderr:
						stderrLines = append(stderrLines, line.Line)
						target = stderrChan
					default:
						continue
					}

					select {
					case <-ctx.Done():
						resultChan <- CommandResult{
							ExitCode: -1,
							Error:    errors.WithDetails(ctx.Err(), "command", command, "error", "context cancelled"),
						}
						return
					case target <- line.Line:
					}
				}
				nextLineIndex = len(snapshot.Lines)

				if snapshot.Running {
					continue
				}

				result := CommandResult{ExitCode: snapshot.ExitCode}
				if !snapshot.Success {
					exitCode := snapshot.ExitCode
					if exitCode == 0 {
						exitCode = -1
					}
					stderrText := strings.TrimSpace(strings.Join(stderrLines, "\n"))
					stdoutText := strings.TrimSpace(strings.Join(stdoutLines, "\n"))
					errMessage := strings.TrimSpace(snapshot.Error)
					if errMessage == "" {
						errMessage = "command execution failed"
					}
					if stderrText != "" && !strings.Contains(errMessage, stderrText) {
						errMessage += ": " + stderrText
					} else if stdoutText != "" && !strings.Contains(errMessage, stdoutText) {
						errMessage += ": " + stdoutText
					}
					result.ExitCode = exitCode
					result.Error = errors.WithDetails(
						errors.New(errMessage),
						"command", command,
						"args", strings.Join(args, " "),
						"exitCode", exitCode,
						"stdout", stdoutText,
						"stderr", stderrText,
					)
				}

				resultChan <- result
				return
			}
		}
	}()

	return stdoutChan, stderrChan, resultChan
}

// drainCommandOutput reads stdoutChan and stderrChan concurrently, collecting
// output lines and invoking progress on each. If notes is non-nil, lines are
// appended to *notes and the accumulated slice is passed to progress (cumulative
// pattern). If notes is nil, a fresh single-element slice is used per line.
func drainCommandOutput(
	stdoutChan, stderrChan <-chan string,
	progress dto.ProgressCallback,
	pct int,
	notes *[]string,
) (outputLines, errorLines []string) {
	var wg sync.WaitGroup
	wg.Go(func() {
		for line := range stdoutChan {
			outputLines = append(outputLines, line)
			if progress != nil {
				if notes != nil {
					*notes = append(*notes, line)
					progress("running", pct, *notes)
				} else {
					progress("running", pct, []string{line})
				}
			}
		}
	})
	wg.Go(func() {
		for line := range stderrChan {
			errorLines = append(errorLines, line)
			if progress != nil {
				if notes != nil {
					*notes = append(*notes, "ERROR: "+line)
					progress("running", pct, *notes)
				} else {
					progress("running", pct, []string{"ERROR: " + line})
				}
			}
		}
	})
	wg.Wait()
	return
}
