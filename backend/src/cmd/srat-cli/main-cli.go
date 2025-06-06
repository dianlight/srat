package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/m1/go-generate-password/generator"
	"github.com/mattn/go-isatty"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/templates"
	"github.com/dianlight/srat/unixsamba"

	"github.com/lmittmann/tint"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// var options *config.Options
var smbConfigFile *string

// var globalRouter *mux.Router
//var optionsFile *string

// var wait time.Duration
var dockerInterface *string
var dockerNetwork *string
var configFile *string
var dbfile *string
var supervisorURL *string
var supervisorToken *string
var logLevel slog.Level
var logLevelString *string

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	w := os.Stderr

	supervisorToken = flag.String("ha-token", os.Getenv("SUPERVISOR_TOKEN"), "HomeAssistant Supervisor Token")
	supervisorURL = flag.String("ha-url", "http://supervisor/", "HomeAssistant Supervisor URL")
	dbfile = flag.String("db", "file::memory:?cache=shared&_pragma=foreign_keys(1)", "Database file")
	logLevelString = flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")

	// set global logger with custom options
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	/*optionsFile = */ startCmd.String("opt", "/data/options.json", "Addon Options json file (Unused for now)")
	configFile = startCmd.String("conf", "", "Addon SambaNas bootconfig json file to migrate from")
	smbConfigFile = startCmd.String("out", "", "Output samba conf file")
	dockerInterface = startCmd.String("docker-interface", "", "Docker interface")
	dockerNetwork = startCmd.String("docker-network", "", "Docker network")
	if !internal.Is_embed {
		//		internal.Frontend = flag.String("frontend", "", "Frontend path - if missing the internal is used")
		internal.TemplateFile = startCmd.String("template", "", "Template file")
	}

	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)

	//	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")

	// Headless CLI mode (execute a command and exit)
	//show_volumes := flag.Bool("show-volumes", false, "Show volumes in headless CLI mode and exit")

	internal.Banner("srat-cli")
	flag.Usage = func() {
		fmt.Printf("Usage %s <config_options...> <command> <command_options...>\n", os.Args[0])
		fmt.Println("Config Options:")
		flag.PrintDefaults()
		fmt.Println("Command start:")
		startCmd.PrintDefaults()
		fmt.Println("Command stop:")
		stopCmd.PrintDefaults()
	}
	startCmd.Usage = func() {
		fmt.Println("Usage:")
		startCmd.PrintDefaults()
	}
	stopCmd.Usage = func() {
		fmt.Println("Usage:")
		stopCmd.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) < 1 {
		slog.Error("Expected 'start' or 'stop' subcommands")
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Args()[0]
	switch command {
	case "start":
		startCmd.Parse(flag.Args()[1:])
	case "stop":
		stopCmd.Parse(flag.Args()[1:])
	default:
		slog.Error("Unknwon command", "command", command)
		flag.Usage()
		os.Exit(1)
	}

	switch *logLevelString {
	case "trace", "debug":
		logLevel = slog.LevelDebug
	case "info", "notice":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error", "fatal":
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

	slog.Debug("Startup Options", "Flags", os.Args)
	//	slog.Debug("Starting SRAT", "version", config.Version, "pid", state.ID, "address", state.Address, "listeners", fmt.Sprintf("%T", state.Listener))

	if command == "start" && *smbConfigFile == "" {
		log.Fatalf("Missing samba config! %s", *smbConfigFile)
	}

	if !strings.Contains(*dbfile, "?") {
		*dbfile = *dbfile + "?cache=shared&_pragma=foreign_keys(1)"
	}

	//dbom.InitDB(*dbfile)

	// Get options
	//options = config.ReadOptionsFile(*optionsFile)

	var apiContext, apiContextCancel = context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
	staticConfig := dto.ContextState{}
	staticConfig.SupervisorURL = *supervisorURL
	staticConfig.SambaConfigFile = *smbConfigFile
	staticConfig.Template = internal.GetTemplateData()
	staticConfig.DockerInterface = *dockerInterface
	staticConfig.DockerNet = *dockerNetwork

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
			//			func() *overseer.State { return &state },
			internal.GetFrontend,
			fx.Annotate(
				func() bool { return true },
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
		fx.Invoke(func(s service.ShareServiceInterface, v service.VolumeServiceInterface) {
			s.SetVolumeService(v) // Bypass block for cyclic dep in FX
		}),
		fx.Invoke(func(
			mount_repo repository.MountPointPathRepositoryInterface,
			props_repo repository.PropertyRepositoryInterface,
			exported_share_repo repository.ExportedShareRepositoryInterface,
			hardwareClient hardware.ClientWithResponsesInterface,
			samba_user_repo repository.SambaUserRepositoryInterface,
			volume_service service.VolumeServiceInterface,
			fs_service service.FilesystemServiceInterface,
		) {
			versionInDB, err := props_repo.Value("ConfigSpecVersion", true)
			if err != nil || versionInDB == nil {
				// Migrate from JSON to DB
				var config config.Config
				if *configFile != "" {
					err := config.LoadConfig(*configFile)
					if err != nil {
						log.Fatalf("Cant load config file %#+v", err)
					}
				} else {
					buffer, err := templates.Default_Config_content.ReadFile("default_config.json")
					if err != nil {
						log.Fatalf("Cant read default config file %#+v", err)
					}
					config.LoadConfigBuffer(buffer)
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

			}
		}),
		fx.Invoke(func(
			mount_repo repository.MountPointPathRepositoryInterface,
			props_repo repository.PropertyRepositoryInterface,
			exported_share_repo repository.ExportedShareRepositoryInterface,
			hardwareClient hardware.ClientWithResponsesInterface,
			samba_user_repo repository.SambaUserRepositoryInterface,
			volume_service service.VolumeServiceInterface,
			fs_service service.FilesystemServiceInterface,
			samba_service service.SambaServiceInterface,
			supervisor_service service.SupervisorServiceInterface,
		) {
			// Setting the actual LogLevel
			err := props_repo.SetValue("LogLevel", *logLevelString)
			if err != nil {
				log.Fatalf("Cant set log level - %#+v", err)
			}

			if command == "start" {
				// Autocreate users
				slog.Info("******* Autocreating users ********")
				_ha_mount_user_password_, err := props_repo.Value("_ha_mount_user_password_", true)
				if err != nil {
					log.Fatalf("Cant get password for _ha_mount_user_ user - %#+v", err)
				}
				err = unixsamba.CreateSambaUser("_ha_mount_user_", _ha_mount_user_password_.(string), unixsamba.UserOptions{
					CreateHome:    false,
					SystemAccount: false,
					Shell:         "/sbin/nologin",
				})
				if err != nil {
					log.Fatalf("Cant create samba user %#+v", err)
				}
				users, err := samba_user_repo.All()
				if err != nil {
					log.Fatalf("Cant load users - %#+v", err)
				}
				for _, user := range users {
					slog.Info("Autocreating user", "name", user.Username)
					err = unixsamba.CreateSambaUser(user.Username, user.Password, unixsamba.UserOptions{
						CreateHome:    false,
						SystemAccount: false,
						Shell:         "/sbin/nologin",
					})
					if err != nil {
						slog.Error("Error autocreating user", "name", user.Username, "err", err)
					} else {
						slog.Info("User created successfully", "name", user.Username)
					}
				}
				slog.Info("******* Autocreating users done! ********")

				// Automount all volumes
				slog.Info("******* Automounting all shares! ********")
				all, err := mount_repo.All()
				if err != nil {
					log.Fatalf("Cant load mounts - %#+v", err)
				}
				for _, mnt := range all {
					if mnt.Type == "ADDON" && *mnt.IsToMountAtStartup {
						slog.Info("Automounting share", "path", mnt.Path)
						conv := converter.DtoToDbomConverterImpl{}
						mpd := dto.MountPointData{}
						conv.MountPointPathToMountPointData(mnt, &mpd)
						err := volume_service.MountVolume(&mpd)
						if err != nil {
							if errors.Is(err, dto.ErrorAlreadyMounted) {
								slog.Info("Share already mounted", "path", mnt.Path)
							} else {
								slog.Error("Error automounting share", "path", mnt.Path, "err", err)
							}
						}
						slog.Debug("Share automounted", "path", mnt.Path)
					}
				}
				slog.Info("******* Automounting all shares done! ********")

				// Apply config to samba
				slog.Info("******* Applying Samba config ********")
				err = samba_service.WriteAndRestartSambaConfig()
				if err != nil {
					log.Fatalf("Cant apply samba config - %#+v", err)
				}
				slog.Info("******* Samba config applied! ********")
			} else if command == "stop" {
				slog.Info("******* Unmounting all shares from Homeassistant ********")
				// remount network share on ha_core
				shares, err := exported_share_repo.All()
				if err != nil {
					log.Fatalf("Can't get Shares - %#+v", err)
				}

				for _, share := range *shares {
					if share.Disabled {
						continue
					}
					switch share.Usage {
					case "media", "share", "backup":
						err = supervisor_service.NetworkUnmountShare(share)
						if err != nil {
							slog.Error("UnMounting error", "share", share, "err", err)
						}
					}
				}
				slog.Info("******* Unmounted all shares from Homeassistant ********")
			}
		}),
	)
	app.Start(context.Background())
	apiContextCancel()
	app.Stop(context.Background())

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
		share.MountPointData.IsToMountAtStartup = pointer.Bool(false)
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
