package filesystem

import (
	"context"
	"os/exec"
	"strings"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// baseAdapter provides common functionality for all filesystem adapters
type baseAdapter struct {
	name          string
	description   string
	alpinePackage string
	mkfsCommand   string
	fsckCommand   string
	labelCommand  string
	signatures    []dto.FsMagicSignature
}

// commandExists checks if a command is available in the system PATH
func commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// runCommand executes a command and returns the output
func runCommand(ctx context.Context, name string, args ...string) (string, int, errors.E) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	exitCode := 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, errors.WithDetails(err, "Command", name, "Args", strings.Join(args, " "))
		}
	}

	return strings.TrimSpace(string(output)), exitCode, nil
}

// checkCommandAvailability checks if required commands are available
func (b *baseAdapter) checkCommandAvailability() dto.FilesystemSupport {
	support := dto.FilesystemSupport{
		CanMount:      true, // Most filesystems can be mounted if kernel supports them
		AlpinePackage: b.alpinePackage,
		MissingTools:  []string{},
	}

	if b.mkfsCommand != "" {
		support.CanFormat = commandExists(b.mkfsCommand)
		if !support.CanFormat {
			support.MissingTools = append(support.MissingTools, b.mkfsCommand)
		}
	}

	if b.fsckCommand != "" {
		support.CanCheck = commandExists(b.fsckCommand)
		if !support.CanCheck {
			support.MissingTools = append(support.MissingTools, b.fsckCommand)
		}
	}

	if b.labelCommand != "" {
		support.CanSetLabel = commandExists(b.labelCommand)
		if !support.CanSetLabel {
			support.MissingTools = append(support.MissingTools, b.labelCommand)
		}
	}

	// For now, state checking is not supported by default
	support.CanGetState = false

	return support
}

// GetName returns the filesystem type name
func (b *baseAdapter) GetName() string {
	return b.name
}

// GetDescription returns the filesystem description
func (b *baseAdapter) GetDescription() string {
	return b.description
}

// GetFsSignatureMagic returns the magic number signatures for this filesystem
func (b *baseAdapter) GetFsSignatureMagic() []dto.FsMagicSignature {
	return b.signatures
}

// IsDeviceSupported checks if a device can be mounted with this filesystem
// by examining magic numbers. This is a default implementation that uses
// the magic signature detection system.
func (b *baseAdapter) IsDeviceSupported(ctx context.Context, devicePath string) (bool, errors.E) {
	// Check if device matches any of the adapter's signatures
	return checkDeviceMatchesSignatures(devicePath, b.signatures)
}
