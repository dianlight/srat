package converter

import (
	"github.com/dianlight/srat/dto"
	"github.com/jaypipes/ghw/pkg/net"
)

// goverter:converter
// goverter:output:file ./net_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type NetToDto interface {
	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	NetInfoToNetworkInfo(source net.Info, target *dto.NetworkInfo) error
}
