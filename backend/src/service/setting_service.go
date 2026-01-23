package service

import (
	"log/slog"
	"os/exec"
	"reflect"
	"strings"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

// settingService handles business logic for settings.
type settingService struct {
	repo repository.PropertyRepositoryInterface
	//telemetryService TelemetryServiceInterface
	eventBus        events.EventBusInterface
	converter       converter.DtoToDbomConverterImpl
	defaultSettings dto.Settings
}

// SettingServiceInterface defines the interface for setting services.
type SettingServiceInterface interface {
	Load() (setting *dto.Settings, err errors.E)
	UpdateSettings(setting *dto.Settings) errors.E
	// HasValue checks if a specific property exists (has a stored value)
	// Accepts only the property key to check.
	HasValue(prop string) (bool, errors.E)
	// HasDefaultValue checks if a specific property exists in the default settings
	// Accepts only the property key to check.
	HasDefaultValue(prop string) (bool, errors.E)
	// GetValue retrieves a property value from the repository.
	// If the value is not found and a default exists, returns the default value.
	// The return type depends on the type of the property.
	GetValue(prop string) (interface{}, errors.E)
	// SetValue sets a property value in the repository.
	// Validates that the value type is compatible with the existing value type (if set)
	// and with the default value type (if exists).
	SetValue(prop string, value interface{}) errors.E
}

// NewSettingService creates a new issue service.
func NewSettingService(
	repo repository.PropertyRepositoryInterface,
	//telemetryService TelemetryServiceInterface,
	eventBus events.EventBusInterface,
	defaultConfig *config.DefaultConfig,
) SettingServiceInterface {
	s := &settingService{
		repo: repo,
		//telemetryService: telemetryService,
		eventBus:        eventBus,
		converter:       converter.DtoToDbomConverterImpl{},
		defaultSettings: dto.Settings{},
	}
	conv := converter.ConfigToDtoConverterImpl{}
	err := conv.ConfigToSettings(defaultConfig.Config, &s.defaultSettings)
	if err != nil {
		slog.Error("Cant convert default config to settings", "error", err)
	}
	return s
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

// ValidateSettings validates and potentially modifies settings based on system capabilities and constraints.
// This is the central point for all settings validation logic.
func (self *settingService) ValidateSettings(setting *dto.Settings) {
	// Validate HAUseNFS setting - NFS must be available if enabled
	if setting.HAUseNFS != nil && *setting.HAUseNFS {
		if _, err := exec.LookPath("exportfs"); err != nil {
			// NFS is not available, force the setting to false
			slog.Warn("NFS is not available on this system (exportfs command not found). Setting ha_use_nfs to false.")
			falseVal := false
			setting.HAUseNFS = &falseVal
		}
	}
}

func (self *settingService) UpdateSettings(setting *dto.Settings) errors.E {
	// Validate settings before saving
	self.ValidateSettings(setting)

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

	/*
		// Configure telemetry service when settings are updated
		if self.telemetryService != nil {
			err = self.telemetryService.Configure(setting.TelemetryMode)
			if err != nil {
				// Log error but don't fail the settings update
				slog.Error("Failed to configure telemetry service", "error", err)
			}
		}
	*/

	self.eventBus.EmitSetting(events.SettingEvent{Setting: setting})
	return nil
}

// HasValue checks if a property exists in the repository.
// It accepts only the property key and returns true if present, false if not found.
// Any error different from NotFound is propagated.
func (s *settingService) HasValue(prop string) (bool, errors.E) {
	val, err := s.repo.Value(prop, true)
	if err != nil {
		if errors.Is(err, dto.ErrorNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	// Consider a property present if it exists, even if its value is nil
	// (nil can be a legitimate stored value). Presence is determined by lookup success.
	_ = val // value not used beyond existence check
	return true, nil
}

// HasDefaultValue checks if a property exists in the default settings.
// It accepts only the property key and uses reflection to check if the corresponding
// field exists in the defaultSettings struct.
func (s *settingService) HasDefaultValue(prop string) (bool, errors.E) {
	if prop == "" {
		return false, nil
	}

	// Use reflection to check if the field exists in defaultSettings
	v := reflect.ValueOf(s.defaultSettings)
	t := v.Type()

	// Iterate through all fields to find a matching json tag or field name
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Check json tag first (most common case)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			// Remove omitempty and other options
			jsonName := strings.Split(jsonTag, ",")[0]
			if jsonName == prop {
				return true, nil
			}
		}

		// Also check if the field name matches (case-insensitive)
		if strings.EqualFold(field.Name, prop) {
			return true, nil
		}
	}

	return false, nil
}

// GetValue retrieves a property value from the repository.
// If the value is not found and a default exists in defaultSettings, returns the default value.
// The return type is interface{} and depends on the type of the property.
func (s *settingService) GetValue(prop string) (interface{}, errors.E) {
	if prop == "" {
		return nil, errors.New("property name cannot be empty")
	}

	// Try to get the value from the repository first
	val, err := s.repo.Value(prop, true)
	if err != nil {
		// If not found, try to get the default value
		if errors.Is(err, dto.ErrorNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			defaultVal, errDefault := s.getDefaultValue(prop)
			if errDefault != nil {
				// No default value exists either
				return nil, errors.WithStack(dto.ErrorNotFound)
			}
			return defaultVal, nil
		}
		// Other errors are propagated
		return nil, err
	}

	return val, nil
}

// getDefaultValue retrieves the default value for a property from defaultSettings.
// Uses reflection to find and return the field value.
func (s *settingService) getDefaultValue(prop string) (interface{}, errors.E) {
	if prop == "" {
		return nil, errors.New("property name cannot be empty")
	}

	// Use reflection to find the field in defaultSettings
	v := reflect.ValueOf(s.defaultSettings)
	t := v.Type()

	// Iterate through all fields to find a matching json tag or field name
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Check json tag first (most common case)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			// Remove omitempty and other options
			jsonName := strings.Split(jsonTag, ",")[0]
			if jsonName == prop {
				// Return the field value
				return fieldValue.Interface(), nil
			}
		}

		// Also check if the field name matches (case-insensitive)
		if strings.EqualFold(field.Name, prop) {
			return fieldValue.Interface(), nil
		}
	}

	return nil, errors.WithStack(dto.ErrorNotFound)
}

