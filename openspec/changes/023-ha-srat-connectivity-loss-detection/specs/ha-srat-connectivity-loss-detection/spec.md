# Spec Delta

## Purpose

HA SRAT Connectivity Loss Detection behaviour for the SRAT project.

## ADDED Requirements

### Requirement: HA SRAT Connectivity Loss Detection

The system SHALL detect and surface connectivity loss between Home Assistant and SRAT (especially WebSocket disruption and prolonged reconnection failure), then trigger clear Repairs guidance so users can quickly restore control-plane communication.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Define connectivity states and outage thresholds

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
