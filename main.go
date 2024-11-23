package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/dianlight/srat/docs"
)

var SRATVersion string
var config *Config
var options *Options

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// HealthCheckHandler godoc
// @Summary      HealthCheck
// @Description  HealthCheck
// @Tags         system
// @Produce      json
// @Success      200
// @Failure      405  {object}  ResponseError
// @Router       /healt [get]
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// A very simple health check.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// In the future we could report back on the status of our DB, or our cache
	// (e.g. Redis) by performing a simple PING, and include them in the response.
	io.WriteString(w, `{"alive": true}`)
}

type ResponseError struct {
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

	// Get config
	config = readConfig(*configFile)

	// Get options
	options = readOptionsFile(*optionsFile)

	r := mux.NewRouter()
	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(loggingMiddleware)
	//r.Use(optionMiddleware)

	// Swagger
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), //The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)

	// HealtCheck
	r.HandleFunc("/health", HealthCheckHandler).Methods(http.MethodGet)

	// Shares
	r.HandleFunc("/shares", listShares).Methods(http.MethodGet)
	r.HandleFunc("/share/{share_name}", getShare).Methods(http.MethodGet)
	r.HandleFunc("/share/{share_name}", createShare).Methods(http.MethodPut)
	r.HandleFunc("/share/{share_name}", updateShare).Methods(http.MethodPost, http.MethodPatch)
	r.HandleFunc("/share/{share_name}", deleteShare).Methods(http.MethodDelete)

	// Volumes TODO:

	// Users TODO:

	// Connections TODO:

	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	})

	/*
		r.HandleFunc("/books/{title}/page/{page}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			if r.Method == http.MethodOptions {
				return
			}
			vars := mux.Vars(r)
			title := vars["title"]
			page := vars["page"]

			fmt.Fprintf(w, "You've requested the book: %s on page %s\n", title, page)
		}).Methods(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodOptions)

		r.HandleFunc("/books/{title}", CreateBook).Methods(http.MethodPost).Schemes("https")
		r.HandleFunc("/books/{title}", ReadBook).Methods("GET")
		r.HandleFunc("/books/{title}", UpdateBook).Methods("PUT")
		r.HandleFunc("/books/{title}", DeleteBook).Methods("DELETE")
	*/

	srv := &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%d", *http_port),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
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
