package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/m1/go-generate-password/generator"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom" // Keep for firstTimeJSONImporter
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/internal/appsetup"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/templates"
	"github.com/dianlight/srat/tlog"
	"github.com/dianlight/srat/unixsamba"

	"go.uber.org/fx"
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
var logLevelString *string

func main() {
	silentMode := flag.Bool("silent", false, "Silent Mode. Remove unecessary banner")
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

	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)
	shortVersion := versionCmd.Bool("short", false, "Short version")

	upgradeCmd := flag.NewFlagSet("upgrade", flag.ExitOnError)
	upgradeChannel := upgradeCmd.String("channel", "release", "Upgrade channel (release, prerelease, develop)")
	updateFilePath := flag.String("update-file-path", os.TempDir()+"/"+filepath.Base(os.Args[0]), "Update file path - used for addon updates")

	flag.Usage = func() {
		fmt.Printf("Usage %s <config_options...> <command> <command_options...>\n", os.Args[0])
		fmt.Println("Config Options:")
		flag.PrintDefaults()
		fmt.Println("Command start:")
		startCmd.PrintDefaults()
		fmt.Println("Command stop:")
		stopCmd.PrintDefaults()
		fmt.Println("Command version:")
		versionCmd.PrintDefaults()
		fmt.Println("Command upgrade:")
		upgradeCmd.PrintDefaults()
	}
	startCmd.Usage = func() {
		fmt.Println("Usage:")
		startCmd.PrintDefaults()
	}
	stopCmd.Usage = func() {
		fmt.Println("Usage:")
		stopCmd.PrintDefaults()
	}
	versionCmd.Usage = func() {
		fmt.Println("Usage:")
		versionCmd.PrintDefaults()
	}
	upgradeCmd.Usage = func() {
		fmt.Println("Usage:")
		upgradeCmd.PrintDefaults()
	}

	flag.Parse()

	if !*silentMode {
		internal.Banner("srat-cli")
	}

	if len(flag.Args()) < 1 {
		slog.Error("Expected 'start','stop' or 'version' subcommands")
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Args()[0]
	switch command {
	case "start":
		startCmd.Parse(flag.Args()[1:])
	case "stop":
		stopCmd.Parse(flag.Args()[1:])
	case "upgrade":
		upgradeCmd.Parse(flag.Args()[1:])
		if *upgradeChannel != "release" && *upgradeChannel != "prerelease" && *upgradeChannel != "develop" {
			slog.Error("Invalid upgrade channel", "channel", *upgradeChannel)
			upgradeCmd.PrintDefaults()
			os.Exit(1)
		}
	case "version":
		versionCmd.Parse(flag.Args()[1:])
		if *shortVersion {
			fmt.Printf("%s\n", config.Version)
		} else {
			fmt.Printf("Version: %s (%s) - %s\n", config.Version, config.CommitHash, config.BuildTimestamp)
		}
		os.Exit(0)
	default:
		slog.Error("Unknwon command", "command", command)
		flag.Usage()
		os.Exit(1)
	}

	err := tlog.SetLevelFromString(*logLevelString)
	if err != nil {
		log.Fatalf("Invalid log level: %s", *logLevelString)
	}

	slog.Debug("Startup Options", "Flags", os.Args)
	//	slog.Debug("Starting SRAT", "version", config.Version, "pid", state.ID, "address", state.Address, "listeners", fmt.Sprintf("%T", state.Listener))

	if command == "start" && *smbConfigFile == "" {
		log.Fatalf("Missing samba config! %s", *smbConfigFile)
	}

	if !strings.Contains(*dbfile, "?") {
		*dbfile = *dbfile + "?cache=shared&_pragma=foreign_keys(1)"
	}

	apiCtx, apiCancel := context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
	defer apiCancel() // Ensure context is cancelled on exit

	staticConfig := dto.ContextState{}
	staticConfig.SupervisorURL = *supervisorURL
	staticConfig.SambaConfigFile = *smbConfigFile
	staticConfig.Template = internal.GetTemplateData()
	staticConfig.DockerInterface = *dockerInterface
	staticConfig.DockerNet = *dockerNetwork
	staticConfig.UpdateFilePath = *updateFilePath
	staticConfig.DatabasePath = *dbfile
	staticConfig.SupervisorURL = *supervisorURL
	staticConfig.SupervisorToken = *supervisorToken

	appParams := appsetup.BaseAppParams{
		Ctx:          apiCtx,
		CancelFn:     apiCancel,
		StaticConfig: &staticConfig,
	}

	// New FX
	app := fx.New(
		appsetup.NewFXLoggerOption(),
		appsetup.ProvideCoreDependencies(appParams),
		appsetup.ProvideHAClientDependencies(appParams), // CLI needs HA clients
		appsetup.ProvideFrontendOption(),                // For template data, etc.
		appsetup.ProvideCyclicDependencyWorkaroundOption(),
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
			if err != nil || versionInDB == nil { // Assuming error means not found or actual DB error
				// Migrate from JSON to DB
				var config config.Config
				if *configFile != "" {
					err = config.LoadConfig(*configFile) // Assign to existing err
					if err != nil {
						log.Fatalf("Cant load config file %#+v", err)
					}
				} else {
					buffer, err := templates.Default_Config_content.ReadFile("default_config.json")
					if err != nil {
						log.Fatalf("Cant read default config file %#+v", err)
					}
					err = config.LoadConfigBuffer(buffer) // Assign to existing err
					if err != nil {
						log.Fatalf("Cant load default config from buffer %#+v", err)
					}
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
					log.Fatalf("Cant import json settings - %#+v", errors.WithStack(err))
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
			upgrade_service service.UpgradeServiceInterface,
			apiContext *dto.ContextState,
		) {
			// Setting the actual LogLevel
			err := props_repo.SetValue("LogLevel", *logLevelString) // Use existing err
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
				if !apiContext.ProtectedMode {
					slog.Info("******* Automounting all shares! ********")
					all, err := mount_repo.All()
					if err != nil {
						log.Fatalf("Cant load mounts - %#+v", err)
					}
					for _, mnt := range all {
						if mnt.Type == "ADDON" && mnt.IsToMountAtStartup != nil && *mnt.IsToMountAtStartup {
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
				} else {
					slog.Info("******* Protected mode is ON, skipping automounting shares! ********")
				}

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
					if share.Disabled != nil && *share.Disabled {
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
			} else if command == "upgrade" {
				slog.Info("Starting upgrade process", "channel", *upgradeChannel)
				updch, ett := dto.ParseUpdateChannel(*upgradeChannel)
				if ett != nil {
					slog.Error("Error parsing upgrade channel", "err", ett)
					return
				}
				ett = props_repo.SetValue("UpdateChannel", updch)
				if ett != nil {
					slog.Error("Error setting upgrade channel", "err", ett)
					return
				}

				if updch == dto.UpdateChannels.DEVELOP {
					slog.Info("Attempting local update for DEVELOP channel.")
					err = upgrade_service.InstallUpdateLocal(&updch)
					if err != nil {
						if errors.Is(err, dto.ErrorNoUpdateAvailable) {
							slog.Info("No local update found or directory missing.", "error", err)
						} else {
							slog.Error("Error during local update process", "err", err)
						}
					} else {
						slog.Info("Local update installed successfully. Please restart the application.")
					}
				} else {
					err = props_repo.SetValue("UpdateChannel", updch)
					if err != nil {
						slog.Error("Error setting upgrade channel", "err", err)
						return
					}

					asset, err := upgrade_service.GetUpgradeReleaseAsset(&updch)
					if err != nil {
						if errors.Is(err, dto.ErrorNoUpdateAvailable) {
							slog.Info("No update available for the requested channel.")
						} else {
							slog.Error("Error checking for updates", "err", err)
						}
					} else if asset != nil {
						slog.Info("Update available", "version", asset.LastRelease, "asset_name", asset.ArchAsset.Name)
						updatePkg, errDownload := upgrade_service.DownloadAndExtractBinaryAsset(asset.ArchAsset)
						if errDownload != nil {
							slog.Error("Error downloading or extracting update", "err", errDownload)
							// os.RemoveAll(updatePkg.TempDirPath) // Ensure cleanup on error if updatePkg is not nil
						} else {
							slog.Info("Update downloaded and extracted successfully", "temp_dir", updatePkg.TempDirPath)
							if updatePkg.CurrentExecutablePath != nil {
								slog.Info("Matching executable found", "path", *updatePkg.CurrentExecutablePath)
								errInstall := upgrade_service.InstallUpdatePackage(updatePkg)
								if errInstall != nil {
									slog.Error("Error installing update for overseer", "err", errInstall)
								}
							} else {
								slog.Warn("Update downloaded, but no directly matching executable found by name. Check extracted files.", "paths", updatePkg.OtherFilesPaths)
							}
							slog.Debug("Cleaning up temporary update directory", "path", updatePkg.TempDirPath)
							if err := os.RemoveAll(updatePkg.TempDirPath); err != nil {
								slog.Warn("Failed to remove temporary update directory", "path", updatePkg.TempDirPath, "err", err)
							}
						}
					} else {
						slog.Info("No update available (asset was nil).")
					}
				}
			}
		}),
	)

	if err := app.Err(); err != nil { // Check for errors from Provide functions
		log.Fatalf("Error during FX setup: %v", err)
	}

	app.Start(context.Background())
	// apiCancel is deferred
	app.Stop(context.Background())
	// os.Exit(0) is implicit if main returns
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
