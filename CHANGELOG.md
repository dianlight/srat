<!-- DOCTOC SKIP -->

# Changelog

## [ 🚧 Unreleased ]

### 🧑‍🏫 Documentation

- **Frontend Testing Standards**: Updated documentation and Copilot instructions to mandate `@testing-library/user-event` for all user interactions in tests. The deprecated `fireEvent` API is now strictly prohibited in all new and modified tests. Updated files:
  - `.github/copilot-instructions.md`: Added userEvent requirements to Testing Library Standards and Component Testing Pattern sections
  - `docs/TEST_COVERAGE.md`: Added userEvent to Framework & Tools and Frontend Testing Best Practices
  - `frontend/README.md`: Added Testing Standards section with userEvent requirement

### 🐛 Bug Fixes

- **Udev Event Parsing Error Handling**: Improved handling of malformed udev events to prevent spurious error reports to Rollbar. Malformed events with invalid environment data are now logged at debug level instead of error level, reducing noise in error tracking while maintaining visibility for legitimate errors.

### 🔄 Breaking Changes

### 🔧 Maintenance

### ✨ Features

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

- First Fully functional version ready for first merge!
