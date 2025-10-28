// go:build !linux
//go:build !linux

package service

import (
	"github.com/dianlight/srat/dto"
)

// checkSMARTHealth evaluates SMART attributes (cross-platform)
func checkSMARTHealth(smartInfo *dto.SmartInfo, _ interface{}, _ interface{}) *dto.SmartHealthStatus {
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
