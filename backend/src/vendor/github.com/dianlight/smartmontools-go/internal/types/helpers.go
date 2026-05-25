package types

// PopulateSelfTestInfo fills a SelfTestInfo with available test types and durations
// from either ATA SMART data or NVMe capabilities.
func PopulateSelfTestInfo(info *SelfTestInfo, ata *AtaSmartData, nvmeCaps *NvmeControllerCapabilities, nvmeOptional *NvmeOptionalAdminCommands) {
	if ata != nil && ata.Capabilities != nil {
		caps := ata.Capabilities
		if caps.SelfTestsSupported {
			info.Available = append(info.Available, "short", "long")
		}
		if caps.ConveyanceSelfTestSupported {
			info.Available = append(info.Available, "conveyance")
		}
		if caps.ExecOfflineImmediate {
			info.Available = append(info.Available, "offline")
		}
		if ata.SelfTest != nil && ata.SelfTest.PollingMinutes != nil {
			pm := ata.SelfTest.PollingMinutes
			if pm.Short > 0 {
				info.Durations["short"] = pm.Short
			}
			if pm.Extended > 0 {
				info.Durations["long"] = pm.Extended
			}
			if pm.Conveyance > 0 {
				info.Durations["conveyance"] = pm.Conveyance
			}
		}
	}
	if (nvmeCaps != nil && nvmeCaps.SelfTest) || (nvmeOptional != nil && nvmeOptional.SelfTest) {
		info.Available = append(info.Available, "short")
	}
}

// ValidSelfTestTypes lists the supported self-test type names.
var ValidSelfTestTypes = []string{"short", "long", "conveyance", "offline"}
