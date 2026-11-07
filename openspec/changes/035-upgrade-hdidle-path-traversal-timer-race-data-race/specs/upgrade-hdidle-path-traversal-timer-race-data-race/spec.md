# Spec Delta

## Purpose

Upgrade & HDIdle Path Traversal, Timer Race, and Data Race behaviour for the SRAT project.

## ADDED Requirements

### Requirement: Upgrade & HDIdle Path Traversal, Timer Race, and Data Race

The system SHALL fix four related security and reliability defects in the upgrade and HDIdle subsystems: 1. **ZipSlip bypass** — `extractFile` in `upgrade_service.go` has a `dest != "."` escape that disables path traversal protection when the destination is the current directory.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** In `upgrade_service.go extractFile`, remove the `dest != "."` guard — always verify `strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(dest)+string(os.PathSeparator))`

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
