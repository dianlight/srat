# smartmontools-go

A Go library that interfaces with smartmontools to monitor and manage storage device health using S.M.A.R.T. (Self-Monitoring, Analysis, and Reporting Technology) data.

![CI](https://github.com/dianlight/smartmontools-go/actions/workflows/ci.yml/badge.svg)
[![Coverage Status](https://codecov.io/github/dianlight/smartmontools-go/graph/badge.svg?token=1J2VP3FEZ4)](https://codecov.io/github/dianlight/smartmontools-go)
[![CodeFactor](https://www.codefactor.io/repository/github/dianlight/smartmontools-go/badge)](https://www.codefactor.io/repository/github/dianlight/smartmontools-go)
![Stable Release](https://img.shields.io/github/v/release/dianlight/smartmontools-go)
![Prerelease](https://img.shields.io/github/v/release/dianlight/smartmontools-go?include_prereleases)

## Features

- 🔍 **Device Scanning**: Automatically detect available storage devices
- 👀 **Drive Discovery**: `DiscoverDevices` probes each drive's optimal protocol and reports SMART readability
- 💻 **NAS Platform Support**: Automatic `smartctl` discovery across Synology DSM, QNAP, FreeBSD/TrueNAS, macOS, and standard Linux
- 💚 **Health Monitoring**: Check device health status using SMART data
- 🏥 **SMART Health Flags**: Full exit code bit decomposition (`ExecBits` / `HealthBits`) via `ExitCodeInfo`
- 📊 **SMART Attributes**: Read and parse detailed SMART attributes
  - 🔋 **Wear Level**: Normalized wear-level percentage for SSDs and NVMe drives via `WearLevelPercent()`
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

### LibBackend prerequisites (optional, Linux/macOS only)

[LibBackend](#libbackend-purego-ffi--linuxmacos) does **not** require `smartctl`. Instead
it loads a pre-built smartmon wrapper shared library at runtime. The library is available for:

- **macOS (Apple silicon)**: ships pre-built with the module at `backends/lib/sdk/libsmartmon_go.dylib`
- **Linux / other platforms**: build the wrapper once from the repository root:

```bash
scripts/setup-lib-backend.sh
```

The script downloads the correct `libsmartmon.a` static library from
[dianlight/smartmontools-sdk](https://github.com/dianlight/smartmontools-sdk) releases and
compiles the thin C++ wrapper in `backends/lib/csrc/` into
`backends/lib/sdk/libsmartmon_go.so`.

Point the backend at the compiled library via the `SMARTMON_LIB_PATH` environment variable or
`libbackend.WithLibraryPath(...)`. See [Library Resolution Order](#library-resolution-order) below.

## Installation

```bash
go get github.com/dianlight/smartmontools-go
```

To use **LibBackend** also import the backend sub-package in your code:

```go
import libbackend "github.com/dianlight/smartmontools-go/backends/lib"
```

To use **CompareBackend**:

```go
import comparebackend "github.com/dianlight/smartmontools-go/backends/compare"
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

> **NAS / embedded platforms**: `NewClient` automatically searches 11 platform-specific
> locations (Synology DSM, QNAP Entware/QPKG, FreeBSD/TrueNAS, macOS Homebrew, NixOS, …)
> when `smartctl` is not found in `PATH`. `WithSmartctlPath` always takes precedence.

### Drive Discovery

`DiscoverDevices` scans all available drives, probes each with its auto-detected
protocol, and transparently attempts an SAT fallback for drives that cannot be read
with their native protocol (common with USB-to-SATA bridges).

```go
results, err := client.DiscoverDevices(context.Background())
if err != nil {
    log.Fatalf("Discovery failed: %v", err)
}

for _, r := range results {
    fmt.Printf("Device:   %s\n", r.DevicePath)
    fmt.Printf("Protocol: %s\n", r.DetectedProtocol)
    fmt.Printf("Readable: %v\n", r.SMARTReadable)
    if r.SATFallbackRequired {
        fmt.Println("  (SAT fallback was required)")
    }
    if r.SMARTReadable {
        fmt.Printf("  Model:  %s\n", r.Model)
        fmt.Printf("  Serial: %s\n", r.Serial)
    }
}
```

### Exit Code Information

When `smartctl` exits with a non-zero status, `SMARTInfo.ExitCodeInfo` is populated
with a breakdown of the exit bits:

```go
info, err := client.GetSMARTInfo(ctx, "/dev/sda")
if err != nil {
    log.Fatalf("Failed to get SMART info: %v", err)
}

if info.ExitCodeInfo != nil {
    // Bits 0–2: execution failures (device open failed, SMART command failed, …)
    if info.ExitCodeInfo.ExecBits != 0 {
        fmt.Printf("Execution failure bits: 0x%02x\n", info.ExitCodeInfo.ExecBits)
    }
    // Bits 3–7: SMART health flags (disk failing, pre-failure attributes, …)
    if info.ExitCodeInfo.HealthBits != 0 {
        hb := info.ExitCodeInfo.HealthBits
        fmt.Printf("SMART health bits: 0x%02x\n", hb)
        fmt.Printf("  Disk failing:        %v\n", hb&0x08 != 0)
        fmt.Printf("  Pre-failure attrs:   %v\n", hb&0x10 != 0)
        fmt.Printf("  Past prefail:        %v\n", hb&0x20 != 0)
        fmt.Printf("  Error log entries:   %v\n", hb&0x40 != 0)
        fmt.Printf("  Self-test failures:  %v\n", hb&0x80 != 0)
    }
}
```

### Wear Level

`SMARTInfo.WearLevelPercent()` returns a normalized 0–100 value representing the
percentage of drive life *used* (0 = new, 100 = worn out), or `nil` for HDDs and
drives where no wear data is available:

```go
info, err := client.GetSMARTInfo(ctx, "/dev/sda")
if err != nil {
    log.Fatalf("Failed to get SMART info: %v", err)
}

if wear := info.WearLevelPercent(); wear != nil {
    fmt.Printf("Wear level: %d%%\n", *wear)
} else {
    fmt.Println("Wear level: N/A (HDD or unsupported drive)")
}
```

The source used depends on the drive type:

| Drive type | Source                                                                     |
| ---------- | -------------------------------------------------------------------------- |
| NVMe       | `nvme_smart_health_information_log.percentage_used`                        |
| SSD (ATA)  | Attr 231 (SSD Life Left) → 177 (Wear Leveling Count) → 173 (SSD Life Used) |
| HDD        | `nil`                                                                      |

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

### LibBackend (Linux/macOS — no subprocess)

`LibBackend` loads `libsmartmon_go.so` (Linux) or `libsmartmon_go.dylib` (macOS) at runtime
via [purego](https://github.com/ebitengine/purego) — **no CGO, no child process spawned**.
It exposes the same `Client` API as `ExecBackend`.

Build the wrapper library first (see [LibBackend prerequisites](#libbackend-prerequisites-optional-linuxmacos-only)), then wire it in with `WithBackend`:

```go
import (
    smartmontools "github.com/dianlight/smartmontools-go"
    libbackend "github.com/dianlight/smartmontools-go/backends/lib"
)

lib, err := libbackend.New(
    libbackend.WithLibraryPath("backends/lib/sdk/libsmartmon_go.dylib"), // or .so on Linux
)
if err != nil {
    log.Fatalf("Failed to load lib backend: %v", err)
}
defer lib.Close()

client, err := smartmontools.NewClient(smartmontools.WithBackend(lib))
```

#### Library Resolution Order

When `WithLibraryPath` is not set, `libbackend.New()` resolves the library in this order:

1. `SMARTMON_LIB_PATH` environment variable (falls back to step 3 if the file is missing)
2. Dynamic linker (`LD_LIBRARY_PATH` / `DYLD_LIBRARY_PATH` / rpath)
3. Well-known absolute paths: `/usr/local/lib`, `/opt/homebrew/lib`, …

```bash
# macOS — using the pre-built library shipped with the module
SMARTMON_LIB_PATH=backends/lib/sdk/libsmartmon_go.dylib go run .

# Linux — after running scripts/setup-lib-backend.sh
SMARTMON_LIB_PATH=backends/lib/sdk/libsmartmon_go.so go run .
```

### CompareBackend

`CompareBackend` runs two (or more) backends **in parallel** for every request. The first
backend is the master — its result is always returned to the caller. Additional backends run in
shadow mode; any discrepancy or error they produce is written to the logger.

Intended use: validate a new backend implementation against the battle-tested `ExecBackend`
before switching.

```go
import (
    smartmontools "github.com/dianlight/smartmontools-go"
    comparebackend "github.com/dianlight/smartmontools-go/backends/compare"
    execbackend "github.com/dianlight/smartmontools-go/backends/exec"
    libbackend "github.com/dianlight/smartmontools-go/backends/lib"
)

exec, err := execbackend.New()
if err != nil {
    log.Fatalf("Failed to create exec backend: %v", err)
}

lib, err := libbackend.New() // requires SMARTMON_LIB_PATH or system library
if err != nil {
    log.Fatalf("Failed to load lib backend: %v", err)
}
defer lib.Close()

compare, err := comparebackend.NewCompareBackend(
    []smartmontools.Backend{exec, lib}, // exec = master, lib = shadow
)
if err != nil {
    log.Fatalf("Failed to create compare backend: %v", err)
}
defer compare.Close()

client, err := smartmontools.NewClient(smartmontools.WithBackend(compare))
// All calls go to exec (master). lib runs in parallel silently.
// Discrepancies → "compare: result mismatch" warning in the logger.
```

> **Note**: Do not pair two identical `ExecBackend` instances. Both would hit the same
> physical device simultaneously, causing device contention and spurious errors.

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

### Efficient SMART Monitoring (Avoiding Periodic Disk Access)

When building monitoring applications that periodically check SMART status, it's important to avoid unnecessary disk I/O that can wake disks from standby mode. This is especially important for:

- Home NAS systems with idle disk spindown
- Battery-powered devices
- Systems with multiple drives where periodic access causes audible noise

**❌ Inefficient approach (wakes disk every check):**

```go
// DON'T: This queries the disk on every call
ticker := time.NewTicker(10 * time.Second)
for range ticker.C {
    // Error handling omitted for brevity in this anti-pattern example
    support, _ := client.IsSMARTSupported(ctx, "/dev/sda") // Disk access!
    if support.Enabled {
        // ... monitor SMART data
    }
}
```

**✅ Efficient approach (cache and event-driven):**

```go
// DO: Query once, cache the result, update on events
type DiskMonitor struct {
    client     smartmontools.SmartClient
    smartCache map[string]*smartmontools.SMARTInfo
    cacheMutex sync.RWMutex
}

// Initial population or refresh after enable/disable
func (m *DiskMonitor) refreshSMARTInfo(ctx context.Context, devicePath string) error {
    info, err := m.client.GetSMARTInfo(ctx, devicePath)
    if err != nil {
        return err
    }
    
    m.cacheMutex.Lock()
    m.smartCache[devicePath] = info
    m.cacheMutex.Unlock()
    
    return nil
}

// Check SMART status without disk I/O
func (m *DiskMonitor) isSMARTEnabled(devicePath string) bool {
    m.cacheMutex.RLock()
    info, exists := m.smartCache[devicePath]
    m.cacheMutex.RUnlock()
    
    if !exists {
        return false
    }
    
    // Extract status from cached info - no disk access!
    support := m.client.GetSMARTSupportFromInfo(info)
    return support.Available && support.Enabled
}

// Periodic monitoring loop
func (m *DiskMonitor) monitorLoop(ctx context.Context, devicePath string) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // Check from cache - no disk I/O
            if !m.isSMARTEnabled(devicePath) {
                continue // Skip monitoring when SMART is disabled
            }
            
            // Only query disk when SMART is enabled
            info, err := m.client.GetSMARTInfo(ctx, devicePath)
            if err != nil {
                log.Printf("Error getting SMART info: %v", err)
                continue
            }
            
            // Update cache
            m.cacheMutex.Lock()
            m.smartCache[devicePath] = info
            m.cacheMutex.Unlock()
            
            // Process SMART data...
            
        case <-ctx.Done():
            return
        }
    }
}

// Call after enabling SMART to refresh cache
func (m *DiskMonitor) enableSMART(ctx context.Context, devicePath string) error {
    if err := m.client.EnableSMART(ctx, devicePath); err != nil {
        return err
    }
    // Refresh cache immediately after enable
    return m.refreshSMARTInfo(ctx, devicePath)
}

// Call after disabling SMART to refresh cache
func (m *DiskMonitor) disableSMART(ctx context.Context, devicePath string) error {
    if err := m.client.DisableSMART(ctx, devicePath); err != nil {
        return err
    }
    // Refresh cache immediately after disable
    return m.refreshSMARTInfo(ctx, devicePath)
}
```

**Key principles:**
1. Call `GetSMARTInfo` once at startup or when SMART enable/disable state changes
2. Cache the `SMARTInfo` result in your application
3. Use `GetSMARTSupportFromInfo` to check SMART status from the cache (no disk I/O)
4. Only query the disk when SMART is known to be enabled
5. Refresh the cache after calling `EnableSMART()` or `DisableSMART()`

This approach eliminates unnecessary disk access and prevents waking disks from standby mode, resolving issues like [dianlight/hassio-addons#596](https://github.com/dianlight/hassio-addons/issues/596).

## API Reference


See [APIDOC.md](APIDOC.md) for detailed API documentation.

## Examples

See the [examples](./examples) directory for more detailed usage examples:

| Example | Description |
|---|---|
| [basic](./examples/basic/main.go) | Device scanning, health checking, and SMART info retrieval with the default ExecBackend |
| [lib](./examples/lib/main.go) | LibBackend via purego — same API, no subprocess spawned (Linux/macOS) |
| [compare](./examples/compare/main.go) | CompareBackend with a snapshot secondary — safe to run without two physical backends |
| [compare-exec-lib](./examples/compare-exec-lib/main.go) | CompareBackend with ExecBackend as master and LibBackend as shadow (Linux/macOS) |

To run an example (root privileges may be required):

```bash
# ExecBackend (default)
cd examples/basic && go run .

# LibBackend (macOS arm64, pre-built library)
SMARTMON_LIB_PATH=../../backends/lib/sdk/libsmartmon_go.dylib \
  cd examples/lib && go run .

# CompareBackend: ExecBackend master + LibBackend shadow (macOS arm64)
SMARTMON_LIB_PATH=../../backends/lib/sdk/libsmartmon_go.dylib \
  cd examples/compare-exec-lib && go run .
```

## Architecture

This library provides two backend implementations and one meta-backend:

| | ExecBackend | LibBackend | CompareBackend |
|---|---|---|---|
| **Platform** | All | Linux, macOS | Any (wraps other backends) |
| **Prerequisite** | `smartctl` binary | `libsmartmon_go.{so,dylib}` | Depends on wrapped backends |
| **Child process** | ✅ (per call) | ❌ | Depends on wrapped backends |
| **CGO** | ❌ | ❌ (uses purego) | ❌ |
| **USB bridge support** | ✅ | ✅ | ✅ |
| **Default** | ✅ | — | — |

### ExecBackend (default)

Shells out to the `smartctl` binary and parses its JSON output. Maximum compatibility, zero extra dependencies beyond smartmontools itself.

### LibBackend (purego FFI — Linux/macOS)

Loads a pre-built smartmon wrapper shared library (`libsmartmon_go.so` /
`libsmartmon_go.dylib`) at runtime using [ebitengine/purego](https://github.com/ebitengine/purego) — **no CGO required**.

```go
// Automatic resolution (reads SMARTMON_LIB_PATH or searches system paths):
lib, err := lib.New()

// Explicit path:
lib, err := lib.New(lib.WithLibraryPath("/usr/local/lib/libsmartmon_go.so"))
```

### CompareBackend

Runs two or more backends in parallel and logs discrepancies between their results. The first
backend is always the master (its output is returned); secondary backends run as shadows.

```go
compare, err := comparebackend.NewCompareBackend(
    []smartmontools.Backend{exec, lib},
    comparebackend.WithTLogHandler(logger),
)
```

📚 **For a comprehensive analysis of different SMART access approaches**, see our [Architecture Decision Record (ADR-001)](./docs/architecture/ADR-001-smart-access-approaches.md), which covers:
- Command wrapper (current default approach)
- Direct ioctl access
- Shared library with FFI (implemented as LibBackend)
- Hybrid approaches

## Implementation details

- **ExecBackend**: locates (or is given) a `smartctl` binary and executes it (`os/exec`). Commands use `--json` where available and the library parses the resulting JSON output.
- **LibBackend**: `dlopen`s `libsmartmon_go.so`/`.dylib` via purego and calls functions exported by the C++ wrapper. No child process is spawned.
- Configurable path: you can pass a custom path with `NewClientWithPath(path string)` if `smartctl` is not on PATH or you want to use a specific binary (ExecBackend only).
- Permissions: many SMART operations require root/administrator privileges or appropriate device access. Expect `permission denied` errors when running without sufficient rights.
- Error handling: the library returns errors when `smartctl` exits non-zero (ExecBackend), when JSON parsing fails, or when required fields are missing. Consumers should inspect errors and possibly the wrapped `*exec.ExitError` for diagnostics.

Example command run by ExecBackend (illustrative):

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
- Add integration tests for LibBackend against real devices in CI.

Mid-term:
- Implement ioctl-based device access for platforms where direct calls are preferable and safe.
- Provide clearer compatibility matrix and CI jobs for Linux/macOS/Windows.
- Publish pre-built `libsmartmon_go` binaries as release assets for common platforms.

Long-term:
- Optimize performance and reduce process creation overhead for large-scale monitoring setups.
- Full native Go backend (v1.0) with zero runtime dependencies.

How to help:
- Add tests that include representative `smartctl --json` outputs (captured from different smartmontools versions/devices).
- Document platform-specific permission and packaging notes (e.g., macOS notarization, Windows admin requirements).
- Test LibBackend on additional Linux distributions and architectures.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [smartmontools](https://www.smartmontools.org/) — the underlying tool that makes this library possible
- [DAB-LABS/smart-sniffer](https://github.com/DAB-LABS/smart-sniffer) — several reliability improvements in this library (multi-path binary resolution, SAT fallback, `--scan-open` → `--scan` fallback, `DiscoverDevices`, and exit code bit decomposition) were inspired by the patterns used in the smart-sniffer agent

## CI and mise

This repository uses [`mise`](https://mise.jdx.dev/) for task running and a GitHub Actions workflow that runs CI on `push` and `pull_request` to `main`.

Common tasks:

| Command                                  | Description                                                                                      |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `mise run test`                          | Run unit tests for all packages                                                                  |
| `mise run ci`                            | Run all CI checks (tidy, download, lint, test)                                                   |
| `mise run lint`                          | Run staticcheck                                                                                  |
| `mise run fmt`                           | Run gofmt on the project                                                                         |
| `mise run coverage`                      | Run tests and show coverage summary                                                              |
| `mise run apidoc`                        | Generate API documentation (`APIDOC.md`)                                                         |
| `mise run clean`                         | Remove build artifacts                                                                           |
| `mise run release [major\|minor\|patch]` | Create and push a tag — stable on `main`, prerelease on a branch with an open PR (requires `gh`) |

`release` accepts a `--dry-run` flag to preview without pushing:

```sh
mise run release         # patch bump (default)
mise run release minor --dry-run
```

To list all available tasks run `mise tasks`.
