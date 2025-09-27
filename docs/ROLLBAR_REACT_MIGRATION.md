# Migration to @rollbar/react

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [Changes Made](#changes-made)
  - [1. Package Updates](#1-package-updates)
  - [2. Service Layer Changes](#2-service-layer-changes)
  - [3. New Hook Implementation](#3-new-hook-implementation)
  - [4. Error Boundary Updates](#4-error-boundary-updates)
  - [5. App-Level Integration](#5-app-level-integration)
  - [6. Hook Updates](#6-hook-updates)
- [Benefits](#benefits)
- [Usage Examples](#usage-examples)
  - [Basic Error Reporting](#basic-error-reporting)
  - [Using Rollbar Directly](#using-rollbar-directly)
- [Configuration](#configuration)
- [Telemetry Modes](#telemetry-modes)
- [Migration Verification](#migration-verification)
- [Breaking Changes](#breaking-changes)
- [Future Enhancements](#future-enhancements)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

This document outlines the migration from the regular `rollbar` package to `@rollbar/react` in the SRAT frontend.

## Overview

The migration improves error handling and telemetry reporting by using React-specific patterns and built-in error boundaries provided by @rollbar/react.

## Changes Made

### 1. Package Updates

- **Removed**: `rollbar` package
- **Removed**: `react-use-error-boundary` package
- **Added**: `@rollbar/react@^1.0.0`

### 2. Service Layer Changes

**File**: `src/services/telemetryService.ts`

- Removed direct Rollbar instance management
- Added `createRollbarConfig()` function to generate Rollbar configuration
- Simplified service to only manage telemetry mode state
- Added helper methods: `getIsConfigured()`, `getAccessToken()`

### 3. New Hook Implementation

**File**: `src/hooks/useRollbarTelemetry.ts` (new)

- Combines `useRollbar` hook with telemetry mode logic
- Provides `reportError()` and `reportEvent()` methods that respect telemetry settings
- Returns current telemetry state information

### 4. Error Boundary Updates

**File**: `src/components/ErrorBoundaryWrapper.tsx`

- Replaced custom error boundary with `@rollbar/react`'s `ErrorBoundary` component
- Improved fallback UI with development error details
- Updated `useErrorReporting` hook to use new telemetry system

### 5. App-Level Integration

**File**: `src/index.tsx`

- Added `Provider` from `@rollbar/react` as the root-level provider (wrapping all other providers)
- Added `ErrorBoundaryWrapper` at the top level to catch all application errors
- Structured provider hierarchy: RollbarProvider → ErrorBoundaryWrapper → ThemeProvider → ... → App

**File**: `src/App.tsx`

- Removed RollbarProvider and ErrorBoundaryWrapper (moved to index.tsx)
- Simplified to focus on core application logic
- Cleaned up unused imports related to Rollbar

### 6. Hook Updates

**File**: `src/hooks/useTelemetryInitialization.ts`

- Updated to use new `useRollbarTelemetry` hook
- Enhanced event reporting for telemetry configuration

## Benefits

1. **Better React Integration**: Uses React patterns like Context and hooks
2. **Automatic Error Boundary**: Built-in error boundary handles React errors seamlessly at the application root level
3. **Improved Error Handling**: Better fallback UI and error information
4. **Simplified Code**: Less manual instance management
5. **Type Safety**: Better TypeScript integration with @rollbar/react
6. **Proper Provider Hierarchy**: RollbarProvider at the root ensures all components have access to error reporting

## Usage Examples

### Basic Error Reporting

```typescript
import { useErrorReporting } from "../components/ErrorBoundaryWrapper";

function MyComponent() {
  const { reportError, reportEvent } = useErrorReporting();

  const handleError = (error: Error) => {
    reportError(error, { context: "user-action" });
  };

  const handleEvent = () => {
    reportEvent("user-clicked-button", { buttonId: "save" });
  };
}
```

### Using Rollbar Directly

```typescript
import { useRollbarTelemetry } from "../hooks/useRollbarTelemetry";

function MyComponent() {
  const { reportError, isEnabled, currentMode } = useRollbarTelemetry();

  if (isEnabled) {
    // Telemetry is enabled based on user settings
  }
}
```

## Configuration

The Rollbar configuration is automatically generated based on:

- Environment variables (`ROLLBAR_CLIENT_ACCESS_TOKEN`)
- Package.json version
- Node environment
- User's telemetry mode settings

## Telemetry Modes

The system continues to respect the existing telemetry modes:

- **Ask**: No reporting (user needs to configure)
- **All**: Report both errors and events
- **Errors**: Report only errors
- **Disabled**: No reporting

## Migration Verification

The migration has been tested with:

- ✅ Successful build (`bun run build`)
- ✅ No TypeScript errors
- ✅ All existing functionality preserved
- ✅ Improved error boundary behavior

## Breaking Changes

None for end users. The API for error reporting (`useErrorReporting`) remains the same.

## Future Enhancements

With @rollbar/react in place, future enhancements could include:

1. Advanced error boundary configuration
2. React-specific error context (component stack traces)
3. Performance monitoring integration
4. Custom error filtering and grouping
