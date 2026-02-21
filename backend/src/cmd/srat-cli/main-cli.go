package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"

	"github.com/dianlight/srat/config" // Keep for firstTimeJSONImporter
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/internal/appsetup"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v83/github"

	"go.uber.org/fx"
)

var smbConfigFile *string
var dockerInterface *string
var dockerNetwork *string
var dbfile *string
var supervisorURL *string
var supervisorToken *string
var logLevelString *string
var upgradeDataDir *string

func formatVersionMessage(short bool) string {
	if short {
		return fmt.Sprintf("%s\n", config.Version)
	}
	return fmt.Sprintf("Version: %s (%s) - %s\n", config.Version, config.CommitHash, config.BuildTimestamp)
}

func parseCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("expected 'start','stop','upgrade','hdidle' or 'version' subcommands")
	}
	switch args[0] {
	case "start", "stop", "upgrade", "version", "hdidle":
		return args[0], nil
	default:
		return "", fmt.Errorf("unknown command: %s", args[0])
	}
}

func main() {
	silentMode := flag.Bool("silent", false, "Silent Mode. Remove unecessary banner")
	supervisorToken = flag.String("ha-token", os.Getenv("SUPERVISOR_TOKEN"), "HomeAssistant Supervisor Token")
	supervisorURL = flag.String("ha-url", "http://supervisor/", "HomeAssistant Supervisor URL")
	dbfile = flag.String("db", "file::memory:?cache=shared&_pragma=foreign_keys(1)", "Database file")
	logLevelString = flag.String("loglevel", "info", "Log level string (debug, info, warn, error)")
	protectedMode := flag.Bool("protected-mode", false, "Addon protected mode")
	upgradeDataDir = flag.String("upgrade-data-dir", "/data/upgrade", "Persistent upgrades data directory")

	// set global logger with custom options
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	smbConfigFile = startCmd.String("out", "", "Output samba conf file")
	dockerInterface = startCmd.String("docker-interface", "", "Docker interface")
	dockerNetwork = startCmd.String("docker-network", "", "Docker network")
	if !internal.Is_embed {
		internal.TemplateFile = startCmd.String("template", "", "Template file")
	}

	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)

	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)
	shortVersion := versionCmd.Bool("short", false, "Short version")

	upgradeCmd := flag.NewFlagSet("upgrade", flag.ExitOnError)
	upgradeChannel := upgradeCmd.String("channel", "release", "Upgrade channel (release, prerelease, develop)")

	flag.Usage = func() {
		fmt.Printf("Usage %s <config_options...> <command> <command_options...>\n", os.Args[0])
		fmt.Println("Config Options:")
		flag.PrintDefaults()
		fmt.Println("Command start (deprecated):")
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

	err := tlog.SetLevelFromString(*logLevelString)
	if err != nil {
		log.Fatalf("Invalid log level: %s", *logLevelString)
	}

	// Test Logger
	/*
		tlog.Trace("Trace log")
		tlog.Debug("Debug log")
		tlog.Info("Info log")
		tlog.Warn("Warn log")
		tlog.Error("Error log")
	*/

	command, cmdErr := parseCommand(flag.Args())
	if cmdErr != nil {
		slog.Error(cmdErr.Error())
		flag.Usage()
		os.Exit(1)
	}
	if !*silentMode {
		internal.Banner("srat-cli", command)
	}

	updch := dto.UpdateChannels.RELEASE
	switch command {
	case "start":
		os.Exit(0) // Deprecated
		startCmd.Parse(flag.Args()[1:])
	case "stop":
		os.Exit(0) // Deprecated
		stopCmd.Parse(flag.Args()[1:])
	case "upgrade":
		upgradeCmd.Parse(flag.Args()[1:])
		var ett error
		updch, ett = dto.ParseUpdateChannel(*upgradeChannel)
		if ett != nil {
			slog.Error("Invalid upgrade channel", "channel", *upgradeChannel, "err", ett)
			upgradeCmd.PrintDefaults()
			os.Exit(1)
		}
	case "version":
		versionCmd.Parse(flag.Args()[1:])
		fmt.Print(formatVersionMessage(*shortVersion))
		if !*shortVersion {
			fmt.Print("Build Metadata: ", config.GetCurrentBinaryVersion().String(), "\n")
		}
		os.Exit(0)
	default:
		slog.Error("unknown command", "command", command)
		flag.Usage()
		os.Exit(1)
	}

	slog.Debug("Startup Options", "Flags", os.Args)
	//	slog.Debug("Starting SRAT", "version", config.Version, "pid", state.ID, "address", state.Address, "listeners", fmt.Sprintf("%T", state.Listener))

	if command == "start" && *smbConfigFile == "" {
		log.Fatalf("Missing samba config! %s", *smbConfigFile)
	}

	//if !strings.Contains(*dbfile, "?") {
	//	*dbfile = *dbfile + "?cache=shared&_pragma=foreign_keys(1)"
	//}

	apiCtx, apiCancel := context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
	defer apiCancel() // Ensure context is cancelled on exit

	staticConfig := dto.ContextState{
		SupervisorURL:   *supervisorURL,
		SambaConfigFile: *smbConfigFile,
		Template:        internal.GetTemplateData(),
		DockerInterface: *dockerInterface,
		DockerNet:       *dockerNetwork,
		UpdateDataDir:   *upgradeDataDir,
		//UpdateFilePath:  "",
		UpdateChannel:   updch,
		DatabasePath:    *dbfile,
		SupervisorToken: *supervisorToken,
		ProtectedMode:   *protectedMode,
		StartTime:       time.Now(),
	}

	appParams := appsetup.BaseAppParams{
		Ctx:          apiCtx,
		CancelFn:     apiCancel,
		StaticConfig: &staticConfig,
	}

	// Determine if we need database access based on command
	// Only version and hdidle commands don't need DB
	// upgrade command needs DB but can use in-memory DB (default flag value)
	needsDB := (command != "version" && command != "upgrade")

	// Build FX options based on command requirements
	var fxOptions []fx.Option
	fxOptions = append(fxOptions, appsetup.NewFXLoggerOption())

	if needsDB {
		// Commands that need database access (start, stop, upgrade)
		fxOptions = append(fxOptions,
			appsetup.ProvideCoreDependencies(appParams),
			appsetup.ProvideFrontendOption(),
			appsetup.ProvideCyclicDependencyWorkaroundOption(),
		)

		// Only include HA client dependencies for start and stop commands
		// Upgrade command doesn't need websocket client
		switch command {
		case "start", "stop":
			fxOptions = append(fxOptions, appsetup.ProvideHAClientDependencies(appParams))
			//		case "upgrade":
			//			fxOptions = append(fxOptions, appsetup.ProvideHAClientDependenciesWithoutWebSocket(appParams))
		}
	} else {
		switch command {
		case "upgrade":
			fxOptions = append(fxOptions,
				appsetup.ProvideCoreDependenciesWithoutDB(appParams),
				fx.Provide(
					func() *github.Client {
						rateLimiter := github_ratelimit.New(nil)
						return github.NewClient(&http.Client{
							Transport: rateLimiter,
						})
					},
					service.NewUpgradeService,
				),
			)
		case "version":
			fxOptions = append(fxOptions,
				appsetup.ProvideCoreDependenciesWithoutDB(appParams),
			)
		}
	}

	/*
		// Add command-specific invocations
		// First Invoke: JSON migration (only for start command)
		if command == "start" {
			fxOptions = append(fxOptions, fx.Invoke(func(
				mount_repo repository.MountPointPathRepositoryInterface,
				props_repo repository.PropertyRepositoryInterface,
				share_repo repository.ExportedShareRepositoryInterface,
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
					err = firstTimeJSONImporter(config, mount_repo, props_repo, share_repo, samba_user_repo)
					if err != nil {
						log.Fatalf("Cant import json settings - %#+v", errors.WithStack(err))
					}

				}
			}))
		}
	*/

	// Second Invoke: Main command logic
	if needsDB {
		fxOptions = append(fxOptions, fx.Invoke(func(
			lc fx.Lifecycle,
			//mount_repo repository.MountPointPathRepositoryInterface,
			//props_repo repository.PropertyRepositoryInterface,
			share_service service.ShareServiceInterface,
			hardwareClient hardware.ClientWithResponsesInterface,
			volume_service service.VolumeServiceInterface,
			fs_service service.FilesystemServiceInterface,
			samba_service service.ServerServiceInterface,
			supervisor_service service.SupervisorServiceInterface,
			upgrade_service service.UpgradeServiceInterface,
			apiContext *dto.ContextState,
		) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// Setting the actual LogLevel
					/*
						err := props_repo.SetValue("LogLevel", *logLevelString) // Use existing err
						if err != nil {
							log.Fatalf("Cant set log level - %#+v", err)
						}
					*/

					switch command {
					case "start":
					case "stop":
						slog.Info("******* Unmounting all shares from Homeassistant ********")
						err = supervisor_service.NetworkUnmountAllShares(ctx)
						if err != nil {
							slog.Error("Error unmounting all shares from Homeassistant", "err", err)
						}
						slog.Info("******* Unmounted all shares from Homeassistant ********")
						/*
							case "upgrade":
								if updch == dto.UpdateChannels.DEVELOP {
									slog.Info("Attempting local update for DEVELOP channel.")
									err := upgrade_service.InstallUpdateLocal(&updch)
									if err != nil {
										if errors.Is(err, dto.ErrorNoUpdateAvailable) {
											slog.Info("No local update found or directory missing.")
										} else {
											slog.Error("Error during local update process", "err", err)
										}
									} else {
										slog.Info("Local update installed successfully. Please restart the application.")
									}
								} else {
									asset, err := upgrade_service.GetUpgradeReleaseAsset(&updch)
									if err != nil {
										if errors.Is(err, dto.ErrorNoUpdateAvailable) {
											slog.Info("No update available for the requested channel.", "channel", updch)
										} else {
											slog.Error("Error checking for updates", "err", err)
										}
									} else if asset != nil {
										slog.Info("Update available", "version", asset.LastRelease, "asset_name", asset.ArchAsset.Name)
										updatePkg, errDownload := upgrade_service.DownloadAndExtractBinaryAsset(asset.ArchAsset)
										if errDownload != nil {
											slog.Error("Error downloading or extracting update", "err", errDownload)
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
						*/
					}
					return nil
				},
			})

		}))
	} else {
		fxOptions = append(fxOptions, fx.Invoke(func(
			lc fx.Lifecycle,
			//mount_repo repository.MountPointPathRepositoryInterface,
			//props_repo repository.PropertyRepositoryInterface,
			//share_service service.ShareServiceInterface,
			//hardwareClient hardware.ClientWithResponsesInterface,
			//samba_user_repo repository.SambaUserRepositoryInterface,
			//volume_service service.VolumeServiceInterface,
			//fs_service service.FilesystemServiceInterface,
			//samba_service service.SambaServiceInterface,
			//supervisor_service service.SupervisorServiceInterface,
			upgrade_service service.UpgradeServiceInterface,
			//apiContext *dto.ContextState,
		) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					switch command {
					case "upgrade":
						asset, err := upgrade_service.GetUpgradeReleaseAsset()
						if err != nil {
							if errors.Is(err, dto.ErrorNoUpdateAvailable) {
								slog.Info("No update available for the requested channel.", "channel", updch)
							} else {
								slog.Error("Error checking for updates", "err", err)
							}
						} else if asset != nil {
							slog.Info("Update available", "version", asset.LastRelease, "asset_name", asset.ArchAsset.Name)
							updatePkg, errDownload := upgrade_service.DownloadAndExtractBinaryAsset(asset.ArchAsset)
							if errDownload != nil {
								slog.Error("Error downloading or extracting update", "err", errDownload)
							} else {
								slog.Info("Update downloaded and extracted successfully")
								if updatePkg.FilesPaths != nil {
									slog.Info("Matching executable found", "paths", updatePkg.FilesPaths)
									errInstall := upgrade_service.InstallUpdatePackage(updatePkg)
									if errInstall != nil {
										slog.Error("Error installing update for overseer", "err", errInstall)
									}
								} else {
									slog.Warn("Update downloaded, but no directly matching executable found by name. Check extracted files.", "paths", updatePkg.FilesPaths)
								}
							}
						} else {
							slog.Info("No update available (asset was nil).")
						}
					}
					return nil
				},
			})

		}))
	}

	// Create FX app with all options
	app := fx.New(fxOptions...)

	if err := app.Err(); err != nil { // Check for errors from Provide functions
		log.Fatalf("Error during FX setup: %v", err)
	}

	app.Start(context.Background())
	// apiCancel is deferred
	app.Stop(context.Background())
	// os.Exit(0) is implicit if main returns
}

/*
func firstTimeJSONImporter(config config.Config,
	mount_repository repository.MountPointPathRepositoryInterface,
	props_repository repository.PropertyRepositoryInterface,
	share_repository repository.ExportedShareRepositoryInterface,
	users_repository repository.SambaUserRepositoryInterface,
) (err error) {

	var conv converter.ConfigToDbomConverterImpl
	shares := &[]dbom.ExportedShare{}
	properties := &dbom.Properties{}
	users := &dbom.SambaUsers{}

	err = conv.ConfigToDbomObjects(config, properties, users, shares)
	if err != nil {
		return errors.WithStack(err)
	}
	err = props_repository.SaveAll(properties)
	if err != nil {
		return errors.WithStack(err)
	}
	err = users_repository.SaveAll(users)
	if err != nil {
		return errors.WithStack(err)
	}
	for i, share := range *shares {
		share.MountPointData.IsToMountAtStartup = new(false)
		err = mount_repository.Save(&share.MountPointData)
		if err != nil {
			return errors.WithStack(err)
		}
		(*shares)[i] = share
	}
	err = share_repository.SaveAll(shares)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
*/
