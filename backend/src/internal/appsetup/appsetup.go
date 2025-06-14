package appsetup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/host"
	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v72/github"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// ConfigureGlobalLogger sets up the global slog logger.
// It takes the desired logLevel string and an io.Writer for output.
// It returns the parsed slog.Level or an error if the level string is invalid.
func ConfigureGlobalLogger(logLevelString string, w io.Writer) (slog.Level, error) {
	var level slog.Level
	switch logLevelString {
	case "trace", "debug":
		level = slog.LevelDebug
	case "info", "notice":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error", "fatal":
		level = slog.LevelError
	default:
		return level, fmt.Errorf("invalid log level: %s", logLevelString)
	}

	isTerminal := false
	if f, ok := w.(*os.File); ok {
		isTerminal = isatty.IsTerminal(f.Fd())
	}

	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      level,
			TimeFormat: time.RFC3339,
			NoColor:    !isTerminal,
			AddSource:  true,
		}),
	))
	return level, nil
}

// NewFXLoggerOption provides the FX logger configuration.
func NewFXLoggerOption() fx.Option {
	return fx.WithLogger(func(l *slog.Logger) fxevent.Logger { // l is provided by FX
		l.Debug("Starting FX") // Use the logger FX provides for this initial message
		fxlog := &fxevent.SlogLogger{
			Logger: l, // Use the logger FX provides to this function for FX events
		}
		fxlog.UseLogLevel(slog.LevelDebug)   // FX events at Debug
		fxlog.UseErrorLevel(slog.LevelError) // FX errors at Error
		return fxlog
	})
}

// BaseAppParams holds common parameters for initializing FX application components.
type BaseAppParams struct {
	Ctx             context.Context
	CancelFn        context.CancelFunc
	StaticConfig    *dto.ContextState
	HAMode          bool
	DBPath          string
	SupervisorURL   string // Optional: Only for apps needing HA clients
	SupervisorToken string // Optional: Only for apps needing HA clients
}

// ProvideCoreDependencies provides FX options for core services and repositories.
func ProvideCoreDependencies(params BaseAppParams) fx.Option {
	return fx.Provide(
		dbom.NewDB,
		func() *slog.Logger { return slog.Default() },
		func() (context.Context, context.CancelFunc) { return params.Ctx, params.CancelFn },
		func() *dto.ContextState { return params.StaticConfig },
		fx.Annotate(
			func() bool { return params.HAMode },
			fx.ResultTags(`name:"ha_mode"`),
		),
		fx.Annotate(
			func() string { return params.DBPath },
			fx.ResultTags(`name:"db_path"`),
		),
		func() *github.Client {
			rateLimiter := github_ratelimit.New(nil)
			return github.NewClient(&http.Client{
				Transport: rateLimiter,
			})
		},
		lsblk.NewLSBKInterpreter,
		service.NewBroadcasterService,
		service.NewVolumeService,
		service.NewSambaService,
		service.NewUpgradeService,
		service.NewDirtyDataService,
		service.NewSupervisorService,
		service.NewFilesystemService,
		service.NewShareService,
		service.NewUserService,
		service.NewHostService,
		repository.NewMountPointPathRepository,
		repository.NewExportedShareRepository,
		repository.NewPropertyRepositoryRepository,
		repository.NewSambaUserRepository,
	)
}

// ProvideHAClientDependencies provides FX options for Home Assistant API clients.
func ProvideHAClientDependencies(params BaseAppParams) fx.Option {
	return fx.Provide(
		func() (*securityprovider.SecurityProviderBearerToken, error) {
			return securityprovider.NewSecurityProviderBearerToken(params.SupervisorToken)
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (ingress.ClientWithResponsesInterface, error) {
			return ingress.NewClientWithResponses(params.SupervisorURL, ingress.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (hardware.ClientWithResponsesInterface, error) {
			return hardware.NewClientWithResponses(params.SupervisorURL, hardware.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (mount.ClientWithResponsesInterface, error) {
			return mount.NewClientWithResponses(params.SupervisorURL, mount.WithRequestEditorFn(bearerAuth.Intercept))
		},
		func(bearerAuth *securityprovider.SecurityProviderBearerToken) (host.ClientWithResponsesInterface, error) {
			return host.NewClientWithResponses(params.SupervisorURL, host.WithRequestEditorFn(bearerAuth.Intercept))
		},
	)
}

// ProvideFrontendOption provides the FX option for the static frontend file system.
func ProvideFrontendOption() fx.Option {
	return fx.Provide(internal.GetFrontend)
}

// ProvideCyclicDependencyWorkaroundOption provides the FX option for the ShareService/VolumeService cyclic dependency.
func ProvideCyclicDependencyWorkaroundOption() fx.Option {
	return fx.Invoke(func(s service.ShareServiceInterface, v service.VolumeServiceInterface) {
		s.SetVolumeService(v) // Bypass block for cyclic dep in FX
	})
}
