# Spec Delta

## Purpose

commandexec Snapshot Memory Leak and Busy-Wait Elimination behaviour for the SRAT project.

## ADDED Requirements

### Requirement: commandexec Snapshot Memory Leak and Busy-Wait Elimination

The system SHALL fix two related performance defects in `backend/src/internal/commandexec/runner.go`: 1. **Memory leak** — completed execution snapshots are stored in `Service.snapshots` indefinitely and never evicted. Each entry can hold up to 500 `CommandOutputLineSnapshot` structs. Long-running servers accumulate this data from filesystem checks, SMART tests, samba restarts, and upgrade flows.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Add `doneCh chan struct{}` to `executionState` struct alongside `snapshot` and `quiet`

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
