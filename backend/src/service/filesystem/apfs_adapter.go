package filesystem

import (
	"context"
	"strings"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// ApfsAdapter implements FilesystemAdapter for APFS filesystems (read-only)
type ApfsAdapter struct {
	baseAdapter
}

// NewApfsAdapter creates a new ApfsAdapter instance
func NewApfsAdapter() FilesystemAdapter {
	return &ApfsAdapter{
		baseAdapter: baseAdapter{
			name:          "apfs",
			description:   "Apple File System (read-only)",
			alpinePackage: "apfs-fuse",
			mkfsCommand:   "",
			fsckCommand:   "",
			labelCommand:  "",
			signatures: []dto.FsMagicSignature{
				{Offset: 0x20, Magic: []byte{'N', 'X', 'S', 'B'}}, // APFS container superblock
			},
		},
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
	support.CanFormat = false                       // APFS formatting not supported on Linux
	support.CanCheck = false                        // apfsutil provides read-only access, cannot check filesystem
	support.CanSetLabel = false                     // APFS label modification not supported on Linux
	support.CanGetState = commandExists("apfsutil") // apfsutil can provide filesystem state/info

	if !support.CanGetState {
		support.MissingTools = append(support.MissingTools, "apfsutil")
	}

	return support, nil
}

// Format formats a device with APFS filesystem
func (a *ApfsAdapter) Format(ctx context.Context, device string, options dto.FormatOptions, progress dto.ProgressCallback) errors.E {
	if progress != nil {
		progress("failure", 0, []string{"APFS formatting is not supported on Linux"})
	}
	return errors.Errorf("APFS formatting is not supported on Linux. APFS is read-only on this system")
}

// Check runs filesystem check on an APFS device
func (a *ApfsAdapter) Check(ctx context.Context, device string, options dto.CheckOptions, progress dto.ProgressCallback) (dto.CheckResult, errors.E) {
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
	outputMount, _, _ := runCommand(ctx, "mount")
	state.IsMounted = strings.Contains(outputMount, device)

	// Add note about read-only status
	state.AdditionalInfo["readOnly"] = true
	state.AdditionalInfo["note"] = "APFS is read-only on Linux. No format/check tools available."

	return state, nil
}
