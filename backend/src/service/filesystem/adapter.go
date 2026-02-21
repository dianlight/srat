package filesystem

import (
	"context"

	"github.com/dianlight/srat/dto"
	"github.com/u-root/u-root/pkg/mount"
	"gitlab.com/tozd/go/errors"
)

// FilesystemAdapter defines the interface for filesystem-specific operations.
// Each supported filesystem type implements this interface to provide
// filesystem-specific functionality like formatting, checking, labeling, etc.
type FilesystemAdapter interface {
	// GetName returns the filesystem type name (e.g., "ext4", "ntfs", "xfs")
	GetName() string

	// GetLinuxFsModule returns the Linux filesystem module/fstype name to use for mounting
	GetLinuxFsModule() string

	// GetDescription returns a human-readable description of the filesystem
	GetDescription() string

	// GetMountFlags returns filesystem-specific mount flags
	GetMountFlags() []dto.MountFlag

	// Mount mounts a source device/file to target using filesystem-specific behavior.
	// If fsType is empty, adapter should try auto-detection when supported.
	// prepareTarget is an optional callback used to prepare the mount target directory.
	Mount(ctx context.Context, source, target, fsType, data string, flags uintptr, prepareTarget func() error) (*mount.MountPoint, errors.E)

	// Unmount unmounts the target using filesystem-specific behavior.
	Unmount(ctx context.Context, target string, force, lazy bool) errors.E

	// IsSupported checks if the filesystem is supported on the system
	// and returns detailed support information
	IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E)

	// Format formats a device with this filesystem
	// progress callback receives status updates (start/running/success/failure),
	// percentual (0-100, or 999 for unsupported), and notes
	Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E

	// Check runs filesystem check (fsck)
	// progress callback receives status updates (start/running/success/failure),
	// percentual (0-100, or 999 for unsupported), and notes
	Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E)

	// GetLabel retrieves the filesystem label
	GetLabel(ctx context.Context, device string) (string, errors.E)

	// SetLabel sets the filesystem label
	SetLabel(ctx context.Context, device string, label string) errors.E

	// GetState returns the current state of the filesystem if supported
	GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E)

	// IsDeviceSupported checks if a device can be mounted with this filesystem
	// by examining magic numbers or using filesystem-specific detection
	IsDeviceSupported(ctx context.Context, devicePath string) (bool, errors.E)

	// GetFsSignatureMagic returns the magic number signatures for this filesystem
	GetFsSignatureMagic() []dto.FsMagicSignature

	// Additional test for moc
	SetMountOpsForTesting(
		tryMount func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
		mountFn func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
		unmountFn func(target string, force, lazy bool) error,
	) (reset func())

	SetExecOpsForTesting(
		lookPath func(string) (string, error),
		command func(ctx context.Context, cmd string, args ...string) ExecCmd,
	) (reset func())

	SetGetFilesystemsForTesting(getFilesystems func() ([]string, error)) (reset func())
}
