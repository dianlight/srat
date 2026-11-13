# Spec Delta

## Purpose

Defines the volumes UI and API behavior for Home Assistant-managed mount points: badge display, hidden mount/format/label/unmount actions, retained Create Share / Go to Share, and hidden Mount Settings card.

## ADDED Requirements

### Requirement: HA-managed volumes show a badge

The system SHALL display a "Managed by Home Assistant" badge on partitions that have any HA-managed mount point.

#### Scenario: badge shown
- **WHEN** a partition's `mount_point_data` contains an entry with `is_ha_mounted = true`
- **THEN** the partition information card renders a "Managed by Home Assistant" chip

### Requirement: HA-managed volumes hide mount and filesystem actions

The system SHALL hide mount, unmount, automount, format, check and label actions for HA-managed partitions, while retaining Create Share (when no share exists) and Go to Share (when a share exists).

#### Scenario: no mount actions
- **WHEN** a partition is HA-managed and has no share
- **THEN** the partition action list contains a "Create Share" action and no mount/unmount/automount/format/check/label actions

#### Scenario: go to share
- **WHEN** a partition is HA-managed and has a share
- **THEN** the partition action list contains a "Go to Share" action and no mount/unmount/automount/format/check/label actions

### Requirement: HA-managed volumes hide Mount Settings card

The system SHALL not render the Mount Settings card for HA-managed partitions.

#### Scenario: mount settings hidden
- **WHEN** a partition's `mount_point_data` contains an entry with `is_ha_mounted = true`
- **THEN** the "Mount Settings" card is not rendered