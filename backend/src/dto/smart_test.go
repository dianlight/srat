package dto_test

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestSmartRangeValue_Fields(t *testing.T) {
	smart := dto.SmartRangeValue{
		Code:       9,
		Value:      1000,
		Min:        0,
		Worst:      900,
		Thresholds: 0,
	}

	assert.Equal(t, 9, smart.Code)
	assert.Equal(t, 1000, smart.Value)
	assert.Equal(t, 0, smart.Min)
	assert.Equal(t, 900, smart.Worst)
	assert.Equal(t, 0, smart.Thresholds)
}

func TestSmartRangeValue_ZeroValues(t *testing.T) {
	smart := dto.SmartRangeValue{}

	assert.Equal(t, 0, smart.Code)
	assert.Equal(t, 0, smart.Value)
	assert.Equal(t, 0, smart.Min)
	assert.Equal(t, 0, smart.Worst)
	assert.Equal(t, 0, smart.Thresholds)
}

func TestSmartTempValue_Fields(t *testing.T) {
	temp := dto.SmartTempValue{
		Value:           45,
		Min:             20,
		Max:             80,
		OvertempCounter: 0,
	}

	assert.Equal(t, 45, temp.Value)
	assert.Equal(t, 20, temp.Min)
	assert.Equal(t, 80, temp.Max)
	assert.Equal(t, 0, temp.OvertempCounter)
}

func TestSmartTempValue_ZeroValues(t *testing.T) {
	temp := dto.SmartTempValue{}

	assert.Equal(t, 0, temp.Value)
	assert.Equal(t, 0, temp.Min)
	assert.Equal(t, 0, temp.Max)
	assert.Equal(t, 0, temp.OvertempCounter)
}

func TestSmartInfo_AllFields(t *testing.T) {
	/*
		additional := map[string]dto.SmartRangeValue{
			"reallocated_sectors": {
				Code:  5,
				Value: 0,
			},
			"seek_error_rate": {
				Code:  7,
				Value: 100,
			},
		}
	*/

	smart := dto.SmartInfo{
		DiskType:     "SATA",
		RotationRate: 7200,
		Supported:    true,
		//Additional:   additional,
	}

	assert.Equal(t, "SATA", smart.DiskType)
	assert.Equal(t, 7200, smart.RotationRate)
	assert.True(t, smart.Supported)
	//assert.Len(t, smart.Additional, 2)
	//assert.Contains(t, smart.Additional, "reallocated_sectors")
	//assert.Contains(t, smart.Additional, "seek_error_rate")
	//assert.Equal(t, 5, smart.Additional["reallocated_sectors"].Code)
	//assert.Equal(t, 0, smart.Additional["reallocated_sectors"].Value)
	//assert.Equal(t, 7, smart.Additional["seek_error_rate"].Code)
	//assert.Equal(t, 100, smart.Additional["seek_error_rate"].Value)
}

func TestSmartInfo_DiskTypes(t *testing.T) {
	tests := []struct {
		name     string
		diskType string
	}{
		{"SATA", "SATA"},
		{"NVMe", "NVMe"},
		{"SCSI", "SCSI"},
		{"Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smart := dto.SmartInfo{
				DiskType: tt.diskType,
			}
			assert.Equal(t, tt.diskType, smart.DiskType)
		})
	}
}

func TestSmartInfo_ZeroValues(t *testing.T) {
	smart := dto.SmartInfo{}

	assert.Empty(t, smart.DiskType)
	assert.Equal(t, 0, smart.RotationRate)
	assert.False(t, smart.Supported)
	//assert.Nil(t, smart.Additional)
}

func TestSmartInfo_EmptyAdditional(t *testing.T) {
	smart := dto.SmartInfo{
		DiskType: "NVMe",
		//Additional: make(map[string]dto.SmartRangeValue),
	}

	//assert.NotNil(t, smart.Additional)
	//assert.Empty(t, smart.Additional)
	assert.Equal(t, "NVMe", smart.DiskType)
}

