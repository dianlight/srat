package main

//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.8.0 gen ./converter

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gorilla/mux"
	"github.com/jpillora/overseer"
	"github.com/mattn/go-isatty"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbutil"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"

	"github.com/jpillora/overseer/fetcher"
	"github.com/lmittmann/tint"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"moul.io/banner"
)

var options *config.Options
var smbConfigFile *string

// var globalRouter *mux.Router
var templateData []byte
var optionsFile *string
var http_port *int
var templateFile *string
var wait time.Duration
var hamode *bool
var dockerInterface *string
var dockerNetwork *string
var roMode *bool
var updateFilePath *string
var configFile *string
var dbfile *string
var frontend *string
var supervisorURL *string
var supervisorToken *string
var logLevel slog.Level
var automount *bool

// Static files
//
//go:embed static/*
var content embed.FS

//go:embed templates/smb.gtpl
var defaultTemplate embed.FS

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	w := os.Stderr
	// set global logger with custom options

	optionsFile = flag.String("opt", "/data/options.json", "Addon Options json file")
	configFile = flag.String("conf", "", "Config json file, can be omitted if used in a pipe")
	http_port = flag.Int("port", 8080, "Http Port on listen to")
	templateFile = flag.String("template", "", "Template file")
	smbConfigFile = flag.String("out", "", "Output samba conf file")
	roMode = flag.Bool("ro", false, "Read only mode")
	hamode = flag.Bool("addon", false, "Run in addon mode")
	dbfile = flag.String("db", "file::memory:?cache=shared&_pragma=foreign_keys(1)", "Database file")
	dockerInterface = flag.String("docker-interface", "", "Docker interface")
	dockerNetwork = flag.String("docker-network", "", "Docker network")
	frontend = flag.String("frontend", "", "Frontend path - if missing the internal is used")
	supervisorToken = flag.String("ha-token", os.Getenv("SUPERVISOR_TOKEN"), "HomeAssistant Supervisor Token")
	supervisorURL = flag.String("ha-url", "http://supervisor/", "HomeAssistant Supervisor URL")
	logLevelString := flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")
	singleInstance := flag.Bool("single-instance", false, "Single instance mode - only one instance of the addon can run ***ONLY FOR DEBUG***")
	automount = flag.Bool("automount", false, "Automount mode - mount all shares automatically")
	updateFilePath = flag.String("update-file-path", os.TempDir()+"/"+filepath.Base(os.Args[0]), "Update file path - used for addon updates")

	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")

	// Headless CLI mode (execute a command and exit)
	//show_volumes := flag.Bool("show-volumes", false, "Show volumes in headless CLI mode and exit")

	flag.Usage = func() {
		writer := flag.CommandLine.Output()

		fmt.Fprint(writer, "SRAT: SambaNAS Rest Administration Interface\n")
		fmt.Fprintf(writer, "Version: %s\n", config.Version)
		fmt.Fprintf(writer, "Commit Hash: %s\n", config.CommitHash)
		fmt.Fprintf(writer, "Build Timestamp: %s\n", config.BuildTimestamp)
		fmt.Fprintf(writer, "Documentation: https://github.com/dianlight/SRAT\n\n")

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

	fmt.Println(banner.Inline("srat"))
	fmt.Printf("SambaNAS Rest Administration Interface (%s)\n", state.ID)
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Commit Hash: %s\n", config.CommitHash)
	fmt.Printf("Build Timestamp: %s\n", config.BuildTimestamp)
	fmt.Printf("Listening on %v\n\n", state.Addresses)
	slog.Debug("Startup Options", "Flags", os.Args)

	slog.Debug("Starting SRAT", "version", config.Version, "pid", state.ID, "address", state.Address, "listeners", fmt.Sprintf("%T", state.Listener))

	// Check template file
	if *templateFile == "" {
		templateDatan, err := defaultTemplate.ReadFile("templates/smb.gtpl")
		if err != nil {
			log.Fatal(err)
		}
		templateData = templateDatan
	} else {
		templateDatan, err := os.ReadFile(*templateFile)
		if err != nil {
			log.Fatalf("Cant read template file %s - %s", *templateFile, err)
		}
		templateData = templateDatan
	}

	if len(templateData) == 0 {
		log.Fatal("Missing template file")
	}

	if *smbConfigFile == "" {
		log.Fatalf("Missing samba config! %s", *smbConfigFile)
	}

	if *roMode {
		log.Println("Read only mode")
	}

	dbom.InitDB(*dbfile + "?cache=shared&_pragma=foreign_keys(1)")

	// Get options
	options = config.ReadOptionsFile(*optionsFile)

	var apiContext, apiContextCancel = context.WithCancel(context.Background())
	sharedResources := dto.ContextState{}
	sharedResources.UpdateFilePath = *updateFilePath
	sharedResources.ReadOnlyMode = *roMode
	sharedResources.SambaConfigFile = *smbConfigFile
	sharedResources.Template = templateData
	sharedResources.DockerInterface = *dockerInterface
	sharedResources.DockerNet = *dockerNetwork

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
			func() *gorm.DB { return dbom.GetDB() },
			func() *slog.Logger { return slog.Default() },
			func() (context.Context, context.CancelFunc) { return apiContext, apiContextCancel },
			func() *dto.ContextState { return &sharedResources },
			func() *overseer.State { return &state },
			func() fs.FS {
				if frontend == nil || *frontend == "" {
					return content
				} else {
					_, err := os.Stat(*frontend)
					if err != nil {
						log.Fatalf("Cant access frontend folder %s - %s", *frontend, err)
					}
					return os.DirFS(*frontend)
				}
			},
			fx.Annotate(
				func() bool { return *hamode },
				fx.ResultTags(`name:"ha_mode"`),
			),
			service.NewBroadcasterService,
			service.NewVolumeService,
			service.NewSambaService,
			service.NewUpgradeService,
			service.NewDirtyDataService,
			repository.NewMountPointPathRepository,
			repository.NewExportedShareRepository,
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
		),
		fx.Invoke(func(
			mount_repo repository.MountPointPathRepositoryInterface,
			exported_share_repo repository.ExportedShareRepositoryInterface,
			hardwareClient hardware.ClientWithResponsesInterface,
		) {
			// JSON Config  Migration if necessary
			// Get config and migrate if DB is empty
			var properties dbom.Properties
			err := properties.Load()
			if err != nil {
				log.Fatalf("Cant load properties - %#+v", err)
			}
			versionInDB, err := properties.GetValue("version")
			if err != nil || versionInDB.(string) == "" {
				// Migrate from JSON to DB
				var config config.Config
				err := config.LoadConfig(*configFile)
				// Setting/Properties
				if err != nil {
					log.Fatalf("Cant load config file %#+v", err)
				}
				err = dbutil.FirstTimeJSONImporter(config, mount_repo, exported_share_repo)
				if err != nil {
					log.Fatalf("Cant import json settings - %#+v", err)
				}
			} else {
				if automount != nil && *automount {
					// Automount all shares
					slog.Info("******* Automounting all shares! ********")
				}
			}
		}),
		fx.Invoke(
			fx.Annotate(
				func(
					_ *http.Server,
					api huma.API,
					router *mux.Router,
					static fs.FS,
					ha_mode bool,
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
					_, err := fs.ReadDir(static, "static")
					if err != nil {
						slog.Warn("Static directory not found:", "err", err)
						router.Path("/{file}.html").Handler(http.FileServerFS(static)).Methods(http.MethodGet)
					} else {
						fsRoot, _ := fs.Sub(static, "static")
						router.PathPrefix("/").Handler(http.FileServerFS(fsRoot)).Methods(http.MethodGet)
					}

					//
					router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
						template, err := route.GetPathTemplate()
						if err != nil {
							return errors.WithMessage(err)
						}
						slog.Debug("Route:", "template", template)
						return nil
					})

					if !ha_mode {
						yaml, err := api.OpenAPI().YAML()
						if err != nil {
							slog.Error("Unable to generate YAML", "err", err)
						}
						err = os.WriteFile("docs/openapi.yaml", yaml, 0644)
						if err != nil {
							slog.Error("Unable to write YAML", "err", err)
						}
						json, err := api.OpenAPI().MarshalJSON()
						if err != nil {
							slog.Error("Unable to generate JSON", "err", err)
						}
						err = os.WriteFile("docs/openapi.json", json, 0644)
						if err != nil {
							slog.Error("Unable to write JSON", "err", err)
						}
					}

				},
				fx.ParamTags("", "", "", "", `name:"ha_mode"`),
			),
		),
	).Run()

	dbom.CloseDB()

	log.Println("shutting down ")
	os.Exit(0)
}
