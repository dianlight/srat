<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents** _generated with [DocToc](https://github.com/thlorenz/doctoc)_

- [Telemetry Configuration Guide](#telemetry-configuration-guide)
  - [Overview](#overview)
  - [Environment Variables](#environment-variables)
    - [Server (Go)](#server-go)
    - [Frontend (TypeScript)](#frontend-typescript)
    - [Source map upload (frontend)](#source-map-upload-frontend)
  - [Build-Time Configuration](#build-time-configuration)
    - [Server linker flags](#server-linker-flags)
    - [Frontend build injection](#frontend-build-injection)
  - [Environment Detection](#environment-detection)
  - [Local Development](#local-development)
  - [Continuous Integration and Delivery (GitHub Actions)](#continuous-integration-and-delivery-github-actions)
  - [Fallback Behavior](#fallback-behavior)
  - [Troubleshooting](#troubleshooting)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Telemetry Configuration Guide

Telemetry in SRAT is powered by **Sentry** and remains controlled by the existing four consent modes (`ask`, `all`, `errors`, `disabled`).

## Overview

SRAT uses build-time configuration for telemetry DSN values:

- **Server**: `SENTRY_DSN` is embedded via Go linker flags into `config.SentryDSN`
- **Frontend**: `VITE_SENTRY_DSN` is injected at build time and read by the frontend macro layer. It must point at a dedicated frontend Sentry project, distinct from the backend `SENTRY_DSN` (#1344).

Environment (`development`, `prerelease`, `production`) is detected at runtime from the version string.

## Environment Variables

### Server (Go)

| Variable     | Required | Description                                | Default    |
| ------------ | -------- | ------------------------------------------ | ---------- |
| `SENTRY_DSN` | No       | Server Sentry DSN (embedded at build time) | `disabled` |

### Frontend (TypeScript)

| Variable          | Required | Description                                | Default    |
| ----------------- | -------- | ------------------------------------------ | ---------- |
| `VITE_SENTRY_DSN` | No       | Frontend Sentry DSN                        | `disabled` |

It must point at a dedicated frontend Sentry project, distinct from the
backend `SENTRY_DSN`.

### Source map upload (frontend)

| Variable                  | Required      | Description                                                | Default      |
| ------------------------- | ------------- | ---------------------------------------------------------- | ------------ |
| `SENTRY_AUTH_TOKEN`       | No (secret)   | Uploads `*.js.map` after production frontend builds        | empty (skip) |
| `SENTRY_ORG`              | No (variable) | Sentry org for source map upload                           | empty (skip) |
| `SENTRY_PROJECT_FRONTEND` | No (variable) | Frontend project slug, no fallback                         | empty (skip) |

Production frontend builds emit external `*.js.map` files (watch/serve stay
inline) and `mise run //frontend:build` uploads them via
`frontend/scripts/upload-sourcemaps.mjs`. The script is best-effort and skips
when secrets are missing, so forks and local builds are unaffected. Use a
dedicated frontend Sentry project: backend events must not land in the same
project as `index-*.js` frontend events (#1344).

## Build-Time Configuration

### Server linker flags

The server build task sets:

- `-X github.com/dianlight/srat/config.SentryDSN=${SENTRY_DSN:-disabled}`

Version metadata is also embedded at build time and used for environment detection.

### Frontend build injection

Set `VITE_SENTRY_DSN` in your environment before frontend build. Keep it
distinct from the backend `SENTRY_DSN`: sharing one DSN routes `index-*.js`
frontend frames into the backend project and breaks triage (#1344).

## Environment Detection

Environment is determined from version automatically:

- `*-dev.*` or `0.0.0-dev.0` → `development`
- `*-rc.*` → `prerelease`
- otherwise → `production`

## Local Development

Optional `.env` example:

- `SENTRY_DSN=disabled`
- `VITE_SENTRY_DSN=disabled`

Then run normal build/test tasks.

## Continuous Integration and Delivery (GitHub Actions)

Recommended secrets:

- `SENTRY_DSN`
- `VITE_SENTRY_DSN` (must differ from backend `SENTRY_DSN`)

Recommended variables (non-sensitive slugs):

- `SENTRY_ORG`, `SENTRY_PROJECT`, `SENTRY_PROJECT_FRONTEND`
- plus `SENTRY_AUTH_TOKEN` as a secret for uploads and release tracking

These are consumed by the build workflow environment.

## Fallback Behavior

When DSN values are `disabled` or empty:

- no telemetry is sent
- consent UI and telemetry modes still function normally
- app behavior remains unchanged

## Troubleshooting

- **Telemetry disabled**: confirm `SENTRY_DSN` / `VITE_SENTRY_DSN` values at build time
- **Wrong environment in Sentry**: verify version naming (`-dev`, `-rc`, release)
- **No frontend events**: ensure `VITE_SENTRY_DSN` is present during frontend build
- **Minified `Object.current` / `withScope` frames**: confirm `*.js.map` upload ran (requires `SENTRY_AUTH_TOKEN`/`SENTRY_ORG`/`SENTRY_PROJECT_FRONTEND`) and the release matches `package.json` version
