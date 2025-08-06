# Changelog

## 2025.08.\* [ 🚧 Unreleased ]

### ✨ Features

- Manage `recycle bin`option for share
- Manage WSDD2 service
- Manage Avahi service
- Veto files for share not global [#79](https://github.com/dianlight/srat/issues/79)
- Ingress security validation [#89](https://github.com/dianlight/srat/issues/89)
- Dashboard
- Add Rollbar telemetry service for error tracking and monitoring

#### **🚧 Work in progress**

- [ ] Manage `local master` option (?)
- [ ] Help screen or overlay help/tour [#82](https://github.com/dianlight/srat/issues/82)
- [ ] Custom component [#83](https://github.com/dianlight/srat/issues/83)
- [x] Smart Control [#100](https://github.com/dianlight/srat/issues/100)
- [x] HDD Spin down [#101](https://github.com/dianlight/srat/issues/101)

### 🐛 Bug Fixes

- `enable`/`disable` share functionality is not working as expected.
- Renaming the admin user does not correctly create the new user or rename the existing one; issues persist until a full addon reboot.
- Fix dianlight/hassio-addons#448 [SambaNAS2] Unable to create share for mounted volume
- Fix dianlight/hassio-addons#447 [SambaNAS2] Unable to mount external drive

#### **🚧 Work in progress**

- [W] Addon protected mode check [#85](https://github.com/dianlight/srat/issues/85)

### 🏗 Chore

- Implement watchdog
- Align UI elements to HA [#81](https://github.com/dianlight/srat/issues/81)

#### **🚧 Work in progress**

- [ ] Create the base documentation [#80](https://github.com/dianlight/srat/issues/80)
- [ ] Display version from ADDON

## 2025.06.1-dev.801 [ 🧪 Pre-release ]

### ✨ Features

- First Fully functional version ready for first merge!
