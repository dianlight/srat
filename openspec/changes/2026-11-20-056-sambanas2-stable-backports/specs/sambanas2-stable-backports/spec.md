# Spec Delta

## Purpose

Tracks upstream SambaNAS2 add-on fixes that are backported to the 2026.11.0 stable branch. No SRAT runtime behavior changes.

## ADDED Requirements

### Requirement: Stable branch includes the 2026.10.2 mount fix

The 2026.11.0 stable add-on SHALL mount `/root/nfs_root.img` at `/mnt` only when `/mnt` is not already a mount point, and SHALL start `srat` only after `init-nfsd` completes.

#### Scenario: USB volume already mounted at /mnt

- **WHEN** a USB volume owns `/mnt` at container startup
- **THEN** `init-nfsd` skips the image mount and logs a warning instead of aborting startup

### Requirement: Single-dash SRAT flags are passed

The `srat` startup script SHALL probe `srat-server --help` for the single-dash forms `-auto-update` and `-ipv4-only` and SHALL pass them when the corresponding add-on options are enabled.

#### Scenario: auto_update enabled

- **WHEN** `auto_update` is true and `--help` reports `-auto-update`
- **THEN** `SRAT_ARGS` contains `-auto-update`

### Requirement: Four-part binary versions are accepted

The binary self-upgrade check SHALL accept versions with three or four numeric components, with an optional leading `v` and an optional prerelease suffix.

#### Scenario: staged 4-part binary

- **WHEN** an upgrade binary is versioned `v2026.11.19.4` and the installed binary is `v2026.11.18.1`
- **THEN** the upgrade is applied

#### Scenario: malformed version

- **WHEN** a version has fewer than three numeric components
- **THEN** it is rejected

### Requirement: No stale bashio caches

The built image SHALL NOT contain `/tmp/.bashio` cache files, and each s6 service that reads ingress port or options SHALL flush stale bashio caches before the first read.

#### Scenario: fresh container from a released image

- **WHEN** the add-on starts from a freshly pulled stable image
- **THEN** ingress port and options are read from the live Supervisor API

#### Scenario: cache helper unavailable

- **WHEN** `bashio::cache.flush_all` is not defined
- **THEN** the service deletes `/tmp/.bashio/*.cache` directly
