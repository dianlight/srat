//go:build pprof

package server

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	_ "net/http/pprof"
)

func init() {
	slog.Warn("PPROF Enabled in build")
}

// RegisterPprof exposes /debug/pprof/ only in pprof-tagged builds.
func RegisterPprof(router *mux.Router) {
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
}
