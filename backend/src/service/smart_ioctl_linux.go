package service

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/anatol/smart.go"
	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/sys/unix"
)

// ATA SMART command constants
const (
	_ATA_SMART                       = 0xb0
	_SMART_ENABLE_OPERATIONS         = 0xd8
	_SMART_DISABLE_OPERATIONS        = 0xd9
	_SMART_EXECUTE_OFFLINE_IMMEDIATE = 0xd4
	_SMART_RETURN_STATUS             = 0xda
)

// SMART test types
const (
	_SMART_SHORT_SELFTEST      = 0x01
	_SMART_LONG_SELFTEST       = 0x02
	_SMART_CONVEYANCE_SELFTEST = 0x03
	_SMART_ABORT_SELFTEST      = 0x7f
)

// getDevicePathForIOCTL converts a symlink device path to the actual device name
// For example: /dev/disk/by-id/ata-KINGSTON... -> /dev/sda
func getDevicePathForIOCTL(devicePath string) (string, error) {
	// If it's already a simple device path like /dev/sda, use it directly
	if strings.HasPrefix(devicePath, "/dev/sd") || strings.HasPrefix(devicePath, "/dev/nvme") {
		return devicePath, nil
	}

	// For symlinks like /dev/disk/by-id/..., resolve the entire symlink chain
	realPath, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		// If symlink resolution fails, try to use the path as-is
		return devicePath, nil
	}

	return realPath, nil
}

// ioctlSMARTCommand executes a SMART command via ioctl using HDIO_DRIVE_CMD
// It opens the device directly to ensure proper permissions and access
func ioctlSMARTCommand(devicePath string, feature byte, lbaLow byte) error {
	// Get the actual device path (resolve symlinks)
	actualPath, err := getDevicePathForIOCTL(devicePath)
	if err != nil {
		return errors.Wrap(err, "failed to resolve device path")
	}

	// Open the device with read/write access
	// SMART commands require write access to the device
	fd, err := os.OpenFile(actualPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			return errors.WithDetails(dto.ErrorSMARTOperationFailed, "reason", "permission denied opening device for SMART operations")
		}
		return errors.Wrapf(err, "failed to open device %s for SMART operations", actualPath)
	}
	defer fd.Close()

	// Get the file descriptor
	fileFD := int(fd.Fd())

	// Create a buffer for the HDIO_DRIVE_CMD ioctl
	// The structure expected by kernel is: unsigned char args[4+512]
	var cmdBuffer [516]byte

	// Set up the 4-byte ATA command
	// Format: [0] = command (0xB0), [1] = feature, [2] = lba_low, [3] = 0x4f (SMART signature)
	cmdBuffer[0] = _ATA_SMART // ATA command 0xB0
	cmdBuffer[1] = feature    // Features register
	cmdBuffer[2] = lbaLow     // Sector count or LBA Low (depends on command)
	cmdBuffer[3] = 0x4f       // SMART signature byte (LBA Mid for SMART operations)

	// HDIO_DRIVE_CMD ioctl number
	// Defined as: _IOC(_IOC_READ | _IOC_WRITE, 'h', 31, 4)
	// Which evaluates to 0x0000031f on most systems
	const HDIO_DRIVE_CMD = 0x0000031f

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fileFD),
		uintptr(HDIO_DRIVE_CMD),
		uintptr(unsafe.Pointer(&cmdBuffer[0])),
	)

	if errno != 0 {
		// Common errno values for SMART commands:
		// EACCES (13): Permission denied
		// EIO (5): Input/output error - device doesn't support SMART or device error
		// EOPNOTSUPP (95): Operation not supported
		// ENODEV (19): No such device
		return errors.Errorf("SMART ioctl command failed on %s: errno=%d", actualPath, errno)
	}

	return nil
}

// enableSMART enables SMART on a SATA device
func enableSMART(dev *smart.SataDevice, devicePath string) errors.E {
	if err := ioctlSMARTCommand(devicePath, _SMART_ENABLE_OPERATIONS, 0); err != nil {
		return errors.Wrap(err, "failed to enable SMART")
	}

	return nil
}

// disableSMART disables SMART on a SATA device
func disableSMART(dev *smart.SataDevice, devicePath string) errors.E {
	if err := ioctlSMARTCommand(devicePath, _SMART_DISABLE_OPERATIONS, 0); err != nil {
		return errors.Wrap(err, "failed to disable SMART")
	}

	return nil
}

