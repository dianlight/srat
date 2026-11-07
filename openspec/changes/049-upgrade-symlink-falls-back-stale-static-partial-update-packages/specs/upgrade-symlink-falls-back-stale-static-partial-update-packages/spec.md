# Spec Delta

## Purpose

Upgrade symlink falls back to stale static on partial update packages behaviour for the SRAT project.

## ADDED Requirements

### Requirement: Upgrade symlink falls back to stale static on partial update packages

The system SHALL fix `detectBestServerVariant` in `backend/src/service/upgrade_service.go` so a develop-channel update containing only a subset of binaries (e.g. `srat-cli` alone) does not re-point the `srat-server` symlink at a stale `srat-server-static` from a previous installation.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Reproduce — unit test where `updatePkg` contains only `srat-cli`, target dir has newer musl + older static on a musl-linker system; assert current code returns `srat-server-static` (bug)

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
