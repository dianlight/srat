package smartmontools

import smtypes "github.com/dianlight/smartmontools-go/internal/types"

// Device represents a storage device.
type Device = smtypes.Device

// NvmeControllerCapabilities represents NVMe controller capabilities.
type NvmeControllerCapabilities = smtypes.NvmeControllerCapabilities

// NvmeSmartHealth represents NVMe SMART health information.
type NvmeSmartHealth = smtypes.NvmeSmartHealth

// NvmeSmartTestLog represents the NVMe self-test log.
type NvmeSmartTestLog = smtypes.NvmeSmartTestLog

// UserCapacity represents storage device capacity information.
type UserCapacity = smtypes.UserCapacity

// SMARTInfo represents comprehensive SMART information for a storage device.
type SMARTInfo = smtypes.SMARTInfo

// SmartStatus represents the overall SMART health status.
type SmartStatus = smtypes.SmartStatus

// SmartSupport represents SMART availability and enablement status.
type SmartSupport = smtypes.SmartSupport

// AtaSmartData represents ATA SMART attributes.
type AtaSmartData = smtypes.AtaSmartData

// StatusField represents a SMART status field.
type StatusField = smtypes.StatusField

// OfflineDataCollection represents offline data collection status.
type OfflineDataCollection = smtypes.OfflineDataCollection

// PollingMinutes represents polling minutes for different test types.
type PollingMinutes = smtypes.PollingMinutes

// SelfTest represents self-test information.
type SelfTest = smtypes.SelfTest

// Capabilities represents SMART capabilities.
type Capabilities = smtypes.Capabilities

// SelfTestInfo represents available self-tests and their durations.
type SelfTestInfo = smtypes.SelfTestInfo

// NvmeOptionalAdminCommands represents NVMe optional admin commands.
type NvmeOptionalAdminCommands = smtypes.NvmeOptionalAdminCommands

// CapabilitiesOutput represents the output of smartctl -c -j.
type CapabilitiesOutput = smtypes.CapabilitiesOutput

// SmartAttribute represents a single SMART attribute.
type SmartAttribute = smtypes.SmartAttribute

// Flags represents SMART attribute flags.
type Flags = smtypes.Flags

// Raw represents a raw SMART attribute value.
type Raw = smtypes.Raw

// Temperature represents device temperature.
type Temperature = smtypes.Temperature

// PowerOnTime represents power-on time.
type PowerOnTime = smtypes.PowerOnTime

// Message represents a message from smartctl.
type Message = smtypes.Message

// SmartctlInfo represents smartctl metadata and messages.
type SmartctlInfo = smtypes.SmartctlInfo

// ProgressCallback reports self-test progress.
type ProgressCallback = smtypes.ProgressCallback

// ExitCodeInfo breaks down the smartctl exit status into semantic groups.
type ExitCodeInfo = smtypes.ExitCodeInfo

// DiscoveryResult holds the outcome of probing a single device during discovery.
type DiscoveryResult = smtypes.DiscoveryResult
