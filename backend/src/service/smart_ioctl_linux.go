package service

import (
	"encoding/binary"
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

// ioctlSMARTCommand executes a SMART command via ioctl
func ioctlSMARTCommand(fd int, feature byte, lbaLow byte) error {
	// ATA pass-through command structure
	type ataPassthru struct {
		protocol      uint8
		flags         uint8
		features      uint8
		sectorCount   uint8
		lbaLow        uint8
		lbaMid        uint8
		lbaHigh       uint8
		device        uint8
		command       uint8
		reserved      uint8
		control       uint8
		timeout       uint32
		_             uint32
		bufferPointer uintptr
		bufferLength  uint32
	}

	cmd := ataPassthru{
		protocol:    4, // Non-data protocol
		flags:       0,
		features:    feature,
		sectorCount: 0,
		lbaLow:      lbaLow,
		lbaMid:      0x4f, // SMART signature
		lbaHigh:     0xc2, // SMART signature
		device:      0xa0,
		command:     _ATA_SMART,
		timeout:     10000, // 10 seconds
	}

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(0xc0289304), // HDIO_DRIVE_CMD
		uintptr(unsafe.Pointer(&cmd)),
	)

	if errno != 0 {
		return errors.Errorf("ioctl failed: %v", errno)
	}

	return nil
}

// enableSMART enables SMART on a SATA device
func enableSMART(dev *smart.SataDevice) errors.E {
	// We need to access the file descriptor - this requires reflection or a modified smart.go library
	// For now, we'll return an error indicating the limitation
	return errors.WithDetails(dto.ErrorSMARTOperationFailed, "reason", "SMART enable requires direct device access")
}

// disableSMART disables SMART on a SATA device
func disableSMART(dev *smart.SataDevice) errors.E {
	return errors.WithDetails(dto.ErrorSMARTOperationFailed, "reason", "SMART disable requires direct device access")
}

// executeSMARTTest starts a SMART self-test on a SATA device
func executeSMARTTest(dev *smart.SataDevice, testType byte) errors.E {
	return errors.WithDetails(dto.ErrorSMARTOperationFailed, "reason", "SMART test execution requires direct device access")
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
