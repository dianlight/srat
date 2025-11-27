package appsetup

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/addons"
	"github.com/dianlight/srat/homeassistant/core_api"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/host"
	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/homeassistant/resolution"
	"github.com/dianlight/srat/homeassistant/root"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/templates"
	"github.com/dianlight/tlog"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v76/github"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// NewFXLoggerOption provides the FX logger configuration.
func NewFXLoggerOption() fx.Option {
	return fx.WithLogger(func(l *slog.Logger) fxevent.Logger { // l is provided by FX
		l.Debug("Starting FX") // Use the logger FX provides for this initial message
		fxlog := &fxevent.SlogLogger{
			Logger: l.WithGroup("fx"), // Use the logger FX provides to this function for FX events
		}
		fxlog.UseLogLevel(tlog.LevelTrace)   // FX events at Trace
		fxlog.UseErrorLevel(tlog.LevelError) // FX errors at Error
		return fxlog
	})
}

// BaseAppParams holds common parameters for initializing FX application components.
type BaseAppParams struct {
	Ctx          context.Context
	CancelFn     context.CancelFunc
	StaticConfig *dto.ContextState
}

// ProvideCoreDependencies provides FX options for core services and repositories.
func ProvideCoreDependencies(params BaseAppParams) fx.Option {
	return fx.Provide(
		dbom.NewDB,
		func() *slog.Logger { return slog.Default() },
		func() (context.Context, context.CancelFunc) { return params.Ctx, params.CancelFn },
		func() *dto.ContextState { return params.StaticConfig },
		func(ctx context.Context) events.EventBusInterface { return events.NewEventBus(ctx) },
		func() *github.Client {
			rateLimiter := github_ratelimit.New(nil)
			return github.NewClient(&http.Client{
				Transport: rateLimiter,
			})
		},
		func() *config.DefaultConfig {
			var nconfig config.Config
			buffer, err := templates.Default_Config_content.ReadFile("default_config.json")
			if err != nil {
				log.Fatalf("Cant read default config file %#+v", err)
			}
			err = nconfig.LoadConfigBuffer(buffer) // Assign to existing err
			if err != nil {
				log.Fatalf("Cant load default config from buffer %#+v", err)
			}
			return &config.DefaultConfig{Config: nconfig}
		},
		service.NewAddonsService,
		service.NewHomeAssistantService,
		service.NewBroadcasterService,
		service.NewVolumeService,
		service.NewSambaService,
		service.NewUpgradeService,
		service.NewDirtyDataService,
		service.NewSupervisorService,
		service.NewHaRootService,
		service.NewFilesystemService,
		service.NewShareService,
		service.NewUserService,
		service.NewHostService,
		service.NewDiskStatsService,
		service.NewNetworkStatsService,
		service.NewSmartService,
		service.NewIssueService,
		service.NewTelemetryService,
		service.NewHaWsService,
		service.NewHardwareService,
		service.NewHDIdleService,
		service.NewSettingService,
		//repository.NewMountPointPathRepository,
		repository.NewExportedShareRepository,
		repository.NewPropertyRepositoryRepository,
		repository.NewSambaUserRepository,
		repository.NewIssueRepository,
	)
}

// ProvideCoreDependenciesWithoutDB provides FX options for core services without database dependencies.
// This is useful for commands that truly don't need database access (e.g., version only).
func ProvideCoreDependenciesWithoutDB(params BaseAppParams) fx.Option {
	return fx.Provide(
		func() *slog.Logger { return slog.Default() },
		func() (context.Context, context.CancelFunc) { return params.Ctx, params.CancelFn },
		func() *dto.ContextState { return params.StaticConfig },
	)
}

// ProvideHAClientDependencies provides FX options for Home Assistant API clients.
func ProvideHAClientDependencies(params BaseAppParams) fx.Option {
	return fx.Provide(
		func() (*securityprovider.SecurityProviderBearerToken, error) {
			return securityprovider.NewSecurityProviderBearerToken(params.StaticConfig.SupervisorToken)
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (ingress.ClientWithResponsesInterface, error) {
			return ingress.NewClientWithResponses(params.StaticConfig.SupervisorURL, ingress.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (hardware.ClientWithResponsesInterface, error) {
			return hardware.NewClientWithResponses(params.StaticConfig.SupervisorURL, hardware.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (mount.ClientWithResponsesInterface, error) {
			return mount.NewClientWithResponses(params.StaticConfig.SupervisorURL, mount.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (host.ClientWithResponsesInterface, error) {
			return host.NewClientWithResponses(params.StaticConfig.SupervisorURL, host.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (addons.ClientWithResponsesInterface, error) {
			return addons.NewClientWithResponses(params.StaticConfig.SupervisorURL, addons.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (resolution.ClientWithResponsesInterface, error) {
			return resolution.NewClientWithResponses(params.StaticConfig.SupervisorURL, resolution.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (core_api.ClientWithResponsesInterface, error) {
			return core_api.NewClientWithResponses(params.StaticConfig.SupervisorURL, core_api.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (root.ClientWithResponsesInterface, error) {
			return root.NewClientWithResponses(params.StaticConfig.SupervisorURL, root.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (websocket.ClientInterface, error) {
			return websocket.NewClient(strings.Replace(params.StaticConfig.SupervisorURL, "http", "ws", 1), params.StaticConfig.SupervisorToken), nil
		},
	)
}

// ProvideHAClientDependenciesWithoutWebSocket provides FX options for Home Assistant API clients excluding the websocket client.
func ProvideHAClientDependenciesWithoutWebSocket(params BaseAppParams) fx.Option {
	return fx.Provide(
		func() (*securityprovider.SecurityProviderBearerToken, error) {
			return securityprovider.NewSecurityProviderBearerToken(params.StaticConfig.SupervisorToken)
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (ingress.ClientWithResponsesInterface, error) {
			return ingress.NewClientWithResponses(params.StaticConfig.SupervisorURL, ingress.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (hardware.ClientWithResponsesInterface, error) {
			return hardware.NewClientWithResponses(params.StaticConfig.SupervisorURL, hardware.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (mount.ClientWithResponsesInterface, error) {
			return mount.NewClientWithResponses(params.StaticConfig.SupervisorURL, mount.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (host.ClientWithResponsesInterface, error) {
			return host.NewClientWithResponses(params.StaticConfig.SupervisorURL, host.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (addons.ClientWithResponsesInterface, error) {
			return addons.NewClientWithResponses(params.StaticConfig.SupervisorURL, addons.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (resolution.ClientWithResponsesInterface, error) {
			return resolution.NewClientWithResponses(params.StaticConfig.SupervisorURL, resolution.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (core_api.ClientWithResponsesInterface, error) {
			return core_api.NewClientWithResponses(params.StaticConfig.SupervisorURL, core_api.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (root.ClientWithResponsesInterface, error) {
			return root.NewClientWithResponses(params.StaticConfig.SupervisorURL, root.WithRequestEditorFn(bearerAuth.Intercept))
		},
		// Note: websocket client is intentionally excluded
	)
}

// ProvideFrontendOption provides the FX option for the static frontend file system.
func ProvideFrontendOption() fx.Option {
	return fx.Provide(internal.GetFrontend)
}

// ProvideCyclicDependencyWorkaroundOption provides the FX option for the ShareService/VolumeService cyclic dependency.
func ProvideCyclicDependencyWorkaroundOption() fx.Option {
	return fx.Invoke(func(s service.ShareServiceInterface, v service.VolumeServiceInterface) {
		//		s.SetVolumeService(v) // Bypass block for cyclic dep in FX
	})
}
