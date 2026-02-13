<!-- DOCTOC SKIP -->

# Changelog

## [ 🚧 Unreleased ]

### 🙏 Thanks

We would like to thank all supporters for their contributions and donations.
With your donations, we are able to continue developing and improving this project. Your support is greatly appreciated.

### ✨ Features

- **HACS Custom Component**: Added a Home Assistant custom component (`custom_components/srat/`) compatible with HACS for direct integration with Home Assistant. Supports UI configuration wizard, Supervisor add-on autodiscovery via slug whitelist, WebSocket-based real-time updates, and exposes sensors compatible with the existing SRAT HA integration (samba status, process status, volume status, disk health, per-disk I/O, and per-partition health). Includes full test suite using `pytest-homeassistant-custom-component` and Python code quality tooling (ruff, mypy) integrated into CI.

### 🧑‍🏫 Documentation

### 🐛 Bug Fixes

- **HA Config Flow Discovery Import**: Fixed Supervisor discovery flow import errors by using the new `HassioServiceInfo` module path with a compatibility fallback for older Home Assistant versions.
- **Udev Event Parsing Error Handling**: Improved handling of malformed udev events to prevent spurious error reports to Rollbar. Malformed events with invalid environment data are now logged at debug level instead of error level, reducing noise in error tracking while maintaining visibility for legitimate errors.
- **Issue Report Gist Offloading**: Fixed oversized issue report URLs by replacing large addon log and console error parameters with Gist links, preventing runaway URL growth when diagnostics are large.
- **Mount Point Type Defaulting**: Default missing mount point types on events to avoid NOT NULL constraint failures when persisting mount points.
- **Mount Conversion Type Derivation**: Ensure mount conversions derive mount point type from the mount path to prevent missing type values.
- **WebSocket Loading State**: Report WebSocket SSE loading as active until the socket is connected, and re-enable loading after disconnects.

### 🔄 Breaking Changes

- **Update Engine Replacement**: Replaced jpillora/overseer with minio/selfupdate for binary updates. The new implementation provides more reliable updates with cryptographic signature verification using minisign. Updates will now properly restart the service when running under s6 supervision.
- **SMB over QUIC Default Behavior Change**: The SMB over QUIC feature is now disabled by default. Users must explicitly enable it in the settings to use this functionality. This change aims to enhance security and stability by preventing unintended use of the experimental protocol.
- **Telemetry Service Update**: The telemetry service has been updated to use Rollbar for error tracking and monitoring. This change may require users to review their privacy settings and consent to data collection, as Rollbar collects different types of data compared to the previous telemetry solution.
- **Auto-Update Service Modification**: The auto-update service has been modified to support multiple update channels (stable, beta, dev) and local development builds. Users may need to reconfigure their update preferences to align with the new channel system.

### 🔧 Maintenance

- **Custom Component Build Tooling**: Added ruff (lint + format) and mypy (type checking) tooling for the HA custom component with `pyproject.toml` configuration, `Makefile` targets (`make check`, `make lint`, `make format`, `make typecheck`), and CI integration in `validate-hacs.yaml`. Fixed all lint and type issues in existing code.
- **Go 1.26 Migration**: Upgraded Go version from 1.25.7 to 1.26.0, adopting new language features:
  - Replaced all `pointer.Bool/String/Int/Uint64/Of/Any()` calls with Go 1.26's built-in `new(expr)` syntax (~268 occurrences) and removed the `xorcare/pointer` dependency
  - Replaced all `interface{}` with `any` alias (147 occurrences) following Go modernizer patterns
  - Replaced `sync.WaitGroup` `Add(1)/Done()` patterns with `WaitGroup.Go()` method in production code
- **TypeScript 6.0/7.0 Preparation**: Updated frontend TypeScript configuration for compatibility with TypeScript 6.0 Beta and preparation for TypeScript 7.0 (Go-based):
  - Removed deprecated compiler flags: `experimentalDecorators` and `useDefineForClassFields`
  - Updated ECMAScript target from ES2021 to ES2022 for better modern feature alignment
  - Enabled `noImplicitOverride` strict flag (code already compliant)
  - Updated `peerDependencies` to support TypeScript 6.0 beta
  - Created comprehensive migration guide (`frontend/TYPESCRIPT_MIGRATION.md`) documenting completed work and remaining tasks for full TS 7.0 readiness
  - Project uses `@typescript/native-preview` (tsgo) for type checking
- Updated dependencies to latest versions to ensure security and compatibility.

### ✨ Features

- **Report Issue on GitHub**: Added new "Report Issue" functionality allowing users to easily create GitHub issues with automated diagnostic data collection:
  - Button in top navigation bar to open issue reporting dialog
  - Problem type selector (Frontend UI, HA Integration, Addon, or Samba problems)
  - Markdown-compatible description field
  - Optional data collection: contextual data (URL, navigation history, browser info, console errors), addon logs, and sanitized SRAT configuration
  - Automatic routing to appropriate repository (dianlight/srat or dianlight/hassos-addon) based on problem type
  - Pre-populated GitHub issue URL with diagnostic information
  - Downloads diagnostic files for attachment to the issue
- **Auto-Update with Signature Verification (#358)**: Implemented a new auto-update mechanism using minio/selfupdate with cryptographic signature verification
  - Added `--auto-update` flag to automatically download and apply updates without user acceptance
  - Updates are signed with minisign (Ed25519) signatures for security
  - Automatic restart when running under s6 supervision
  - Public key is embedded in the binary for signature verification
  - Build workflow automatically signs all release binaries
- **Allow Guest Setting**: Added new `Allow Guest` boolean setting in Settings → General section to enable anonymous guest access to Samba shares. When enabled, configures Samba with `guest account = nobody` and `map to guest = Bad User` for secure guest authentication.
- **Enhanced SMART Service [#234](https://github.com/dianlight/srat/issues/234)**: Implemented comprehensive SMART disk monitoring and control features:
- **SMB over QUIC Support [#227](https://github.com/dianlight/srat/issues/227)**: Added comprehensive support for SMB over QUIC transport protocol with intelligent system detection
- **Auto-Update Service**: Implemented a backend service for automatic updates from GitHub releases, with support for multiple channels and local development builds.
- **Telemetry Configuration**: Added UI in Settings to configure telemetry modes, dependent on internet connectivity.
- Manage `local master` option (?)
- Add Rollbar telemetry service for error tracking and monitoring
- Help screen or overlay help/tour [#82](https://github.com/dianlight/srat/issues/82)
- Smart Control [#100](https://github.com/dianlight/srat/issues/100)
- HDD Spin down [#101](https://github.com/dianlight/srat/issues/101)

### 🏗 Chore

- Replace snapd osutil dependency with internal mount utilities based on moby/sys/mountinfo <!-- cspell:disable-line -->
- Align UI elements to HA [#81](https://github.com/dianlight/srat/issues/81)
- Create the base documentation [#80](https://github.com/dianlight/srat/issues/80)
- Display version from ADDON

## 2025.06.1-dev.801 [ 🧪 Pre-release ]

### ✨ Features

- First Fully functional version ready for first merge.
