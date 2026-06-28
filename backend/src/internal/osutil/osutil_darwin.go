//go:build darwin

package osutil

import (
	"context"
	"fmt"
)

// IsKernelModuleLoaded returns false on macOS (no Linux kernel modules).
// Returns nil error since "module not loaded" is a valid state, not an error.
func IsKernelModuleLoaded(moduleName string) (bool, error) {
	return false, nil
}

// CreateBlockDevice is not supported on macOS.
func CreateBlockDevice(ctx context.Context, device string) error {
	return fmt.Errorf("CreateBlockDevice is not supported on macOS")
}

// GetFileSystems returns an empty list on macOS.
// The override mechanism (set via MockFileSystems) is respected for testing,
// allowing tests that mock filesystem lists to work on macOS.
func GetFileSystems() ([]string, error) {
	filesystemsOverrideMu.RLock()
	override := filesystemsOverride
	filesystemsOverrideMu.RUnlock()

	if override != nil {
		return override()
	}

	return nil, nil
}

// LoadMountInfo returns an empty list on macOS since /proc is not available.
// This allows code that depends on mount info to function gracefully during
// development on macOS. The override mechanism (setMountsOverride) is still
// respected for testing.
func LoadMountInfo() ([]*MountInfoEntry, error) {
	overrideMu.RLock()
	override := mountsOverride
	overrideMu.RUnlock()

	if override != nil {
		mounts, err := override()
		if err != nil {
			return nil, err
		}
		return convertFromProcs(mounts), nil
	}

	return []*MountInfoEntry{}, nil
}
