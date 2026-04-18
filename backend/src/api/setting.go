package api

import (
	"context"
	"errors"

	"github.com/Masterminds/semver/v3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
)

type SettingsHanler struct {
	settingService service.SettingServiceInterface
	addonsService  service.AddonsServiceInterface
	haComponentSvc service.HomeAssistantComponentServiceInterface
	haService      service.HomeAssistantServiceInterface
	upgradeService service.UpgradeServiceInterface
	eventBus       events.EventBusInterface
}

// NewSettingsHanler creates a new settings handler.
func NewSettingsHanler(
	settingService service.SettingServiceInterface,
	addonsService service.AddonsServiceInterface,
	haComponentSvc service.HomeAssistantComponentServiceInterface,
	haService service.HomeAssistantServiceInterface,
	upgradeService service.UpgradeServiceInterface,
	eventBus events.EventBusInterface,
) *SettingsHanler {
	p := new(SettingsHanler)
	p.settingService = settingService
	p.addonsService = addonsService
	p.haComponentSvc = haComponentSvc
	p.haService = haService
	p.upgradeService = upgradeService
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
	huma.Put(api, "/restart", self.RestartAddon, huma.OperationTags("system"))
	huma.Get(api, "/settings/homeassistant/custom-component/status", self.GetHomeAssistantCustomComponentStatus, huma.OperationTags("system"))
	huma.Post(api, "/settings/homeassistant/custom-component/install", self.InstallHomeAssistantCustomComponent, huma.OperationTags("system"))
	huma.Post(api, "/settings/homeassistant/custom-component/upgrade", self.UpgradeHomeAssistantCustomComponent, huma.OperationTags("system"))
	huma.Delete(api, "/settings/homeassistant/custom-component", self.UninstallHomeAssistantCustomComponent, huma.OperationTags("system"))
	huma.Post(api, "/settings/homeassistant/restart-core", self.RestartHACore, huma.OperationTags("system"))
	huma.Get(api, "/settings/app-config", self.GetAppConfig, huma.OperationTags("system"))
	huma.Put(api, "/settings/app-config", self.UpdateAppConfig, huma.OperationTags("system"))
	huma.Get(api, "/settings/app-config/schema", self.GetAppConfigSchema, huma.OperationTags("system"))
}

// GetHomeAssistantCustomComponentStatus reports install/version/connection state
// for the SRAT Home Assistant custom component.
func (self *SettingsHanler) GetHomeAssistantCustomComponentStatus(ctx context.Context, input *struct{}) (*struct {
	Body dto.HomeAssistantCustomComponentStatus
}, error) {
	status, err := self.haComponentSvc.GetStatus()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to inspect Home Assistant custom component status: %v", err)
	}

	syncErr := self.haComponentSvc.SyncIssueStatus(status)
	if syncErr != nil {
		return nil, huma.Error500InternalServerError("Failed to synchronize Home Assistant component issue state: %v", syncErr)
	}

	if self.upgradeService != nil {
		if ass, assErr := self.upgradeService.GetUpgradeReleaseAsset(); assErr == nil && ass != nil && ass.LastRelease != "" {
			status.LatestVersion = &ass.LastRelease
			if status.InstalledVersion != nil {
				if iv, err := semver.NewVersion(*status.InstalledVersion); err == nil {
					if lv, err := semver.NewVersion(*status.LatestVersion); err == nil {
						status.CanUpgrade = iv.LessThan(lv)
					}
				}
			} else {
				status.CanUpgrade = false
			}
		}
	}

	return &struct {
		Body dto.HomeAssistantCustomComponentStatus
	}{Body: *status}, nil
}

func (self *SettingsHanler) InstallHomeAssistantCustomComponent(ctx context.Context, input *struct{}) (*struct {
	Body dto.HomeAssistantCustomComponentStatus
}, error) {
	err := self.haComponentSvc.InstallOrUpgrade(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to install Home Assistant custom component: %v", err)
	}

	status, err := self.haComponentSvc.GetStatus()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to inspect Home Assistant custom component status after install/upgrade: %v", err)
	}

	/*
		syncErr := self.haComponentSvc.SyncIssueStatus(status)
		if syncErr != nil {
			return nil, huma.Error500InternalServerError("Failed to synchronize Home Assistant component issue state: %v", syncErr)
		}

		if self.upgradeService != nil {
			if ass, assErr := self.upgradeService.GetUpgradeReleaseAsset(); assErr == nil && ass != nil && ass.LastRelease != "" {
				status.LatestVersion = &ass.LastRelease
			}
		}
	*/

	//status.CanInstall = !status.Installed
	//	status.CanUpgrade = status.Installed
	//	status.CanUninstall = status.Installed
	/*
		repairErr := self.haComponentSvc.UpsertRestartRequiredRepair(ctx)
		if repairErr != nil {
			return nil, huma.Error500InternalServerError("Failed to create Home Assistant restart repair: %v", repairErr)
		}
	*/
	//	_ = ctx

	return &struct {
		Body dto.HomeAssistantCustomComponentStatus
	}{Body: *status}, nil
}

