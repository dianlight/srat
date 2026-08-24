package converter

import (
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDtoToDbomConverterImpl_RcloneLinkDTOToRcloneLink(t *testing.T) {
	lastSync := time.Date(2026, time.August, 24, 12, 30, 0, 0, time.UTC)
	conv := DtoToDbomConverterImpl{}
	source := dto.RcloneLink{
		TargetKind:      dto.RcloneTargetKindVolume,
		TargetID:        "/mnt/media",
		Provider:        "dropbox",
		RemotePath:      "backups/srat",
		Status:          dto.RcloneStatusAuthorized,
		LastSyncAt:      &lastSync,
		LastSyncResult:  "ok",
		LastSyncMessage: "sync completed",
		AutoSync:        true,
		ScheduleMinutes: 60,
	}

	result, err := conv.RcloneLinkDTOToRcloneLink(source)

	require.NoError(t, err)
	assert.Equal(t, source.TargetKind, result.TargetKind)
	assert.Equal(t, source.TargetID, result.TargetID)
	assert.Equal(t, source.Provider, result.Provider)
	assert.Equal(t, source.RemotePath, result.RemotePath)
	assert.Equal(t, source.Status, result.Status)
	assert.Same(t, source.LastSyncAt, result.LastSyncAt)
	assert.Equal(t, source.LastSyncResult, result.LastSyncResult)
	assert.Equal(t, source.LastSyncMessage, result.LastSyncMessage)
	assert.Equal(t, source.AutoSync, result.AutoSync)
	assert.Equal(t, source.ScheduleMinutes, result.ScheduleMinutes)
}

func TestDtoToDbomConverterImpl_RcloneLinkToRcloneLinkDTO(t *testing.T) {
	lastSync := time.Date(2026, time.August, 24, 12, 30, 0, 0, time.UTC)
	conv := DtoToDbomConverterImpl{}
	source := dbom.RcloneLink{
		TargetKind:      dto.RcloneTargetKindHassosData,
		TargetID:        "hassos-data",
		Provider:        "dropbox",
		RemotePath:      "backups/hassos",
		Status:          dto.RcloneStatusError,
		LastSyncAt:      &lastSync,
		LastSyncResult:  "failed",
		LastSyncMessage: "remote unavailable",
		AutoSync:        false,
		ScheduleMinutes: 0,
	}

	result, err := conv.RcloneLinkToRcloneLinkDTO(source)

	require.NoError(t, err)
	expected := dto.RcloneLink{
		TargetKind:      source.TargetKind,
		TargetID:        source.TargetID,
		Provider:        source.Provider,
		RemotePath:      source.RemotePath,
		Status:          source.Status,
		LastSyncAt:      source.LastSyncAt,
		LastSyncResult:  source.LastSyncResult,
		LastSyncMessage: source.LastSyncMessage,
		AutoSync:        source.AutoSync,
		ScheduleMinutes: source.ScheduleMinutes,
	}
	assert.Equal(t, expected, result)
}

func TestDtoToDbomConverterImpl_RcloneLinkRoundTrip_ZeroValues(t *testing.T) {
	conv := DtoToDbomConverterImpl{}

	dbomLink, err := conv.RcloneLinkDTOToRcloneLink(dto.RcloneLink{})
	require.NoError(t, err)
	back, err := conv.RcloneLinkToRcloneLinkDTO(dbomLink)
	require.NoError(t, err)
	assert.Equal(t, dto.RcloneLink{}, back)
}
