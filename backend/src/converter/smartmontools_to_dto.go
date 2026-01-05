package converter

import (
	"github.com/dianlight/smartmontools-go"
	"github.com/dianlight/srat/dto"
)

// goverter:converter
// goverter:output:file ./smartmontools_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:extend intToSmartRangeValue
// goverter:default:update
type SmartMonToolsToDto interface {
	// goverter:map SmartSupport.Available Supported
	// goverter:ignore DiskId
	SmartMonToolsSmartInfoToSmartInfo(source *smartmontools.SMARTInfo) (target *dto.SmartInfo, err error)

	// goverter:ignore Additional
	// goverter:map SmartSupport.Enabled Enabled
	// goverter:map SmartStatus.Running IsTestRunning
	// goverter:map SmartStatus.Passed IsTestPassed
	// goverter:map SmartStatus.Damaged IsInWarning
	// goverter:map SmartStatus.Critical IsInDanger
	// goverter:map PowerOnTime PowerOnHours
	SmartMonToolsSmartInfoToSmartStatus(source *smartmontools.SMARTInfo) (target *dto.SmartStatus, err error)

	// goverter:ignore Min Max OvertempCounter
	// goverter:map Current Value
	smartMonToolsTemperatureToSmartTempValue(source *smartmontools.Temperature) (target dto.SmartTempValue, err error)

	// goverter:ignore Code Min Worst Thresholds
	// goverter:map Hours Value
	smartMonToolsPowerOnTimeToSmartRangeValue(source *smartmontools.PowerOnTime) (target dto.SmartRangeValue, err error)
}

func intToSmartRangeValue(source int) (target dto.SmartRangeValue, err error) {
	target = dto.SmartRangeValue{
		Value: source,
	}
	return target, nil
}
