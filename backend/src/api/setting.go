package api

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
)

type SettingsHanler struct {
	apiContext     *dto.ContextState
	settingService service.SettingServiceInterface
	addonsService  service.AddonsServiceInterface
	eventBus       events.EventBusInterface
}

// NewSettingsHanler creates a new instance of SettingsHanler with the provided
// apiContext and dirtyService. It initializes the SettingsHanler with the given
// context state and dirty data service interface.
//
// Parameters:
//   - apiContext: A pointer to dto.ContextState which holds the context state for the API.
//   - dirtyService: An implementation of the DirtyDataServiceInterface which handles dirty data operations.
//   - props_repo: An implementation of the PropertyRepositoryInterface which handles property operations.
//   - telemetryService: An implementation of the TelemetryServiceInterface which handles telemetry operations.
//
// Returns:
//   - A pointer to the newly created SettingsHanler instance.
func NewSettingsHanler(
	apiContext *dto.ContextState,
	settingService service.SettingServiceInterface,
	addonsService service.AddonsServiceInterface,
	eventBus events.EventBusInterface,
) *SettingsHanler {
	p := new(SettingsHanler)
	p.apiContext = apiContext
	p.settingService = settingService
	p.addonsService = addonsService
	p.eventBus = eventBus

	return p
}

// RegisterSettings registers the settings-related endpoints with the provided API.
// It sets up the following routes:
// - GET /settings: Retrieves the current settings.
// - PUT /settings: Updates the current settings.
//
// Parameters:
// - api: The huma.API instance to register the routes with.
func (self *SettingsHanler) RegisterSettings(api huma.API) {
	huma.Get(api, "/settings", self.GetSettings, huma.OperationTags("system"))
	huma.Put(api, "/settings", self.UpdateSettings, huma.OperationTags("system"))
	huma.Get(api, "/settings/app-config", self.GetAppConfig, huma.OperationTags("system"))
	huma.Put(api, "/settings/app-config", self.UpdateAppConfig, huma.OperationTags("system"))
	huma.Get(api, "/settings/app-config/schema", self.GetAppConfigSchema, huma.OperationTags("system"))
}

// UpdateSettings updates the settings based on the provided input.
// It loads the current database configuration, converts the input settings
// to the database properties format, saves the updated configuration, and
// then converts the updated properties back to the settings format.
// Finally, it marks the settings as dirty to indicate that they have been changed.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: A struct containing the settings to be updated.
//
// Returns:
//   - A struct containing the updated settings.
//   - An error if any step in the process fails.
func (self *SettingsHanler) UpdateSettings(ctx context.Context, input *struct {
	//Name string `path:"name" maxLength:"30" example:"world" doc:"Name to greet"`
	Body dto.Settings
}) (*struct{ Body dto.Settings }, error) {
	config := input.Body

	err := self.settingService.UpdateSettings(&config)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update settings: %v", err)
	}

	return &struct{ Body dto.Settings }{Body: config}, nil
}

// GetSettings retrieves the application settings from the database,
// converts them to the DTO format, and returns them.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: An empty struct as input.
//
// Returns:
//   - A struct containing the settings in the Body field.
//   - An error if there is any issue loading or converting the settings.
func (self *SettingsHanler) GetSettings(ctx context.Context, input *struct{}) (*struct{ Body dto.Settings }, error) {

	settings, err := self.settingService.Load()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to load settings: %v", err)
	}

	return &struct{ Body dto.Settings }{Body: *settings}, nil
}

// GetAppConfig retrieves current app options and rendered runtime config.
func (self *SettingsHanler) GetAppConfig(ctx context.Context, input *struct{}) (*struct{ Body dto.AppConfigData }, error) {
	config, err := self.addonsService.GetAppConfig(ctx)
	if err != nil {
		tlog.ErrorContext(ctx, "Failed to load app configuration", "error", errors.Unwrap(err))
		return nil, huma.Error500InternalServerError("Failed to load app configuration: %v", err)
	}

	return &struct{ Body dto.AppConfigData }{Body: *config}, nil
}

// GetAppConfigSchema retrieves app options schema and app descriptions.
func (self *SettingsHanler) GetAppConfigSchema(ctx context.Context, input *struct{}) (*struct{ Body dto.AppConfigSchema }, error) {
	schema, err := self.addonsService.GetAppConfigSchema(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to load app configuration schema: %v", err)
	}

	return &struct{ Body dto.AppConfigSchema }{Body: *schema}, nil
}

// UpdateAppConfig updates app options and marks app configuration as dirty.
func (self *SettingsHanler) UpdateAppConfig(ctx context.Context, input *struct {
	Body dto.AppConfigUpdateRequest
}) (*struct{ Body dto.AppConfigData }, error) {
	err := self.addonsService.SetAppConfig(ctx, input.Body.Options)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to update app configuration: %v", err)
	}

	self.eventBus.EmitAppConfig(events.AppConfigEvent{Config: &input.Body})

	config, getErr := self.addonsService.GetAppConfig(ctx)
	if getErr != nil {
		return nil, huma.Error500InternalServerError("App configuration updated but reload failed: %v", getErr)
	}

	return &struct{ Body dto.AppConfigData }{Body: *config}, nil
}
