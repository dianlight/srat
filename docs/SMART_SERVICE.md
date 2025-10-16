<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [SMART Service Enhancement](#smart-service-enhancement)
  - [Overview](#overview)
  - [API Methods](#api-methods)
    - [GetHealthStatus(devicePath string)](#gethealthstatusdevicepath-string)
    - [StartSelfTest(devicePath string, testType SmartTestType)](#startselftestdevicepath-string-testtype-smarttesttype)
    - [AbortSelfTest(devicePath string)](#abortselftestdevicepath-string)
    - [GetTestStatus(devicePath string)](#getteststatusdevicepath-string)
    - [EnableSMART(devicePath string) / DisableSMART(devicePath string)](#enablesmartdevicepath-string--disablesmartdevicepath-string)
  - [Pre-Failure Alert System](#pre-failure-alert-system)
    - [How It Works](#how-it-works)
    - [Registering a Callback](#registering-a-callback)
    - [Example Integration](#example-integration)
  - [Platform Support](#platform-support)
    - [Linux](#linux)
    - [Other Platforms](#other-platforms)
  - [Error Handling](#error-handling)
  - [Limitations](#limitations)
  - [Testing](#testing)
  - [Future Enhancements](#future-enhancements)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# SMART Service Enhancement

This document describes the enhanced SMART service functionality for monitoring and controlling disk S.M.A.R.T. attributes.

## Overview

The SMART service has been extended with comprehensive disk health monitoring and control capabilities:

- **Health Status Monitoring**: Evaluate disk health by comparing SMART attributes against thresholds
- **Self-Test Execution**: Initiate, abort, and monitor SMART self-tests (short, long, conveyance)
- **SMART Control**: Enable or disable SMART functionality on devices
- **Pre-Failure Alerts**: Automatic warnings when disk attributes indicate potential failure

**Implementation Note**: This service uses a **patched version** of `github.com/anatol/smart.go` that exposes file descriptors for direct device control. The patch is managed via `gohack` and applied automatically during the build process (`make patch`). See [Platform Support](#platform-support) and [Limitations](#limitations) sections for details.

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

### AbortSelfTest(devicePath string)

Aborts a currently running SMART self-test on a SATA device.

**Example:**

```go
err := smartService.AbortSelfTest("/dev/sda")
if err != nil {
    return err
}
log.Info("SMART self-test aborted")
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

- **Health Status Monitoring**: ✅ Fully supported
- **Self-Test Execution**: ✅ Fully supported (start, abort, status monitoring)
- **SMART Control**: ✅ Fully supported (enable/disable SMART functionality)
- **Implementation**: Uses ioctl commands for direct device control via patched `github.com/anatol/smart.go` library
- **Requirements**: Appropriate permissions (typically root or disk group)

**Library Patching:**

The SMART service uses a patched version of `github.com/anatol/smart.go` to expose file descriptors needed for direct device control. The patches are managed via `gohack` and applied automatically during the build process:

```bash
cd backend
make patch  # Applies all library patches including smart.go
```

Multiple patches are applied to smart.go in alphabetical order:
1. `smart.go-#010.patch` - Fix for SATA power_hours parsing (from wuxingzhong/smart.go)
2. `smart.go-srat#999.patch` - Adds `FileDescriptor()` method to Device interface

These patches enable:
- Correct power-on hours calculation using duration parsing
- Direct ioctl access for operations like enabling/disabling SMART and starting/aborting self-tests
- Advanced device control

**Patch Details:**
- Location: `backend/patches/smart.go-*.patch`
- Applied libraries: `github.com/anatol/smart.go`, `github.com/zarldev/goenums`, `github.com/jpillora/overseer`
- Tool: `gohack` (manages local library modifications)
- Automatic: Patches are applied during `make all` or `make patch`

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

### Library Patching Requirement

The SMART service implementation relies on a **patched version** of the `github.com/anatol/smart.go` library. The original library provides excellent read-only SMART access but does not expose file descriptors needed for direct device control operations.

**What the patch provides:**
- `FileDescriptor()` method added to the `Device` interface
- Implementation in all device types (SataDevice, NVMeDevice, ScsiDevice)
- Direct access to underlying file descriptors for ioctl operations

**How to use:**
```bash
cd backend
make patch  # Downloads libraries via gohack and applies patches
make build  # Build with patched libraries
```

The patch is automatically applied when running `make all` and is required for the following operations:
- EnableSMART/DisableSMART
- StartSelfTest/AbortSelfTest
- Any operation requiring direct ioctl access

### Linux

All SMART operations are fully implemented and functional for Linux platforms using ioctl commands with direct device access via the patched library.

### Other Platforms

The current implementation provides full API structure and health monitoring capabilities for all platforms. However, direct device control operations (enable/disable SMART, execute/abort tests) are limited on non-Linux platforms and return appropriate error messages indicating platform limitations.

The underlying `github.com/anatol/smart.go` library provides cross-platform support for basic SMART attribute reading. The patch extends this with file descriptor access, but the ioctl implementations remain platform-specific.

**Platform-specific limitations:**

- **macOS/Windows**: Direct device control operations not implemented (returns `ErrorSMARTNotSupported`)
- **BSD variants**: May have limited support depending on specific implementation
- **Embedded systems**: Depends on kernel ioctl support

The health checking and test log reading functionality is fully operational across all supported platforms using the existing library features.

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

1. **NVMe Support**: Extend self-test capabilities to NVMe devices
2. **Real-time Monitoring**: WebSocket/SSE streaming of SMART attribute changes
3. **Historical Tracking**: Store SMART data over time for trend analysis
4. **Predictive Failure**: Machine learning-based failure prediction using historical data
5. **Email Notifications**: Built-in email alerts for failing disks
