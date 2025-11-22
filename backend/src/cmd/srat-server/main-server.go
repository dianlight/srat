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
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/smartmontools-go"
	"github.com/dianlight/srat/internal/appsetup"
	"github.com/dianlight/tlog"
	"github.com/gorilla/mux"
	"gitlab.com/tozd/go/errors"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/server"

	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
	"go.uber.org/fx"
)

var smbConfigFile *string

var http_port *int
var secureMode *bool
var dockerInterface *string
var dockerNetwork *string
var roMode *bool
var updateFilePath *string
var dbfile *string
var supervisorURL *string
var supervisorToken *string
var addonIpAddress *string
var logLevelString *string
var protectedMode *bool

func validateSambaConfig(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("missing samba config")
	}
	return nil
}

type serverContextOptions struct {
	AddonIPAddress  string
	ReadOnlyMode    bool
	ProtectedMode   bool
	SecureMode      bool
	UpdateFilePath  string
	SambaConfigFile string
	Template        []byte
	DockerInterface string
	DockerNetwork   string
	DatabasePath    string
	SupervisorToken string
	SupervisorURL   string
	Heartbeat       int
	StartTime       time.Time
}

func buildServerContextState(opts serverContextOptions) dto.ContextState {
	return dto.ContextState{
		AddonIpAddress:  opts.AddonIPAddress,
		ReadOnlyMode:    opts.ReadOnlyMode,
		ProtectedMode:   opts.ProtectedMode,
		SecureMode:      opts.SecureMode,
		UpdateFilePath:  opts.UpdateFilePath,
		SambaConfigFile: opts.SambaConfigFile,
		Template:        opts.Template,
		DockerInterface: opts.DockerInterface,
		DockerNet:       opts.DockerNetwork,
		Heartbeat:       opts.Heartbeat,
		SupervisorURL:   opts.SupervisorURL,
		SupervisorToken: opts.SupervisorToken,
		DatabasePath:    opts.DatabasePath,
		StartTime:       opts.StartTime,
	}
}

func main() {

	http_port = flag.Int("port", 8080, "Http Port on listen to")
	smbConfigFile = flag.String("out", "", "Output samba conf file")
	roMode = flag.Bool("ro", false, "Read only mode")
	secureMode = flag.Bool("addon", false, "Run in addon mode - this will enable HomeAssistant Supervisor API and ingress support and authenticate via Supervisor Token")
	dbfile = flag.String("db", "file::memory:?cache=shared&_pragma=foreign_keys(1)", "Database file")
	dockerInterface = flag.String("docker-interface", "", "Docker interface")
	dockerNetwork = flag.String("docker-network", "", "Docker network")
	protectedMode = flag.Bool("protected-mode", false, "Addon protected mode")

	if !internal.Is_embed {
		internal.Frontend = flag.String("frontend", "", "Frontend path - if missing the internal is used")
		internal.TemplateFile = flag.String("template", "", "Template file")
	}
	supervisorToken = flag.String("ha-token", os.Getenv("SUPERVISOR_TOKEN"), "HomeAssistant Supervisor Token")
	supervisorURL = flag.String("ha-url", "http://supervisor/", "HomeAssistant Supervisor URL")
	logLevelString = flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")
	singleInstance := flag.Bool("single-instance", false, "Single instance mode - only one instance of the addon can run ***ONLY FOR DEBUG***")
	updateFilePath = flag.String("update-file-path", os.TempDir()+"/"+filepath.Base(os.Args[0]), "Update file path - used for addon updates")
	addonIpAddress = flag.String("ip-address", "127.0.0.1", "Addon IP address // $(bashio::addon.ip_address)")

	flag.Parse()

	err := tlog.SetLevelFromString(*logLevelString)
	if err != nil {
		log.Fatalf("Invalid log level: %s", *logLevelString)
	}

	flag.Usage = func() {
		internal.Banner("srat")
		flag.PrintDefaults()
	}

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
		slog.Debug("Stopping main process", "pid", os.Getpid())
	}
}

//type writeDeadliner interface {
//	SetWriteDeadline(time.Time) error
//}

