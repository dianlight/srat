package dto

//go:generate go tool goenums smart_attribute.go

type smartAttributeCode int // Code int,Type string

const (
	// SMART attribute IDs
	smartAttributeUndefined     smartAttributeCode = iota // Undefined 0,"Unknown"
	smartAttrTemperatureCelsius                           // Temperature 194,"Old_age"
	smartAttrPowerOnHours                                 // PowerOnHours 9,"Old_age"
	smartAttrPowerCycleCount                              // PowerCycleCount 12,"Old_age"
	//smartAttrReallocatedSectors                             // ReallocatedSectors 5,"Pre-fail"
	//smartAttrCurrentPending                                 // CurrentPending 197,"Old_age"
	//smartAttrOfflineUncorrectable                           // OfflineUncorrectable 198,"Old_age"
	//smartAttrUDMA_CRC_ErrorCount                            // UDMA_CRC_ErrorCount 199,"Old_age"
)
