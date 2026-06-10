package exec

import smtypes "github.com/dianlight/smartmontools-go/types"

// Shared interface aliases keep the exec backend decoupled from the root package.
type (
	LogAdapter       = smtypes.LogAdapter
	Backend          = smtypes.Backend
	DiscoveryBackend = smtypes.DiscoveryBackend
	Commander        = smtypes.Commander
	Cmd              = smtypes.Cmd
)

// Shared type aliases reuse the module's SMART domain model in the exec backend.
type (
	Device                     = smtypes.Device
	SMARTInfo                  = smtypes.SMARTInfo
	NvmeControllerCapabilities = smtypes.NvmeControllerCapabilities
	NvmeSmartHealth            = smtypes.NvmeSmartHealth
	NvmeSmartTestLog           = smtypes.NvmeSmartTestLog
	UserCapacity               = smtypes.UserCapacity
	SmartStatus                = smtypes.SmartStatus
	SmartSupport               = smtypes.SmartSupport
	AtaSmartData               = smtypes.AtaSmartData
	StatusField                = smtypes.StatusField
	OfflineDataCollection      = smtypes.OfflineDataCollection
	PollingMinutes             = smtypes.PollingMinutes
	SelfTest                   = smtypes.SelfTest
	Capabilities               = smtypes.Capabilities
	SelfTestInfo               = smtypes.SelfTestInfo
	NvmeOptionalAdminCommands  = smtypes.NvmeOptionalAdminCommands
	CapabilitiesOutput         = smtypes.CapabilitiesOutput
	SmartAttribute             = smtypes.SmartAttribute
	Flags                      = smtypes.Flags
	Raw                        = smtypes.Raw
	Temperature                = smtypes.Temperature
	PowerOnTime                = smtypes.PowerOnTime
	Message                    = smtypes.Message
	SmartctlInfo               = smtypes.SmartctlInfo
	ProgressCallback           = smtypes.ProgressCallback
	ExitCodeInfo               = smtypes.ExitCodeInfo
	DiscoveryResult            = smtypes.DiscoveryResult
)

// Shared SMART attribute constants used by exec backend helpers.
const (
	SmartAttrSSDLifeUsed       = smtypes.SmartAttrSSDLifeUsed
	SmartAttrWearLevelingCount = smtypes.SmartAttrWearLevelingCount
	SmartAttrSSDLifeLeft       = smtypes.SmartAttrSSDLifeLeft
	SmartAttrSandForceInternal = smtypes.SmartAttrSandForceInternal
	SmartAttrTotalLBAsWritten  = smtypes.SmartAttrTotalLBAsWritten
)

var validSelfTestTypes = smtypes.ValidSelfTestTypes

func populateSelfTestInfo(info *SelfTestInfo, ata *AtaSmartData, nvmeCaps *NvmeControllerCapabilities, nvmeOptional *NvmeOptionalAdminCommands) {
	smtypes.PopulateSelfTestInfo(info, ata, nvmeCaps, nvmeOptional)
}
