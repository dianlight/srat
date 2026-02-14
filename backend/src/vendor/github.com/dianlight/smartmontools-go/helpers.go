package smartmontools

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// isATADevice checks if a device type is ATA-based (ata, sat, sata, etc.)
func isATADevice(deviceType string) bool {
	if deviceType == "" {
		return false
	}
	dt := strings.ToLower(deviceType)
	return strings.Contains(dt, "ata") || strings.Contains(dt, "sat") || dt == "scsi"
}

// determineDiskType determines the type of disk based on available information.
// Optimized to check conditions in order of likelihood and cost.
func determineDiskType(info *SMARTInfo) string {
	// Check for NVMe devices first
	if info.Device.Type == "nvme" || info.NvmeSmartHealth != nil || info.NvmeControllerCapabilities != nil {
		return "NVMe"
	}

	// Check rotation rate for ATA/SATA devices (most reliable indicator)
	if info.RotationRate != nil {
		if *info.RotationRate == 0 {
			return "SSD"
		}
		return "HDD"
	}

	// Check device type from smartctl
	deviceType := strings.ToLower(info.Device.Type)
	if strings.Contains(deviceType, "nvme") {
		return "NVMe"
	}

	if strings.Contains(deviceType, "sata") || strings.Contains(deviceType, "ata") || strings.Contains(deviceType, "sat") {
		// If we have ATA SMART data but no rotation rate, try to infer
		if info.AtaSmartData != nil {
			// Look for SSD-specific attributes
			for _, attr := range info.AtaSmartData.Table {
				if attr.ID == SmartAttrSSDLifeLeft || attr.ID == SmartAttrSandForceInternal || attr.ID == SmartAttrTotalLBAsWritten {
					return "SSD"
				}
			}
		}
	}

	// If we can't determine, return Unknown
	return "Unknown"
}

// ensureCompatibleSmartctl runs "smartctl -V" and checks the version is supported.
// The library depends on JSON output (-j), which requires smartctl >= 7.0.
func ensureCompatibleSmartctl(smartctlPath string) error {
	out, err := exec.Command(smartctlPath, "-V").Output()
	if err != nil {
		return fmt.Errorf("failed to check smartctl version: %w", err)
	}
	major, minor, err := parseSmartctlVersion(string(out))
	if err != nil {
		return fmt.Errorf("unable to parse smartctl version: %w", err)
	}
	const minMajor, minMinor = 7, 0
	if major < minMajor || (major == minMajor && minor < minMinor) {
		return fmt.Errorf("unsupported smartctl version %d.%d; require >= %d.%d", major, minor, minMajor, minMinor)
	}
	return nil
}

// parseSmartctlVersion extracts the major and minor version numbers from
// the output of "smartctl -V". Expected forms include lines like:
//
//	"smartctl 7.3 2022-02-28 r5338 ..." or "smartctl 7.5 ...".
func parseSmartctlVersion(output string) (int, int, error) {
	// Find first occurrence of "smartctl X.Y"
	re := regexp.MustCompile(`(?m)\bsmartctl\s+(\d+)\.(\d+)\b`)
	m := re.FindStringSubmatch(output)
	if len(m) != 3 {
		return 0, 0, fmt.Errorf("version pattern not found in output")
	}
	// Convert captures to ints using strconv for better performance
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse major version: %w", err)
	}
	minor, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse minor version: %w", err)
	}
	return major, minor, nil
}
