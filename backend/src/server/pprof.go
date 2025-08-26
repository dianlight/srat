//go:build pprof

package server

import (
	"log/slog"
	_ "net/http/pprof"
)

func init() {
	slog.Warn("PPROF Enabled in build")
}
