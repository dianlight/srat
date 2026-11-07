# Spec Delta

## Purpose

Missing SambaNAS2 Addon Detection behaviour for the SRAT project.

## ADDED Requirements

### Requirement: Missing SambaNAS2 Addon Detection

The system SHALL detect when the SambaNAS2 add-on is missing, disabled, or unavailable from the Home Assistant Supervisor, and surface a clear, actionable Repair flow so users can recover quickly without manual log inspection.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Define the missing/unavailable add-on state model and detection rules

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