// SetValue sets a property value in the repository.
// Validates that the value type is compatible with the existing value type (if set)
// and with the default value type (if exists).
func (s *settingService) SetValue(prop string, value interface{}) errors.E {
	if prop == "" {
		return errors.New("property name cannot be empty")
	}

	if value == nil {
		return errors.New("value cannot be nil")
	}

	// Get the type of the new value
	newType := reflect.TypeOf(value)

	// Check if there's an existing value and validate type compatibility
	existingVal, err := s.repo.Value(prop, true)
	if err == nil {
		// Existing value found, check type compatibility
		if existingVal != nil {
			existingType := reflect.TypeOf(existingVal)
			if !s.areTypesCompatible(existingType, newType) {
				return errors.Errorf("type mismatch: cannot set %s (type %s) to property with existing type %s",
					prop, newType.String(), existingType.String())
			}
		}
	} else if !errors.Is(err, dto.ErrorNotFound) && !errors.Is(err, gorm.ErrRecordNotFound) {
		// Propagate any error that's not "not found"
		return err
	}

	// If no existing value or NotFound, check against default value type
	if errors.Is(err, dto.ErrorNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
		defaultVal, errDefault := s.getDefaultValue(prop)
		if errDefault == nil && defaultVal != nil {
			// Default value exists, validate type compatibility
			defaultType := reflect.TypeOf(defaultVal)
			if !s.areTypesCompatible(defaultType, newType) {
				return errors.Errorf("type mismatch: cannot set %s (type %s) to property with default type %s",
					prop, newType.String(), defaultType.String())
			}
		}
		// If no default value exists, allow any type for new properties
	}

	// Type validation passed, save the value
	return s.repo.SetValue(prop, value)
}

// areTypesCompatible checks if two types are compatible for assignment.
// Handles special cases like pointer types, slices, and basic type compatibility.
func (s *settingService) areTypesCompatible(existing, new reflect.Type) bool {
	// Exact match
	if existing == new {
		return true
	}

	// Handle pointer types - compare the underlying types
	if existing.Kind() == reflect.Ptr && new.Kind() == reflect.Ptr {
		return s.areTypesCompatible(existing.Elem(), new.Elem())
	}

	// Allow assigning non-pointer to pointer of same type (and vice versa)
	if existing.Kind() == reflect.Ptr && existing.Elem() == new {
		return true
	}
	if new.Kind() == reflect.Ptr && new.Elem() == existing {
		return true
	}

	// For slices, check element types
	if existing.Kind() == reflect.Slice && new.Kind() == reflect.Slice {
		return s.areTypesCompatible(existing.Elem(), new.Elem())
	}

	// Check if new type is assignable to existing type (handles interfaces, etc.)
	if new.AssignableTo(existing) {
		return true
	}

	// Check if types have the same kind and are convertible
	if existing.Kind() == new.Kind() {
		switch existing.Kind() {
		case reflect.String, reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return true
		}
	}

	return false
}

// GetValueAs returns the property value as the requested generic type T.
// It wraps GetValue and performs a runtime type check/convert to T.
func GetValueAs[T any](svc SettingServiceInterface, prop string) (T, errors.E) {
	var zero T
	v, err := svc.GetValue(prop)
	if err != nil {
		return zero, err
	}
	if v == nil {
		// Cannot convert nil to concrete type
		return zero, errors.Errorf("value for %s is nil and cannot be converted", prop)
	}
	vt := reflect.TypeOf(v)
	tt := reflect.TypeOf((*T)(nil)).Elem()
	// Direct assignable
	if vt.AssignableTo(tt) {
		return v.(T), nil
	}
	// Convertible types (e.g., int -> int64 of same kind not always convertible; reflect handles safe cases)
	if vt.ConvertibleTo(tt) {
		converted := reflect.ValueOf(v).Convert(tt).Interface()
		return converted.(T), nil
	}
	return zero, errors.Errorf("type mismatch: cannot convert %s to %s for %s", vt.String(), tt.String(), prop)
}

// SetValueAs sets the property value using a typed generic value T.
// It wraps SetValue which performs runtime compatibility checks with existing/default types.
func SetValueAs[T any](svc SettingServiceInterface, prop string, value T) errors.E {
	return svc.SetValue(prop, any(value))
}
