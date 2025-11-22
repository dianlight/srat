package main

import (
	"context"
	"flag"
	"log"
	"os"
	"sync"

	"gorm.io/gen"
	"gorm.io/gorm"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/internal/appsetup"
	"github.com/dianlight/tlog"

	"go.uber.org/fx"
)

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
	output := flag.String("out", "./src/repository/dao/", "Output directory where create generated files")
	mockMode := flag.Bool("mock", true, "Use mock data for generation")

	flag.Usage = func() {
		flag.PrintDefaults()
	}

	flag.Parse()
	applyMockEnv(*mockMode)

	internal.Banner("srat-gormgen")

	err := tlog.SetLevelFromString(*logLevelString)
	if err != nil {
		log.Fatalf("Invalid log level: %s", *logLevelString)
	}

	apiCtx, apiCancel := context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
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
		appsetup.ProvideCoreDependenciesWithoutDB(appParams),
		fx.Provide(dbom.NewDB),
		fx.Invoke(
			func(
				shutdowner fx.Shutdowner,
				gormdb *gorm.DB,
			) {
				g := gen.NewGenerator(gen.Config{
					OutPath:        *output,
					Mode:           gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
					WithUnitTest:   false,
					FieldNullable:  true,
					FieldCoverable: true,
					FieldSignable:  true,
				})

				g.UseDB(gormdb) // reuse your gorm db

				// Generate basic type-safe DAO API for struct following conventions
				g.ApplyBasic(dbom.HDIdleDevice{})
				g.ApplyBasic(dbom.MountPointPath{})

				// Generate Type Safe API with Dynamic SQL defined on Querier interface for `model.User` and `model.Company`
				//g.ApplyInterface(func(Querier) {}, model.User{}, model.Company{})

				// Generate the code
				g.Execute()

			},
		),
	)

	app.Start(context.Background())
	// apiCancel is deferred
	app.Stop(context.Background())

	os.Exit(0)
}
