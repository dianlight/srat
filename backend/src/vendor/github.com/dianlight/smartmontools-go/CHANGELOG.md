# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.1] — 2025-05-28
- **Hotfix release**: Moved `internal/types` to `types` for public access. This was an oversight in the v0.4.0 refactor and the `internal` package prevented access to key types like `SMARTInfo` and `DiscoveryResult`. The v0.4.0 release is unaffected and remains available for users who don't need the new features.

## [0.4.0] — 2025-05-21

### Breaking Changes
- `ExecBackend`, `ExecBackendOption`, `NewExecBackend`, and related `WithExec*` options are now implemented by the `backends/exec` package. The root package keeps backward-compatible aliases and wrappers.
- `Commander.Command()` now accepts the exported `LogAdapter` type, making the interface implementable outside this module.

### Added
- `backends/exec/` package containing the `ExecBackend` implementation
- `internal/types/` shared type hub for domain types, interfaces, constants, and helpers
- `ExecBackend.SmartctlPath()` accessor
- `ExecBackend.SetDeviceTypeHint(path, deviceType string)` cache seeding helper
- `ExecBackend.DeviceTypeHint(path string) (string, bool)` cache inspection helper
- `backends/exec.WithLogHandler(logger LogAdapter) Option`
- Exported `LogAdapter` type in the root package
- `backends/exec/drivedb_version.go`: generated file exposing `DrivedbUpstreamCommit` and `DrivedbUpstreamDate` constants tracking the embedded `drivedb.h` upstream provenance
- Root-package re-exports `DrivedbUpstreamCommit` and `DrivedbUpstreamDate` for easy access
- `.github/workflows/drivedb-update.yml`: daily GitHub Actions workflow that detects upstream `drivedb.h` changes and opens automated PRs
- `.github/workflows/drivedb-fetch.yml`: companion workflow that downloads `drivedb.h` when Renovate updates `drivedb_version.go` in a PR
- `.github/renovate.json` custom datasource and regex manager for Renovate-based drivedb tracking

### Changed
- The root package is now a thin facade over `internal/types` and `backends/exec`
- Exec-specific helpers and drivedb parsing moved out of the root package

##  [v0.3.1] — 2025-05-16

### Added

- **Multi-path `smartctl` resolution** (`helpers.go`): `NewClient` now searches 11
  platform-specific locations when `smartctl` is not found in `PATH` — including
  Synology DSM, SynoCommunity QPKG, QNAP Entware/QPKG, macOS Homebrew (Intel &
  Apple Silicon), MacPorts, FreeBSD/TrueNAS CORE, and NixOS. An actionable error
  message with per-platform install instructions is returned when no binary is found.
  The `WithSmartctlPath` option continues to take full precedence and bypasses the
  search entirely.

- **SAT protocol automatic fallback** (`client.go`): When `GetSMARTInfo` encounters
  execution-failure exit bits (bits 0–2) and no device type is cached, the library
  automatically retries with `-d sat`. On success the detected type is written to
  the device type cache so subsequent calls skip the re-probe. This transparently
  handles many USB-to-SATA bridges, Synology `/dev/sata*` paths, and RAID passthrough
  devices where auto-detection fails.

- **Full exit code bit decomposition** (`types.go`, `client.go`): `SMARTInfo` now
  carries an `ExitCodeInfo *ExitCodeInfo` field that is populated whenever the
  `smartctl` exit status is non-zero. The struct exposes two fields:
  - `ExecBits int` — bits 0–2 (mask `0x07`): execution failures (device open
    failed, command parse error, SMART command failed).
  - `HealthBits int` — bits 3–7 (mask `0xF8`): SMART health flags (disk failing,
    pre-failure attributes, past-threshold prefail, error log, self-test log).

  This lets consumers programmatically distinguish "device could not be queried"
  from "device is reporting degraded health" without parsing error strings.

- **Per-device health-bit deduplication logging** (`client.go`): An internal
  `healthBitsCache` (keyed by device path) records the last-seen `HealthBits` value
  per device. A `WARN` log line with per-bit structured fields (`diskFailing`,
  `prefailAttr`, `pastPrefail`, `errorLog`, `selfTestLog`) is emitted only when the
  health-bit pattern changes, preventing log flooding for drives in a stable degraded
  state.

- **`--scan-open` → `--scan` fallback in `ScanDevices`** (`client.go`): `ScanDevices`
  now automatically falls back to plain `--scan --json` when `--scan-open --json`
  fails. This ensures device enumeration works in container sandboxes, on older
  kernels, and in environments where the caller lacks the privileges required by
  `--scan-open`.

- **`DiscoverDevices` method** (`client.go`, `types.go`): New
  `DiscoverDevices(ctx context.Context) ([]DiscoveryResult, error)` method added to
  the `SmartClient` interface. It scans all drives, probes each with its
  auto-detected protocol, and automatically attempts an explicit SAT fallback per
  drive when the initial read fails. Each `DiscoveryResult` carries:
  - `DevicePath string` — kernel device path (e.g. `/dev/sda`)
  - `DetectedProtocol string` — protocol used for a successful read (`ata`, `sat`, …)
  - `SMARTReadable bool` — whether SMART data could be read at all
  - `SATFallbackRequired bool` — whether the SAT fallback was needed
  - `Model string` — model name or model family (whichever is available)
  - `Serial string` — serial number

  Useful for diagnosing protocol-detection issues and generating device override
  configurations without writing application code.

- **`WearLevelPercent` method on `SMARTInfo`** (`types.go`): New
  `WearLevelPercent() *int` method returns the percentage of drive life *used*
  (0 = new, 100 = worn out) for SSDs and NVMe drives, or `nil` for HDDs and
  unknown types. Sources by drive type:
  - NVMe: `nvme_smart_health_information_log.percentage_used`
  - ATA SSD: SMART attributes in priority order — 231 (SSD Life Left),
    177 (Wear Leveling Count), 173 (SSD Life Used)
  - HDD / Unknown: `nil`

  The returned value is always clamped to [0, 100]. New package constants
  `SmartAttrSSDLifeUsed = 173` and `SmartAttrWearLevelingCount = 177` are
  exported alongside the existing SSD detection constants.

### Changed

- `ScanDevices` now logs a `WARN` and retries with `--scan` instead of returning an
  error immediately when `--scan-open` is unavailable or fails.
- `NewClient` uses the new `resolveSmartctlPath` helper (multi-path search) instead
  of `exec.LookPath` when no explicit path is provided.

---

## [v0.2.7] — 2025-04-16

See [GitHub release](https://github.com/dianlight/smartmontools-go/releases/tag/v0.2.7).

## [v0.2.6] — 2025-04-07

See [GitHub release](https://github.com/dianlight/smartmontools-go/releases/tag/v0.2.6).

## [v0.2.5] — 2025-03-28

See [GitHub release](https://github.com/dianlight/smartmontools-go/releases/tag/v0.2.5).

## [v0.2.4] — 2025-03-21

See [GitHub release](https://github.com/dianlight/smartmontools-go/releases/tag/v0.2.4).

## [v0.2.3] — 2025-03-14

See [GitHub release](https://github.com/dianlight/smartmontools-go/releases/tag/v0.2.3).
