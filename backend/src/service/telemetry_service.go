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

	sentry "github.com/getsentry/sentry-go"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/time/rate"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/urlutil"
	"github.com/dianlight/tlog"
)

// skipSentryFlushForTest disables sentry.Flush() in tests to avoid blocking.
var skipSentryFlushForTest bool

// TelemetryServiceInterface defines the interface for telemetry services
type TelemetryServiceInterface interface {
	// Configure configures the telemetry service with the given mode
	Configure(mode dto.TelemetryMode) errors.E
	// ReportError reports an error to the telemetry service
	ReportError(interfaces ...any) errors.E
	// ReportEvent reports a telemetry event to the service
	ReportEvent(event string, data map[string]any) errors.E
	// IsConnectedToInternet checks if internet connection is available
	IsConnectedToInternet() bool
	// Shutdown shuts down the telemetry service
	Shutdown()
}

type TelemetryService struct {
	ctx              context.Context
	mode             dto.TelemetryMode
	sentryConfigured bool
	sentryMu         sync.Mutex // Protect Configure re-entrance
	accessToken      string     // Sentry DSN
	environment      string
	version          string

	settingService SettingServiceInterface
	haroot         HaRootServiceInterface

	// tlog callback management
	tlogErrorCallbackID string
	tlogFatalCallbackID string

	// Limiter
	errorSessionLimiter *rate.Sometimes

	// testTransport is injected by tests to capture Sentry events without real HTTP calls.
	testTransport sentry.Transport
}

// SetTestTransport injects a custom Sentry transport for unit tests. FOR TESTING ONLY.
func (ts *TelemetryService) SetTestTransport(t sentry.Transport) {
	ts.testTransport = t
}

// NewTelemetryService creates a new telemetry service instance
func NewTelemetryService(lc fx.Lifecycle, Ctx context.Context,
	settingService SettingServiceInterface,
	haroot HaRootServiceInterface,
	eventBus events.EventBusInterface,
) (TelemetryServiceInterface, errors.E) {
	accessToken := config.SentryDSN
	if accessToken == "" {
		accessToken = "disabled" // Use placeholder if not set at build time
	}

	// Determine environment from build version.
	environment := config.Environment()
	errorSessionLimiter := rate.Sometimes{First: 10}
	switch environment {
	case "development":
		errorSessionLimiter = rate.Sometimes{First: 2}
	case "prerelease":
		errorSessionLimiter = rate.Sometimes{First: 5}
	}

	setting, err := settingService.Load()
	if err != nil {
		slog.ErrorContext(Ctx, "Error loading settings for telemetry service", "error", err)
	}
	if setting == nil {
		setting = &dto.Settings{}
	}
	if !setting.TelemetryMode.IsValid() {
		setting.TelemetryMode = dto.TelemetryModes.TELEMETRYMODEASK
	}

	tm := &TelemetryService{
		ctx:                 Ctx,
		settingService:      settingService,
		mode:                setting.TelemetryMode,
		accessToken:         accessToken,
		environment:         environment,
		version:             config.Version,
		haroot:              haroot,
		errorSessionLimiter: &errorSessionLimiter,
	}

	unsubscribe := eventBus.OnSetting(func(ctx context.Context, event events.SettingEvent) errors.E {
		if err := tm.Configure(event.Setting.TelemetryMode); err != nil {
			slog.WarnContext(ctx, "Failed to reconfigure telemetry", "error", err)
		}
		return nil
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := tm.Configure(tm.mode); err != nil {
				slog.WarnContext(ctx, "Failed to configure telemetry on start", "error", err)
			}
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
	ts.sentryMu.Lock()
	defer ts.sentryMu.Unlock()

	ts.mode = mode

	// Always clear callbacks before (re)configuring
	ts.unregisterTlogCallbacks()

	if mode == dto.TelemetryModes.TELEMETRYMODEDISABLED || mode == dto.TelemetryModes.TELEMETRYMODEASK {
		if ts.sentryConfigured {
			// Disable Sentry by re-initialising with an empty DSN
			_ = sentry.Init(sentry.ClientOptions{Dsn: ""})
			ts.sentryConfigured = false
		}
		slog.InfoContext(ts.ctx, "Sentry telemetry disabled", "mode", mode.String())
		return nil
	}

	sysinfo, err := ts.haroot.GetSystemInfo()
	if err != nil {
		slog.WarnContext(ts.ctx, "Error getting system info", "error", errors.WithStack(err))
	}

	// Only initialise Sentry if mode is All or Errors and internet is available
	if (mode == dto.TelemetryModes.TELEMETRYMODEALL || mode == dto.TelemetryModes.TELEMETRYMODEERRORS) && ts.IsConnectedToInternet() {
		dsn := ts.accessToken
		if dsn == "disabled" {
			dsn = ""
		}

		opts := sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      ts.environment,
			Release:          ts.version,
			ServerName:       "github.com/" + config.Repository,
			AttachStacktrace: true,
			BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				// Anonymise IP address
				event.User.IPAddress = ""
				// Enhance stack traces for tozd/go/errors and pkg/errors when sentry-go
				// hasn't already extracted one.
				if hint != nil && hint.OriginalException != nil && len(event.Exception) > 0 {
					ex := &event.Exception[0]
					if ex.Stacktrace == nil {
						ex.Stacktrace = extractSentryStacktrace(hint.OriginalException)
					}
				}
				return event
			},
		}
		if ts.testTransport != nil {
			opts.Transport = ts.testTransport
		}
		if initErr := sentry.Init(opts); initErr != nil {
			return errors.WithStack(fmt.Errorf("sentry.Init: %w", initErr))
		}

		// Set global scope tags/user once
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("arch", runtime.GOARCH)
			scope.SetTag("os", runtime.GOOS)
			scope.SetTag("version", ts.version)
			scope.SetContext("app", sentry.Context{
				"cpu": runtime.NumCPU(),
			})
			if sysinfo != nil {
				scope.SetContext("sysinfo", sentry.Context{"data": sysinfo})
				if sysinfo.MachineId != nil {
					scope.SetUser(sentry.User{ID: *sysinfo.MachineId})
				}
			}
		})

		ts.sentryConfigured = true
		slog.InfoContext(ts.ctx, "Sentry telemetry configured", "mode", mode.String(), "environment", ts.environment, "version", ts.version)

		// Register tlog callbacks for Error and Fatal levels
		ts.registerTlogCallbacks()

		// Send a test event if mode is All
		if mode == dto.TelemetryModes.TELEMETRYMODEALL {
			if err := ts.ReportEvent("telemetry_enabled", map[string]any{
				"version":     ts.version,
				"environment": ts.environment,
			}); err != nil {
				slog.WarnContext(ts.ctx, "Failed to report telemetry_enabled event", "error", err)
			}
		}
	} else {
		slog.InfoContext(ts.ctx, "Sentry telemetry disabled", "mode", mode.String(), "internet", ts.IsConnectedToInternet())
		ts.unregisterTlogCallbacks()
	}

	return nil
}

