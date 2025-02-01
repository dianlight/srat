package main

//go:generate go run github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc4 init --pd --parseInternal --outputTypes json,yaml
//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.7.0 gen ./converter

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

	"github.com/jpillora/overseer"
	"github.com/mattn/go-isatty"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbutil"
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

// Static files
//
//go:embed static/* docs/swagger.json
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
	var configFile = flag.String("conf", "", "Config json file, can be omitted if used in a pipe")
	http_port = flag.Int("port", 8080, "Http Port on listen to")
	templateFile = flag.String("template", "", "Template file")
	smbConfigFile = flag.String("out", "", "Output file, if not defined output will be to console")
	roMode = flag.Bool("ro", false, "Read only mode")
	hamode = flag.Bool("addon", false, "Run in addon mode")
	dbfile := flag.String("db", "file::memory:?cache=shared&_pragma=foreign_keys(1)", "Database file")
	dockerInterface = flag.String("docker-interface", "", "Docker interface")
	dockerNetwork = flag.String("docker-network", "", "Docker network")

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
	//log.Printf("Update file: %s\n", data.UpdateFilePath)

	/* FIXME: Migrate to service
	if *show_volumes {
		volume := api.NewVolumeHandler(context.Background())
		volumes, err := volume.GetVolumesData()
		if err != nil {
			log.Fatalf("Error fetching volumes: %v", err)
			os.Exit(1)
		}
		pretty.Printf("\n%v\n", volumes)
		os.Exit(0)
	}
	*/

	dbom.InitDB(*dbfile)

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
		dbutil.FirstTimeJSONImporter(config)
		if err != nil {
			log.Fatalf("Cant import json settings - %#v", err)
		}
	}
	//data.Config = aconfig

	// End

	overseer.Run(overseer.Config{
		Program: prog,
		Address: fmt.Sprintf(":%d", *http_port),
		Fetcher: &fetcher.File{
			Path:     updateFilePath,
			Interval: 1 * time.Second,
		},
		//Debug: true,
	})
}

