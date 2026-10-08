package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/commandexec"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

type recordingExecutor struct {
	mu         sync.Mutex
	timeouts   []time.Duration
	calls      int
	snapshot   dto.CommandExecutionSnapshot
	err        error
	commandIDs []string
}

func (f *recordingExecutor) record(ctx context.Context, commandID string) {
	timeout := time.Duration(0)
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.timeouts = append(f.timeouts, timeout)
	f.calls++
	f.commandIDs = append(f.commandIDs, commandID)
}

func (f *recordingExecutor) Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	f.record(ctx, commandID)
	if f.err != nil {
		return f.snapshot, f.err
	}
	return dto.CommandExecutionSnapshot{ExecutionID: "test-exec", CommandID: commandID, Success: true}, nil
}

func (f *recordingExecutor) LastTimeout() time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.timeouts) == 0 {
		return 0
	}
	return f.timeouts[len(f.timeouts)-1]
}

func (f *recordingExecutor) LookPath(command string) (string, error) { return command, nil }
func (f *recordingExecutor) Start(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	return "test-exec", nil
}
func (f *recordingExecutor) StartQuiet(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	return "test-exec", nil
}
func (f *recordingExecutor) StartWithInput(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (string, error) {
	return "test-exec", nil
}
func (f *recordingExecutor) StartWithInputQuiet(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (string, error) {
	return "test-exec", nil
}
func (f *recordingExecutor) ExecuteQuiet(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return f.Execute(ctx, commandID, label, command, args...)
}
func (f *recordingExecutor) ExecuteWithInput(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return f.Execute(ctx, commandID, label, command, args...)
}
func (f *recordingExecutor) ExecuteWithInputQuiet(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return f.Execute(ctx, commandID, label, command, args...)
}
func (f *recordingExecutor) GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool) {
	return dto.CommandExecutionSnapshot{}, false
}

var _ commandexec.Executor = (*recordingExecutor)(nil)

func smbdLikeConfig() serviceConfig {
	return serviceConfig{
		Name:                 "smbd",
		SoftResetServiceMask: dto.DataDirtyTracker{Users: true, Shares: true},
		HardResetServiceMask: dto.DataDirtyTracker{Settings: true},
		Managed:              true,
		HardResetCommand:     []string{"s6-svc", "-rwR", "/run/s6-rc/servicedirs/smbd"},
		SoftResetCommand:     []string{"smbcontrol", "smbd", "reload-config"},
		StartCommand:         []string{"s6-svc", "-uwU", "/run/s6-rc/servicedirs/smbd"},
	}
}

func TestRestartOneService_HardRestartUsesDedicatedTimeout(t *testing.T) {
	fake := &recordingExecutor{}
	svc := &ServerService{commandRunner: fake}
	dirty := dto.DataDirtyTracker{Settings: true}
	procStatus := &dto.ProcessStatus{Pid: 123}

	err := svc.restartOneService(context.Background(), "smbd", smbdLikeConfig(), procStatus, dirty)
	require.NoError(t, err)
	require.Equal(t, 1, fake.calls)
	got := fake.LastTimeout()
	require.Greater(t, got, 60*time.Second, "hard restart must not reuse the old 30s shared timeout")
	require.LessOrEqual(t, got, 125*time.Second)
}

func TestRestartOneService_SoftRestartUsesShortTimeout(t *testing.T) {
	fake := &recordingExecutor{}
	svc := &ServerService{commandRunner: fake}
	dirty := dto.DataDirtyTracker{Shares: true}
	procStatus := &dto.ProcessStatus{Pid: 123}

	err := svc.restartOneService(context.Background(), "smbd", smbdLikeConfig(), procStatus, dirty)
	require.NoError(t, err)
	require.Equal(t, 1, fake.calls)
	got := fake.LastTimeout()
	require.Greater(t, got, 20*time.Second)
	require.LessOrEqual(t, got, 35*time.Second)
}

func TestRestartOneService_SequentialHardRestartsKeepFullTimeout(t *testing.T) {
	fake := &recordingExecutor{}
	svc := &ServerService{commandRunner: fake}
	dirty := dto.DataDirtyTracker{Settings: true}
	procStatus := &dto.ProcessStatus{Pid: 123}

	require.NoError(t, svc.restartOneService(context.Background(), "smbd", smbdLikeConfig(), procStatus, dirty))
	require.NoError(t, svc.restartOneService(context.Background(), "nmbd", smbdLikeConfig(), procStatus, dirty))
	require.Equal(t, 2, fake.calls)
	for i, timeout := range fake.timeouts {
		require.Greater(t, timeout, 60*time.Second, "restart %d must get a fresh hard-restart timeout", i)
		require.LessOrEqual(t, timeout, 125*time.Second)
	}
}

func TestRestartOneService_DeadPidForcesHardRestart(t *testing.T) {
	fake := &recordingExecutor{}
	svc := &ServerService{commandRunner: fake}
	procStatus := &dto.ProcessStatus{Pid: 0}

	require.NoError(t, svc.restartOneService(context.Background(), "smbd", smbdLikeConfig(), procStatus, dto.DataDirtyTracker{}))
	require.Equal(t, 1, fake.calls)
	require.Equal(t, "service-hard-restart-smbd", fake.commandIDs[0])
	got := fake.LastTimeout()
	require.Greater(t, got, 60*time.Second)
	require.LessOrEqual(t, got, 125*time.Second)
}

func TestRestartOneService_NoRestartNeededSkipsCommand(t *testing.T) {
	fake := &recordingExecutor{}
	svc := &ServerService{commandRunner: fake}
	procStatus := &dto.ProcessStatus{Pid: 123}

	require.NoError(t, svc.restartOneService(context.Background(), "smbd", smbdLikeConfig(), procStatus, dto.DataDirtyTracker{}))
	require.Equal(t, 0, fake.calls)
}

func TestRestartOneService_HardRestartErrorIncludesOutput(t *testing.T) {
	fake := &recordingExecutor{
		snapshot: dto.CommandExecutionSnapshot{
			ExecutionID: "exec-1",
			Success:     false,
			Lines: []dto.CommandOutputLineSnapshot{
				{Channel: dto.CommandOutputChannelStderr, Line: "s6-svc failed: timeout"},
			},
		},
		err: errors.New("exit status 1"),
	}
	svc := &ServerService{commandRunner: fake}
	dirty := dto.DataDirtyTracker{Settings: true}
	procStatus := &dto.ProcessStatus{Pid: 123}

	err := svc.restartOneService(context.Background(), "smbd", smbdLikeConfig(), procStatus, dirty)
	require.Error(t, err)
	require.Contains(t, err.Error(), "hard restart")
	require.Contains(t, err.Error(), "s6-svc failed: timeout")
}

func TestRestartOneService_SoftRestartErrorPropagates(t *testing.T) {
	fake := &recordingExecutor{err: errors.New("connection refused")}
	svc := &ServerService{commandRunner: fake}
	dirty := dto.DataDirtyTracker{Shares: true}
	procStatus := &dto.ProcessStatus{Pid: 123}

	err := svc.restartOneService(context.Background(), "smbd", smbdLikeConfig(), procStatus, dirty)
	require.Error(t, err)
	require.Contains(t, err.Error(), "soft restart")
}

func TestRestartOneService_StartSkippedWhenCommandMissing(t *testing.T) {
	fake := &recordingExecutor{}
	svc := &ServerService{commandRunner: fake}
	cfg := smbdLikeConfig()
	cfg.StartCommand = []string{"definitely-missing-binary-xyz"}

	require.NoError(t, svc.restartOneService(context.Background(), "smbd", cfg, nil, dto.DataDirtyTracker{}))
	require.Equal(t, 0, fake.calls)
}

func TestRestartOneService_StartErrorPropagates(t *testing.T) {
	fake := &recordingExecutor{err: errors.New("start failed")}
	svc := &ServerService{commandRunner: fake}
	cfg := smbdLikeConfig()
	cfg.StartCommand = []string{"sh", "-c", "true"}

	err := svc.restartOneService(context.Background(), "smbd", cfg, nil, dto.DataDirtyTracker{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "starting service")
	require.Equal(t, 1, fake.calls)
	got := fake.LastTimeout()
	require.Greater(t, got, 20*time.Second)
	require.LessOrEqual(t, got, 35*time.Second)
}

func TestRunCommandWithRunner_PreservesOutputOnError(t *testing.T) {
	fake := &recordingExecutor{
		snapshot: dto.CommandExecutionSnapshot{
			ExecutionID: "exec-1",
			Success:     false,
			Lines: []dto.CommandOutputLineSnapshot{
				{Channel: dto.CommandOutputChannelStderr, Line: "partial log line"},
			},
		},
		err: context.DeadlineExceeded,
	}
	svc := &ServerService{commandRunner: fake}

	out, err := svc.runCommandWithRunner(context.Background(), "id", "label", []string{"sleep", "60"})
	require.Error(t, err)
	require.Contains(t, out, "partial log line")
}

func TestRestartServerServices_SkipsMissingStartCommands(t *testing.T) {
	orig := serviceConfigMap
	serviceConfigMap = map[string]serviceConfig{
		"testsvc": {
			Name:                 "testsvc",
			Managed:              true,
			StartCommand:         []string{"definitely-missing-binary-xyz"},
			SoftResetServiceMask: dto.DataDirtyTracker{Shares: true},
			HardResetServiceMask: dto.DataDirtyTracker{Settings: true},
		},
		"unmanaged": {
			Name:    "unmanaged",
			Managed: false,
		},
	}
	defer func() { serviceConfigMap = orig }()

	fake := &recordingExecutor{}
	svc := &ServerService{ctx: context.Background(), commandRunner: fake, eventBus: events.NewEventBus(context.Background())}

	require.NoError(t, svc.restartServerServices(context.Background(), dto.DataDirtyTracker{}))
	require.Equal(t, 0, fake.calls)
}

func TestRestartServerServices_StartErrorAbortsRestart(t *testing.T) {
	orig := serviceConfigMap
	serviceConfigMap = map[string]serviceConfig{
		"testsvc": {
			Name:                 "testsvc",
			Managed:              true,
			StartCommand:         []string{"sh", "-c", "true"},
			SoftResetServiceMask: dto.DataDirtyTracker{Shares: true},
			HardResetServiceMask: dto.DataDirtyTracker{Settings: true},
		},
	}
	defer func() { serviceConfigMap = orig }()

	fake := &recordingExecutor{err: errors.New("start failed")}
	svc := &ServerService{ctx: context.Background(), commandRunner: fake, eventBus: events.NewEventBus(context.Background())}

	err := svc.restartServerServices(context.Background(), dto.DataDirtyTracker{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "starting service")
}
