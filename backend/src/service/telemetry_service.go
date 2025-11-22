package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	oerrors "github.com/pkg/errors"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/time/rate"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/tlog"
	"github.com/rollbar/rollbar-go"
)

// Global state to track if rollbar has been closed to prevent double-close panics
var (
	rollbarGlobalMu         sync.Mutex
	rollbarGlobalClosed     bool
	skipRollbarCloseForTest bool // Set to true in tests to prevent closing rollbar
)

// TelemetryServiceInterface defines the interface for telemetry services
type TelemetryServiceInterface interface {
	// Configure configures the telemetry service with the given mode
	Configure(mode dto.TelemetryMode) errors.E
	// ReportError reports an error to the telemetry service
	ReportError(interfaces ...interface{}) errors.E
	// ReportEvent reports a telemetry event to the service
	ReportEvent(event string, data map[string]interface{}) errors.E
	// IsConnectedToInternet checks if internet connection is available
	IsConnectedToInternet() bool
	// Shutdown shuts down the telemetry service
	Shutdown()
}

type TelemetryService struct {
	ctx               context.Context
	mode              dto.TelemetryMode
	rollbarConfigured bool
	rollbarClosed     bool       // Track if rollbar has been explicitly closed
	rollbarMu         sync.Mutex // Protect rollbar close operations
	accessToken       string
	environment       string
	version           string

	settingService SettingServiceInterface
	haroot         HaRootServiceInterface
	eventBus       events.EventBusInterface

	// tlog callback management
	tlogErrorCallbackID string
	tlogFatalCallbackID string

	// Limiter
	errorSessionLimiter *rate.Sometimes
}

// NewTelemetryService creates a new telemetry service instance
func NewTelemetryService(lc fx.Lifecycle, Ctx context.Context,
	settingService SettingServiceInterface,
	haroot HaRootServiceInterface,
	eventBus events.EventBusInterface,
) (TelemetryServiceInterface, errors.E) {
	accessToken := config.RollbarToken
	if accessToken == "" {
		accessToken = "disabled" // Use placeholder if not set at build time
	}

	// Determine environment: use build-time setting or auto-detect from version
	environment := config.RollbarEnvironment
	errorSessionLimiter := rate.Sometimes{First: 10}
	if environment == "" {
		if config.Version == "0.0.0-dev.0" || strings.Contains(config.Version, "-dev.") {
			environment = "development"
			errorSessionLimiter = rate.Sometimes{First: 2}
		} else if strings.Contains(config.Version, "-rc.") {
			environment = "prerelease"
			errorSessionLimiter = rate.Sometimes{First: 5}
		} else {
			environment = "production"
		}
	}

	telemetryMode, _ := settingService.GetValue("TelemetryMode")
	if mode, ok := telemetryMode.(dto.TelemetryMode); !ok || !mode.IsValid() {
		telemetryMode = dto.TelemetryModes.TELEMETRYMODEASK
	}

	tm := &TelemetryService{
		ctx:                 Ctx,
		settingService:      settingService,
		mode:                telemetryMode.(dto.TelemetryMode),
		accessToken:         accessToken,
		environment:         environment,
		version:             config.Version,
		haroot:              haroot,
		errorSessionLimiter: &errorSessionLimiter,
	}

	unsubscribe := eventBus.OnSetting(func(ctx context.Context, event events.SettingEvent) {
		tm.Configure(event.Setting.TelemetryMode)
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			tm.Configure(tm.mode)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			tm.Shutdown()
			if unsubscribe != nil {
				unsubscribe()
			}
			return nil
		},
	})

	return tm, nil
}