func prog(state overseer.State) {

	log.Printf("SRAT: SambaNAS Rest Administration Interface\n")
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
		log.Println("Missing samba config going in test mode")
	}

	if *roMode {
		log.Println("Read only mode")
	}

	// Get options
	options = config.ReadOptionsFile(*optionsFile)

	var apiContext, apiContextCancel = context.WithCancel(context.Background())
	sharedResources := api.ContextState{}
	sharedResources.UpdateFilePath = updateFilePath
	sharedResources.ReadOnlyMode = *roMode
	sharedResources.SambaConfigFile = *smbConfigFile
	sharedResources.Template = templateData
	sharedResources.DockerInterface = *dockerInterface
	sharedResources.DockerNet = *dockerNetwork
	//sharedResources.SSEBroker = api.NewSSEBroker()

	//sharedResources.FromJSONConfig(*aconfig)
	//apiContext = sharedResources.ToContext(apiContext)
	apiContext = api.StateToContext(&sharedResources, apiContext)

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
			func() *slog.Logger { return logger },
			func() (context.Context, context.CancelFunc) { return apiContext, apiContextCancel },
			func() *api.ContextState { return &sharedResources },
			func() *overseer.State { return &state },
			fx.Annotate(
				func() fs.FS { return content },
				fx.ResultTags(`name:"static_fs"`),
			),
			fx.Annotate(
				func() bool { return *hamode },
				fx.ResultTags(`name:"ha_mode"`),
			),
			service.NewBroadcasterService,
			service.NewVolumeService,
			service.NewSambaService,
			server.AsRoute(api.NewSSEBroker),
			server.AsRoute(api.NewHealthHandler),
			server.AsRoute(api.NewShareHandler),
			server.AsRoute(api.NewVolumeHandler),
			server.AsRoute(api.NewSettingsHanler),
			fx.Annotate(
				server.NewMuxRouter,
				fx.ParamTags(`group:"routes"`, `name:"ha_mode"`, `name:"static_fs"`),
			),
			server.NewHTTPServer,
		),
		fx.Invoke(func(*http.Server) {}),
	).Run()

	/*
		globalRouter := mux.NewRouter()
		if hamode != nil && *hamode {
			globalRouter.Use(HAMiddleware)
		}
	*/

	/*
		ok
			// Health check
			health := api.NewHealth(apiContext, *roMode)
			globalRouter.HandleFunc("/health", health.HealthCheckHandler).Methods(http.MethodGet)

			ok
			// Shares
			share := api.NewShareHandler(apiContext)
			globalRouter.HandleFunc("/shares", share.ListShares).Methods(http.MethodGet)
			globalRouter.HandleFunc("/share/{share_name}", share.GetShare).Methods(http.MethodGet)
			globalRouter.HandleFunc("/share", share.CreateShare).Methods(http.MethodPost)
			globalRouter.HandleFunc("/share/{share_name}", share.UpdateShare).Methods(http.MethodPut)
			globalRouter.HandleFunc("/share/{share_name}", share.DeleteShare).Methods(http.MethodDelete)

			ok
			// Volumes
			volumes := api.NewVolumeHandler(apiContext)
			globalRouter.HandleFunc("/volumes", volumes.ListVolumes).Methods(http.MethodGet)
			globalRouter.HandleFunc("/volume/{id}/mount", volumes.MountVolume).Methods(http.MethodPost)
			globalRouter.HandleFunc("/volume/{id}/mount", volumes.UmountVolume).Methods(http.MethodDelete)
	*/
	/*
			// ---------------------------------------- OLAPI --------------------------------

			globalRouter.HandleFunc("/update", api.UpdateHandler).Methods(http.MethodPut)
			globalRouter.HandleFunc("/restart", api.RestartHandler).Methods(http.MethodPut)
			globalRouter.HandleFunc("/nics", api.GetNICsHandler).Methods(http.MethodGet)
			globalRouter.HandleFunc("/filesystems", api.GetFSHandler).Methods(http.MethodGet)
			//globalRouter.HandleFunc("/sse", sharedResources.SSEBroker.Stream).Methods(http.MethodGet)

			// Users
			globalRouter.HandleFunc("/admin/user", api.GetAdminUser).Methods(http.MethodGet)
			globalRouter.HandleFunc("/admin/user", api.UpdateAdminUser).Methods(http.MethodPut, http.MethodPatch)
			globalRouter.HandleFunc("/users", api.ListUsers).Methods(http.MethodGet)
			//	globalRouter.HandleFunc("/user/{username}", api.GetUser).Methods(http.MethodGet)
			globalRouter.HandleFunc("/user", api.CreateUser).Methods(http.MethodPost)
			globalRouter.HandleFunc("/user/{username}", api.UpdateUser).Methods(http.MethodPut, http.MethodPatch)
			globalRouter.HandleFunc("/user/{username}", api.DeleteUser).Methods(http.MethodDelete)

			// Samba
			globalRouter.HandleFunc("/samba", api.GetSambaConfig).Methods(http.MethodGet)
			globalRouter.HandleFunc("/samba/apply", api.ApplySamba).Methods(http.MethodPut)
			//globalRouter.HandleFunc("/samba/status", api.GetSambaProcessStatus).Methods(http.MethodGet)

			// Global
			ok
			globalRouter.HandleFunc("/global", api.GetSettings).Methods(http.MethodGet)
			globalRouter.HandleFunc("/global", api.UpdateSettings).Methods(http.MethodPut, http.MethodPatch)

			// Configuration

			//globalRouter.HandleFunc("/config", api.PersistAllConfig).Methods(http.MethodPut, http.MethodPatch)
			//globalRouter.HandleFunc("/config", api.RollbackConfig).Methods(http.MethodDelete)

			// WebSocket
			globalRouter.HandleFunc("/events", api.WSChannelEventsList).Methods(http.MethodGet)
			globalRouter.HandleFunc("/ws", api.WSChannelHandler)
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

	// Print all routes
	/*
		globalRouter.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			template, err := route.GetPathTemplate()
			if err != nil {
				return tracerr.Wrap(err)
			}
			log.Printf("Route: %s\n", template)
			return nil
		})
	*/
	/*
		handler := cors.New(
			cors.Options{
				//AllowedOrigins:   []string{"*"},
				AllowOriginFunc:  func(origin string) bool { return true },
				AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
				AllowedHeaders:   []string{"*"},
				AllowCredentials: true,
				MaxAge:           300,
			},
		).Handler(globalRouter)
		loggedRouter := handlers.LoggingHandler(os.Stdout, handler)

		srv := &http.Server{
			//Addr: fmt.Sprintf("%s:%d",state.Address, *http_port),
			// Good practice to set timeouts to avoid Slowloris attacks.
			//WriteTimeout: time.Second * 15,
			ReadTimeout: time.Second * 15,
			IdleTimeout: time.Second * 60,
			Handler:     loggedRouter, // Pass our instance of gorilla/mux in.
			ConnContext: func(ctx context.Context, c net.Conn) context.Context {
				log.Printf("New connection: %s\n", c.RemoteAddr())
				ctx = api.StateToContext(&sharedResources, ctx)
				return ctx
			},
		}
	*/
	/*
		// Run the backgrounde services
		//go api.HealthAndUpdateDataRefeshHandlers(apiContext)
		// Run our server in a goroutine so that it doesn't block.
		go func() {
			log.Printf("Starting Server... \n GoTo: http://localhost:%d/", *http_port)

			if err := srv.Serve(state.Listener); err != nil {
				log.Fatal(err)
			}
		}()
	*/
	/*
		c := make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
		// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
		signal.Notify(c, os.Interrupt)

		// Block until we receive our signal.
		<-c
		log.Println("Shutting down server...")

		// Create a deadline to wait for.
		//ctx, cancel := context.WithTimeout(context.Background(), wait)
		//defer cancel()
		// Doesn't block if no connections, but will otherwise wait
		// until the timeout deadline.
		// srv.Shutdown(ctx)
		// Optionally, you could run srv.Shutdown in a goroutine and block on
		// <-ctx.Done() if your application should wait for other services
		// to finalize based on context cancellation.
	*/
	log.Println("shutting down")
	os.Exit(0)
}
