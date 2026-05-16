<!-- DOCTOC SKIP -->

# Changelog

## 2026.5.0-rc7

### 🙏 Thanks

We would like to thank all supporters for their contributions and donations.
With your donations, we are able to continue developing and improving this project. Your support is greatly appreciated.

> **Note**: This section tracks development progress and changes planned for the Release Candidate (RC). The final release notes will be organized and consolidated once the RC is ready for public testing.

### 🏗 Chore

- Change the frontend testing engine to vitest to be more stable and realistic.
- Add a new test on browser directly.

### 🐛 Bug Fixes

- **Reduce continuous disk access (#636)**: Optimized backend services to significantly reduce redundant disk I/O:
  - `DiskStatsService`: Heavy tick (every 60s) fetches SMART data and partition metadata via `syscall.Statfs`; lightweight ticks (5 of every 6) reuse cached data from the previous tick, eliminating `smartctl` invocations and VFS probes on every 10s poll.
  - `NetworkStatsService`: Settings are loaded from disk only on heavy ticks (every 60s); lightweight ticks reuse the in-memory cached settings.
  - `HealthHandler`: Expensive `smbstatus` subprocess and samba process status broadcasts are gated to heavy ticks (~every 60s) instead of every 5s.
  - `HDIdleService`: Disk power state (spun-up/spun-down) is tracked in memory; DB writes only occur on state transitions rather than on every polling cycle.
  - `AddonConfigWatcherService`: File modification timestamp (`mtime`) is checked before reading and hashing `options.json`, skipping the full read when the file has not changed.
  - Fixed `EnableSMART`/`DisableSMART` and `GetSmartInfo` emitting/returning `DiskId` set to the raw device path (`/dev/sda`) instead of the canonical device ID used to index the `DiskMap`. This caused SMART info to be silently lost after toggling SMART and caused the health API `per_disk_info[id].smart_info.disk_id` to show the raw path.
  - Fixed `volume_service` `OnSmart` handler calling `AddSmartInfo` for self-test progress events (which carry an empty `SmartInfo.DiskId`), producing hundreds of spurious `WARN` log entries every 5s during a running self-test.

## 2026.5.0-rc6

### 🐛 Bug Fixes

- Fix compile issue in github actions that was cause of freezed UI in some cases.

## 2026.5.0-rc5

### ✨ Features

- New startup wizard for first-run configuration of essential Samba settings (hostname, workgroup, admin password) and optional telemetry opt-in. The wizard is implemented as a multi-step dialog with a progress stepper and integrated with the existing guided tour system for contextual help. It is accessible from the Settings page and automatically shown on first run.

## 2026.04.0-rc4

### 🏗 Chore

- Correct release process.

## 2026.4.0-rc3

### 🔧 Maintenance

- **Backend Code Quality Refactor**: We are undertaking a comprehensive refactor to eliminate recurring Go anti-patterns across the codebase. This includes replacing `interface{}` with `any`, updating error handling to use `errors.AsType[T]`, modernizing goroutine management with `wg.Go`, extracting common helper functions, and improving context-aware logging. This refactor will enhance code readability, maintainability, and performance while adhering to modern Go best practices.
- **TypeScript 6.0 Migration**: The frontend codebase is being updated for compatibility with TypeScript 6.0 final, including removal of deprecated compiler flags, updating ECMAScript target, enabling new strict flags, and optimizing code for improved type inference. A comprehensive migration guide is being created to document the changes and prepare for the upcoming TypeScript 7.0 Go-based compiler.

## v2026.4.0-rc2 [ 🧪 Release Candidate ]

### ✨ Features

- **Interface IP Resolution**: Samba configuration now resolves network interface names to IP addresses at generation time, ensuring IPv4 preference is honored. The `--ipv4-only` CLI flag allows disabling IPv6 addresses in the `interfaces` directive. This prevents issues where interface names could resolve to IPv6 addresses, causing connectivity problems when IPv4 is preferred.
- **HACS Custom Component**: Added a Home Assistant custom component (`custom_components/srat/`) compatible with HACS for direct integration with Home Assistant. Supports UI configuration wizard, Supervisor add-on autodiscovery via slug whitelist, WebSocket-based real-time updates, and exposes sensors compatible with the existing SRAT HA integration (samba status, process status, volume status, disk health, per-disk I/O, and per-partition health). Includes full test suite using `pytest-homeassistant-custom-component` and Python code quality tooling (ruff, mypy) integrated into CI. *Early internal implementation serving as the foundation for upcoming releases.*
- **Report Issue on GitHub**: Added new "Report Issue" functionality allowing users to easily create GitHub issues with automated diagnostic data collection:
  - Button in top navigation bar to open issue reporting dialog
  - Problem type selector (Frontend UI, HA Integration, Addon, or Samba problems)
  - Markdown-compatible description field
  - Optional data collection: contextual data (URL, navigation history, browser info, console errors), addon logs, and sanitized SRAT configuration
  - Automatic routing to appropriate repository (dianlight/srat or dianlight/hassos-addon) based on problem type
  - Pre-populated GitHub issue URL with diagnostic information
  - Downloads diagnostic files for attachment to the issue
- **Autoupdate with Signature Verification (#358)**: Implemented a new autoupdate mechanism using minio/selfupdate with cryptographic signature verification:
  - Added `--auto-update` flag to automatically download and apply updates without user acceptance
  - Updates are signed with minisign (Ed25519) signatures for security
  - Automatic restart when running under s6 supervision
  - Public key is embedded in the binary for signature verification
  - Build workflow automatically signs all release binaries
- **Allow Guest Setting**: Added new `Allow Guest` boolean setting in Settings → General section to enable anonymous guest access to Samba shares. When enabled, configures Samba with `guest account = nobody` and `map to guest = Bad User` for secure guest authentication.
- **Enhanced SMART Service [#234](https://github.com/dianlight/srat/issues/234)**: Implemented comprehensive SMART disk monitoring and control features including health assessment, temperature monitoring, and attribute tracking.
- **SMB over QUIC Support [#227](https://github.com/dianlight/srat/issues/227)**: Added comprehensive support for SMB over QUIC transport protocol with intelligent system detection and automatic fallback to TCP when QUIC is unavailable.
- **Autoupdate Service**: Implemented a back-end service for automatic updates from GitHub releases, with support for multiple channels (stable, beta, dev) and local development builds.
- **Telemetry Configuration**: Added UI in Settings to configure telemetry modes (Rollbar error tracking), dependent on internet connectivity and user consent.
- **Volume Mount Intelligence**: Enriched volume mount structs with partition and filesystem metadata to enable informed NFS vs CIFS export decisions and implemented proper volume-event handling for cache retry and invalidation. ([#500](https://github.com/dianlight/srat/issues/500))
- **Bidirectional Home Assistant WebSocket**: Introduced client-to-server WebSocket messaging, starting with a `helo` handshake that allows the custom component to announce its identity and version to the backend. ([#508](https://github.com/dianlight/srat/issues/508))
- **Disable SMART Integration Setting**: Added a new setting to disable SMART integration, helping mitigate excessive disk wake-ups in sleeping-disk scenarios. ([#499](https://github.com/dianlight/srat/issues/499))
- **Home Assistant Repairs Proxy Service**: Implemented a backend service to manage Home Assistant repairs via the custom component, with queued commands and lifecycle synchronization over WebSocket. ([#518](https://github.com/dianlight/srat/issues/518))
- **Overlay Helper System Improvements**: Refactored the TourEvents system for better accuracy and type safety, added comprehensive tests, and established frontend maintenance guidelines. ([#515](https://github.com/dianlight/srat/issues/515))
- Add repair service and proxy for Home Assistant integration

### 🐛 Bug Fixes

- **HA Config Flow Discovery Import**: Fixed Supervisor discovery flow import errors by using the new `HassioServiceInfo` module path with a compatibility fallback for older Home Assistant versions.
- **Udev Event Parsing Error Handling**: Improved handling of malformed udev events to prevent spurious error reports to Rollbar. Malformed events with invalid environment data are now logged at debug level instead of error level, reducing noise in error tracking while maintaining visibility for legitimate errors.
- **Issue Report Gist Offloading**: Fixed oversized issue report URLs by replacing large addon log and console error parameters with Gist links, preventing runaway URL growth when diagnostics are large.
- **Mount Point Type Defaulting**: Default missing mount point types on events to avoid NOT NULL constraint failures when persisting mount points.
- **Mount Conversion Type Derivation**: Ensure mount conversions derive mount point type from the mount path to prevent missing type values.
- **WebSocket Loading State**: Report WebSocket SSE loading as active until the socket is connected, and re-enable loading after disconnects.
- **Deterministic Mount Flag Metadata**: Ensure mount-flag metadata for shared options (for example, uid/gid) is derived from a consistent preferred adapter source to avoid nondeterministic descriptions and regex values.
- **Volumes TreeView ID Collisions**: Namespace volume tree item IDs by disk to prevent duplicate partition identifiers from crashing the Volumes tab.
- **Disk FSCK Status Population**: Populate fsck supported/needed fields in disk stats using filesystem service capability and state information.

### 🔄 Breaking Changes

- **Update Engine Replacement**: Replaced jpillora/overseer with minio/selfupdate for binary updates. The new implementation provides more reliable updates with cryptographic signature verification using minisign. Updates will now properly restart the service when running under s6 supervision.
- **SMB over QUIC Default Behavior Change**: The SMB over QUIC feature is now disabled by default. Users must explicitly enable it in the settings to use this functionality. This change aims to enhance security and stability by preventing unintended use of the experimental protocol.
- **Telemetry Service Update**: The telemetry service has been updated to use Rollbar for error tracking and monitoring. This change may require users to review their privacy settings and consent to data collection, as Rollbar collects different types of data compared to the previous telemetry solution.
- **Autoupdate Service Modification**: The autoupdate service has been modified to support multiple update channels (stable, beta, dev) and local development builds. Users may need to reconfigure their update preferences to align with the new channel system.
- **Disk Health Payload Update**: Per-partition disk health now reports `filesystem_state` and no longer includes the redundant `fsck_needed` field.
- **Partition Filesystem Support**: Per-partition disk health no longer includes `fsck_supported`; filesystem support is now reported on partitions as `filesystem_support`.

### 🔧 Maintenance

- **Custom Component Build Tooling**: Added ruff (lint + format) and mypy (type checking) tooling for the HA custom component with `pyproject.toml` configuration, `Makefile` targets (`make check`, `make lint`, `make format`, `make typecheck`), and CI integration in `validate-hacs.yaml`. Fixed all lint and type issues in existing code.
- **Go 1.26 Migration**: Upgraded Go version from 1.25.7 to 1.26.0, adopting new language features:
  - Replaced all `pointer.Bool/String/Int/Uint64/Of/Any()` calls with Go 1.26's built-in `new(expr)` syntax (~268 occurrences) and removed the `xorcare/pointer` dependency
  - Replaced all `interface{}` with `any` alias (147 occurrences) following Go modernizer patterns
  - Replaced `sync.WaitGroup` `Add(1)/Done()` patterns with `WaitGroup.Go()` method in production code
- **TypeScript 6.0 Final Migration**: Updated frontend TypeScript configuration for compatibility with TypeScript 6.0 final (March 23, 2026) and preparation for TypeScript 7.0 (Go-based):
  - Removed all deprecated compiler flags (`experimentalDecorators`, `useDefineForClassFields`, `baseUrl`, `outFile`)
  - Updated ECMAScript target from ES2021 to ES2022 for better modern feature alignment
  - Enabled `noImplicitOverride` strict flag (code already compliant)
  - Code optimizations leveraging TS 6.0 improved type inference (removed 11 unnecessary type assertions)
  - Updated `peerDependencies` to support TypeScript 6.0 final
  - Created comprehensive migration guide (`frontend/TYPESCRIPT_MIGRATION.md`) documenting completed work and remaining tasks for full TS 7.0 readiness
  - Project uses `@typescript/native-preview` (tsgo) for type checking
  - TypeScript 6.0 final is the last JavaScript-based version before the Go-native 7.0 compiler
- Updated dependencies to latest versions to ensure security and compatibility.

### 🏗 Chore

- Replace snapd osutil dependency with internal mount utilities based on moby/sys/mountinfo <!-- cspell:disable-line -->
- Align UI elements to HA [#81](https://github.com/dianlight/srat/issues/81)
- Create the base documentation [#80](https://github.com/dianlight/srat/issues/80)
- Display version from ADDON

## 2025.06.1-dev.801 [ 🧪 Pre-release ]

### ✨ Features

- First Fully functional version ready for first merge.
