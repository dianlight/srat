package filesystem

import (
	"context"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// ZfsAdapter implements FilesystemAdapter for ZFS filesystems.
//
// ZFS management is pool-centric (zpool/zfs), while SRAT filesystem APIs are
// partition/device-centric. For this reason SRAT currently supports ZFS mount
// capability only; format/check/label operations are intentionally unsupported.
type ZfsAdapter struct {
	baseAdapter
}

// NewZfsAdapter creates a new ZfsAdapter instance.
func NewZfsAdapter() FilesystemAdapter {
	return &ZfsAdapter{
		baseAdapter: newBaseAdapter(
			"zfs",
			"ZFS Filesystem",
			true,
			"zfs",
			"",
			"",
			"",
			"",
			nil,
		),
	}
}

// GetMountFlags returns ZFS-specific mount flags.
func (a *ZfsAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "zfsutil", Description: "Enable Linux ZFS mount helper integration"},
		{Name: "xattr", Description: "Enable extended attributes"},
		{Name: "noxattr", Description: "Disable extended attributes"},
		{Name: "atime", Description: "Update inode access times"},
		{Name: "noatime", Description: "Do not update inode access times"},
	}
}

// IsSupported checks if ZFS is supported on the system.
func (a *ZfsAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	support := a.checkCommandAvailability()

	// ZFS operations in SRAT are intentionally limited to mount support.
	support.CanFormat = false
	support.CanCheck = false
	support.CanSetLabel = false
	support.CanGetState = true

	return support, nil
}

// Format formats a device with ZFS filesystem.
func (a *ZfsAdapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	defer a.invalidateCommandResultCache()

	_ = ctx
	_ = device
	_ = options

	if progress != nil {
		progress("failure", 0, []string{"ZFS format is not supported by SRAT"})
	}

	return errors.Errorf("ZFS format is not supported by SRAT. ZFS provisioning is pool-level and managed outside partition format operations")
}

// Check runs filesystem check on a ZFS device.
func (a *ZfsAdapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	defer a.invalidateCommandResultCache()

	_ = ctx
	_ = device
	_ = options

	if progress != nil {
		progress("failure", 0, []string{"ZFS check is not supported by SRAT"})
	}

	result := dto.CheckResult{
		Success:     false,
		ErrorsFound: false,
		ErrorsFixed: false,
		Message:     "ZFS check is not supported by SRAT",
		ExitCode:    1,
	}

	return result, errors.Errorf("ZFS check is not supported by SRAT. ZFS health checks are pool-level and not available via partition check API")
}

// GetLabel retrieves the ZFS filesystem label.
func (a *ZfsAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	_ = ctx
	_ = device

	return "", errors.Errorf("ZFS label retrieval is not supported by SRAT. ZFS naming is dataset/pool-level")
}

// SetLabel sets the ZFS filesystem label.
func (a *ZfsAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	defer a.invalidateCommandResultCache()

	_ = ctx
	_ = device
	_ = label

	return errors.Errorf("ZFS label modification is not supported by SRAT. ZFS naming is dataset/pool-level")
}

// GetState returns the state of a ZFS filesystem.
func (a *ZfsAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	_ = ctx

	state := dto.FilesystemState{
		AdditionalInfo: map[string]any{
			"note": "ZFS health and repair are pool-level operations and are not exposed through SRAT partition state APIs",
		},
		IsClean:          false,
		HasErrors:        false,
		StateDescription: "Unknown (pool-level checks required)",
	}

	state.IsMounted = a.isDeviceMounted(device)

	return state, nil
}
