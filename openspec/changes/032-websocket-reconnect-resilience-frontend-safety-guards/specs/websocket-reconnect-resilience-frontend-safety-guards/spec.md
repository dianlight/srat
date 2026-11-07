# Spec Delta

## Purpose

WebSocket Reconnect Resilience and Frontend Safety Guards behaviour for the SRAT project.

## ADDED Requirements

### Requirement: WebSocket Reconnect Resilience and Frontend Safety Guards

The system SHALL fix four related issues in the frontend WebSocket layer and `App.tsx` command-output handling: 1. **No exponential backoff** — all tabs reconnect at exactly 1s after disconnect, causing synchronized storms on HA supervisor restarts.

#### Scenario: Change applied

- **WHEN** the change is applied as described in the proposal
- **THEN** Add `reconnectAttempt` counter to the `onCacheEntryAdded` scope; implement exponential backoff: `delay = Math.min(base * 2 ** attempt, maxDelay) + Math.random() * 200`; cap at 30s; reset counter on successful `open`

#### Scenario: Regression check

- **WHEN** the change has been implemented
- **THEN** the relevant test suites pass
