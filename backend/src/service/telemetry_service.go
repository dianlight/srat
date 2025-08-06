package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/rollbar/rollbar-go"
)

// TelemetryServiceInterface defines the interface for telemetry services
type TelemetryServiceInterface interface {
	// Configure configures the telemetry service with the given mode
	Configure(mode dto.TelemetryMode) error
	// ReportError reports an error to the telemetry service
	ReportError(err error) error
	// ReportEvent reports a telemetry event to the service
	ReportEvent(event string, data map[string]interface{}) error
	// IsConnectedToInternet checks if internet connection is available
	IsConnectedToInternet() bool
	// Shutdown shuts down the telemetry service
	Shutdown()
}

type TelemetryService struct {
	mode              dto.TelemetryMode
	rollbarConfigured bool
	accessToken       string
	environment       string
	version           string
}

// NewTelemetryService creates a new telemetry service instance
func NewTelemetryService(accessToken, environment, version string) TelemetryServiceInterface {
	return &TelemetryService{
		mode:        dto.TelemetryModes.TELEMETRYMODEASK, // Default to Ask
		accessToken: accessToken,
		environment: environment,
		version:     version,
	}
}

// Configure configures the telemetry service with the given mode
func (ts *TelemetryService) Configure(mode dto.TelemetryMode) error {
	ts.mode = mode

	// Shutdown existing configuration
	if ts.rollbarConfigured {
		rollbar.Close()
		ts.rollbarConfigured = false
	}

	// Only initialize Rollbar if mode is All or Errors and internet is available
	if (mode == dto.TelemetryModes.TELEMETRYMODEALL || mode == dto.TelemetryModes.TELEMETRYMODEERRORS) && ts.IsConnectedToInternet() {
		rollbar.SetToken(ts.accessToken)
		rollbar.SetEnvironment(ts.environment)
		rollbar.SetCodeVersion(ts.version)
		rollbar.SetServerHost("srat-server")
		rollbar.SetServerRoot("/")

		ts.rollbarConfigured = true
		slog.Info("Rollbar telemetry configured", "mode", mode.String())

		// Send a test event if mode is All
		if mode == dto.TelemetryModes.TELEMETRYMODEALL {
			ts.ReportEvent("telemetry_enabled", map[string]interface{}{
				"version":     ts.version,
				"environment": ts.environment,
			})
		}
	} else {
		slog.Info("Rollbar telemetry disabled", "mode", mode.String(), "internet", ts.IsConnectedToInternet())
	}

	return nil
}

// ReportError reports an error to the telemetry service
func (ts *TelemetryService) ReportError(err error) error {
	if !ts.rollbarConfigured {
		return nil // Silently ignore if not configured
	}

	if ts.mode == dto.TelemetryModes.TELEMETRYMODEDISABLED || ts.mode == dto.TelemetryModes.TELEMETRYMODEASK {
		return nil // Don't report if disabled or asking
	}

	// Report errors for both All and Errors modes
	if ts.mode == dto.TelemetryModes.TELEMETRYMODEALL || ts.mode == dto.TelemetryModes.TELEMETRYMODEERRORS {
		rollbar.Error(err)
		slog.Debug("Error reported to Rollbar", "error", err.Error())
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
}
