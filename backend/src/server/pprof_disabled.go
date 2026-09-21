//go:build !pprof

package server

import (
	"github.com/gorilla/mux"
)

// RegisterPprof is a no-op in production builds so /debug/pprof/ returns 404.
func RegisterPprof(_ *mux.Router) {}
