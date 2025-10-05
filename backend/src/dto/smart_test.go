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

	smart := dto.SmartInfo{
		DiskType: "SATA",
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
		Additional: additional,
	}

	assert.Equal(t, "SATA", smart.DiskType)
	assert.Equal(t, 45, smart.Temperature.Value)
	assert.Equal(t, 10000, smart.PowerOnHours.Value)
	assert.Equal(t, 500, smart.PowerCycleCount.Value)
	assert.Len(t, smart.Additional, 2)
	assert.Contains(t, smart.Additional, "reallocated_sectors")
	assert.Contains(t, smart.Additional, "seek_error_rate")
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
	assert.Equal(t, 0, smart.Temperature.Value)
	assert.Equal(t, 0, smart.PowerOnHours.Value)
	assert.Equal(t, 0, smart.PowerCycleCount.Value)
	assert.Nil(t, smart.Additional)
}

func TestSmartInfo_EmptyAdditional(t *testing.T) {
	smart := dto.SmartInfo{
		DiskType:   "NVMe",
		Additional: make(map[string]dto.SmartRangeValue),
	}

	assert.NotNil(t, smart.Additional)
	assert.Empty(t, smart.Additional)
}
