// +build pprof

package main

import (
    "log"
    "log/slog"
    "net/http"
    _ "net/http/pprof"
)

func init() {
   slog.Warn("PPROF Enabled in build")
}