// Configure configures the telemetry service with the given mode
func (ts *TelemetryService) Configure(mode dto.TelemetryMode) errors.E {
	ts.mode = mode
	if mode == dto.TelemetryModes.TELEMETRYMODEDISABLED || mode == dto.TelemetryModes.TELEMETRYMODEASK {
		return nil
	}

	// Shutdown existing configuration only if we're reconfiguring
	ts.rollbarMu.Lock()
	if ts.rollbarConfigured && !ts.rollbarClosed {
		// Check global state before closing
		rollbarGlobalMu.Lock()
		if !rollbarGlobalClosed {
			// Wrap in recover to handle potential panic from rollbar library
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.WarnContext(ts.ctx, "Recovered from rollbar.Close() panic during reconfiguration", "panic", r)
					}
				}()
				rollbar.Close()
			}()
			rollbarGlobalClosed = true
		}
		rollbarGlobalMu.Unlock()
		ts.rollbarConfigured = false
		ts.rollbarClosed = true
	}
	ts.rollbarMu.Unlock()

	// Always ensure callbacks are cleared before (re)configuring
	ts.unregisterTlogCallbacks()

	sysinfo, err := ts.haroot.GetSystemInfo()
	if err != nil {
		slog.WarnContext(ts.ctx, "Error getting system info", "error", errors.WithStack(err))
	}

	// Only initialize Rollbar if mode is All or Errors and internet is available
	if (mode == dto.TelemetryModes.TELEMETRYMODEALL || mode == dto.TelemetryModes.TELEMETRYMODEERRORS) && ts.IsConnectedToInternet() {
		rollbar.SetToken(ts.accessToken)
		rollbar.SetEnvironment(ts.environment)
		rollbar.SetCaptureIp(rollbar.CaptureIpAnonymize)
		rollbar.SetCodeVersion(ts.version)
		rollbar.SetPlatform("client")
		rollbar.SetServerRoot("github.com/" + config.Repository)
		rollbar.SetCustom(map[string]interface{}{
			"version":     ts.version,
			"environment": ts.environment,
			"arch":        runtime.GOARCH,
			"os":          runtime.GOOS,
			"cpu":         runtime.NumCPU(),
			"sysinfo":     sysinfo,
		})
		if sysinfo != nil && sysinfo.MachineId != nil {
			rollbar.SetPerson(*sysinfo.MachineId, "", "")
		}
		rollbar.SetStackTracer(func(err error) ([]runtime.Frame, bool) {
			// preserve the default behavior for other types of errors
			if trace, ok := rollbar.DefaultStackTracer(err); ok {
				return trace, ok
			}

			type stackTracer interface {
				StackTrace() oerrors.StackTrace
			}

			if nerr, ok := oerrors.Cause(err).(stackTracer); ok && nerr != nil {
				// Convert github.com/pkg/errors StackTrace -> []uintptr -> []runtime.Frame
				st := nerr.StackTrace()
				pcs := make([]uintptr, 0, len(st))
				for _, f := range st {
					pcs = append(pcs, uintptr(f))
				}
				frames := runtime.CallersFrames(pcs)
				var out []runtime.Frame
				for {
					f, more := frames.Next()
					out = append(out, f)
					if !more {
						break
					}
				}
				return out, true
			}

			if cerr, ok := err.(errors.E); ok {
				frames := runtime.CallersFrames(cerr.StackTrace())
				var out []runtime.Frame
				for {
					f, more := frames.Next()
					out = append(out, f)
					if !more {
						break
					}
				}
				return out, true
			}

			return nil, false
		})

		ts.rollbarMu.Lock()
		ts.rollbarConfigured = true
		ts.rollbarClosed = false // Reset the closed flag since we've reconfigured
		// Reset global closed flag since we've reconfigured
		rollbarGlobalMu.Lock()
		rollbarGlobalClosed = false
		rollbarGlobalMu.Unlock()
		ts.rollbarMu.Unlock()
		slog.InfoContext(ts.ctx, "Rollbar telemetry configured", "mode", mode.String(), "platform", rollbar.Platform(), "environment", ts.environment, "version", ts.version)

		// Register tlog callbacks for Error and Fatal levels
		ts.registerTlogCallbacks()

		// Send a test event if mode is All
		if mode == dto.TelemetryModes.TELEMETRYMODEALL {
			ts.ReportEvent("telemetry_enabled", map[string]interface{}{
				"version":     ts.version,
				"environment": ts.environment,
			})
		}
	} else {
		slog.InfoContext(ts.ctx, "Rollbar telemetry disabled", "mode", mode.String(), "internet", ts.IsConnectedToInternet())
		// Ensure callbacks are not registered when disabled
		ts.unregisterTlogCallbacks()
	}

	return nil
}

