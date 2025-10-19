package service

import (
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"gitlab.com/tozd/go/errors"
)

// settingService handles business logic for settings.
type settingService struct {
	repo      repository.PropertyRepositoryInterface
	converter converter.DtoToDbomConverterImpl
}

// SettingServiceInterface defines the interface for setting services.
type SettingServiceInterface interface {
	Load() (setting *dto.Settings, err errors.E)
}

// NewSettingService creates a new issue service.
func NewSettingService(repo repository.PropertyRepositoryInterface) SettingServiceInterface {
	return &settingService{repo: repo, converter: converter.DtoToDbomConverterImpl{}}
}

// Create creates a new issue.
func (s *settingService) Load() (setting *dto.Settings, err errors.E) {
	props, err := s.repo.All(true)
	if err != nil {
		return nil, err
	}
	set := &dto.Settings{}

	s.converter.PropertiesToSettings(props, set)
	return set, nil
}