func (self *SettingsHanler) UpgradeHomeAssistantCustomComponent(ctx context.Context, input *struct{}) (*struct {
	Body dto.HomeAssistantCustomComponentStatus
}, error) {
	err := self.haComponentSvc.InstallOrUpgrade(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to install Home Assistant custom component: %v", err)
	}

	status, err := self.haComponentSvc.GetStatus()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to inspect Home Assistant custom component status after install/upgrade: %v", err)
	}
	/*

		syncErr := self.haComponentSvc.SyncIssueStatus(status)
		if syncErr != nil {
			return nil, huma.Error500InternalServerError("Failed to synchronize Home Assistant component issue state: %v", syncErr)
		}

		if self.upgradeService != nil {
			if ass, assErr := self.upgradeService.GetUpgradeReleaseAsset(); assErr == nil && ass != nil && ass.LastRelease != "" {
				status.LatestVersion = &ass.LastRelease
			}
		}

		status.CanInstall = !status.Installed
		status.CanUpgrade = status.Installed
		status.CanUninstall = status.Installed
		repairErr := self.haComponentSvc.UpsertRestartRequiredRepair(ctx)
		if repairErr != nil {
			return nil, huma.Error500InternalServerError("Failed to create Home Assistant restart repair: %v", repairErr)
		}

		_ = ctx
	*/
	return &struct {
		Body dto.HomeAssistantCustomComponentStatus
	}{Body: *status}, nil
}

func (self *SettingsHanler) UninstallHomeAssistantCustomComponent(ctx context.Context, input *struct{}) (*struct {
	Body dto.HomeAssistantCustomComponentStatus
}, error) {
	err := self.haComponentSvc.Uninstall(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to uninstall Home Assistant custom component: %v", err)
	}

	status, err := self.haComponentSvc.GetStatus()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to inspect Home Assistant custom component status after uninstall: %v", err)
	}

	/*
		syncErr := self.haComponentSvc.SyncIssueStatus(status)
		if syncErr != nil {
			return nil, huma.Error500InternalServerError("Failed to synchronize Home Assistant component issue state: %v", syncErr)
		}

		repairErr := self.haComponentSvc.UpsertRestartRequiredRepair(ctx)
		if repairErr != nil {
			return nil, huma.Error500InternalServerError("Failed to create Home Assistant restart repair: %v", repairErr)
		}
	*/
	return &struct {
		Body dto.HomeAssistantCustomComponentStatus
	}{Body: *status}, nil
}

// RestartAddon triggers a Home Assistant Supervisor restart for the current addon.
func (self *SettingsHanler) RestartAddon(ctx context.Context, input *struct{}) (*struct{ Body string }, error) {
	err := self.addonsService.RestartSelfApp(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to restart addon: %v", err)
	}

	repairErr := self.haComponentSvc.DismissRestartRequiredRepair(ctx)
	if repairErr != nil {
		return nil, huma.Error500InternalServerError("Failed to dismiss Home Assistant restart repair: %v", repairErr)
	}

	return &struct{ Body string }{Body: "addon restart requested"}, nil
}

// RestartHACore triggers a Home Assistant Core restart via the HA service call API.
func (self *SettingsHanler) RestartHACore(ctx context.Context, input *struct{}) (*struct{ Body string }, error) {
	if self.haService == nil {
		return nil, huma.Error503ServiceUnavailable("Home Assistant service is not available")
	}
	err := self.haService.RestartHomeAssistant(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to restart Home Assistant core: %v", err)
	}
	return &struct{ Body string }{Body: "Home Assistant core restart requested"}, nil
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

	if !config.RequiresRestart {
		dismissErr := self.haComponentSvc.DismissAddonConfigIssue(ctx)
		if dismissErr != nil {
			return nil, huma.Error500InternalServerError("Failed to dismiss app-config repair issue: %v", dismissErr)
		}
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

	dismissErr := self.haComponentSvc.DismissAddonConfigIssue(ctx)
	if dismissErr != nil {
		return nil, huma.Error500InternalServerError("Failed to dismiss app-config repair issue: %v", dismissErr)
	}

	return &struct{ Body dto.AppConfigData }{Body: *config}, nil
}
