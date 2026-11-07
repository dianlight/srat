# Spec Delta

## Purpose

Addon Restart Device-Not-Found Race behaviour for the SRAT project.

## ADDED Requirements

### Requirement: Addon Restart Device-Not-Found Race

The system SHALL fix transient `Device 'ata-...' not found during startup` on soft add-on Restart. The by-id symlink (or Supervisor hardware snapshot) is not yet visible when the first automount runs, the failure is terminal (no retry), and only a full host reboot recovers.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Create task doc with hassio-addons#767 + srat#1266 links

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
