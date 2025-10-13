# SMART Service Enhancement

This document describes the enhanced SMART service functionality for monitoring and controlling disk S.M.A.R.T. attributes.

## Overview

The SMART service has been extended with comprehensive disk health monitoring and control capabilities:

- **Health Status Monitoring**: Evaluate disk health by comparing SMART attributes against thresholds
- **Self-Test Execution**: Initiate and monitor SMART self-tests (short, long, conveyance)
- **SMART Control**: Enable or disable SMART functionality on devices
- **Pre-Failure Alerts**: Automatic warnings when disk attributes indicate potential failure

## API Methods

### GetHealthStatus(devicePath string)

Evaluates the current health status of a disk by analyzing SMART attributes and comparing them against failure thresholds.

**Returns:** `SmartHealthStatus`
- `Passed`: Boolean indicating if all attributes are within acceptable limits
- `FailingAttributes`: List of attribute names below their thresholds
- `OverallStatus`: "healthy", "warning", or "failing"

**Example:**
```go
health, err := smartService.GetHealthStatus("/dev/sda")
if err != nil {
    return err
}

if !health.Passed {
    log.Warn("Disk health check failed", 
        "attributes", health.FailingAttributes,
        "status", health.OverallStatus)
}
```

### StartSelfTest(devicePath string, testType SmartTestType)

Initiates a SMART self-test on a SATA device.

**Test Types:**
- `SmartTestTypeShort`: Quick test (~2 minutes)
- `SmartTestTypeLong`: Comprehensive test (hours, varies by disk)
- `SmartTestTypeConveyance`: Transport damage test (minutes)

**Example:**
```go
err := smartService.StartSelfTest("/dev/sda", dto.SmartTestTypeShort)
if err != nil {
    return err
}
log.Info("SMART self-test started")
```

### GetTestStatus(devicePath string)

Retrieves the status of the currently running or most recently completed SMART self-test.

**Returns:** `SmartTestStatus`
- `Status`: "idle", "running", "completed", "failed"
- `TestType`: Type of test that was/is running
- `PercentComplete`: Progress indicator (0-100)
- `LBAOfFirstError`: Location of first error if test failed

**Example:**
```go
status, err := smartService.GetTestStatus("/dev/sda")
if err != nil {
    return err
}

if status.Status == "running" {
    log.Info("Test in progress", "percent", status.PercentComplete)
}
```

### EnableSMART(devicePath string) / DisableSMART(devicePath string)

Control SMART functionality on SATA devices.

**Example:**
```go
// Enable SMART monitoring
if err := smartService.EnableSMART("/dev/sda"); err != nil {
    return err
}

// Disable SMART monitoring
if err := smartService.DisableSMART("/dev/sda"); err != nil {
    return err
}
```

## Pre-Failure Alert System

The SMART service automatically integrates with the tlog callback system to provide notifications when disk health issues are detected.

### How It Works

1. When `GetHealthStatus()` detects failing attributes, it automatically logs a warning via `tlog.Warn()`
2. The tlog system can have callbacks registered at the `WARN` level to take action
3. Applications can register custom callbacks to send notifications, create alerts, etc.

### Registering a Callback

To receive notifications when SMART pre-failure conditions are detected:

```go
import (
    "log/slog"
    "github.com/dianlight/srat/tlog"
)

// Register a callback for WARN-level logs (includes SMART failures)
callbackID := tlog.RegisterCallback(slog.LevelWarn, func(event tlog.LogEvent) {
    // Check if this is a SMART-related warning
    event.Record.Attrs(func(attr slog.Attr) bool {
        if attr.Key == "device" {
            // This is a device-related warning, likely SMART
            device := attr.Value.String()
            
            // Take action: send notification, create alert, etc.
            sendDiskHealthAlert(device)
        }
        return true
    })
})

// Later, unregister the callback if needed
tlog.UnregisterCallback(slog.LevelWarn, callbackID)
```

### Example Integration

Here's a complete example of setting up SMART monitoring with alerts:

```go
package main

import (
    "log/slog"
    "time"
    
    "github.com/dianlight/srat/service"
    "github.com/dianlight/srat/dto"
    "github.com/dianlight/srat/tlog"
)

func main() {
    smartService := service.NewSmartService()
    
    // Register callback for SMART pre-failure alerts
    tlog.RegisterCallback(slog.LevelWarn, func(event tlog.LogEvent) {
        var device string
        var failingAttrs []string
        
        event.Record.Attrs(func(attr slog.Attr) bool {
            switch attr.Key {
            case "device":
                device = attr.Value.String()
            case "failing_attributes":
                // Extract failing attributes
                // (implementation depends on attribute format)
            }
            return true
        })
        
        if device != "" {
            // Send alert to monitoring system
            sendAlert("Disk Health Alert", 
                "Device %s has failing SMART attributes: %v", 
                device, failingAttrs)
        }
    })
    
    // Periodically check disk health
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    devices := []string{"/dev/sda", "/dev/sdb"}
    
    for range ticker.C {
        for _, device := range devices {
            health, err := smartService.GetHealthStatus(device)
            if err != nil {
                slog.Error("Failed to check SMART health", 
                    "device", device, "error", err)
                continue
            }
            
            if !health.Passed {
                // Warning will be logged automatically by GetHealthStatus
                // and our callback will be triggered
                slog.Info("Detected failing disk", 
                    "device", device,
                    "status", health.OverallStatus)
            }
        }
    }
}
```

## Platform Support

### Linux
- Full support for all operations
- Uses ioctl commands for direct device control
- Requires appropriate permissions (typically root or disk group)

### Other Platforms
- Health status monitoring: ✅ Supported
- Self-test execution: ⚠️ Limited (returns error indicating platform limitation)
- SMART enable/disable: ⚠️ Limited (returns error indicating platform limitation)

## Error Handling

The service uses specific error types for different failure scenarios:

- `ErrorSMARTNotSupported`: Device doesn't support SMART or operation not available
- `ErrorSMARTOperationFailed`: SMART operation failed at device level
- `ErrorSMARTTestInProgress`: Cannot start new test while one is running
- `ErrorInvalidParameter`: Invalid test type or parameter provided

## Limitations

The current implementation provides the full API structure and health monitoring capabilities. Direct device control operations (enable/disable SMART, execute tests) are partially implemented with placeholders that return appropriate errors. This is because the underlying `github.com/anatol/smart.go` library doesn't expose the file descriptor needed for ioctl commands.

To fully enable these operations, one of the following approaches is needed:
1. Fork and modify `anatol/smart.go` to expose file descriptors
2. Use a different SMART library that provides control operations
3. Implement direct device opening and ioctl calls without using smart.go for control operations

The health checking and test log reading functionality is fully operational using the existing library features.

## Testing

Comprehensive tests are provided in `smart_service_test.go`:

```bash
cd backend
make test
```

Tests cover:
- Health status evaluation
- Test type validation
- Device error handling
- Cache behavior
- Platform-specific behavior

## Future Enhancements

Potential improvements for future versions:

1. **Complete ioctl Implementation**: Full implementation of enable/disable/test execution with proper file descriptor handling
2. **NVMe Support**: Extend self-test capabilities to NVMe devices
3. **Real-time Monitoring**: WebSocket/SSE streaming of SMART attribute changes
4. **Historical Tracking**: Store SMART data over time for trend analysis
5. **Predictive Failure**: Machine learning-based failure prediction using historical data
6. **Email Notifications**: Built-in email alerts for failing disks
