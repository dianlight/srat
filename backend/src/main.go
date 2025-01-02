package main

//go:generate swag init --pd --parseInternal

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jpillora/overseer"
	"github.com/kr/pretty"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	"github.com/dianlight/srat/dbom"
	_ "github.com/dianlight/srat/docs"
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
var sambaConfigFile *string
var wait time.Duration
var hamode *bool

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

type ResponseError struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
	Body  any    `json:"body"`
}

// DoResponse writes a JSON response to the provided http.ResponseWriter.
// It sets the HTTP status code and marshals the given body into JSON format.
//
// Parameters:
//   - code: The HTTP status code to be set in the response.
//   - w: The http.ResponseWriter to write the response to.
//   - body: The data to be marshaled into JSON and written as the response body.
//
// If there's an error marshaling the body into JSON, it calls DoResponseError
// with an internal server error status.
func DoResponse(code int, w http.ResponseWriter, body any) {
	w.WriteHeader(code)
	jsonResponse, jsonError := json.Marshal(body)
	if jsonError != nil {
		DoResponseError(http.StatusInternalServerError, w, "Unable to encode JSON", jsonError)
	} else {
		w.Write(jsonResponse)
	}
	return
}

// DoResponseError writes a JSON error response to the provided http.ResponseWriter.
// It sets the HTTP status code and marshals an error object into JSON format.
//
// Parameters:
//   - code: The HTTP status code to be set in the response.
//   - w: The http.ResponseWriter to write the response to.
//   - message: A string describing the error message.
//   - body: Additional data to be included in the error response.
//
// The function doesn't return any value. It writes the error response directly to the provided http.ResponseWriter.
// If there's an error marshaling the response into JSON, it writes an internal server error status
// and the error message as plain text.
func DoResponseError(code int, w http.ResponseWriter, message string, body any) {
	w.WriteHeader(code)
	jsonResponse, jsonError := json.Marshal(ResponseError{Error: message, Body: body})
	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.Write(jsonResponse)
	}
	return
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
	optionsFile = flag.String("opt", "/data/options.json", "Addon Options json file")
	data.ConfigFile = flag.String("conf", "", "Config json file, can be omitted if used in a pipe")
	http_port = flag.Int("port", 8080, "Http Port on listen to")
	templateFile = flag.String("template", "", "Template file")
	smbConfigFile = flag.String("out", "", "Output file, if not defined output will be to console")
	data.ROMode = flag.Bool("ro", false, "Read only mode")
	hamode = flag.Bool("addon", false, "Run in addon mode")
	dbfile := flag.String("db", ":memory:?cache=shared", "Database file")

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

	data.UpdateFilePath = os.TempDir() + "/" + filepath.Base(os.Args[0])
	//log.Printf("Update file: %s\n", data.UpdateFilePath)

	if *show_volumes {
		volumes, err := GetVolumesData()
		if err != nil {
			log.Fatalf("Error fetching volumes: %v", err)
			os.Exit(1)
		}
		pretty.Printf("\n%v\n", volumes)
		os.Exit(0)
	}

	dbom.InitDB(*dbfile)

	overseer.Run(overseer.Config{
		Program: prog,
		Address: fmt.Sprintf(":%d", *http_port),
		Fetcher: &fetcher.File{
			Path:     data.UpdateFilePath,
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

	if *data.ROMode {
		log.Println("Read only mode")
	}

	// Get config
	aconfig, cerr := config.LoadConfig(*data.ConfigFile)
	if cerr != nil {
		log.Fatalf("Cant load config file %s - %s", *data.ConfigFile, cerr)
	}
	data.Config = aconfig

	// Get options
	options = config.ReadOptionsFile(*optionsFile)

	globalRouter := mux.NewRouter()
	if hamode != nil && *hamode {
		globalRouter.Use(HAMiddleware)
	}

	// System
	globalRouter.HandleFunc("/health", HealthCheckHandler).Methods(http.MethodGet)
	globalRouter.HandleFunc("/update", UpdateHandler).Methods(http.MethodPut)
	globalRouter.HandleFunc("/restart", RestartHandler).Methods(http.MethodPut)
	globalRouter.HandleFunc("/nics", GetNICsHandler).Methods(http.MethodGet)
	globalRouter.HandleFunc("/filesystems", GetFSHandler).Methods(http.MethodGet)

	// Shares
	globalRouter.HandleFunc("/shares", listShares).Methods(http.MethodGet)
	globalRouter.HandleFunc("/share/{share_name}", getShare).Methods(http.MethodGet)
	globalRouter.HandleFunc("/share", createShare).Methods(http.MethodPost)
	globalRouter.HandleFunc("/share/{share_name}", updateShare).Methods(http.MethodPut, http.MethodPatch)
	globalRouter.HandleFunc("/share/{share_name}", deleteShare).Methods(http.MethodDelete)

	// Volumes
	globalRouter.HandleFunc("/volumes", listVolumes).Methods(http.MethodGet)
	globalRouter.HandleFunc("/volume/{volume_name}/mount", mountVolume).Methods(http.MethodPost)
	globalRouter.HandleFunc("/volume/{volume_name}/mount", umountVolume).Methods(http.MethodDelete)

	// Users
	globalRouter.HandleFunc("/admin/user", getAdminUser).Methods(http.MethodGet)
	globalRouter.HandleFunc("/admin/user", updateAdminUser).Methods(http.MethodPut, http.MethodPatch)
	globalRouter.HandleFunc("/users", listUsers).Methods(http.MethodGet)
	globalRouter.HandleFunc("/user/{username}", getUser).Methods(http.MethodGet)
	globalRouter.HandleFunc("/user", createUser).Methods(http.MethodPost)
	globalRouter.HandleFunc("/user/{username}", updateUser).Methods(http.MethodPut, http.MethodPatch)
	globalRouter.HandleFunc("/user/{username}", deleteUser).Methods(http.MethodDelete)

	// Samba
	globalRouter.HandleFunc("/samba", getSambaConfig).Methods(http.MethodGet)
	globalRouter.HandleFunc("/samba/apply", applySamba).Methods(http.MethodPut)
	globalRouter.HandleFunc("/samba/status", getSambaProcessStatus).Methods(http.MethodGet)

	// Global
	globalRouter.HandleFunc("/global", api.GetGlobalConfig).Methods(http.MethodGet)
	globalRouter.HandleFunc("/global", api.UpdateGlobalConfig).Methods(http.MethodPut, http.MethodPatch)

	// Configuration
	globalRouter.HandleFunc("/config", persistConfig).Methods(http.MethodPut, http.MethodPatch)
	globalRouter.HandleFunc("/config", rollbackConfig).Methods(http.MethodDelete)

	// WebSocket
	globalRouter.HandleFunc("/events", WSChannelEventsList).Methods(http.MethodGet)
	globalRouter.HandleFunc("/ws", WSChannelHandler)

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

	// Print all routes
	globalRouter.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		template, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		log.Printf("Route: %s\n", template)
		return nil
	})

	handler := cors.New(
		cors.Options{
			AllowedOrigins:   []string{"*"},
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
			ctx = context.WithValue(ctx, "addon_config", ctx.Value("addon_config"))
			ctx = context.WithValue(ctx, "addon_option", ctx.Value("addon_option"))
			return ctx
		},
	}

	// Run the backgrounde services
	go HealthAndUpdateDataRefeshHandlers()
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
