package main

//go:generate go run github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc4 init --pd --parseInternal --outputTypes json,yaml
//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.7.0 gen ./converter

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jpillora/overseer"
	"github.com/kr/pretty"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbutil"

	//_ "github.com/dianlight/srat/docs"
	"github.com/jpillora/overseer/fetcher"
	"github.com/rs/cors"
)

var SRATVersion string
var options *config.Options
var smbConfigFile *string
var globalRouter *mux.Router
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

// HAMiddleware is a middleware function for handling HomeAssistant authentication.
// It checks for the presence and validity of the X-Supervisor-Token in the request header.
//
// Parameters:
//   - next: The next http.Handler in the chain to be called if authentication is successful.
//
// Returns:
//   - http.Handler: A new http.Handler that wraps the authentication logic around the next handler.
//     If authentication fails, it returns a 401 Unauthorized status.
//     If successful, it adds the token to the request context and calls the next handler.
func HAMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("X-Supervisor-Token")
		if tokenString == "" {
			log.Printf("Not in a HomeAssistant environment!")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if tokenString != os.Getenv("SUPERVISOR_TOKEN") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "auth_token", tokenString)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
	show_volumes := flag.Bool("show-volumes", false, "Show volumes in headless CLI mode and exit")

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

	if *show_volumes {
		volumes, err := api.GetVolumesData()
		if err != nil {
			log.Fatalf("Error fetching volumes: %v", err)
			os.Exit(1)
		}
		pretty.Printf("\n%v\n", volumes)
		os.Exit(0)
	}

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

	var apiContext = context.Background()
	sharedResources := api.ContextState{}
	sharedResources.UpdateFilePath = updateFilePath
	sharedResources.ReadOnlyMode = *roMode
	sharedResources.SambaConfigFile = *smbConfigFile
	sharedResources.Template = templateData
	sharedResources.DockerInterface = *dockerInterface
	sharedResources.DockerNet = *dockerNetwork
	sharedResources.SSEBroker = api.NewSSEBroker()

	//sharedResources.FromJSONConfig(*aconfig)
	//apiContext = sharedResources.ToContext(apiContext)
	apiContext = api.StateToContext(&sharedResources, apiContext)

	globalRouter := mux.NewRouter()
	if hamode != nil && *hamode {
		globalRouter.Use(HAMiddleware)
	}

	// System
	health := api.NewHealth(apiContext, *roMode)
	globalRouter.HandleFunc("/health", health.HealthCheckHandler).Methods(http.MethodGet)
	globalRouter.HandleFunc("/update", api.UpdateHandler).Methods(http.MethodPut)
	globalRouter.HandleFunc("/restart", api.RestartHandler).Methods(http.MethodPut)
	globalRouter.HandleFunc("/nics", api.GetNICsHandler).Methods(http.MethodGet)
	globalRouter.HandleFunc("/filesystems", api.GetFSHandler).Methods(http.MethodGet)
	globalRouter.HandleFunc("/sse", sharedResources.SSEBroker.Stream).Methods(http.MethodGet)

	// Shares
	globalRouter.HandleFunc("/shares", api.ListShares).Methods(http.MethodGet)
	globalRouter.HandleFunc("/share/{share_name}", api.GetShare).Methods(http.MethodGet)
	globalRouter.HandleFunc("/share", api.CreateShare).Methods(http.MethodPost)
	globalRouter.HandleFunc("/share/{share_name}", api.UpdateShare).Methods(http.MethodPut)
	globalRouter.HandleFunc("/share/{share_name}", api.DeleteShare).Methods(http.MethodDelete)

	// Volumes
	globalRouter.HandleFunc("/volumes", api.ListVolumes).Methods(http.MethodGet)
	globalRouter.HandleFunc("/volume/{id}/mount", api.MountVolume).Methods(http.MethodPost)
	globalRouter.HandleFunc("/volume/{id}/mount", api.UmountVolume).Methods(http.MethodDelete)

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
	/*
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
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      loggedRouter, // Pass our instance of gorilla/mux in.
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			log.Printf("New connection: %s\n", c.RemoteAddr())
			ctx = api.StateToContext(&sharedResources, ctx)
			return ctx
		},
	}

	// Run the backgrounde services
	go api.HealthAndUpdateDataRefeshHandlers(apiContext)
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Printf("Starting Server... \n GoTo: http://localhost:%d/", *http_port)

		if err := srv.Serve(state.Listener); err != nil {
			log.Fatal(err)
		}
	}()

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
	log.Println("shutting down")
	os.Exit(0)
}
