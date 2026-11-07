# Spec Delta

## Purpose

Frontend Data Correctness — isLoading Bug, Hook Rules, Password Exposure behaviour for the SRAT project.

## ADDED Requirements

### Requirement: Frontend Data Correctness — isLoading Bug, Hook Rules, Password Exposure

The system SHALL fix three data-correctness and security defects in the frontend that cause silent incorrect behavior: 1. **`isLoading` uses `&&` instead of `||`** — the three data hooks (`healthHook`, `volumeHook`, `shareHook`) combine REST and WebSocket loading flags with `&&`, which means a REST API error makes `isLoading: false` while `evloading` is still `true`. Dashboard metrics, volumes, and shares render silently empty instead of showing an...

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Change `isLoading: isLoading && evloading` to `isLoading: isLoading || evloading` in `healthHook.ts:48`, `volumeHook.ts:29`, and `shareHook.ts:30`

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