func prog(state overseer.State) {

	internal.Banner("srat-server")

	slog.Debug("Startup Options", "Flags", os.Args)
	slog.Debug("Starting SRAT", "version", config.Version, "pid", state.ID, "address", state.Address, "listeners", fmt.Sprintf("%T", state.Listener))

	if err := validateSambaConfig(*smbConfigFile); err != nil {
		log.Fatalf("Missing samba config! %s", *smbConfigFile)
	}

	if *roMode {
		log.Println("Read only mode")
	}

	//if !strings.Contains(*dbfile, "?") {
	//	*dbfile = *dbfile + "?cache=shared&_pragma=foreign_keys(1)"
	//}

	apiCtx, apiCancel := context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
	// apiCancel is called at the end of Run() by FX lifecycle or explicitly if Run errors

	staticConfig := buildServerContextState(serverContextOptions{
		AddonIPAddress:  *addonIpAddress,
		ReadOnlyMode:    *roMode,
		ProtectedMode:   *protectedMode,
		SecureMode:      *secureMode,
		UpdateFilePath:  *updateFilePath,
		SambaConfigFile: *smbConfigFile,
		Template:        internal.GetTemplateData(),
		DockerInterface: *dockerInterface,
		DockerNetwork:   *dockerNetwork,
		DatabasePath:    *dbfile,
		SupervisorToken: *supervisorToken,
		SupervisorURL:   *supervisorURL,
		Heartbeat:       5,
		StartTime:       time.Now(),
	})

	appParams := appsetup.BaseAppParams{
		Ctx:          apiCtx,
		CancelFn:     apiCancel,
		StaticConfig: &staticConfig,
	}

	// New FX
	app := fx.New(
		fx.StartTimeout(120*time.Second), // Increase timeout for slow startups (e.g., HA websocket connection)
		appsetup.NewFXLoggerOption(),
		appsetup.ProvideCoreDependencies(appParams),
		appsetup.ProvideHAClientDependencies(appParams),
		appsetup.ProvideFrontendOption(),
		appsetup.ProvideCyclicDependencyWorkaroundOption(),
		fx.Provide(
			func() *overseer.State { return &state },
			server.AsHumaRoute(api.NewSSEBroker),
			smartmontools.NewClient,
			api.NewWebSocketBroker,
			server.AsHumaRoute(api.NewHealthHandler),
			server.AsHumaRoute(api.NewShareHandler),
			server.AsHumaRoute(api.NewVolumeHandler),
			server.AsHumaRoute(api.NewSmartHandler),
			server.AsHumaRoute(api.NewSettingsHanler),
			server.AsHumaRoute(api.NewUserHandler),
			server.AsHumaRoute(api.NewSambaHanler),
			server.AsHumaRoute(api.NewUpgradeHanler),
			server.AsHumaRoute(api.NewSystemHanler),
			server.AsHumaRoute(api.NewIssueAPI),
			server.AsHumaRoute(api.NewTelemetryHandler),
			server.AsHumaRoute(api.NewHDIdleHandler),
			server.NewMuxRouter,
			server.NewHTTPServer,
			server.NewHumaAPI,
		),
		fx.Invoke(
			func(
				_ *http.Server,
				_ huma.API,
				router *mux.Router,
				static http.FileSystem,
				wsHandler *api.WebSocketHandler,
				//apiCtx *dto.ContextState,
				//shutdowner fx.Shutdowner,
			) {
				// WebSocket route
				router.HandleFunc("/ws", wsHandler.HandleWebSocket).Methods(http.MethodGet)

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
		),
		fx.Invoke(func(
			lc fx.Lifecycle,
			props_repo repository.PropertyRepositoryInterface,
			//samba_service service.SambaServiceInterface,
			//			hdidle_service service.HDIdleServiceInterface,
		) {
			// Setting the actual LogLevel
			err := props_repo.SetValue("LogLevel", *logLevelString)
			if err != nil {
				log.Fatalf("Cant set log level - %#+v", err)
			}

			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// Apply config to samba
					/*
						slog.Info("******* Applying Samba config ********")
						err = samba_service.WriteAndRestartSambaConfig()
						if err != nil {
							log.Fatalf("Cant apply samba config - %#+v", err)
						}
						slog.Info("******* Samba config applied! ********")
					*/
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return nil
				},
			})

		}),
	)

	if err := app.Err(); err != nil { // Check for errors from Provide functions
		log.Fatalf("Error during FX setup: %v", err)
	}

	app.Run() // This blocks until the application stops

	slog.Info("Stopping SRAT", "pid", state.ID)
	apiCtx.Value("wg").(*sync.WaitGroup).Wait() // Ensure background tasks complete
	apiCancel()                                 // Explicitly cancel context
	slog.Info("SRAT stopped", "pid", state.ID)
}
