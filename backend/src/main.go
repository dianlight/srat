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
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gorilla/mux"
	"github.com/jpillora/overseer"
	"github.com/mattn/go-isatty"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbutil"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"

	"github.com/jpillora/overseer/fetcher"
	"github.com/lmittmann/tint"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

var SRATVersion string
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
var updateFilePath string
var configFile *string
var dbfile *string
var frontend *string

// Static files
//
//go:embed static/*
var content embed.FS

//go:embed templates/smb.gtpl
var defaultTemplate embed.FS

// @title						SRAT API
// @version					1.0
// @description				This are samba rest admin API
// @contact.name				Lucio Tarantino
// @contact.url				https://github.com/dianlight
// @contact.email				lucio.tarantino@gmail.com
// @license.name				Apache 2.0
// @license.url				http://www.apache.org/licenses/LICENSE-2.0.html
// @securitydefinitions.apikey	ApiKeyAuth
// @in							header
// @name						X-Supervisor-Token
// @description				HomeAssistant Supervisor Token
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	w := os.Stderr

	// create a new logger
	//logger := slog.New(tint.NewHandler(w, nil))

	// set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.RFC3339,
			NoColor:    !isatty.IsTerminal(w.Fd()),
			AddSource:  true,
		}),
	))

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

	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")

	// Headless CLI mode (execute a command and exit)
	//show_volumes := flag.Bool("show-volumes", false, "Show volumes in headless CLI mode and exit")

	flag.Usage = func() {
		writer := flag.CommandLine.Output()

		fmt.Fprint(writer, "SRAT: SambaNAS Rest Administration Interface\n")
		fmt.Fprintf(writer, "Version: %s\n", SRATVersion)
		fmt.Fprintf(writer, "Documentation: https://github.com/dianlight/SRAT\n\n")

		flag.PrintDefaults()
	}

	flag.Parse()

	updateFilePath = os.TempDir() + "/" + filepath.Base(os.Args[0])

	overseer.Run(overseer.Config{
		Program: prog,
		Address: fmt.Sprintf(":%d", *http_port),
		Fetcher: &fetcher.File{
			Path:     updateFilePath,
			Interval: 1 * time.Second,
		},
		TerminateTimeout: 60,
		Debug:            false,
	})
}

func prog(state overseer.State) {

	log.Printf("SRAT: SambaNAS Rest Administration Interface (%s)\n", state.ID)
	log.Printf("SRAT Version: %s\n", SRATVersion)
	log.Printf("\nFlags: %v\n", os.Args)

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
		log.Fatal("Missing samba config!")
	}

	if *roMode {
		log.Println("Read only mode")
	}

	dbom.InitDB(*dbfile + "?cache=shared&_pragma=foreign_keys(1)")

	// Get options
	options = config.ReadOptionsFile(*optionsFile)

	var apiContext, apiContextCancel = context.WithCancel(context.Background())
	sharedResources := dto.ContextState{}
	sharedResources.UpdateFilePath = updateFilePath
	sharedResources.ReadOnlyMode = *roMode
	sharedResources.SambaConfigFile = *smbConfigFile
	sharedResources.Template = templateData
	sharedResources.DockerInterface = *dockerInterface
	sharedResources.DockerNet = *dockerNetwork

	w := os.Stderr

	// create a new logger
	logger := slog.New(tint.NewHandler(w, &tint.Options{
		NoColor:    !isatty.IsTerminal(w.Fd()),
		Level:      slog.LevelDebug,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))

	// New FX
	fx.New(
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
		fx.Provide(
			func() *gorm.DB { return dbom.GetDB() },
			func() *slog.Logger { return logger },
			func() (context.Context, context.CancelFunc) { return apiContext, apiContextCancel },
			func() *dto.ContextState { return &sharedResources },
			func() *overseer.State { return &state },
			fx.Annotate(
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
				fx.ResultTags(`name:"static_fs"`),
			),
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
			server.AsRoute(api.NewSambaHanler),
			server.AsRoute(api.NewUpgradeHanler),
			server.AsRoute(api.NewSystemHanler),
			fx.Annotate(
				server.NewMuxRouter,
				fx.ParamTags(`group:"routes"`, `name:"ha_mode"`, `name:"static_fs"`),
			),
			server.NewHTTPServer,
			server.NewHumaAPI,
		),
		fx.Invoke(func(
			mount_repo repository.MountPointPathRepositoryInterface,
			exported_share_repo repository.ExportedShareRepositoryInterface,
		) {
			// JSON Config  Migration if necessary
			// Get config and migrate if DB is empty
			var properties dbom.Properties
			err := properties.Load()
			if err != nil {
				log.Fatalf("Cant load properties - %s", err)
			}
			versionInDB, err := properties.GetValue("version")
			if err != nil || versionInDB.(string) == "" {
				// Migrate from JSON to DB
				var config config.Config
				err := config.LoadConfig(*configFile)
				// Setting/Properties
				if err != nil {
					log.Fatalf("Cant load config file %s", err)
				}
				err = dbutil.FirstTimeJSONImporter(config, mount_repo, exported_share_repo)
				if err != nil {
					log.Fatalf("Cant import json settings - %s", tracerr.SprintSourceColor(err))
				}
			}
		}),
		fx.Invoke(func(_ *http.Server, api huma.API, router *mux.Router) {
			router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
				template, err := route.GetPathTemplate()
				if err != nil {
					return tracerr.Wrap(err)
				}
				slog.Debug("Route:", "template", template)
				return nil
			})

			// FIXME: Disattivare quando compilazione
			yaml, err := api.OpenAPI().YAML()
			if err != nil {
				slog.Error("Unable to generate YAML", "err", err)
			}
			err = os.WriteFile("src/docs/openapi.yaml", yaml, 0644)
			if err != nil {
				slog.Error("Unable to write YAML", "err", err)
			}
			json, err := api.OpenAPI().MarshalJSON()
			if err != nil {
				slog.Error("Unable to generate JSON", "err", err)
			}
			err = os.WriteFile("src/docs/openapi.json", json, 0644)
			if err != nil {
				slog.Error("Unable to write JSON", "err", err)
			}

		}),
	).Run()

	dbom.CloseDB()
	/*
			// Static files
		globalRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/static/", http.StatusPermanentRedirect)
		})
		globalRouter.PathPrefix("/").Handler(http.FileServerFS(content)).Methods(http.MethodGet)

		// Print content directory recursively
			fs.WalkDir(content, ".", func(p string, d fs.DirEntry, err error) error {
				log.Printf("dir=%s, path=%s\n", path.Dir(p), p)
				return nil
			})
	*/

	log.Println("shutting down")
	os.Exit(0)
}
