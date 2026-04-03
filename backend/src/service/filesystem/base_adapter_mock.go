package filesystem

import (
	"context"

	"github.com/dianlight/srat/internal/osutil"
	"github.com/u-root/u-root/pkg/mount"
)

type testFilesystemCommandExecutor struct {
	lookPath func(string) (string, error)
	command  func(context.Context, string, ...string) ExecCmd
}

func (t *testFilesystemCommandExecutor) LookPath(command string) (string, error) {
	return t.lookPath(command)
}

func (t *testFilesystemCommandExecutor) Command(ctx context.Context, name string, args ...string) ExecCmd {
	return t.command(ctx, name, args...)
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

// SetExecOpsForTesting allows overriding command execution operations for tests.
func (b *baseAdapter) SetExecOpsForTesting(
	lookPath func(string) (string, error),
	command func(context.Context, string, ...string) ExecCmd,
) (reset func()) {
	original := b.commandExecutor

	testExecutor := &testFilesystemCommandExecutor{}
	if lookPath != nil {
		testExecutor.lookPath = lookPath
	} else {
		testExecutor.lookPath = b.commandExecutor.LookPath
	}
	if command != nil {
		testExecutor.command = command
	} else {
		testExecutor.command = b.commandExecutor.Command
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
