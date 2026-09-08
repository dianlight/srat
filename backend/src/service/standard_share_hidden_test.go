package service

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

type stubSettingService struct {
	settings *dto.Settings
	err      errors.E
}

func (s *stubSettingService) Load() (*dto.Settings, errors.E) {
	return s.settings, s.err
}

func (s *stubSettingService) UpdateSettings(_ *dto.Settings) errors.E { return nil }

func (s *stubSettingService) SetCommandExists(_ func(cmd []string) bool) {}

func (s *stubSettingService) DumpTable() (string, errors.E) { return "", nil }

func TestAnnotateStandardShareHiddenWithMode(t *testing.T) {
	svc := &ShareService{}

	share := &dto.SharedResource{Name: "local_apps"}
	svc.annotateStandardShareHiddenWithMode(share, dto.StandardShareNamesModeOld)
	require.NotNil(t, share.Status)
	assert.True(t, share.Status.IsHidden)

	share = &dto.SharedResource{Name: "addons", Status: &dto.SharedResourceStatus{IsValid: true}}
	svc.annotateStandardShareHiddenWithMode(share, dto.StandardShareNamesModeOld)
	assert.False(t, share.Status.IsHidden)
	assert.True(t, share.Status.IsValid)

	share = &dto.SharedResource{Name: "addons"}
	svc.annotateStandardShareHiddenWithMode(share, dto.StandardShareNamesModeNew)
	assert.True(t, share.Status.IsHidden)

	share = &dto.SharedResource{Name: "media"}
	svc.annotateStandardShareHiddenWithMode(share, dto.StandardShareNamesModeNew)
	assert.False(t, share.Status.IsHidden)

	svc.annotateStandardShareHiddenWithMode(nil, dto.StandardShareNamesModeOld)
}

func TestCurrentStandardShareNamesMode(t *testing.T) {
	svc := &ShareService{}
	assert.Equal(t, dto.StandardShareNamesModeBoth, svc.currentStandardShareNamesMode())

	svc = &ShareService{settingService: &stubSettingService{settings: &dto.Settings{StandardShareNames: dto.StandardShareNamesModeOld}}}
	assert.Equal(t, dto.StandardShareNamesModeOld, svc.currentStandardShareNamesMode())

	svc = &ShareService{settingService: &stubSettingService{settings: &dto.Settings{StandardShareNames: dto.StandardShareNamesModeNew}}}
	assert.Equal(t, dto.StandardShareNamesModeNew, svc.currentStandardShareNamesMode())

	svc = &ShareService{settingService: &stubSettingService{settings: nil, err: errors.New("db down")}}
	assert.Equal(t, dto.StandardShareNamesModeBoth, svc.currentStandardShareNamesMode())

	svc = &ShareService{settingService: &stubSettingService{settings: nil}}
	assert.Equal(t, dto.StandardShareNamesModeBoth, svc.currentStandardShareNamesMode())
}

func TestAnnotateStandardShareHiddenUsesSettings(t *testing.T) {
	svc := &ShareService{settingService: &stubSettingService{settings: &dto.Settings{StandardShareNames: dto.StandardShareNamesModeNew}}}
	share := &dto.SharedResource{Name: "addon_configs"}
	svc.annotateStandardShareHidden(share)
	require.NotNil(t, share.Status)
	assert.True(t, share.Status.IsHidden)
}
