# Spec Delta

## Purpose

Allow Share Subfolders behaviour for the SRAT project.

## ADDED Requirements

### Requirement: Allow Share Subfolders

The system SHALL allow users to configure a Samba share that exposes a specific subdirectory of a mounted volume rather than its root. Currently every share maps directly to the mount-point root (`/mnt/<device>`). The goal is to add an optional `subfolder` (or `path`) field to the share configuration so that `smb.conf` uses `path = /mnt/<device>/<subfolder>` instead.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Add `Subfolder string` (optional) field to `dto.SharedResource` and `dbom.ExportedShare`; add DB migration for the new column

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
