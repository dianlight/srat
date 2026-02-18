package filesystem

import (
	"context"

	"github.com/dianlight/srat/dto"
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
}
