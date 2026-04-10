package filesystem

import (
	"context"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/u-root/u-root/pkg/mount"
)

type testFilesystemCommandExecutor struct {
	lookPath         func(string) (string, error)
	start            func(context.Context, string, string, string, ...string) (string, error)
	execute          func(context.Context, string, string, string, ...string) (dto.CommandExecutionSnapshot, error)
	executeWithInput func(context.Context, string, string, string, string, ...string) (dto.CommandExecutionSnapshot, error)
	getSnapshot      func(string) (dto.CommandExecutionSnapshot, bool)
}

func (t *testFilesystemCommandExecutor) LookPath(command string) (string, error) {
	return t.lookPath(command)
}

func (t *testFilesystemCommandExecutor) Start(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	return t.start(ctx, commandID, label, command, args...)
}

func (t *testFilesystemCommandExecutor) StartWithInput(ctx context.Context, commandID, label, _ string, command string, args ...string) (string, error) {
	return t.Start(ctx, commandID, label, command, args...)
}

func (t *testFilesystemCommandExecutor) Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return t.execute(ctx, commandID, label, command, args...)
}

func (t *testFilesystemCommandExecutor) ExecuteWithInput(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	if t.executeWithInput != nil {
		return t.executeWithInput(ctx, commandID, label, stdinContent, command, args...)
	}
	return t.Execute(ctx, commandID, label, command, args...)
}

func (t *testFilesystemCommandExecutor) GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool) {
	return t.getSnapshot(executionID)
}

// SetMountOpsForTesting allows overriding generic mount operations for tests.
func (b *baseAdapter) SetMountOpsForTesting(
	tryMount func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	mountFn func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	unmountFn func(target string, force, lazy bool) error,
) (reset func()) {
	if tryMount != nil {
		b.baseTryMountFunc = tryMount
	}
	if mountFn != nil {
		b.baseDoMountFunc = mountFn
	}
	if unmountFn != nil {
		b.baseUnmountFunc = unmountFn
	}
	return func() {
		b.baseTryMountFunc = mount.TryMount
		b.baseDoMountFunc = mount.Mount
		b.baseUnmountFunc = mount.Unmount
	}
}

// SetExecOpsForTesting allows overriding command discovery operations for tests.
func (b *baseAdapter) SetExecOpsForTesting(lookPath func(string) (string, error)) (reset func()) {
	original := b.commandExecutor
	current := b.resolveCommandExecutor()

	testExecutor := &testFilesystemCommandExecutor{
		lookPath:    current.LookPath,
		start:       current.Start,
		execute:     current.Execute,
		getSnapshot: current.GetSnapshot,
	}
	if lookPath != nil {
		testExecutor.lookPath = lookPath
	}

	b.commandExecutor = testExecutor

	return func() {
		b.commandExecutor = original
	}
}

func (b *baseAdapter) SetGetFilesystemsForTesting(getFilesystems func() ([]string, error)) (reset func()) {
	if getFilesystems != nil {
		b.getFilesystems = getFilesystems
	}
	return func() {
		b.getFilesystems = osutil.GetFileSystems
	}
}

func (b *baseAdapter) SetIsDeviceMountedForTesting(isDeviceMounted func(device string) bool) (reset func()) {
	b.isDeviceMountedF = isDeviceMounted
	return func() {
		b.isDeviceMountedF = nil
	}
}
