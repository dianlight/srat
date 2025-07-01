package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/internal/appsetup"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/tlog"

	"go.uber.org/fx"
)

var output *string

func main() {
	// set global logger with custom options
	logLevelString := flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")
	output = flag.String("out", "./docs/", "Output directory where create openapi.* files")
	internal.Banner("srat-openapi")

	flag.Usage = func() {
		flag.PrintDefaults()
	}

	flag.Parse()

	err := tlog.SetLevelFromString(*logLevelString)
	if err != nil {
		log.Fatalf("Invalid log level: %s", *logLevelString)
	}

	apiCtx, apiCancel := context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
	defer apiCancel() // Ensure context is cancelled on exit
	staticConfig := dto.ContextState{
		ReadOnlyMode: false,
		DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)",
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
			server.AsHumaRoute(api.NewSettingsHanler),
			server.AsHumaRoute(api.NewUserHandler),
			server.AsHumaRoute(api.NewSambaHanler),
			server.AsHumaRoute(api.NewUpgradeHanler),
			server.AsHumaRoute(api.NewSystemHanler),
			server.AsHumaRoute(api.NewIssueAPI),
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
				err = os.WriteFile(*output+"/openapi.yaml", yaml, 0644)
				if err != nil {
					slog.Error("Unable to write YAML", "error", err)
				}
				json, err := api.OpenAPI().MarshalJSON()
				if err != nil {
					slog.Error("Unable to generate JSON", "error", err)
				}
				err = os.WriteFile(*output+"/openapi.json", json, 0644)
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
