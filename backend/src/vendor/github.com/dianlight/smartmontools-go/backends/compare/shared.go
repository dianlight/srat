package compare

import smtypes "github.com/dianlight/smartmontools-go/types"

// Shared interface aliases keep the compare backend decoupled from the root package.
type (
	LogAdapter       = smtypes.LogAdapter
	Backend          = smtypes.Backend
	DiscoveryBackend = smtypes.DiscoveryBackend
)

// Shared type aliases reuse the module's SMART domain model in the compare backend.
type (
	Device          = smtypes.Device
	SMARTInfo       = smtypes.SMARTInfo
	SmartctlInfo    = smtypes.SmartctlInfo
	SelfTestInfo    = smtypes.SelfTestInfo
	DiscoveryResult = smtypes.DiscoveryResult
)
