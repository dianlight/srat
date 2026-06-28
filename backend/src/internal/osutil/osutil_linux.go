//go:build linux

package osutil

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// IsKernelModuleLoaded checks if a kernel module is currently loaded.
// It reads /proc/modules to determine if the module exists.
func IsKernelModuleLoaded(moduleName string) (bool, error) {
	file, err := os.Open("/proc/modules")
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Module format: modulename size refcount dependencies state offset
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == moduleName {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

// CreateBlockDevice creates a loop block device node using mknod.
// Returns nil if the device already exists.
func CreateBlockDevice(ctx context.Context, device string) error {
	// Check if the device already exists
	if _, err := os.Stat(device); !os.IsNotExist(err) {
		slog.DebugContext(ctx, "Loop device already exists", "device", device)
		return nil
	}

	// Extract major and minor numbers from device name
	major, minor, err := extractMajorMinor(device)
	if err != nil {
		return fmt.Errorf("failed to extract major and minor numbers: %w", err)
	}

	// Create the block device using mknod syscall
	dev := (major << 8) | minor
	if err := syscall.Mknod(device, syscall.S_IFBLK|0660, dev); err != nil {
		return fmt.Errorf("failed to create block device: %w", err)
	}

	return nil
}

// extractMajorMinor parses a loop device path (e.g. /dev/loop0) to extract major and minor numbers.
// The major number for loop devices is always 7.
func extractMajorMinor(device string) (int, int, error) {
	re := regexp.MustCompile(`/dev/loop(\d+)`)
	matches := re.FindStringSubmatch(device)
	if len(matches) != 2 {
		return 0, 0, fmt.Errorf("invalid device format: %s", device)
	}

	minor, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to convert minor number: %w", err)
	}

	// The major number for loop devices is 7
	major := 7
	return major, minor, nil
}

// Source: https://github.com/shirou/gopsutil
func GetFileSystems() ([]string, error) {
	filesystemsOverrideMu.RLock()
	override := filesystemsOverride
	filesystemsOverrideMu.RUnlock()

	if override != nil {
		return override()
	}

	filename := "/proc/filesystems"
	lines, err := readLinesOffsetN(filename, 0, -1)
	if err != nil {
		return nil, err
	}
	var ret []string
	seen := make(map[string]struct{})
	allowedNodev := map[string]struct{}{
		"zfs":     {},
		"fuse":    {},
		"fuse3":   {},
		"fuseblk": {},
	}
	for _, line := range lines {
		cleaned := strings.TrimSpace(line)
		if cleaned == "" {
			continue
		}
		if !strings.HasPrefix(cleaned, "nodev") {
			if _, exists := seen[cleaned]; !exists {
				ret = append(ret, cleaned)
				seen[cleaned] = struct{}{}
			}
			continue
		}
		fields := strings.Fields(cleaned)
		if len(fields) != 2 {
			continue
		}
		fsType := strings.TrimSpace(fields[1])
		if _, ok := allowedNodev[fsType]; ok {
			if _, exists := seen[fsType]; !exists {
				ret = append(ret, fsType)
				seen[fsType] = struct{}{}
			}
		}
	}

	return ret, nil
}
