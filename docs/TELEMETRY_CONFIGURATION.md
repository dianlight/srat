# Telemetry Configuration Guide

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [Environment Variables](#environment-variables)
  - [Backend (Go)](#backend-go)
  - [Frontend (TypeScript)](#frontend-typescript)
- [Version Management](#version-management)
  - [Backend](#backend)
  - [Frontend](#frontend)
- [Environment Detection](#environment-detection)
  - [Automatic Environment Detection](#automatic-environment-detection)
  - [Manual Override](#manual-override)
- [Build-time Configuration](#build-time-configuration)
  - [Local Development](#local-development)
  - [CI/CD (GitHub Actions)](#cicd-github-actions)
  - [Docker Builds](#docker-builds)
- [Token Types](#token-types)
  - [Unified Rollbar Token](#unified-rollbar-token)
- [Security Considerations](#security-considerations)
- [Fallback Behavior](#fallback-behavior)
- [Testing Configuration](#testing-configuration)
- [Example Configurations](#example-configurations)
  - [Development Setup](#development-setup)
  - [Production CI/CD](#production-cicd)
  - [Prerelease Environment](#prerelease-environment)
  - [Staging Environment](#staging-environment)
- [Troubleshooting](#troubleshooting)
  - [Common Issues](#common-issues)
  - [Debug Steps](#debug-steps)
  - [Version-based Environment Examples](#version-based-environment-examples)
- [Frontend integration notes](#frontend-integration-notes)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

This document explains how to configure Rollbar telemetry tokens, environment, and version information for the SRAT project at build time.

## Overview

The SRAT telemetry system uses Rollbar for error reporting and analytics. Configuration values are set at build time through environment variables and automatically determined from the build context.

## Environment Variables

### Backend (Go)

| Variable                      | Required | Description                                  | Default                                                                   |
| ----------------------------- | -------- | -------------------------------------------- | ------------------------------------------------------------------------- |
| `ROLLBAR_CLIENT_ACCESS_TOKEN` | No       | Rollbar access token (set at build time)     | `""` (disabled)                                                           |
| `ROLLBAR_ENVIRONMENT`         | No       | Rollbar environment name (set at build time) | Auto-detected from version (`development`, `prerelease`, or `production`) |

**Note**: Backend telemetry configuration is set at **build time** via ldflags, not runtime environment variables.

### Frontend (TypeScript)

| Variable                      | Required | Description                                                              | Default    |
| ----------------------------- | -------- | ------------------------------------------------------------------------ | ---------- |
| `ROLLBAR_CLIENT_ACCESS_TOKEN` | No       | Client-side Rollbar access token (injected at build time via Bun define) | `disabled` |

## Version Management

### Backend

The backend version and telemetry configuration are automatically set at build time using Go ldflags:

```bash
-ldflags="-X github.com/dianlight/srat/config.Version=$(VERSION) \
          -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) \
          -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT)"
```

The version is sourced from:

1. `VERSION` environment variable (for CI/CD)
2. Git tags via `git describe --tags --always --abbrev=0 --match='[0-9]*.[0-9]*.[0-9]*'`

The telemetry tokens are read from environment variables at **build time** and embedded into the binary.

### Frontend

The frontend version comes from `package.json` and can be updated by the `scripts/update-frontend-version.sh` script based on Git tags.

## Environment Detection

### Automatic Environment Detection

If `ROLLBAR_ENVIRONMENT` is not set at build time, the environment is automatically determined based on semantic versioning:

- **Development**: Version contains `-dev.` suffix or equals `0.0.0-dev.0`
- **Prerelease**: Version contains `-rc.` suffix (release candidates)
- **Production**: All other versions

### Manual Override

Set `ROLLBAR_ENVIRONMENT` at build time to override automatic detection:

```bash
export ROLLBAR_ENVIRONMENT="staging"
make build
```

## Build-time Configuration

### Local Development

1. **Create `.env` file** (optional):

   ```bash
   # Unified Rollbar token (used for both backend and frontend)
   ROLLBAR_CLIENT_ACCESS_TOKEN=your_rollbar_token_here

   # Override environment (embedded at build time)
   ROLLBAR_ENVIRONMENT=development
   ```

2. **Source environment and build**:

   ```bash
   source .env
   make build
   ```

   Or set variables inline:

   ```bash
   ROLLBAR_CLIENT_ACCESS_TOKEN=token make build
   ```

### CI/CD (GitHub Actions)

Add secrets to your GitHub repository:

1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Add the following secrets:
   - `ROLLBAR_CLIENT_ACCESS_TOKEN` (unified Rollbar token)

The build workflow will automatically use these tokens.

### Docker Builds

Pass environment variables to the Docker build:

```bash
docker build \
  --build-arg ROLLBAR_CLIENT_ACCESS_TOKEN="your_rollbar_token" \
  .
```

## Token Types

### Unified Rollbar Token

- Single token can be used by both backend and frontend
- Simplifies configuration and deployment
- Can be either server-side or client-side token depending on your Rollbar setup
- Required for error reporting from both components

Note: Using a single token simplifies configuration but consider security implications in your specific deployment. If preferred, you can supply different tokens for backend and frontend by customizing your build pipelines.

## Security Considerations

1. **Never commit tokens** to the repository
2. **Use different tokens** for development and production environments
3. **Token will be visible** in both server logs and client JavaScript
4. **Consider token scope** when choosing between server-side and client-side tokens

## Fallback Behavior

When tokens are not configured at build time:

- Backend telemetry service initializes with empty token (disabled)
- Frontend telemetry service initializes with `accessToken: "disabled"`
- No data is sent to Rollbar
- Application functions normally
- Users can still configure telemetry mode in settings (but it won't work without valid tokens)

## Testing Configuration

1. **Check token presence** (before building):

   ```bash
   # Unified token for both backend and frontend
   echo $ROLLBAR_CLIENT_ACCESS_TOKEN
   ```

2. **Build with tokens**:

   ```bash
   ROLLBAR_CLIENT_ACCESS_TOKEN=your_token make build
   ```

3. **Verify version detection**:
   - Check backend logs for "Rollbar telemetry configured" message
   - Check frontend console for telemetry configuration message

4. **Test error reporting**:
   - Enable telemetry in settings
   - Trigger an error
   - Check Rollbar dashboard for received events

## Example Configurations

### Development Setup

```bash
export ROLLBAR_CLIENT_ACCESS_TOKEN="dev_rollbar_token_abc123"
export ROLLBAR_ENVIRONMENT="development"
```

### Production CI/CD

```yaml
env:
  ROLLBAR_CLIENT_ACCESS_TOKEN: ${{ secrets.ROLLBAR_CLIENT_ACCESS_TOKEN }}
  ROLLBAR_ENVIRONMENT: "production"
```

### Prerelease Environment

```bash
export ROLLBAR_CLIENT_ACCESS_TOKEN="prerelease_rollbar_token_def456"
export ROLLBAR_ENVIRONMENT="prerelease"
```

### Staging Environment

```bash
export ROLLBAR_CLIENT_ACCESS_TOKEN="staging_rollbar_token_def456"
export ROLLBAR_ENVIRONMENT="staging"
```

## Troubleshooting

### Common Issues

1. **"Rollbar telemetry disabled"**: Token not set at build time or set to empty string
2. **Version shows as "0.0.0-dev.0"**: Build ldflags not properly configured
3. **Frontend telemetry not working**: `ROLLBAR_CLIENT_ACCESS_TOKEN` not defined at build time
4. **Wrong environment detected**: Set `ROLLBAR_ENVIRONMENT` explicitly at build time

### Debug Steps

1. Check environment variables are set before building
2. Verify tokens are valid in Rollbar dashboard
3. Rebuild with proper environment variables set
4. Check browser network tab for Rollbar API calls
5. Review server logs for telemetry service messages
6. Ensure internet connectivity for Rollbar API

### Version-based Environment Examples

| Version Example    | Detected Environment | Description                  |
| ------------------ | -------------------- | ---------------------------- |
| `1.0.0`            | `production`         | Standard release             |
| `1.0.0-dev.1`      | `development`        | Development build            |
| `1.0.0-rc.1`       | `prerelease`         | Release candidate            |
| `2.1.3-dev.abc123` | `development`        | Development with commit hash |
| `2.1.3-rc.2`       | `prerelease`         | Second release candidate     |
| `0.0.0-dev.0`      | `development`        | Default development version  |

## Frontend integration notes

- Rollbar is provided via `@rollbar/react` provider using `createRollbarConfig()`
- Use `useRollbarTelemetry` hook to send errors/events respecting user-selected mode
- Events are sent only in `All` mode; errors in `All` and `Errors`; no data is sent in `Ask` or `Disabled`

1. Check environment variables are set before building
2. Verify tokens are valid in Rollbar dashboard
3. Rebuild with proper environment variables set
4. Check browser network tab for Rollbar API calls
5. Review server logs for telemetry service messages
6. Ensure internet connectivity for Rollbar API
