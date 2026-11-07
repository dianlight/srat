# Spec Delta

## Purpose

Defines the shares UI behavior for shares backed by Home Assistant-managed volumes: annotation badge and mount-point chip coloring.

## ADDED Requirements

### Requirement: Shares on HA-managed volumes show a badge

The system SHALL display a "Managed by Home Assistant" chip on a share whose status reports `is_ha_mounted = true`.

#### Scenario: badge shown
- **WHEN** a share's `status.is_ha_mounted` is `true`
- **THEN** the share details panel renders a "Managed by Home Assistant" chip

### Requirement: Usage chip reflects HA ownership

The system SHALL color the usage chip as primary when the share is HA-managed.

#### Scenario: primary color
- **WHEN** a share's `status.is_ha_mounted` is `true`
- **THEN** the usage chip uses the primary color