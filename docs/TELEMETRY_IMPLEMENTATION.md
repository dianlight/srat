<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Sentry Telemetry Implementation](#sentry-telemetry-implementation)
  - [Consent Model (unchanged)](#consent-model-unchanged)
  - [Backend (Go)](#backend-go)
    - [Key behaviors](#key-behaviors)
    - [Stack trace handling](#stack-trace-handling)
    - [Tests](#tests)
  - [Frontend (React/TypeScript)](#frontend-reacttypescript)
    - [Key modules](#key-modules)
    - [Key behaviors](#key-behaviors-1)
  - [Build and Continuous Integration](#build-and-continuous-integration)
  - [Privacy Notes](#privacy-notes)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Sentry Telemetry Implementation

This document summarizes SRAT telemetry/error reporting after migration from the previous telemetry provider to Sentry.

## Consent Model (unchanged)

SRAT preserves the same user-facing telemetry modes:

- **Ask**: no data until user chooses
- **All**: errors + telemetry events
- **Errors**: errors only
- **Disabled**: no telemetry

## Backend (Go)

Core implementation is in `backend/src/service/telemetry_service.go`.

### Key behaviors

- Uses `github.com/getsentry/sentry-go`
- Initializes with runtime-derived environment (`development` / `prerelease` / `production`)
- Sends exceptions with `CaptureException`
- Sends telemetry events with `CaptureEvent`
- Flushes on shutdown with `sentry.Flush(...)`
- Connectivity checks target `https://sentry.io`

### Stack trace handling

A fallback stack trace extraction path is used for `tozd/go/errors` style stacks to improve exception context when Sentry does not extract one automatically.

### Tests

`backend/src/service/telemetry_service_test.go` uses a custom in-memory Sentry transport to capture emitted events without network calls.

## Frontend (React/TypeScript)

### Key modules

- `frontend/src/hooks/useSentryTelemetry.ts`
- `frontend/src/components/ConsoleErrorToSentry.tsx`
- `frontend/src/components/ErrorBoundaryWrapper.tsx`
- `frontend/src/index.tsx`

### Key behaviors

- Initializes Sentry in the app entrypoint
- Uses telemetry mode guards before reporting
- Captures manual errors/events via hook
- Forwards `console.error` calls through a dedicated component
- Uses `@sentry/react` error boundary

## Build and Continuous Integration

- Backend DSN: `SENTRY_DSN` → `config.SentryDSN` via linker flag
- Frontend DSN: `VITE_SENTRY_DSN`
- CI workflow updated to pass Sentry DSN variables

## Privacy Notes

Telemetry remains optional and user-controlled. See `PRIVACY.md` for details about data usage and third-party processing.
