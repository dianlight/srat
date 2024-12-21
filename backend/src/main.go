package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	_ "github.com/dianlight/srat/docs"
)

var SRATVersion string
var options *config.Options
var smbConfigFile *string
var globalRouter *mux.Router
var templateData []byte

// Static files
//
//go:embed static/*
var content embed.FS

//go:embed templates/smb.gtpl
var defaultTemplate embed.FS

func ACAOMethodMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // Access-Control-Allow-Headers
		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
	})
}

type ResponseError struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
	Body  any    `json:"body"`
}

//	@title			SRAT API
//	@version		1.0
//	@description	This are samba rest admin API
// _termsOfService http://swagger.io/terms/

//	@contact.name	Lucio Tarantino
// _contact.url http://www.swagger.io/support
//	@contact.email	lucio.tarantino@gmail.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// _host petstore.swagger.io
// _BasePath /v2
func main() {

	optionsFile := flag.String("opt", "/data/options.json", "Addon Options json file")
	configFile := flag.String("conf", "", "Config json file, can be omitted if used in a pipe")
	http_port := flag.Int("port", 8080, "Http Port on listen to")
	templateFile := flag.String("template", "", "Template file")
	smbConfigFile := flag.String("out", "", "Output file, if not defined output will be to console")
	data.ROMode = flag.Bool("ro", false, "Read only mode")

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")

	flag.Usage = func() {
		writer := flag.CommandLine.Output()

		fmt.Fprint(writer, "SRAT: Samba Rest Administration Interface\n")
		fmt.Fprintf(writer, "Version: %s\n", SRATVersion)
		fmt.Fprintf(writer, "Documentation: https://github.com/dianlight/SRAT\n\n")

		flag.PrintDefaults()
	}

	flag.Parse()

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
	data.Config = config.ReadConfig(*configFile)
	data.Config = config.MigrateConfig(data.Config)

	// Get options
	options = config.ReadOptionsFile(*optionsFile)

	globalRouter := mux.NewRouter()
	globalRouter.Use(mux.CORSMethodMiddleware(globalRouter))
	globalRouter.Use(ACAOMethodMiddleware)
	//r.Use(optionMiddleware)

	// Swagger
	globalRouter.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), //The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)

	// HealtCheck
	globalRouter.HandleFunc("/health", HealthCheckHandler).Methods(http.MethodGet, http.MethodOptions)

	// Shares
	globalRouter.HandleFunc("/shares", listShares).Methods(http.MethodGet, http.MethodOptions)
	globalRouter.HandleFunc("/share/{share_name}", getShare).Methods(http.MethodGet, http.MethodOptions)
	globalRouter.HandleFunc("/share/{share_name}", createShare).Methods(http.MethodPost)
	globalRouter.HandleFunc("/share/{share_name}", updateShare).Methods(http.MethodPut, http.MethodPatch)
	globalRouter.HandleFunc("/share/{share_name}", deleteShare).Methods(http.MethodDelete)

	// Volumes
	globalRouter.HandleFunc("/volumes", listVolumes).Methods(http.MethodGet, http.MethodOptions)
	globalRouter.HandleFunc("/volume/{volume_name}", getVolume).Methods(http.MethodGet, http.MethodOptions)
	//	globalRouter.HandleFunc("/volume/{volume_name}", updateVolume).Methods(http.MethodPut, http.MethodPatch)
	//	globalRouter.HandleFunc("/volume/{volume_name}/mount", mountVolume).Methods(http.MethodPost)
	//	globalRouter.HandleFunc("/volume/{volume_name}/mount", umountVolume).Methods(http.MethodDelete)

	// Users
	globalRouter.HandleFunc("/admin/user", getAdminUser).Methods(http.MethodGet, http.MethodOptions)
	globalRouter.HandleFunc("/admin/user", updateAdminUser).Methods(http.MethodPut, http.MethodPatch)
	globalRouter.HandleFunc("/users", listUsers).Methods(http.MethodGet, http.MethodOptions)
	globalRouter.HandleFunc("/user/{username}", getUser).Methods(http.MethodGet, http.MethodOptions)
	globalRouter.HandleFunc("/user", createUser).Methods(http.MethodPost, http.MethodOptions)
	globalRouter.HandleFunc("/user/{username}", updateUser).Methods(http.MethodPut, http.MethodPatch)
	globalRouter.HandleFunc("/user/{username}", deleteUser).Methods(http.MethodDelete)

	// Samba
	globalRouter.HandleFunc("/samba", getSambaConfig).Methods(http.MethodGet)
	globalRouter.HandleFunc("/samba/apply", applySamba).Methods(http.MethodPut)

	// WebSocket
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

	loggedRouter := handlers.LoggingHandler(os.Stdout, globalRouter)

	srv := &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%d", *http_port),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      loggedRouter, // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Printf("Starting Server... \n Swagger At: http://localhost:%d/swagger/index.html", *http_port)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}