/*
ReportError reports an error to the telemetry service.

	Error reports an item with level `error`. This function recognizes arguments with the following types:

*http.Request
error
string
map[string]interface{}
int
The string and error types are mutually exclusive. If an error is present then a stack trace is captured. If an int is also present then we skip that number of stack frames. If the map is present it is used as extra custom data in the item. If a string is present without an error, then we log a message without a stack trace. If a request is present we extract as much relevant information from it as we can.
*/
func (ts *TelemetryService) ReportError(interfaces ...interface{}) errors.E {
	if !ts.rollbarConfigured {
		return nil // Silently ignore if not configured
	}

	if ts.mode == dto.TelemetryModes.TELEMETRYMODEDISABLED || ts.mode == dto.TelemetryModes.TELEMETRYMODEASK {
		return nil // Don't report if disabled or asking
	}

	//interfaces = append(interfaces, 3)

	// Report errors for both All and Errors modes
	if ts.mode == dto.TelemetryModes.TELEMETRYMODEALL || ts.mode == dto.TelemetryModes.TELEMETRYMODEERRORS {
		ts.errorSessionLimiter.Do(func() {
			rollbar.Error(interfaces...)
			slog.DebugContext(ts.ctx, "Error reported to Rollbar", "error", interfaces)
		})
	}

	return nil
}

// ReportEvent reports a telemetry event to the service
func (ts *TelemetryService) ReportEvent(event string, data map[string]interface{}) errors.E {
	if !ts.rollbarConfigured {
		return nil // Silently ignore if not configured
	}

	// Only report events in All mode
	if ts.mode != dto.TelemetryModes.TELEMETRYMODEALL {
		return nil
	}

	// Add event type and timestamp to data
	if data == nil {
		data = make(map[string]interface{})
	}
	data["event_type"] = event
	data["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	rollbar.Info(fmt.Sprintf("Event: %s", event), data)
	slog.DebugContext(ts.ctx, "Event reported to Rollbar", "event", event, "data", data)

	return nil
}

// IsConnectedToInternet checks if internet connection is available
func (ts *TelemetryService) IsConnectedToInternet() bool {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create request to test connectivity
	req, err := http.NewRequestWithContext(ctx, "HEAD", "https://api.rollbar.com", nil)
	if err != nil {
		slog.DebugContext(ctx, "Failed to create internet connectivity request", "error", err)
		return false
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		slog.DebugContext(ctx, "Internet connectivity check failed", "error", err)
		return false
	}
	defer resp.Body.Close()

	// Consider successful if we get any response (even 4xx/5xx indicates connectivity)
	connected := resp.StatusCode > 0
	slog.DebugContext(ctx, "Internet connectivity check completed", "connected", connected, "status", resp.StatusCode)

	return connected
}

// Shutdown shuts down the telemetry service
func (ts *TelemetryService) Shutdown() {
	// Unregister any tlog callbacks first to prevent them from trying to use Rollbar
	ts.unregisterTlogCallbacks()

	// Close Rollbar only if it's configured and not already closed
	ts.rollbarMu.Lock()
	defer ts.rollbarMu.Unlock()

	if ts.rollbarConfigured && !ts.rollbarClosed {
		// Wait for any pending rollbar messages to be sent before closing
		rollbar.Wait()

		// Check global state before closing
		rollbarGlobalMu.Lock()
		defer rollbarGlobalMu.Unlock()

		// Skip closing if we're in test mode (to avoid "channel closed" errors across tests)
		if !skipRollbarCloseForTest && !rollbarGlobalClosed {
			// Wrap in recover to handle potential panic from rollbar library
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.WarnContext(ts.ctx, "Recovered from rollbar.Close() panic", "panic", r)
					}
				}()
				rollbar.Close()
			}()
			rollbarGlobalClosed = true
			slog.InfoContext(ts.ctx, "Rollbar telemetry service shutdown")
		}
		ts.rollbarConfigured = false
		ts.rollbarClosed = true
	}
}

// ResetRollbarGlobalState resets the global rollbar state - FOR TESTING ONLY
func ResetRollbarGlobalState() {
	rollbarGlobalMu.Lock()
	defer rollbarGlobalMu.Unlock()
	rollbarGlobalClosed = false
}

