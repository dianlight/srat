package service_test

import (
	"log/slog"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/tlog"
	"github.com/stretchr/testify/assert"
)

// Example_getHealthStatus demonstrates basic usage of SMART health monitoring
func Example_getHealthStatus() {
	smartService := service.NewSmartService()

	// Get health status of a device
	health, err := smartService.GetHealthStatus("/dev/sda")
	if err != nil {
		// Handle error - device may not exist or not support SMART
		return
	}

	if !health.Passed {
		// Disk has failing attributes
		slog.Warn("Disk health check failed",
			"status", health.OverallStatus,
			"failing_attributes", health.FailingAttributes)
	}
}

// Example_startSelfTest demonstrates initiating a SMART self-test
func Example_startSelfTest() {
	smartService := service.NewSmartService()

	// Start a short self-test
	err := smartService.StartSelfTest("/dev/sda", dto.SmartTestTypeShort)
	if err != nil {
		slog.Error("Failed to start SMART self-test", "error", err)
		return
	}

	slog.Info("SMART self-test started successfully")
}

// Example_withCallback demonstrates setting up a callback for pre-failure alerts
func Example_withCallback() {
	smartService := service.NewSmartService()

	// Register a callback to receive SMART pre-failure alerts
	callbackID := tlog.RegisterCallback(slog.LevelWarn, func(event tlog.LogEvent) {
		// Check if this is a SMART-related warning
		var device string
		event.Record.Attrs(func(attr slog.Attr) bool {
			if attr.Key == "device" {
				device = attr.Value.String()
			}
			return true
		})

		if device != "" {
			// Take action: send notification, create ticket, etc.
			slog.Info("SMART pre-failure callback triggered", "device", device)
		}
	})

	// Perform health check - if failing, callback will be triggered
	_, _ = smartService.GetHealthStatus("/dev/sda")

	// Later, unregister the callback
	tlog.UnregisterCallback(slog.LevelWarn, callbackID)
}

// TestSmartServiceCallbackIntegration verifies callback integration
func TestSmartServiceCallbackIntegration(t *testing.T) {
	// This test verifies that the tlog callback registration mechanism works
	// In production, GetHealthStatus triggers callbacks when detecting failures

	callbackID := tlog.RegisterCallback(slog.LevelWarn, func(event tlog.LogEvent) {
		// Callback will be called asynchronously when WARN-level logs occur
		// In production, this would handle SMART pre-failure alerts
	})
	defer tlog.UnregisterCallback(slog.LevelWarn, callbackID)

	// Verify that callback registration succeeded
	assert.NotEmpty(t, callbackID, "Callback ID should be generated")

	// In production use, GetHealthStatus() automatically triggers this callback
	// when it detects failing SMART attributes via tlog.Warn()
}

