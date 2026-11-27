# Spec Delta

## Purpose

Define how a Samba share may expose a specific subdirectory of a mounted volume instead of the mount root, so one volume can carry several shares.

## ADDED Requirements

### Requirement: Share may expose a mount-relative subfolder

The system SHALL support configuring a Samba share with an optional subfolder that is resolved relative to the mounted volume root.

#### Scenario: Subfolder configured

- **WHEN** a user configures a share with a subfolder
- **THEN** the Samba share path resolves to the mount root joined with the subfolder
- **AND** the folder is created with the volume root's mode and owner when it does not exist

#### Scenario: Subfolder defaults to mount root

- **WHEN** a share is configured without a subfolder
- **THEN** the Samba share path resolves to the mounted volume root

#### Scenario: Traversal rejected

- **WHEN** a share configuration sets a subfolder containing an absolute path or parent traversal
- **THEN** the configuration is rejected and the share path is not changed

#### Scenario: Symlink escape rejected

- **WHEN** a subfolder resolves outside the volume through a symlink
- **THEN** the configuration is rejected and nothing is created outside the volume

#### Scenario: Unmounted volume

- **WHEN** a subfolder must be created on a volume that is not mounted
- **THEN** the configuration is rejected and nothing is created in the mount point directory

### Requirement: Several shares per volume

The system SHALL allow several shares on one volume, each exposing a different folder.

#### Scenario: One share per folder

- **WHEN** shares are created for different subfolders of the same volume
- **THEN** each share exposes its own folder with its own users

#### Scenario: Folder already exposed

- **WHEN** a share would expose a folder that another active share already exposes, including the volume root
- **THEN** the configuration is rejected with a conflict

#### Scenario: Mount state

- **WHEN** a volume carrying several shares is mounted or unmounted
- **THEN** every share of the volume is updated

### Requirement: A share is never widened by accident

The system SHALL keep a share's subfolder unless it is explicitly changed.

#### Scenario: Partial update

- **WHEN** a share update omits the subfolder
- **THEN** the stored subfolder is kept

#### Scenario: Folder removed

- **WHEN** the folder of a subfolder share no longer exists
- **THEN** the share is reported invalid and is not published, instead of exposing the volume root

### Requirement: Subfolders are a lab feature

The system SHALL require Lab Mode (`share_subfolders`, beta) to set or change a share subfolder.

#### Scenario: Lab Mode off

- **WHEN** a request sets a new or different subfolder while Lab Mode is off
- **THEN** it is rejected with 403

#### Scenario: Existing subfolder share without Lab Mode

- **WHEN** Lab Mode is off and a share that already uses a subfolder is updated without changing it
- **THEN** the update succeeds and the subfolder is kept
