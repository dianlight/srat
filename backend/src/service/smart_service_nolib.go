//go:build !smartlib

package service

import (
	"log/slog"

	"github.com/dianlight/smartmontools-sdk/bindings/go/v8"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
)

// initSmartClient returns a SMART client using the exec backend (runs smartctl
// as a subprocess). This build path avoids the purego/libdl dynamic-linking
// requirement so that the resulting binary is fully statically linked and
// portable across glibc and musl environments.
//
// To enable the lib backend (purego, requires libsmartmon_go.so), build with
// the "smartlib" tag: go build -tags smartlib ...
func initSmartClient(apiCtx *dto.ContextState) smartmontools.SmartClient {
	recordLibSmartBackendOutcome(apiCtx, false, "static build: lib backend disabled, build with -tags smartlib to enable direct mode")
	slog.Info("SMART exec backend active (static build — lib backend disabled; build with -tags smartlib to enable)")
	client, _ := smartmontools.NewClient(smartmontools.WithTLogHandler(tlog.NewLoggerWithLevel(tlog.LevelInfo)))
	return client
}
