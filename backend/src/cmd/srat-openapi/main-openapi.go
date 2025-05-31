package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mattn/go-isatty"
	"github.com/xorcare/pointer"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"

	"github.com/lmittmann/tint"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

var output *string
var logLevel slog.Level

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	w := os.Stderr
	// set global logger with custom options
	logLevelString := flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")
	output = flag.String("out", "./docs/", "Output directory where create openapi.* files")
	internal.Banner("srat-openapi")

	flag.Usage = func() {
		flag.PrintDefaults()
	}

	flag.Parse()

	switch *logLevelString {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		log.Fatalf("Invalid log level: %s", *logLevelString)
	}

	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      logLevel,
			TimeFormat: time.RFC3339,
			NoColor:    !isatty.IsTerminal(w.Fd()),
			AddSource:  true,
		}),
	))

	var apiContext, apiContextCancel = context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
	staticConfig := dto.ContextState{}

	// New FX
	app := fx.New(
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			log.Debug("Starting FX")
			fxlog := &fxevent.SlogLogger{
				Logger: log,
			}
			fxlog.UseLogLevel(slog.LevelDebug)
			fxlog.UseErrorLevel(slog.LevelError)
			return fxlog
		}),
		fx.Provide(
			dbom.NewDB,
			func() *slog.Logger { return slog.Default() },
			func() (context.Context, context.CancelFunc) { return apiContext, apiContextCancel },
			func() *dto.ContextState { return &staticConfig },
			fx.Annotate(
				func() bool { return true },
				fx.ResultTags(`name:"ha_mode"`),
			),
			fx.Annotate(
				func() string { return *pointer.String("file::memory:?cache=shared&_pragma=foreign_keys(1)") },
				fx.ResultTags(`name:"db_path"`),
			),
			lsblk.NewLSBKInterpreter,
			service.NewBroadcasterService,
			service.NewVolumeService,
			service.NewSambaService,
			service.NewUpgradeService,
			service.NewDirtyDataService,
			service.NewSupervisorService,
			service.NewFilesystemService,
			service.NewShareService,
			repository.NewMountPointPathRepository,
			repository.NewExportedShareRepository,
			repository.NewPropertyRepositoryRepository,
			repository.NewSambaUserRepository,
			server.AsHumaRoute(api.NewSSEBroker),
			server.AsHumaRoute(api.NewHealthHandler),
			server.AsHumaRoute(api.NewShareHandler),
			server.AsHumaRoute(api.NewVolumeHandler),
			server.AsHumaRoute(api.NewSettingsHanler),
			server.AsHumaRoute(api.NewUserHandler),
			server.AsHumaRoute(api.NewSambaHanler),
			server.AsHumaRoute(api.NewUpgradeHanler),
			server.AsHumaRoute(api.NewSystemHanler),
			fx.Annotate(
				server.NewMuxRouter,
				fx.ParamTags(`name:"ha_mode"`),
			), //server.NewHTTPServer,
			server.NewHumaAPI,
		),
		fx.Invoke(
			func(
				api huma.API,
				shutdowner fx.Shutdowner,
			) {
				yaml, err := api.OpenAPI().YAML()
				if err != nil {
					slog.Error("Unable to generate YAML", "err", err)
				}
				err = os.WriteFile(*output+"/openapi.yaml", yaml, 0644)
				if err != nil {
					slog.Error("Unable to write YAML", "err", err)
				}
				json, err := api.OpenAPI().MarshalJSON()
				if err != nil {
					slog.Error("Unable to generate JSON", "err", err)
				}
				err = os.WriteFile(*output+"/openapi.json", json, 0644)
				if err != nil {
					slog.Error("Unable to write JSON", "err", err)
				}
			},
		),
	)

	app.Start(context.Background())
	apiContextCancel()
	app.Stop(context.Background())

	os.Exit(0)
}