/*
ReportError reports an error to the telemetry service.

Accepts variadic arguments matching the legacy telemetry convention:

	*http.Request — attached as request context
	error         — captured as exception
	string        — captured as message (when no error present)
	map[string]any — attached as extra data
	int           — ignored (legacy skip-frames hint)
*/
func (ts *TelemetryService) ReportError(interfaces ...any) errors.E {
	if !ts.sentryConfigured {
		return nil // Silently ignore if not configured
	}

	if ts.mode == dto.TelemetryModes.TELEMETRYMODEDISABLED || ts.mode == dto.TelemetryModes.TELEMETRYMODEASK {
		return nil
	}

	// Report errors for both All and Errors modes
	if ts.mode == dto.TelemetryModes.TELEMETRYMODEALL || ts.mode == dto.TelemetryModes.TELEMETRYMODEERRORS {
		ts.errorSessionLimiter.Do(func() {
			var captureErr error
			var captureMsg string
			var req *http.Request
			extras := make(map[string]any)

			for _, iface := range interfaces {
				switch v := iface.(type) {
				case error:
					captureErr = v
				case string:
					captureMsg = v
				case *http.Request:
					req = v
				case map[string]any:
					for k, val := range v {
						extras[k] = val
					}
				}
			}

			hub := sentry.CurrentHub().Clone()
			if len(extras) > 0 {
				hub.Scope().SetContext("extra", sentry.Context(extras))
			}
			if req != nil {
				hub.Scope().SetRequest(req)
			}

			if captureErr != nil {
				hub.CaptureException(captureErr)
				slog.DebugContext(ts.ctx, "Error reported to Sentry", "error", captureErr)
			} else if captureMsg != "" {
				hub.CaptureMessage(captureMsg)
				slog.DebugContext(ts.ctx, "Message reported to Sentry", "message", captureMsg)
			}
		})
	}

	return nil
}

