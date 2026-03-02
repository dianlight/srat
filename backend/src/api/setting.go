package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
)

type SettingsHanler struct {
	apiContext     *dto.ContextState
	settingService service.SettingServiceInterface
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
) *SettingsHanler {
	p := new(SettingsHanler)
	p.apiContext = apiContext
	p.settingService = settingService

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
