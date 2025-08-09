# Rollbar Telemetry Implementation

This document describes the implementation of Rollbar telemetry and error reporting with configurable privacy modes.

## Backend Environment Variables

- `ROLLBAR_CLIENT_ACCESS_TOKEN`: Unified Rollbar access token (embedded at build time via ldflags)
- `ROLLBAR_ENVIRONMENT`: Override automatic environment detection (embedded at build time via ldflags)
- Version is automatically set from `config.Version` (configured via build ldflags)
- Environment auto-detected: "development" for dev versions, "production" for releases
- Security: Tokens are embedded at build time, not read from runtime environment
- Simplification: Same token can be used for both backend and frontend

## Overview

The telemetry system provides four configuration modes:

- **Ask** (default): User hasn't chosen a preference yet
- **All**: Send errors and telemetry events to Rollbar servers
- **Errors**: Send only errors to Rollbar servers
- **Disabled**: Don't use Rollbar at all

## Features Implemented

### Backend (Go)

1. **TelemetryMode Enum** (`dto/telemetry_mode.go`)
   - Generated enum with values: Ask, All, Errors, Disabled
   - Uses goenums for type safety

2. **TelemetryService** (`service/telemetry_service.go`)
   - Manages Rollbar configuration based on mode
   - Internet connectivity checking
   - Error reporting and event tracking
   - Graceful shutdown

3. **Configuration Integration**
   - Added `telemetry_mode` to Config struct
   - Added migration from config version 3 to 4
   - Converter integration for Settings DTO

4. **API Endpoints**
   - `GET /telemetry/modes` - Returns available telemetry modes
   - `GET /telemetry/internet-connection` - Checks internet connectivity
   - Updated `PUT /settings` to configure telemetry service

5. **Dependency Injection**
   - TelemetryService registered in FX container
   - Integrated with existing settings handler

### Frontend (React/TypeScript)

1. **TelemetryService** (`services/telemetryService.ts`)
   - Singleton service managing Rollbar client
   - Mode-based configuration
   - Error and event reporting
   - TypeScript types for mode safety

2. **useRollbarTelemetry hook** (`hooks/useRollbarTelemetry.ts`)
   - Thin wrapper around `@rollbar/react` that honors current telemetry mode
   - `reportError(error, extraData?)`: sends errors when mode is `All` or `Errors`
   - `reportEvent(event, data?)`: sends events only when mode is `All`
   - Safe no-ops if telemetry is not configured or disabled

3. **TelemetryModal** (`components/TelemetryModal.tsx`)
   - Modal dialog for initial telemetry preference selection
   - Displays only when mode is "Ask" and internet is available
   - Cannot be dismissed without making a choice
   - Includes privacy disclaimer and Rollbar information

4. **Settings Integration** (`pages/Settings.tsx`)
   - Added telemetry mode field to settings form
   - Disabled when no internet connection
   - Excludes "Ask" from user-selectable options
   - Shows helper text when internet is unavailable

5. **Error Boundary Integration** (`components/ErrorBoundaryWrapper.tsx`)
   - Automatic error reporting to telemetry service
   - Additional context (user agent, URL, timestamp)
   - Manual error/event reporting hooks

6. **Hooks**
   - `useTelemetryModal.ts` - Determines when to show telemetry modal
   - `useTelemetryInitialization.ts` - Initializes service on app load
   - `useErrorReporting.ts` - Manual error reporting
   - `useRollbarTelemetry.ts` - Rollbar wrapper honoring telemetry modes

#### Frontend usage examples

```tsx
import { useRollbarTelemetry } from "../hooks/useRollbarTelemetry";

export function SaveButton() {
  const { reportEvent, reportError } = useRollbarTelemetry();

  const onClick = async () => {
    try {
      reportEvent("save_clicked", { source: "toolbar" });
      // ... perform save
    } catch (err) {
      reportError(err instanceof Error ? err : String(err), { action: "save" });
    }
  };

  return <button onClick={onClick}>Save</button>;
}
```