// ReportEvent reports a telemetry event to the service
func (ts *TelemetryService) ReportEvent(event string, data map[string]any) errors.E {
	if !ts.sentryConfigured {
		return nil // Silently ignore if not configured
	}

	// Only report events in All mode
	if ts.mode != dto.TelemetryModes.TELEMETRYMODEALL {
		return nil
	}

	// Add event type and timestamp to data
	if data == nil {
		data = make(map[string]any)
	}
	data["event_type"] = event
	data["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	sentryEvent := &sentry.Event{
		Level:    sentry.LevelInfo,
		Message:  fmt.Sprintf("Event: %s", event),
		Contexts: map[string]sentry.Context{"data": sentry.Context(data)},
	}
	sentry.CaptureEvent(sentryEvent)
	slog.DebugContext(ts.ctx, "Event reported to Sentry", "event", event, "data", data)

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
	const sentryURL = "https://sentry.io"
	if err := urlutil.ValidateURL(sentryURL, []string{"sentry.io"}); err != nil {
		slog.DebugContext(ctx, "Untrusted connectivity URL", "url", sentryURL, "error", err)
		return false
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", sentryURL, nil)
	if err != nil {
		slog.DebugContext(ctx, "Failed to create internet connectivity request", "error", err)
		return false
	}

	// Execute request
	resp, err := client.Do(req) // #nosec G704
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
	// Unregister any tlog callbacks first to prevent them from trying to use Sentry
	ts.unregisterTlogCallbacks()

	ts.sentryMu.Lock()
	defer ts.sentryMu.Unlock()

	if ts.sentryConfigured {
		if !skipSentryFlushForTest {
			sentry.Flush(2 * time.Second)
		}
		ts.sentryConfigured = false
		slog.InfoContext(ts.ctx, "Sentry telemetry service shutdown")
	}
}

// SetSkipSentryFlushForTest sets whether to skip sentry.Flush() in tests - FOR TESTING ONLY
func SetSkipSentryFlushForTest(skip bool) {
	skipSentryFlushForTest = skip
}

// extractSentryStacktrace builds a sentry.Stacktrace from pkg/errors or tozd/go/errors
// when sentry-go hasn't extracted one automatically.
func extractSentryStacktrace(err error) *sentry.Stacktrace {
	type uintptrStackTracer interface {
		StackTrace() []uintptr
	}

	// sentry-go handles pkg/errors natively; only handle tozd/go/errors ([]uintptr) here.
	if cerr, ok := err.(uintptrStackTracer); ok {
		pcs := cerr.StackTrace()
		callersFrames := runtime.CallersFrames(pcs)
		frames := make([]sentry.Frame, 0, len(pcs))
		for {
			f, more := callersFrames.Next()
			frames = append(frames, sentry.NewFrame(f))
			if !more {
				break
			}
		}
		// Sentry expects frames innermost-last; reverse from innermost-first
		for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
			frames[i], frames[j] = frames[j], frames[i]
		}
		return &sentry.Stacktrace{Frames: frames}
	}

	return nil
}

// registerTlogCallbacks registers callbacks to forward tlog Error/Fatal to Sentry
func (ts *TelemetryService) registerTlogCallbacks() {
	// Safety: avoid duplicate registrations
	ts.unregisterTlogCallbacks()

	callback := func(event tlog.LogEvent) {
		// Only forward when configured and mode allows
		if !ts.sentryConfigured {
			return
		}
		if ts.mode != dto.TelemetryModes.TELEMETRYMODEALL && ts.mode != dto.TelemetryModes.TELEMETRYMODEERRORS {
			return
		}

		// Try to extract an error and request from log event attributes
		var extractedErr error
		var request *http.Request
		extraData := make(map[string]any)

		// ANSI escape code remover
		ansiRegexp := regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
		stripANSI := func(s string) string { return ansiRegexp.ReplaceAllString(s, "") }

		// Recursively process attributes into a map, extracting error/request and cleaning strings
		var processAttr func(a slog.Attr, dst map[string]any)
		processAttr = func(a slog.Attr, dst map[string]any) {
			key := strings.ToLower(a.Key)

			// Handle groups first
			if a.Value.Kind() == slog.KindGroup {
				groupMap := map[string]any{}
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
				nested := map[string]any{}
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