func TestSmartStatus_AllFields(t *testing.T) {
	additional := map[string]dto.SmartRangeValue{
		"reallocated_sectors": {
			Code:  5,
			Value: 0,
		},
		"seek_error_rate": {
			Code:  7,
			Value: 100,
		},
	}

	status := dto.SmartStatus{
		Temperature: dto.SmartTempValue{
			Value: 45,
			Min:   20,
			Max:   80,
		},
		PowerOnHours: dto.SmartRangeValue{
			Code:  9,
			Value: 10000,
		},
		PowerCycleCount: dto.SmartRangeValue{
			Code:  12,
			Value: 500,
		},
		Enabled:    true,
		Additional: additional,
	}

	assert.Equal(t, 45, status.Temperature.Value)
	assert.Equal(t, 10000, status.PowerOnHours.Value)
	assert.Equal(t, 500, status.PowerCycleCount.Value)
	assert.True(t, status.Enabled)
	assert.Len(t, status.Additional, 2)
	assert.Contains(t, status.Additional, "reallocated_sectors")
	assert.Contains(t, status.Additional, "seek_error_rate")
	assert.Equal(t, 5, status.Additional["reallocated_sectors"].Code)
	assert.Equal(t, 0, status.Additional["reallocated_sectors"].Value)
	assert.Equal(t, 7, status.Additional["seek_error_rate"].Code)
	assert.Equal(t, 100, status.Additional["seek_error_rate"].Value)
}

func TestSmartStatus_ZeroValues(t *testing.T) {
	status := dto.SmartStatus{}

	assert.Equal(t, 0, status.Temperature.Value)
	assert.Equal(t, 0, status.PowerOnHours.Value)
	assert.Equal(t, 0, status.PowerCycleCount.Value)
	assert.False(t, status.Enabled)
	assert.Nil(t, status.Additional)
}

func TestSmartStatus_EmptyAdditional(t *testing.T) {
	status := dto.SmartStatus{
		Enabled:    true,
		Additional: make(map[string]dto.SmartRangeValue),
	}

	assert.NotNil(t, status.Additional)
	assert.Empty(t, status.Additional)
	assert.True(t, status.Enabled)
}

func TestSmartTestType_Values(t *testing.T) {
	assert.Equal(t, dto.SmartTestTypeShort, dto.SmartTestType("short"))
	assert.Equal(t, dto.SmartTestTypeLong, dto.SmartTestType("long"))
	assert.Equal(t, dto.SmartTestTypeConveyance, dto.SmartTestType("conveyance"))
}

func TestSmartTestStatus_Fields(t *testing.T) {
	status := dto.SmartTestStatus{
		Status:          "running",
		TestType:        "short",
		PercentComplete: 50,
	}

	assert.Equal(t, "running", status.Status)
	assert.Equal(t, "short", status.TestType)
	assert.Equal(t, 50, status.PercentComplete)
}

func TestSmartHealthStatus_Healthy(t *testing.T) {
	health := dto.SmartHealthStatus{
		Passed:        true,
		OverallStatus: "healthy",
	}

	assert.True(t, health.Passed)
	assert.Equal(t, "healthy", health.OverallStatus)
	assert.Nil(t, health.FailingAttributes)
}

func TestSmartHealthStatus_Failing(t *testing.T) {
	health := dto.SmartHealthStatus{
		Passed:            false,
		FailingAttributes: []string{"Reallocated_Sector_Ct", "Current_Pending_Sector"},
		OverallStatus:     "failing",
	}

	assert.False(t, health.Passed)
	assert.Equal(t, "failing", health.OverallStatus)
	assert.Len(t, health.FailingAttributes, 2)
	assert.Contains(t, health.FailingAttributes, "Reallocated_Sector_Ct")
	assert.Contains(t, health.FailingAttributes, "Current_Pending_Sector")
}
