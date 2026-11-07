# Spec Delta

## Purpose

Define how a Samba share may expose a specific subdirectory of a mounted volume instead of the mount root.

## ADDED Requirements

### Requirement: Share may expose a mount-relative subfolder

The system SHALL support configuring a Samba share with an optional subfolder that is resolved relative to the mounted volume root.

#### Scenario: Subfolder configured

- **WHEN** a user configures a share with a subfolder
- **THEN** the Samba share path resolves to the mount root joined with the subfolder

#### Scenario: Subfolder defaults to mount root

- **WHEN** a share is configured without a subfolder
- **THEN** the Samba share path resolves to the mounted volume root

#### Scenario: Traversal rejected

- **WHEN** a share configuration sets a subfolder containing an absolute path or parent traversal
- **THEN** the configuration is rejected and the share path is not changed
