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

## Prerequisites

This library requires `smartctl` (part of smartmontools) to be installed on your system:

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
client := smartmontools.NewClientWithPath("/usr/local/sbin/smartctl")
```

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
