<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** _generated with [DocToc](https://github.com/thlorenz/doctoc)_

- [Telemetry Configuration Guide](#telemetry-configuration-guide)
  - [Overview](#overview)
  - [Environment Variables](#environment-variables)
    - [Server (Go)](#server-go)
    - [Frontend (TypeScript)](#frontend-typescript)
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
- **Frontend**: `VITE_SENTRY_DSN` is injected at build time and read by the frontend macro layer

Environment (`development`, `prerelease`, `production`) is detected at runtime from the version string.

## Environment Variables

### Server (Go)

| Variable     | Required | Description                                | Default    |
| ------------ | -------- | ------------------------------------------ | ---------- |
| `SENTRY_DSN` | No       | Server Sentry DSN (embedded at build time) | `disabled` |

### Frontend (TypeScript)

| Variable          | Required | Description                                  | Default    |
| ----------------- | -------- | -------------------------------------------- | ---------- |
| `VITE_SENTRY_DSN` | No       | Frontend Sentry DSN (injected at build time) | `disabled` |

## Build-Time Configuration

### Server linker flags

The server build task sets:

- `-X github.com/dianlight/srat/config.SentryDSN=${SENTRY_DSN:-disabled}`

Version metadata is also embedded at build time and used for environment detection.

### Frontend build injection

Set `VITE_SENTRY_DSN` in your environment before frontend build.

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
- `VITE_SENTRY_DSN`

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