// SetSkipRollbarCloseForTest sets whether to skip rollbar close in tests - FOR TESTING ONLY
func SetSkipRollbarCloseForTest(skip bool) {
	rollbarGlobalMu.Lock()
	defer rollbarGlobalMu.Unlock()
	skipRollbarCloseForTest = skip
}

// registerTlogCallbacks registers callbacks to forward tlog Error/Fatal to Rollbar
func (ts *TelemetryService) registerTlogCallbacks() {
	// Safety: avoid duplicate registrations
	ts.unregisterTlogCallbacks()

	callback := func(event tlog.LogEvent) {
		// Only forward when configured and mode allows
		if !ts.rollbarConfigured {
			return
		}
		if ts.mode != dto.TelemetryModes.TELEMETRYMODEALL && ts.mode != dto.TelemetryModes.TELEMETRYMODEERRORS {
			return
		}

		// Try to extract an error and request from log event attributes
		var extractedErr error
		var request *http.Request
		extraData := make(map[string]interface{})

		// ANSI escape code remover
		ansiRegexp := regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
		stripANSI := func(s string) string { return ansiRegexp.ReplaceAllString(s, "") }

		// Recursively process attributes into a map, extracting error/request and cleaning strings
		var processAttr func(a slog.Attr, dst map[string]interface{})
		processAttr = func(a slog.Attr, dst map[string]interface{}) {
			key := strings.ToLower(a.Key)

			// Handle groups first
			if a.Value.Kind() == slog.KindGroup {
				groupMap := map[string]interface{}{}
				for _, ga := range a.Value.Group() {
					processAttr(ga, groupMap)
				}
				if len(groupMap) > 0 {
					dst[key] = groupMap
				}
				return
			}

			v := a.Value.Any()
			if v == nil {
				return // Skip nil values
			}
			switch vv := v.(type) {
			case *http.Request:
				// Capture request, do not include in extraData
				request = vv
				return
			case errors.E:
				// Capture structured error, do not include in extraData
				extractedErr = vv
				return
			case error:
				// Capture generic error, do not include in extraData
				extractedErr = vv
				return
			case string:
				dst[key] = stripANSI(vv)
				return
			case fmt.Stringer:
				func() {
					defer func() {
						if recover() != nil {
							dst[key] = fmt.Sprintf("%+v", vv) // Fallback if Stringer panics
						}
					}()
					dst[key] = stripANSI(vv.String())
				}()
				return
			case []slog.Attr:
				// Some formatters may expose groups via Any()
				nested := map[string]interface{}{}
				for _, ga := range vv {
					processAttr(ga, nested)
				}
				if len(nested) > 0 {
					dst[key] = nested
				}
				return
			default:
				// Fallback: store value as-is
				dst[key] = vv
				return
			}
		}

		event.Record.Attrs(func(attr slog.Attr) bool {
			tlog.Trace("Attr:", attr.Key, "=", attr.Value.Any())
			processAttr(attr, extraData)
			return true
		})

		// Use existing telemetry path to report error
		if extractedErr == nil {
			if request != nil {
				_ = ts.ReportError(request, "§ "+event.Record.Message, extraData)
			} else {
				_ = ts.ReportError("§ "+event.Record.Message, extraData)
			}
		} else {
			if request != nil {
				_ = ts.ReportError(request, extractedErr, extraData)
			} else {
				_ = ts.ReportError(extractedErr, extraData)
			}
		}
	}

	ts.tlogErrorCallbackID = tlog.RegisterCallback(tlog.LevelError, callback)
	ts.tlogFatalCallbackID = tlog.RegisterCallback(tlog.LevelFatal, callback)
}

// unregisterTlogCallbacks removes previously registered callbacks, if any
func (ts *TelemetryService) unregisterTlogCallbacks() {
	if ts.tlogErrorCallbackID != "" {
		if tlog.UnregisterCallback(tlog.LevelError, ts.tlogErrorCallbackID) {
			ts.tlogErrorCallbackID = ""
		}
	}
	if ts.tlogFatalCallbackID != "" {
		if tlog.UnregisterCallback(tlog.LevelFatal, ts.tlogFatalCallbackID) {
			ts.tlogFatalCallbackID = ""
		}
	}
}
