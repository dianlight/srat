//go:build smartlib

package service

import (
	"log/slog"

	"github.com/dianlight/smartmontools-go"
	libbackend "github.com/dianlight/smartmontools-go/backends/lib"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
)

// initSmartClient tries the lib backend first (requires libsmartmon_go.so at
// runtime) then falls back to the exec backend. The lib backend enables direct
// SMART data access without spawning smartctl subprocess calls, but it requires
// the smartmon wrapper shared library to be present at runtime.
func initSmartClient(apiCtx *dto.ContextState) smartmontools.SmartClient {
	libBe, libErr := libbackend.New(libbackend.WithTLogHandler(tlog.NewLoggerWithLevel(tlog.LevelInfo)))
	if libErr == nil {
		if apiCtx != nil {
			apiCtx.LibSmartAvailable = true
		}
		slog.Info("SMART lib backend loaded (direct mode available)")
		client, _ := smartmontools.NewClient(
			smartmontools.WithBackend(libBe),
			smartmontools.WithTLogHandler(tlog.NewLoggerWithLevel(tlog.LevelInfo)),
		)
		return client
	}
	slog.Info("SMART lib backend not available, falling back to exec backend", "reason", libErr.Error())
	client, _ := smartmontools.NewClient(smartmontools.WithTLogHandler(tlog.NewLoggerWithLevel(tlog.LevelInfo)))
	return client
}
