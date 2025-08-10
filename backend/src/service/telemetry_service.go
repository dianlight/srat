package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tlog"
	"github.com/rollbar/rollbar-go"
)

// TelemetryServiceInterface defines the interface for telemetry services
type TelemetryServiceInterface interface {
	// Configure configures the telemetry service with the given mode
	Configure(mode dto.TelemetryMode) error
	// ReportError reports an error to the telemetry service
	ReportError(interfaces ...interface{}) error
	// ReportEvent reports a telemetry event to the service
	ReportEvent(event string, data map[string]interface{}) error
	// IsConnectedToInternet checks if internet connection is available
	IsConnectedToInternet() bool
	// Shutdown shuts down the telemetry service
	Shutdown()
}

type TelemetryService struct {
	ctx               context.Context
	mode              dto.TelemetryMode
	rollbarConfigured bool
	accessToken       string
	environment       string
	version           string

	prop repository.PropertyRepositoryInterface

	// tlog callback management
	tlogErrorCallbackID string
	tlogFatalCallbackID string
}

// NewTelemetryService creates a new telemetry service instance
func NewTelemetryService(lc fx.Lifecycle, Ctx context.Context, prop repository.PropertyRepositoryInterface) (TelemetryServiceInterface, errors.E) {
	accessToken := config.RollbarToken
	if accessToken == "" {
		accessToken = "disabled" // Use placeholder if not set at build time
	}

	// Determine environment: use build-time setting or auto-detect from version
	environment := config.RollbarEnvironment
	if environment == "" {
		if config.Version == "0.0.0-dev.0" || strings.Contains(config.Version, "-dev.") {
			environment = "development"
		} else {
			environment = "production"
		}
	}

	dbconfig, err := prop.All(true)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var conv converter.DtoToDbomConverterImpl

	var mconfig dto.Settings
	err = conv.PropertiesToSettings(dbconfig, &mconfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !mconfig.TelemetryMode.IsValid() {
		mconfig.TelemetryMode = dto.TelemetryModes.TELEMETRYMODEASK
	}

	tm := &TelemetryService{
		ctx:         Ctx,
		prop:        prop,
		mode:        mconfig.TelemetryMode,
		accessToken: accessToken,
		environment: environment,
		version:     config.Version,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			tm.Configure(mconfig.TelemetryMode)
			return nil
		},
	})

	return tm, nil
}

// Configure configures the telemetry service with the given mode
func (ts *TelemetryService) Configure(mode dto.TelemetryMode) error {
	ts.mode = mode

	// Shutdown existing configuration
	if ts.rollbarConfigured {
		rollbar.Close()
		ts.rollbarConfigured = false
	}

	// Always ensure callbacks are cleared before (re)configuring
	ts.unregisterTlogCallbacks()

	// Only initialize Rollbar if mode is All or Errors and internet is available
	if (mode == dto.TelemetryModes.TELEMETRYMODEALL || mode == dto.TelemetryModes.TELEMETRYMODEERRORS) && ts.IsConnectedToInternet() {
		rollbar.SetToken(ts.accessToken)
		rollbar.SetEnvironment(ts.environment)
		rollbar.SetCodeVersion(ts.version)
		rollbar.SetPlatform("client")
		rollbar.SetServerRoot("github.com/" + config.Repository)
		rollbar.SetCustom(map[string]interface{}{
			"version":     ts.version,
			"environment": ts.environment,
			// TODO: Add Hassos and Homeassistant data info
		})

		ts.rollbarConfigured = true
		slog.Info("Rollbar telemetry configured", "mode", mode.String(), "platform", rollbar.Platform(), "environment", ts.environment, "version", ts.version)

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
		slog.Info("Rollbar telemetry disabled", "mode", mode.String(), "internet", ts.IsConnectedToInternet())
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
func (ts *TelemetryService) ReportError(interfaces ...interface{}) error {
	if !ts.rollbarConfigured {
		return nil // Silently ignore if not configured
	}

	if ts.mode == dto.TelemetryModes.TELEMETRYMODEDISABLED || ts.mode == dto.TelemetryModes.TELEMETRYMODEASK {
		return nil // Don't report if disabled or asking
	}

	// Report errors for both All and Errors modes
	if ts.mode == dto.TelemetryModes.TELEMETRYMODEALL || ts.mode == dto.TelemetryModes.TELEMETRYMODEERRORS {
		rollbar.Error(interfaces...)
		slog.Debug("Error reported to Rollbar", "error", interfaces)
	}

	return nil
}

// ReportEvent reports a telemetry event to the service
func (ts *TelemetryService) ReportEvent(event string, data map[string]interface{}) error {
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
	slog.Debug("Event reported to Rollbar", "event", event, "data", data)

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
		slog.Debug("Failed to create internet connectivity request", "error", err)
		return false
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("Internet connectivity check failed", "error", err)
		return false
	}
	defer resp.Body.Close()

	// Consider successful if we get any response (even 4xx/5xx indicates connectivity)
	connected := resp.StatusCode > 0
	slog.Debug("Internet connectivity check completed", "connected", connected, "status", resp.StatusCode)

	return connected
}

// Shutdown shuts down the telemetry service
func (ts *TelemetryService) Shutdown() {
	if ts.rollbarConfigured {
		rollbar.Close()
		ts.rollbarConfigured = false
		slog.Info("Rollbar telemetry service shutdown")
	}
	// Unregister any tlog callbacks
	ts.unregisterTlogCallbacks()
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

		// Try to extract an error from the log event attributes
		var extractedErr error
		extraData := make(map[string]interface{})
		event.Record.Attrs(func(attr slog.Attr) bool {
			tlog.Trace("Attr:", attr.Key, "=", attr.Value.Any())
			key := strings.ToLower(attr.Key)
			if key == "error" || key == "err" {
				v := attr.Value.Any()
				extraData[key] = v
				switch vv := v.(type) {
				case error:
					extractedErr = vv
					delete(extraData, key)
					return true
				case []slog.Attr:
					extraData[key] = map[string]interface{}{}
					// When formatted, error may be represented as slice of Attrs, first usually message
					if len(vv) > 0 {
						for _, a := range vv {
							extraData[key].(map[string]interface{})[a.Key] = a.Value.Any()
							if a.Key == "org_error" {
								extractedErr = a.Value.Any().(error)
								delete(extraData[key].(map[string]interface{}), a.Key)
								return true
							}
						}
					}
				}
			}
			return true
		})

		// Use existing telemetry path to report error
		if extractedErr == nil {
			_ = ts.ReportError(event.Record.Message, extraData)
		} else {
			_ = ts.ReportError(extractedErr, extraData)
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
