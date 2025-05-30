package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gorilla/mux"
	"github.com/jpillora/overseer"
	"github.com/mattn/go-isatty"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"gitlab.com/tozd/go/errors"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"

	"github.com/jpillora/overseer/fetcher"
	"github.com/lmittmann/tint"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

var smbConfigFile *string

var http_port *int
var hamode *bool
var dockerInterface *string
var dockerNetwork *string
var roMode *bool
var updateFilePath *string
var dbfile *string
var supervisorURL *string
var supervisorToken *string
var logLevel slog.Level
var addonIpAddress *string

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	w := os.Stderr
	// set global logger with custom options

	http_port = flag.Int("port", 8080, "Http Port on listen to")
	smbConfigFile = flag.String("out", "", "Output samba conf file")
	roMode = flag.Bool("ro", false, "Read only mode")
	hamode = flag.Bool("addon", false, "Run in addon mode")
	dbfile = flag.String("db", "file::memory:?cache=shared&_pragma=foreign_keys(1)", "Database file")
	dockerInterface = flag.String("docker-interface", "", "Docker interface")
	dockerNetwork = flag.String("docker-network", "", "Docker network")
	if !internal.Is_embed {
		internal.Frontend = flag.String("frontend", "", "Frontend path - if missing the internal is used")
		internal.TemplateFile = flag.String("template", "", "Template file")
	}
	supervisorToken = flag.String("ha-token", os.Getenv("SUPERVISOR_TOKEN"), "HomeAssistant Supervisor Token")
	supervisorURL = flag.String("ha-url", "http://supervisor/", "HomeAssistant Supervisor URL")
	logLevelString := flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")
	singleInstance := flag.Bool("single-instance", false, "Single instance mode - only one instance of the addon can run ***ONLY FOR DEBUG***")
	updateFilePath = flag.String("update-file-path", os.TempDir()+"/"+filepath.Base(os.Args[0]), "Update file path - used for addon updates")
	addonIpAddress = flag.String("ip-address", "127.0.0.1", "Addon IP address // $(bashio::addon.ip_address)")

	flag.Usage = func() {
		internal.Banner("srat")
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

	if *singleInstance {
		slog.Debug("Single instance mode")
		// Run the program directly
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *http_port))
		if err != nil {
			log.Fatalf("Failed to listen on port %d: %s", *http_port, err)
		}
		prog(overseer.State{
			Address:  fmt.Sprintf(":%d", *http_port),
			Listener: listener,
		})
		os.Exit(0)
	} else {
		overseer.Run(overseer.Config{
			Program: prog,
			Address: fmt.Sprintf(":%d", *http_port),
			Fetcher: &fetcher.File{
				Path:     *updateFilePath,
				Interval: 1 * time.Second,
			},
			TerminateTimeout: 60,
			Debug:            false,
		})
	}
}

type writeDeadliner interface {
	SetWriteDeadline(time.Time) error
}

func prog(state overseer.State) {

	internal.Banner("srat")

	slog.Debug("Startup Options", "Flags", os.Args)
	slog.Debug("Starting SRAT", "version", config.Version, "pid", state.ID, "address", state.Address, "listeners", fmt.Sprintf("%T", state.Listener))

	if *smbConfigFile == "" {
		log.Fatalf("Missing samba config! %s", *smbConfigFile)
	}

	if *roMode {
		log.Println("Read only mode")
	}

	if !strings.Contains(*dbfile, "?") {
		*dbfile = *dbfile + "?cache=shared&_pragma=foreign_keys(1)"
	}

	var apiContext, apiContextCancel = context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
	staticConfig := dto.ContextState{}
	staticConfig.AddonIpAddress = *addonIpAddress
	staticConfig.SupervisorURL = *supervisorURL
	staticConfig.UpdateFilePath = *updateFilePath
	staticConfig.ReadOnlyMode = *roMode
	staticConfig.SambaConfigFile = *smbConfigFile
	staticConfig.Template = internal.GetTemplateData()
	staticConfig.DockerInterface = *dockerInterface
	staticConfig.DockerNet = *dockerNetwork

	// New FX
	fx.New(
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
			func() *overseer.State { return &state },
			internal.GetFrontend,
			fx.Annotate(
				func() bool { return *hamode },
				fx.ResultTags(`name:"ha_mode"`),
			),
			fx.Annotate(
				func() string { return *dbfile },
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
			),
			server.NewHTTPServer,
			server.NewHumaAPI,
			func() *securityprovider.SecurityProviderBearerToken {
				sp, err := securityprovider.NewSecurityProviderBearerToken(*supervisorToken)
				if err != nil {
					log.Fatalf("Failed to create security provider: %s", err)
				}
				return sp
			},
			func(bearerAuth *securityprovider.SecurityProviderBearerToken) *ingress.ClientWithResponses {
				ingressClient, err := ingress.NewClientWithResponses(*supervisorURL, ingress.WithRequestEditorFn(bearerAuth.Intercept))
				if err != nil {
					log.Fatal(err)
				}
				return ingressClient
			},
			func(bearerAuth *securityprovider.SecurityProviderBearerToken) hardware.ClientWithResponsesInterface {
				hardwareClient, err := hardware.NewClientWithResponses(*supervisorURL, hardware.WithRequestEditorFn(bearerAuth.Intercept))
				if err != nil {
					log.Fatal(err)
				}
				return hardwareClient
			},
			func(bearerAuth *securityprovider.SecurityProviderBearerToken) mount.ClientWithResponsesInterface {
				mountClient, err := mount.NewClientWithResponses(*supervisorURL, mount.WithRequestEditorFn(bearerAuth.Intercept))
				if err != nil {
					log.Fatal(err)
				}
				return mountClient
			},
		),
		fx.Invoke(
			fx.Annotate(
				func(
					_ *http.Server,
					api huma.API,
					router *mux.Router,
					static http.FileSystem,
					ha_mode bool,
					shutdowner fx.Shutdowner,
				) {
					// Addon-LocalUpdate deploy
					if !ha_mode {
						executablePath, err := os.Executable()
						if err != nil {
							slog.Error("Error getting executable path:", "err", err)
						} else {
							slog.Debug("Serving executable", "path", executablePath)
							router.Path("/srat_" + runtime.GOARCH).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								http.ServeFile(w, r, executablePath)
							}).Methods(http.MethodGet)
							if runtime.GOARCH != "amd64" {
								router.Path("/srat_x86_64").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
									http.ServeFile(w, r, executablePath+"_x86_64")
								}).Methods(http.MethodGet)
							}
						}

					}

					// Static Routes
					router.PathPrefix("/").Handler(http.FileServer(static)).Methods(http.MethodGet)
					//
					router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
						template, err := route.GetPathTemplate()
						if err != nil {
							return errors.WithMessage(err)
						}
						slog.Debug("Route:", "template", template)
						return nil
					})
				},
				fx.ParamTags("", "", "", "", `name:"ha_mode"`),
			),
		),
	).Run()

	slog.Info("Stopping SRAT", "pid", state.ID)
	//dbom.CloseDB()
	apiContext.Value("wg").(*sync.WaitGroup).Wait()
	slog.Info("SRAT stopped", "pid", state.ID)

	os.Exit(0)
}
