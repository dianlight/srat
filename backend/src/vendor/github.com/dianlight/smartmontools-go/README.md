> ⚠️ IMPORTANT — Very early development

> This project is in very early development. The current implementation executes the external `smartctl` binary (via exec) and parses its output. It does NOT currently provide native Go bindings, direct ioctl integration, or libgoffi-based integration. Those integrations are planned for future releases. Use this library for experimentation only.

# smartmontools-go

A Go library that interfaces with smartmontools to monitor and manage storage device health using S.M.A.R.T. (Self-Monitoring, Analysis, and Reporting Technology) data.

![CI](https://github.com/dianlight/smartmontools-go/actions/workflows/ci.yml/badge.svg)
[![Coverage Status](https://codecov.io/github/dianlight/smartmontools-go/graph/badge.svg?token=1J2VP3FEZ4)](https://codecov.io/github/dianlight/smartmontools-go)
![Stable Release](https://img.shields.io/github/v/release/dianlight/smartmontools-go)
![Prerelease](https://img.shields.io/github/v/release/dianlight/smartmontools-go?include_prereleases)

## Features

- 🔍 **Device Scanning**: Automatically detect available storage devices
- 💚 **Health Monitoring**: Check device health status using SMART data
- 📊 **SMART Attributes**: Read and parse detailed SMART attributes
- 🌡️ **Temperature Monitoring**: Track device temperature
- ⚙️ **Self-Tests**: Initiate and monitor SMART self-tests
- 🔧 **Device Information**: Retrieve model, serial number, firmware version, and more
- 🔌 **USB Bridge Support**: Automatic fallback for unknown USB bridges with embedded device database

## Prerequisites

This library requires `smartctl` (part of smartmontools) to be installed on your system.

Minimum supported version: smartctl 7.0 (for JSON `-j` output).

### Linux
```bash
# Debian/Ubuntu
sudo apt-get install smartmontools

# RHEL/CentOS/Fedora
sudo yum install smartmontools

# Arch Linux
sudo pacman -S smartmontools
```

### macOS
```bash
brew install smartmontools
```

### Windows
Download and install from [smartmontools.org](https://www.smartmontools.org/)

## Installation

```bash
go get github.com/dianlight/smartmontools-go
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/dianlight/smartmontools-go"
)

func main() {
    // Create a new client
    client, err := smartmontools.NewClient()
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // Scan for devices
    devices, err := client.ScanDevices()
    if err != nil {
        log.Fatalf("Failed to scan devices: %v", err)
    }

    for _, device := range devices {
        fmt.Printf("Device: %s (type: %s)\n", device.Name, device.Type)
        
        // Check health
        healthy, err := client.CheckHealth(device.Name)
        if err != nil {
            log.Printf("Failed to check health: %v", err)
            continue
        }
        
        if healthy {
            fmt.Println("  Health: PASSED ✓")
        } else {
            fmt.Println("  Health: FAILED ✗")
        }
    }
}
```

### Getting SMART Information

```go
// Get detailed SMART information
smartInfo, err := client.GetSMARTInfo("/dev/sda")
if err != nil {
    log.Fatalf("Failed to get SMART info: %v", err)
}

fmt.Printf("Model: %s\n", smartInfo.ModelName)
fmt.Printf("Serial: %s\n", smartInfo.SerialNumber)
fmt.Printf("Temperature: %d°C\n", smartInfo.Temperature.Current)
fmt.Printf("Power On Hours: %d\n", smartInfo.PowerOnTime.Hours)

// Access SMART attributes
if smartInfo.AtaSmartData != nil {
    for _, attr := range smartInfo.AtaSmartData.Table {
        fmt.Printf("Attribute %d (%s): %d\n", attr.ID, attr.Name, attr.Value)
    }
}
```

### Running Self-Tests

```go
// Run a short self-test
err := client.RunSelfTest("/dev/sda", "short")
if err != nil {
    log.Fatalf("Failed to run self-test: %v", err)
}

// Available test types: "short", "long", "conveyance", "offline"
```

### Custom smartctl Path

```go
// If smartctl is not in PATH or you want to use a specific binary
client, err := smartmontools.NewClient(smartmontools.WithSmartctlPath("/usr/local/sbin/smartctl"))
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
```

### Logging

The library uses the [`tlog`](https://github.com/dianlight/tlog) package for structured logging.

Default behavior:

* When you call `smartmontools.NewClient()` without a `WithLogHandler` option, the client creates a debug-level `*tlog.Logger` (via `tlog.NewLoggerWithLevel(tlog.LevelDebug)`) so that diagnostic output (command execution, fallbacks, warnings) is available.
* You can adjust the global log level at runtime using `tlog.SetLevelFromString("info")` or `tlog.SetLevel(tlog.LevelInfo)`. Levels include: `trace`, `debug`, `info`, `notice`, `warn`, `error`, `fatal`.
* All internal logging is key/value structured. Expensive debug operations are guarded; if you perform your own heavy debug logging, first check with `tlog.IsLevelEnabled(tlog.LevelDebug)`.

Override the logger for a specific client instance:

```go
import (
    "context"
    "log"
    "github.com/dianlight/smartmontools-go"
    "github.com/dianlight/tlog"
)

func main() {
    // Create a custom logger which only logs WARN and above.
    customLogger := tlog.NewLoggerWithLevel(tlog.LevelWarn)

    client, err := smartmontools.NewClient(
        smartmontools.WithLogHandler(customLogger),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // Example global level change (applies to package-level functions too)
    if err := tlog.SetLevelFromString("info"); err != nil {
        tlog.Warn("Failed to set log level", "error", err)
    }

    devices, err := client.ScanDevices(context.Background())
    if err != nil {
        tlog.Error("Scan failed", "error", err)
        return
    }
    tlog.Info("Scan complete", "count", len(devices))
}
```

If you need a logger instance with a different minimum level temporarily (without changing globals), use:

```go
traceLogger := tlog.WithLevel(tlog.LevelTrace) // returns *slog.Logger for ad-hoc usage
traceLogger.Log(context.Background(), tlog.LevelTrace, "Detailed trace")
```

For code interacting with the client, prefer passing a `*tlog.Logger` via `WithLogHandler`. For ad-hoc logging outside the client lifecycle, use the package-level helpers (`tlog.Info`, `tlog.DebugContext`, etc.).

Graceful shutdown of callback processor (if you registered callbacks):

```go
defer tlog.Shutdown() // Ensures queued callback events are processed before exit
```

### Custom Default Context

```go
// Set a default context that will be used when methods are called with nil context
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

client, err := smartmontools.NewClient(smartmontools.WithContext(ctx))
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}

// Now when calling with nil context, the client will use the configured default
info, err := client.GetSMARTInfo(nil, "/dev/sda") // Uses the 30s timeout context
```

### Combining Options

```go
// Combine multiple options
client, err := smartmontools.NewClient(
    smartmontools.WithSmartctlPath("/usr/local/sbin/smartctl"),
    smartmontools.WithLogHandler(logger),
    smartmontools.WithContext(ctx),
)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
```

### USB Bridge Support

The library includes automatic support for USB storage devices that use unknown USB bridges. When smartctl reports an "Unknown USB bridge" error, the library:

1. **Checks embedded database**: Looks up the USB vendor:product ID in the embedded standard `drivedb.h` from smartmontools
2. **Automatic fallback**: If found, uses the known device type; otherwise falls back to `-d sat`
3. **Caches results**: Remembers successful device types for faster future access

```go
client, _ := smartmontools.NewClient()

// Works automatically with USB bridges, even if unknown to smartctl
info, err := client.GetSMARTInfo("/dev/disk/by-id/usb-Intenso_Memory_Center-0:0")
if err != nil {
    log.Fatalf("Failed to get SMART info: %v", err)
}

fmt.Printf("Model: %s\n", info.ModelName)
fmt.Printf("Health: %v\n", info.SmartStatus.Passed)
```

The embedded database is the official smartmontools `drivedb.h` which contains USB bridge definitions from the upstream project. See [docs/drivedb.md](./docs/drivedb.md) for details.

## API Reference

See [APIDOC.md](APIDOC.md) for detailed API documentation.

## Examples

See the [examples](./examples) directory for more detailed usage examples:

- [Basic Usage](./examples/basic/main.go) - Demonstrates device scanning, health checking, and SMART info retrieval

To run the basic example:

```bash
cd examples/basic
go run main.go
```

**Note**: Some operations require root/administrator privileges to access disk devices.

## Architecture

This library uses a command-line wrapper approach, executing `smartctl` commands and parsing their JSON output. The library leverages smartmontools' built-in JSON output format for reliable and structured data extraction.

While the project references libgoffi in its description, the current implementation uses the command-line interface for maximum compatibility and reliability. Future versions may incorporate direct library bindings using libgoffi for enhanced performance.

📚 **For a comprehensive analysis of different SMART access approaches**, see our [Architecture Decision Record (ADR-001)](./docs/architecture/ADR-001-smart-access-approaches.md), which covers:
- Command wrapper (current approach)
- Direct ioctl access
- Shared library with FFI
- Hybrid approaches

The ADR includes detailed comparisons, code examples, performance benchmarks, and recommendations for different use cases.

## Implementation details

- Execution model: the library locates (or is given) a `smartctl` binary and executes it (os/exec). Commands use `--json` where available and the library parses the resulting JSON output.
- Configurable path: you can pass a custom path with `NewClientWithPath(path string)` if `smartctl` is not on PATH or you want to use a specific binary.
- Permissions: many SMART operations require root/administrator privileges or appropriate device access. Expect `permission denied` errors when running without sufficient rights.
- Error handling: the library returns errors when `smartctl` exits non-zero, when JSON parsing fails, or when required fields are missing. Consumers should inspect errors and possibly the wrapped `*exec.ExitError` for diagnostics.
- Limitations: because this approach shells out to an external binary, it has higher process overhead and depends on the installed smartmontools version and platform support. It does not (yet) provide direct ioctl access or in-process bindings.

Example command run by the library (illustrative):

```text
smartctl --json -a /dev/sda
```

This will be parsed into the library's `SMARTInfo` structures.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Roadmap

Short-term (current):
- Stabilize the exec-based API surface (ScanDevices, GetSMARTInfo, CheckHealth, RunSelfTest).
- Improve error messages and diagnostics when `smartctl` is missing, incompatible, or returns non-JSON output.
- Add more unit tests that mock `smartctl` JSON output.

Mid-term:
- Add optional libgoffi-based bindings to call smartmontools in-process where supported.
- Implement ioctl-based device access for platforms where direct calls are preferable and safe.
- Provide clearer compatibility matrix and CI jobs for Linux/macOS/Windows.

Long-term:
- Offer a native Go implementation/path that does not require an external `smartctl` binary for common operations.
- Optimize performance and reduce process creation overhead for large-scale monitoring setups.

How to help:
- If you'd like to work on native bindings, start by opening an issue describing the platform and approach (libgoffi vs ioctl-first).
- Add tests that include representative `smartctl --json` outputs (captured from different smartmontools versions/devices).
- Document platform-specific permission and packaging notes (e.g., macOS notarization, Windows admin requirements).

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [smartmontools](https://www.smartmontools.org/) - The underlying tool that makes this library possible
- [libgoffi](https://github.com/noctarius/libgoffi) - FFI adapter library for Go (for future enhancements)

## CI and Makefile

This repository includes a `Makefile` with common targets and a GitHub Actions workflow that runs CI on `push` and `pull_request` to `main`.

Quick Makefile usage:

- Run tests: `make test`
- Run full CI locally (formats, vet, staticcheck, tests): `make ci`
- Format code: `make fmt`
- Build binary: `make build`

Staticcheck will be installed into your Go bin (GOBIN or GOPATH/bin) if not already present when you run `make ci`.
