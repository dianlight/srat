# Spec Delta

## Purpose

Upstream libsmartmon JSON schema gaps (self-test status, smart-status) behaviour for the SRAT project.

## ADDED Requirements

### Requirement: Upstream libsmartmon JSON schema gaps (self-test status, smart-status)

The system SHALL resolve the root cause found in #1196 at its source: the bundled `libsmartmon_go.so` emits an `ata_smart_data.self_test` object containing only `polling_minutes` (verified via strings on the shipped `.so`: `"self_test":{"polling_minutes":{"short":`) and no usable `smart_status`, while its `checkHealth` reports failure on drives that `smartctl -H` passes (Kingston SV300S37A240G, `smartctl -a` exit 0). Either fix upstream...

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Confirm exact gaps against upstream C source (self_test.status, smart_status.passed, checkHealth logic)

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