## User Experience

### First Launch

1. If user hasn't set telemetry preference (mode = "Ask")
2. And internet connection is available
3. Modal dialog appears requiring user to choose preference
4. User cannot proceed without making a choice
5. Choice is saved and telemetry service is configured accordingly

### Settings Page

1. Telemetry Mode field available in settings
2. Field disabled if no internet connection
3. Helper text explains internet requirement
4. "Ask" option not available for selection
5. Changes take effect immediately

### Error Handling

1. Uncaught errors automatically reported (if enabled)
2. Manual error reporting available via hooks
3. Telemetry events reported for feature usage (if All mode)
4. No data sent in Ask or Disabled modes

## Configuration

### Backend Environment Variables

- `ROLLBAR_ACCESS_TOKEN`: Server-side Rollbar access token (embedded at build time via ldflags)
- `ROLLBAR_ENVIRONMENT`: Override automatic environment detection (embedded at build time via ldflags)
- Version is automatically set from `config.Version` (configured via build ldflags)
- Environment auto-detected: "development" for dev versions, "production" for releases
- **Security**: Tokens are embedded at build time, not read from runtime environment

### Frontend Configuration

- `ROLLBAR_CLIENT_ACCESS_TOKEN`: Client-side Rollbar access token (set at build time)
- Environment detection via `NODE_ENV` (development/production)
- Version automatically sourced from `package.json`
- Build system injects environment variables via `define` in bun.build.ts

## Privacy Compliance

1. **Transparent**: Modal clearly explains what data is collected
2. **Consent**: User must explicitly choose their preference
3. **Control**: Settings can be changed at any time
4. **Minimal**: Only errors/usage data, no personal information
5. **Conditional**: Requires internet connection to enable

## Migration Path

Existing installations will:

1. Have `telemetry_mode` set to "Ask" after upgrade
2. Show telemetry modal on next UI access (if internet available)
3. Function normally if user chooses "Disabled"
4. Provide improved error reporting if user opts in

## Security Considerations

1. All data sent over HTTPS to Rollbar servers
2. No sensitive data (passwords, file contents) transmitted
3. Error context limited to technical information
4. User can disable at any time
5. Internet connectivity required prevents accidental data transmission

## Dependencies Added

### Backend

- `github.com/rollbar/rollbar-go` v1.4.8

### Frontend

- `rollbar` v2.26.4

## Files Created/Modified

### Backend

- `dto/telemetry_mode.go` (new)
- `dto/telemetrymodes_enums.go` (generated)
- `service/telemetry_service.go` (new)
- `api/telemetry.go` (new)
- `dto/settings.go` (modified)
- `config/addon_json_config.go` (modified)
- `converter/config_to_dto.go` (modified)
- `api/setting.go` (modified)
- `internal/appsetup/appsetup.go` (modified)
- `cmd/srat-server/main-server.go` (modified)
- `cmd/srat-openapi/main-openapi.go` (modified)

### Frontend

- `services/telemetryService.ts` (new)
- `components/TelemetryModal.tsx` (new)
- `components/ErrorBoundaryWrapper.tsx` (new)
- `hooks/useTelemetryModal.ts` (new)
- `hooks/useTelemetryInitialization.ts` (new)
- `pages/Settings.tsx` (modified)
- `App.tsx` (modified)
- `store/sratApi.ts` (regenerated)

## Testing

To test the implementation:

1. **Build and run backend**: `go run ./cmd/srat-server` with appropriate flags
2. **Build and serve frontend**: `bun run build && bun run dev`
3. **Test scenarios**:
   - Fresh install (should show telemetry modal)
   - Settings page telemetry field
   - Error reporting with different modes
   - Internet connectivity checks
   - Mode switching behavior

The implementation provides a complete, privacy-respecting telemetry system that helps improve the software while giving users full control over their data.
