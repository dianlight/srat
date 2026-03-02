package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/internal/appsetup"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/tlog"

	"go.uber.org/fx"
)

func openAPIFilenames(dir string) (string, string) {
	if dir == "" {
		return "/openapi.yaml", "/openapi.json"
	}
	cleaned := filepath.Clean(dir)
	return filepath.Join(cleaned, "openapi.yaml"), filepath.Join(cleaned, "openapi.json")
}

func applyMockEnv(enabled bool) {
	if enabled {
		os.Setenv("SRAT_MOCK", "true")
	} else {
		os.Unsetenv("SRAT_MOCK")
	}
}

func main() {
	// set global logger with custom options
	logLevelString := flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")
	output := flag.String("out", "./docs/", "Output directory where create openapi.* files")
	mockMode := flag.Bool("mock", true, "Use mock data for generation")

	flag.Usage = func() {
		flag.PrintDefaults()
	}

	flag.Parse()
	applyMockEnv(*mockMode)

	yamlPath, jsonPath := openAPIFilenames(*output)

	internal.Banner("srat-openapi", "")

	err := tlog.SetLevelFromString(*logLevelString)
	if err != nil {
		log.Fatalf("Invalid log level: %s", *logLevelString)
	}

	apiCtx, apiCancel := context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
	defer apiCancel() // Ensure context is cancelled on exit
	staticConfig := dto.ContextState{
		ReadOnlyMode:  false,
		DatabasePath:  "file::memory:?cache=shared&_pragma=foreign_keys(1)",
		HACoreReady:   false, // We don't need HA integration for OpenAPI
		ProtectedMode: true,  // No real services are running
	}

	appParams := appsetup.BaseAppParams{
		Ctx:          apiCtx,
		CancelFn:     apiCancel,
		StaticConfig: &staticConfig,
	}

	// New FX
	app := fx.New(
		appsetup.NewFXLoggerOption(),
		appsetup.ProvideCoreDependencies(appParams),
		appsetup.ProvideCyclicDependencyWorkaroundOption(),
		fx.Provide(
			server.AsHumaRoute(api.NewSSEBroker),
			server.AsHumaRoute(api.NewHealthHandler),
			server.AsHumaRoute(api.NewShareHandler),
			server.AsHumaRoute(api.NewVolumeHandler),
			server.AsHumaRoute(api.NewSmartHandler),
			server.AsHumaRoute(api.NewSettingsHanler),
			server.AsHumaRoute(api.NewUserHandler),
			server.AsHumaRoute(api.NewSambaHanler),
			server.AsHumaRoute(api.NewUpgradeHanler),
			server.AsHumaRoute(api.NewSystemHanler),
			server.AsHumaRoute(api.NewFilesystemHandler),
			server.AsHumaRoute(api.NewIssueAPI),
			server.AsHumaRoute(api.NewTelemetryHandler),
			server.AsHumaRoute(api.NewHDIdleHandler),
			api.NewWebSocketBroker,
			server.NewMuxRouter,
			server.NewHumaAPI,
		),
		fx.Invoke(
			func(
				api huma.API,
				shutdowner fx.Shutdowner,
			) {
				yaml, err := api.OpenAPI().YAML()
				if err != nil {
					slog.Error("Unable to generate YAML", "error", err)
				}
				err = os.WriteFile(yamlPath, yaml, 0o600)
				if err != nil {
					slog.Error("Unable to write YAML", "error", err)
				}
				json, err := api.OpenAPI().MarshalJSON()
				if err != nil {
					slog.Error("Unable to generate JSON", "error", err)
				}
				err = os.WriteFile(jsonPath, json, 0o600)
				if err != nil {
					slog.Error("Unable to write JSON", "error", err)
				}
			},
		),
	)

	app.Start(context.Background())
	// apiCancel is deferred
	app.Stop(context.Background())

	os.Exit(0)
}
