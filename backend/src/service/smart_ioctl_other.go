// go:build !linux
//go:build !linux

package service

import (
	"github.com/anatol/smart.go"
	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// enableSMART enables SMART on a SATA device (not supported on non-Linux platforms)
func enableSMART(dev *smart.SataDevice) errors.E {
	return errors.WithDetails(dto.ErrorSMARTNotSupported, "reason", "SMART enable not supported on this platform")
}

// disableSMART disables SMART on a SATA device (not supported on non-Linux platforms)
func disableSMART(dev *smart.SataDevice) errors.E {
	return errors.WithDetails(dto.ErrorSMARTNotSupported, "reason", "SMART disable not supported on this platform")
}

// executeSMARTTest starts a SMART self-test (not supported on non-Linux platforms)
func executeSMARTTest(dev *smart.SataDevice, testType byte) errors.E {
	return errors.WithDetails(dto.ErrorSMARTNotSupported, "reason", "SMART test execution not supported on this platform")
}

// parseSelfTestLog parses the SMART self-test log (not supported on non-Linux platforms)
func parseSelfTestLog(log interface{}) (*dto.SmartTestStatus, errors.E) {
	return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "reason", "SMART test log parsing not supported on this platform")
}

// checkSMARTHealth evaluates SMART attributes (cross-platform)
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

	if len(failingAttrs) > 0 {
		health.FailingAttributes = failingAttrs
		health.OverallStatus = "failing"
	}

	return health
}
