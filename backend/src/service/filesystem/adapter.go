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

	// GetDescription returns a human-readable description of the filesystem
	GetDescription() string

	// GetMountFlags returns filesystem-specific mount flags
	GetMountFlags() []dto.MountFlag

	// IsSupported checks if the filesystem is supported on the system
	// and returns detailed support information
	IsSupported(ctx context.Context) (FilesystemSupport, errors.E)

	// Format formats a device with this filesystem
	Format(ctx context.Context, device string, options FormatOptions) errors.E

	// Check runs filesystem check (fsck)
	Check(ctx context.Context, device string, options CheckOptions) (CheckResult, errors.E)

	// GetLabel retrieves the filesystem label
	GetLabel(ctx context.Context, device string) (string, errors.E)

	// SetLabel sets the filesystem label
	SetLabel(ctx context.Context, device string, label string) errors.E

	// GetState returns the current state of the filesystem if supported
	GetState(ctx context.Context, device string) (FilesystemState, errors.E)
}

// FilesystemSupport contains information about filesystem support on the system
type FilesystemSupport struct {
	// CanMount indicates if the filesystem can be mounted
	CanMount bool `json:"canMount"`

	// CanFormat indicates if the filesystem can be formatted (mkfs available)
	CanFormat bool `json:"canFormat"`

	// CanCheck indicates if the filesystem can be checked (fsck available)
	CanCheck bool `json:"canCheck"`

	// CanSetLabel indicates if the filesystem label can be changed
	CanSetLabel bool `json:"canSetLabel"`

	// CanGetState indicates if filesystem state can be retrieved
	CanGetState bool `json:"canGetState"`

	// AlpinePackage is the Alpine Linux package name that provides support
	AlpinePackage string `json:"alpinePackage,omitempty"`

	// MissingTools lists the tools that are not available
	MissingTools []string `json:"missingTools,omitempty"`
}

// FormatOptions contains options for formatting a filesystem
type FormatOptions struct {
	// Label is the filesystem label to set during formatting
	Label string `json:"label,omitempty"`

	// Force forces formatting even if the device appears to be in use
	Force bool `json:"force,omitempty"`

	// AdditionalOptions contains filesystem-specific options
	AdditionalOptions map[string]string `json:"additionalOptions,omitempty"`
}

// CheckOptions contains options for checking a filesystem
type CheckOptions struct {
	// AutoFix automatically fixes errors if possible
	AutoFix bool `json:"autoFix,omitempty"`

	// Force forces check even if filesystem appears clean
	Force bool `json:"force,omitempty"`

	// Verbose enables verbose output
	Verbose bool `json:"verbose,omitempty"`
}

// CheckResult contains the result of a filesystem check operation
type CheckResult struct {
	// Success indicates if the check completed successfully
	Success bool `json:"success"`

	// ErrorsFound indicates if errors were found
	ErrorsFound bool `json:"errorsFound"`

	// ErrorsFixed indicates if errors were fixed (when AutoFix is enabled)
	ErrorsFixed bool `json:"errorsFixed"`

	// Message contains a human-readable message about the check result
	Message string `json:"message,omitempty"`

	// ExitCode is the exit code from the check command
	ExitCode int `json:"exitCode"`
}

// FilesystemState represents the current state of a filesystem
type FilesystemState struct {
	// IsClean indicates if the filesystem is in a clean state
	IsClean bool `json:"isClean"`

	// IsMounted indicates if the filesystem is currently mounted
	IsMounted bool `json:"isMounted"`

	// HasErrors indicates if the filesystem has errors
	HasErrors bool `json:"hasErrors"`

	// StateDescription is a human-readable description of the state
	StateDescription string `json:"stateDescription,omitempty"`

	// AdditionalInfo contains filesystem-specific state information
	AdditionalInfo map[string]interface{} `json:"additionalInfo,omitempty"`
}
