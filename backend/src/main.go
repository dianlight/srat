package main

//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.8.3 gen ./converter

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
	"github.com/m1/go-generate-password/generator"
	"github.com/mattn/go-isatty"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"gitlab.com/tozd/go/errors"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/unixsamba"

	"github.com/jpillora/overseer/fetcher"
	"github.com/lmittmann/tint"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"moul.io/banner"
)

var options *config.Options
var smbConfigFile *string

// var globalRouter *mux.Router
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
var addonIpAddress *string

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	w := os.Stderr
	// set global logger with custom options

	optionsFile = flag.String("opt", "/data/options.json", "Addon Options json file")
	configFile = flag.String("conf", "", "Config json file, can be omitted if used in a pipe")
	http_port = flag.Int("port", 8080, "Http Port on listen to")
	smbConfigFile = flag.String("out", "", "Output samba conf file")
	roMode = flag.Bool("ro", false, "Read only mode")
	hamode = flag.Bool("addon", false, "Run in addon mode")
	dbfile = flag.String("db", "file::memory:?cache=shared&_pragma=foreign_keys(1)", "Database file")
	dockerInterface = flag.String("docker-interface", "", "Docker interface")
	dockerNetwork = flag.String("docker-network", "", "Docker network")
	if !is_embed {
		frontend = flag.String("frontend", "", "Frontend path - if missing the internal is used")
		templateFile = flag.String("template", "", "Template file")
	}
	supervisorToken = flag.String("ha-token", os.Getenv("SUPERVISOR_TOKEN"), "HomeAssistant Supervisor Token")
	supervisorURL = flag.String("ha-url", "http://supervisor/", "HomeAssistant Supervisor URL")
	logLevelString := flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")
	singleInstance := flag.Bool("single-instance", false, "Single instance mode - only one instance of the addon can run ***ONLY FOR DEBUG***")
	automount = flag.Bool("automount", false, "Automount mode - mount all shares automatically")
	updateFilePath = flag.String("update-file-path", os.TempDir()+"/"+filepath.Base(os.Args[0]), "Update file path - used for addon updates")
	addonIpAddress = flag.String("ip-address", "$(bashio::addon.ip_address)", "Addon IP address")

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

	if *smbConfigFile == "" {
		log.Fatalf("Missing samba config! %s", *smbConfigFile)
	}

	if *roMode {
		log.Println("Read only mode")
	}

	if !strings.Contains(*dbfile, "?") {
		*dbfile = *dbfile + "?cache=shared&_pragma=foreign_keys(1)"
	}

	//dbom.InitDB(*dbfile)

	// Get options
	options = config.ReadOptionsFile(*optionsFile)

	var apiContext, apiContextCancel = context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
	staticConfig := dto.ContextState{}
	staticConfig.AddonIpAddress = *addonIpAddress
	staticConfig.UpdateFilePath = *updateFilePath
	staticConfig.ReadOnlyMode = *roMode
	staticConfig.SambaConfigFile = *smbConfigFile
	staticConfig.Template = getTemplateData()
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
			getFrontend,
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
		fx.Invoke(func(
			mount_repo repository.MountPointPathRepositoryInterface,
			props_repo repository.PropertyRepositoryInterface,
			exported_share_repo repository.ExportedShareRepositoryInterface,
			hardwareClient hardware.ClientWithResponsesInterface,
			samba_user_repo repository.SambaUserRepositoryInterface,
			volume_service service.VolumeServiceInterface,
		) {
			versionInDB, err := props_repo.Value("version", true)
			if err != nil || versionInDB.(string) == "" {
				// Migrate from JSON to DB
				var config config.Config
				err := config.LoadConfig(*configFile)
				// Setting/Properties
				if err != nil {
					log.Fatalf("Cant load config file %#+v", err)
				}

				pwdgen, err := generator.NewWithDefault()
				if err != nil {
					log.Fatalf("Cant generate password %#+v", err)
				}
				_ha_mount_user_password_, err := pwdgen.Generate()
				if err != nil {
					log.Fatalf("Cant generate password %#+v", err)
				}

				err = unixsamba.CreateSambaUser("_ha_mount_user_", *_ha_mount_user_password_, unixsamba.UserOptions{
					CreateHome:    false,
					SystemAccount: false,
					Shell:         "/sbin/nologin",
				})
				if err != nil {
					log.Fatalf("Cant create samba user %#+v", err)
				}

				err = firstTimeJSONImporter(config, mount_repo, props_repo, exported_share_repo, samba_user_repo, *_ha_mount_user_password_)
				if err != nil {
					log.Fatalf("Cant import json settings - %#+v", err)
				}

			} else {
				if automount != nil && *automount {
					// Automount all shares
					slog.Info("******* Automounting all shares! ********")
					all, err := mount_repo.All()
					if err != nil {
						log.Fatalf("Cant load mounts - %#+v", err)
					}
					for _, mnt := range all {
						if mnt.Type == "ADDON" && !mnt.IsMounted {
							slog.Info("Automounting share", "path", mnt.Path)
							err := volume_service.MountVolume(dto.MountPointData{
								Path:   mnt.Path,
								Device: mnt.Device,
								FSType: mnt.FSType,
								Flags:  mnt.Flags.ToStringSlice(),
							})
							if err != nil {
								slog.Error("Error automounting share", "path", mnt.Path, "err", err)
							}
						}
					}
				}
			}
		}),
		fx.Invoke(
			fx.Annotate(
				func(
					_ *http.Server,
					api huma.API,
					router *mux.Router,
					static http.FileSystem,
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
					/*
						hfiles, err := static.Open(".")
						if err != nil {
							slog.Error("Error reading static directory:", "err", err)
							panic(err)
						}
						files, err := hfiles.Readdir(0)
						if err != nil {
							slog.Error("Error reading static directory:", "err", err)
							panic(err)
						}
						slog.Debug("Static files:", "files", files)
					*/
					router.PathPrefix("/").Handler(http.FileServer(static)).Methods(http.MethodGet)
					/*
						if slices.ContainsFunc(files, func(f fs.DirEntry) bool {
							return f.IsDir() && f.Name() == "static"
						}) {
							fsRoot, _ := fs.Sub(static, "static")
							router.PathPrefix("/").Handler(http.FileServerFS(fsRoot)).Methods(http.MethodGet)
						} else {
							slog.Warn("Static directory not found:", "err", err)
							router.Path("/").Handler(http.FileServerFS(static)).Methods(http.MethodGet)
						}
					*/
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
		fx.Invoke(func(
			samba_service service.SambaServiceInterface,
		) {
			// Generate Samba Config and restart Samba
			samba_service.WriteAndRestartSambaConfig()
		},
		),
	).Run()

	slog.Info("Stopping SRAT", "pid", state.ID)
	//dbom.CloseDB()
	apiContext.Value("wg").(*sync.WaitGroup).Wait()
	slog.Info("SRAT stopped", "pid", state.ID)

	os.Exit(0)
}

func firstTimeJSONImporter(config config.Config,
	mount_repository repository.MountPointPathRepositoryInterface,
	props_repository repository.PropertyRepositoryInterface,
	export_share_repository repository.ExportedShareRepositoryInterface,
	users_repository repository.SambaUserRepositoryInterface,
	_ha_mount_user_password_ string,
) (err error) {

	var conv converter.ConfigToDbomConverterImpl
	shares := &[]dbom.ExportedShare{}
	properties := &dbom.Properties{}
	users := &dbom.SambaUsers{}

	err = conv.ConfigToDbomObjects(config, properties, users, shares)
	if err != nil {
		return errors.WithStack(err)
	}
	properties.AddInternalValue("_ha_mount_user_password_", _ha_mount_user_password_)

	err = props_repository.SaveAll(properties)
	if err != nil {
		return errors.WithStack(err)
	}
	err = users_repository.SaveAll(users)
	if err != nil {
		return errors.WithStack(err)
	}
	for i, share := range *shares {
		err = mount_repository.Save(&share.MountPointData)
		if err != nil {
			return errors.WithStack(err)
		}
		//		slog.Debug("Share ", "id", share.MountPointData.ID)
		(*shares)[i] = share
	}
	err = export_share_repository.SaveAll(shares)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