// executeSMARTTest starts a SMART self-test on a SATA device
func executeSMARTTest(dev *smart.SataDevice, testType byte, devicePath string) errors.E {
	if err := ioctlSMARTCommand(devicePath, _SMART_EXECUTE_OFFLINE_IMMEDIATE, testType); err != nil {
		return errors.Wrap(err, "failed to execute SMART self-test")
	}

	return nil
}

// checkSMARTStatus queries whether SMART is enabled on a SATA device
// Returns error if SMART is not supported or if the query fails
func checkSMARTStatus(dev *smart.SataDevice, devicePath string) (bool, errors.E) {
	// Get the actual device path (resolve symlinks)
	actualPath, err := getDevicePathForIOCTL(devicePath)
	if err != nil {
		return false, errors.Wrap(err, "failed to resolve device path")
	}

	// Open the device with read-only access (status check doesn't need write access)
	fd, err := os.OpenFile(actualPath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsPermission(err) {
			return false, errors.WithDetails(dto.ErrorSMARTOperationFailed, "reason", "permission denied opening device for SMART status check")
		}
		return false, errors.Wrapf(err, "failed to open device %s for SMART status check", actualPath)
	}
	defer fd.Close()

	// Get the file descriptor
	fileFD := int(fd.Fd())

	// Create a buffer for the HDIO_DRIVE_CMD ioctl
	// The structure expected by kernel is: unsigned char args[4+512]
	var cmdBuffer [516]byte

	// Set up the 4-byte ATA command for status check
	// Format: [0] = command (0xB0), [1] = feature (0xDA), [2] = 0, [3] = 0x4f (SMART signature)
	cmdBuffer[0] = _ATA_SMART           // ATA command 0xB0
	cmdBuffer[1] = _SMART_RETURN_STATUS // Feature: 0xDA for status check
	cmdBuffer[2] = 0                    // LBA Low: 0
	cmdBuffer[3] = 0x4f                 // SMART signature byte (LBA Mid for SMART operations)

	// HDIO_DRIVE_CMD ioctl number
	const HDIO_DRIVE_CMD = 0x0000031f

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fileFD),
		uintptr(HDIO_DRIVE_CMD),
		uintptr(unsafe.Pointer(&cmdBuffer[0])),
	)

	if errno != 0 {
		// Check for specific error codes that indicate SMART is not supported/enabled
		if errno == 5 { // EIO - Input/output error, often means device doesn't support SMART
			return false, nil
		}
		// SMART not supported or command failed
		return false, errors.Errorf("SMART status check failed on %s: errno=%d", actualPath, errno)
	}

	// If command succeeds, SMART is enabled
	// The returned status bytes in cmdBuffer[4] indicate SMART status
	// cmdBuffer[4] = LBA Low (returns 0xf1 if SMART failed, 0x2c if passed)
	// For a simple check, if the ioctl succeeds, SMART is supported and enabled
	return true, nil
}

// parseSelfTestLog parses the SMART self-test log to get test status
func parseSelfTestLog(log interface{}) (*dto.SmartTestStatus, errors.E) {
	// This would parse the actual self-test log from smart.go
	// For now, return a basic implementation
	return &dto.SmartTestStatus{
		Status:   "idle",
		TestType: "none",
	}, nil
}

// checkSMARTHealth evaluates SMART attributes to determine disk health
func checkSMARTHealth(smartInfo *dto.SmartInfo, thresholds map[uint8]uint8, attrs map[uint8]interface{}) *dto.SmartHealthStatus {
	health := &dto.SmartHealthStatus{
		Passed:        true,
		OverallStatus: "healthy",
	}

	failingAttrs := []string{}

	// Check if any critical attributes are below threshold
	for code, attr := range smartInfo.Additional {
		if attr.Thresholds > 0 && attr.Value < attr.Thresholds {
			failingAttrs = append(failingAttrs, code)
			health.Passed = false
		}
	}

	// Check power cycle count threshold
	if smartInfo.PowerCycleCount.Thresholds > 0 &&
		smartInfo.PowerCycleCount.Value < smartInfo.PowerCycleCount.Thresholds {
		failingAttrs = append(failingAttrs, "PowerCycleCount")
		health.Passed = false
	}

	// Check power on hours threshold
	if smartInfo.PowerOnHours.Thresholds > 0 &&
		smartInfo.PowerOnHours.Value < smartInfo.PowerOnHours.Thresholds {
		failingAttrs = append(failingAttrs, "PowerOnHours")
		health.Passed = false
	}

	if len(failingAttrs) > 0 {
		health.FailingAttributes = failingAttrs
		health.OverallStatus = "failing"
	}

	return health
}

// Suppress unused import warnings
var (
	_ = binary.LittleEndian
	_ = unsafe.Pointer(nil)
)
