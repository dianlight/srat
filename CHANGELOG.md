<!-- DOCTOC SKIP -->

# Changelog

## [ 🚧 Unreleased ]

### 🙏 Thanks

We would like to thank all supporters for their contributions and donations.
With your donations, we are able to continue developing and improving this project. Your support is greatly appreciated.

### 🧑‍🏫 Documentation

### 🐛 Bug Fixes

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
- **Auto-Update Service Modification**: The auto-update service has been modified to support multiple update channels (stable, beta, dev) and local development builds. Users may need to reconfigure their update preferences to align with the new channel system.
- **Disk Health Payload Update**: Per-partition disk health now reports `filesystem_state` and no longer includes the redundant `fsck_needed` field.
- **Partition Filesystem Support**: Per-partition disk health no longer includes `fsck_supported`; filesystem support is now reported on partitions as `filesystem_support`.

### 🔧 Maintenance

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
