package converter

import (
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/mount"
)

// goverter:converter
// goverter:output:file ./ha_supervisor_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type HaSupervisorToDto interface {
	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:skipCopySameType
	// goverter:map Name Share
	// goverter:map Usage Usage | hAMountUsageToMountUsage
	// goverter:ignore Password Path Port ReadOnly Server State Type Username
	SharedResourceToMount(source dto.SharedResource, target *mount.Mount) error
}
