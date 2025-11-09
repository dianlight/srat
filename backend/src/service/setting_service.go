package service

import (
	"log/slog"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"gitlab.com/tozd/go/errors"
)

// settingService handles business logic for settings.
type settingService struct {
	repo             repository.PropertyRepositoryInterface
	telemetryService TelemetryServiceInterface
	eventBus         events.EventBusInterface
	converter        converter.DtoToDbomConverterImpl
}

// SettingServiceInterface defines the interface for setting services.
type SettingServiceInterface interface {
	Load() (setting *dto.Settings, err errors.E)
	UpdateSettings(setting *dto.Settings) errors.E
}

// NewSettingService creates a new issue service.
func NewSettingService(
	repo repository.PropertyRepositoryInterface,
	telemetryService TelemetryServiceInterface,
	eventBus events.EventBusInterface,
) SettingServiceInterface {
	return &settingService{
		repo:             repo,
		telemetryService: telemetryService,
		eventBus:         eventBus,
		converter:        converter.DtoToDbomConverterImpl{},
	}
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

func (self *settingService) UpdateSettings(setting *dto.Settings) errors.E {
	dbconfig, err := self.repo.All(true)
	if err != nil {
		return errors.WithStack(err)
	}
	var conv converter.DtoToDbomConverterImpl

	err = conv.SettingsToProperties(*setting, &dbconfig)
	if err != nil {
		return errors.WithStack(err)
	}

	err = self.repo.SaveAll(&dbconfig)
	if err != nil {
		return errors.WithStack(err)
	}

	err = conv.PropertiesToSettings(dbconfig, setting)
	if err != nil {
		return errors.WithStack(err)
	}

	// Configure telemetry service when settings are updated
	if self.telemetryService != nil {
		err = self.telemetryService.Configure(setting.TelemetryMode)
		if err != nil {
			// Log error but don't fail the settings update
			slog.Error("Failed to configure telemetry service", "error", err)
		}
	}

	self.eventBus.EmitSetting(events.SettingEvent{Setting: setting})
	return nil
}
