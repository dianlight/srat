package filesystem

import (
	"context"
	"fmt"
	"strings"

	"github.com/dianlight/srat/dto"
	"github.com/u-root/u-root/pkg/mount"
	"gitlab.com/tozd/go/errors"
)

// ApfsAdapter implements FilesystemAdapter for APFS filesystems (read-only)
type ApfsAdapter struct {
	baseAdapter
}

// NewApfsAdapter creates a new ApfsAdapter instance
func NewApfsAdapter() FilesystemAdapter {
	return &ApfsAdapter{
		baseAdapter: newBaseAdapter(
			"apfs",
			"Apple File System (read-only)",
			"apfs-fuse",
			"",
			"",
			"",
			"apfsutil",
			[]dto.FsMagicSignature{
				{Offset: 0x20, Magic: []byte{'N', 'X', 'S', 'B'}}, // APFS container superblock
			},
		),
	}
}

// GetMountFlags returns APFS-specific mount flags
func (a *ApfsAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{
		{Name: "uid", Description: "User ID for files", NeedsValue: true, ValueDescription: "User ID", ValueValidationRegex: `^\d+$`},
		{Name: "gid", Description: "Group ID for files", NeedsValue: true, ValueDescription: "Group ID", ValueValidationRegex: `^\d+$`},
		{Name: "vol", Description: "Volume index to mount", NeedsValue: true, ValueDescription: "Volume index (0-based)", ValueValidationRegex: `^\d+$`},
	}
}

// IsSupported checks if APFS is supported on the system
func (a *ApfsAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	// APFS is read-only on Linux via apfs-fuse package
	// apfsutil provides information/metadata access but not format/check/modify operations
	support := a.checkCommandAvailability()
	support.CanMount = a.commandExists("apfs-fuse")
	if !support.CanMount {
		support.MissingTools = append(support.MissingTools, "apfs-fuse")
	}
	support.CanFormat = false   // APFS formatting not supported on Linux
	support.CanCheck = false    // apfsutil provides read-only access, cannot check filesystem
	support.CanSetLabel = false // APFS label modification not supported on Linux

	return support, nil
}

// Mount mounts APFS with apfs-fuse.
func (a *ApfsAdapter) Mount(
	ctx context.Context,
	source, target, fsType, data string,
	flags uintptr,
	prepareTarget func() error,
) (*mount.MountPoint, errors.E) {
	_ = fsType
	_ = flags

	if prepareTarget != nil {
		if err := prepareTarget(); err != nil {
			return nil, errors.WithDetails(err, "Target", target, "Message", "failed to prepare APFS mount target")
		}
	}

	args := make([]string, 0, 4)
	if strings.TrimSpace(data) != "" {
		args = append(args, "-o", data)
	}
	args = append(args, source, target)

	output, exitCode, err := a.runCommand(ctx, "apfs-fuse", args...)
	if err != nil {
		return nil, errors.WithDetails(err,
			"Source", source,
			"Target", target,
			"Data", data,
			"ExitCode", exitCode,
			"Output", output,
		)
	}
	if exitCode != 0 {
		return nil, errors.WithDetails(errors.New("apfs-fuse mount failed"),
			"Source", source,
			"Target", target,
			"Data", data,
			"ExitCode", exitCode,
			"Output", output,
		)
	}

	return &mount.MountPoint{
		Path:   target,
		Device: source,
		FSType: "apfs",
		Flags:  flags,
		Data:   data,
	}, nil
}

// Unmount unmounts APFS using FUSE-first semantics.
func (a *ApfsAdapter) Unmount(ctx context.Context, target string, force, lazy bool) errors.E {
	if a.commandExists("fusermount3") {
		output, exitCode, err := a.runCommand(ctx, "fusermount3", "-u", target)
		if err == nil && exitCode == 0 {
			return nil
		}

		if err != nil {
			err = errors.WithDetails(err, "Output", output, "ExitCode", exitCode, "Target", target)
			_ = err // fallback to generic unmount below
		}
	}

	if err := a.baseAdapter.Unmount(ctx, target, force, lazy); err != nil {
		return errors.WithDetails(err,
			"Target", target,
			"Message", fmt.Sprintf("APFS unmount failed via fusermount3 and generic unmount fallback: %v", err),
		)
	}

	return nil
}

// Format formats a device with APFS filesystem
func (a *ApfsAdapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("failure", 0, []string{"APFS formatting is not supported on Linux"})
	}
	return errors.Errorf("APFS formatting is not supported on Linux. APFS is read-only on this system")
}

// Check runs filesystem check on an APFS device
func (a *ApfsAdapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
	defer a.invalidateCommandResultCache()

	if progress != nil {
		progress("failure", 0, []string{"APFS filesystem checking is not supported on Linux"})
	}
	result := dto.CheckResult{
		Success:     false,
		ErrorsFound: false,
		ErrorsFixed: false,
		Message:     "APFS filesystem checking is not supported on Linux",
		ExitCode:    1,
	}
	return result, errors.Errorf("APFS filesystem checking is not supported on Linux. APFS is read-only on this system")
}

// GetLabel retrieves the APFS filesystem label
func (a *ApfsAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	return "", errors.Errorf("APFS label retrieval is not supported on Linux. APFS is read-only on this system")
}

// SetLabel sets the APFS filesystem label
func (a *ApfsAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	defer a.invalidateCommandResultCache()

	return errors.Errorf("APFS label modification is not supported on Linux. APFS is read-only on this system")
}

// GetState returns the state of an APFS filesystem
func (a *ApfsAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	state := dto.FilesystemState{
		AdditionalInfo:   make(map[string]interface{}),
		IsClean:          true,  // Assume clean since we can't check
		HasErrors:        false, // Assume no errors since we can't check
		StateDescription: "Read-only (no Linux tools)",
	}

	// Check if filesystem is mounted
	outputMount, _, _ := a.runCommandCached(ctx, "mount")
	state.IsMounted = strings.Contains(outputMount, device)

	// Add note about read-only status
	state.AdditionalInfo["readOnly"] = true
	state.AdditionalInfo["note"] = "APFS is read-only on Linux. No format/check tools available."

	return state, nil
}
