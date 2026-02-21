package filesystem

import (
	"context"
	"os/exec"

	"github.com/dianlight/srat/internal/osutil"
	"github.com/u-root/u-root/pkg/mount"
)

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

// SetExecOpsForTesting allows overriding exec operations for tests
func (b *baseAdapter) SetExecOpsForTesting(
	lookPath func(string) (string, error),
	command func(context.Context, string, ...string) ExecCmd,
) (reset func()) {
	if lookPath != nil {
		b.execLookPath = lookPath
	}
	if command != nil {
		b.execCommand = command
	}
	return func() {
		b.execLookPath = exec.LookPath
		b.execCommand = func(ctx context.Context, name string, args ...string) ExecCmd {
			return &realExecCmd{cmd: exec.CommandContext(ctx, name, args...)}
		}
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
