# Generic Live Command Execution Console - Implementation Summary

<!-- DOCTOC SKIP -->

## Overview

SRAT now supports a **generic live command execution console** that streams real-time command output
over the existing WebSocket transport. Backend services can delegate any system command to
`CommandExecutionServiceInterface` and the resulting stdout/stderr lines, timing, and exit status
are automatically broadcast to connected clients with a bounded in-memory buffer.

This feature also introduced a back-end security hardening pass (Task 14) that wrapped all remaining
direct `os/exec` call sites behind injectable abstractions, added an allowlist for Samba-related
user-management commands, and established patterns for deterministic unit testing without process
execution.

---

## Files Created

### Backend

- **`backend/src/service/command_execution_service.go`** - Core streaming service; emits
  `command_started` → `command_output` → `command_terminated` events; bounded 500-line ring buffer
  per execution.
- **`backend/src/service/command_execution_service_test.go`** - Full test suite for streaming,
  termination status, buffer boundaries, and concurrency behaviour.
- **`backend/src/dto/command_execution.go`** - Event payload DTOs
  (`CommandStartedNotification`, `CommandOutputNotification`, `CommandTerminatedNotification`,
  `CommandExecutionSnapshot`).
- **`backend/src/unixsamba/unixsamba_security_test.go`** - Security validation tests for the
  command allowlist and newline-injection rejection.

### Frontend

- **`frontend/src/components/ReadonlyCommandTerminal.tsx`** - Scrollable monospace terminal
  component; renders `CommandOutputLineSnapshot[]`; colours stderr output red.
- **`frontend/src/components/__tests__/ReadonlyCommandTerminal.test.tsx`** - Component tests.
- **`frontend/src/__tests__/App.commandEvents.test.tsx`** - Integration tests for App
  banner/popup/toast rendering on command events.

---

## Files Modified

### Backend

- **`backend/src/dto/webevent_map.go`** - Added `command_started`, `command_output`,
  `command_terminated` event-type registrations.
- **`backend/src/service/server_process_service.go`** - Migrated Samba start/stop/restart/test
  commands to `CommandExecutionServiceInterface`.
- **`backend/src/service/filesystem/base_adapter.go`** - Introduced `filesystemCommandExecutor`
  interface + `defaultFilesystemCommandExecutor`; all exec calls routed through the abstraction.
- **`backend/src/service/filesystem/base_adapter_mock.go`** - Rewrote `SetExecOpsForTesting` to
  override the `commandExecutor` field via `testFilesystemCommandExecutor` struct.
- **`backend/src/internal/osutil/osutil.go`** - Added injectable `sambaVersionExec` hook +
  `MockSambaVersionExec` helper for deterministic testing.
- **`backend/src/unixsamba/unixsamba.go`** - Added `allowedUnixSambaCommands` allowlist +
  `validateUnixSambaCommand` guard on both execution paths.
- **`backend/src/internal/appsetup/appsetup.go`** - Registered `CommandExecutionServiceInterface`
  in Fx graph.

### Frontend

- **`frontend/src/App.tsx`** - Added command-event WS subscription; toast on start; popup with
  `ReadonlyCommandTerminal` and buffered output on completion.

### Documentation & Instructions

- **`.github/instructions/backend-command-execution.instructions.md`** - New mandatory instruction
  file; rules for all present and future back-end command execution and migration.
- **`docs/EVENT_DRIVEN_ARCHITECTURE.md`** - Added command execution event section (event contract,
  flow, payload shapes).
- **`docs/EVENT_SYSTEM_QUICK_REFERENCE.md`** - Added command execution quick-reference entry.

---

## Architecture

```text
backend service
    └─ CommandExecutionServiceInterface.Start(ctx, commandID, label, command, args...)
            │  emits
            ├─ CommandStartedNotification    → BroadcasterService → WebSocket → frontend
            ├─ CommandOutputNotification ×N  → BroadcasterService → WebSocket → frontend
            └─ CommandTerminatedNotification → BroadcasterService → WebSocket → frontend
```

- **`commandID`** - callers supply a stable identifier (e.g. `"samba:restart"`) that the frontend
  uses to correlate runs across reconnects.
- **`execution_id`** - a UUID generated per invocation; scopes all output and termination events
  for that run.
- **Buffer** - latest 500 lines are kept in `CommandExecutionSnapshot.Lines`; older lines are
  discarded. New clients joining mid-run receive the buffered snapshot.

---

## Security Hardening (Task 14)

All remaining direct `os/exec` usages are confined to default adapter implementations hidden behind
interfaces. No call site exposes shell interpolation or user-controlled command path selection.

- **`unixsamba`** - `defaultCommandExecutor` implementation guarded by
  `allowedUnixSambaCommands` allowlist (pdbedit/useradd/smbpasswd/deluser/usermod) and
  newline-in-arg rejection on every call.
- **`internal/osutil`** - `sambaVersionExec` injectable hook; default implementation runs
  only `samba --version` with no caller-controlled arguments.
- **`service/filesystem`** - `filesystemCommandExecutor` interface; all concrete adapters
  supply hardcoded command names; no user input reaches the exec path.
- **`service`** - `CommandExecutionService` passes `command` and `args...` as discrete argv
  entries (no shell interpolation); callers are internal backend services only.

---

## Testing

```bash
# Core service
go test ./service -run TestCommandExecutionService

# Filesystem abstraction
go test ./service/filesystem -run 'TestBaseAdapterTestSuite|TestNtfsAdapterTestSuite|TestExt4AdapterTestSuite'

# Unixsamba security + behaviour
go test ./unixsamba -run 'TestUnixSambaTestSuite|TestValidateUnixSambaCommand'

# Osutil injection
go test ./internal/osutil -run TestOsutilSuite

# Frontend
mise run //frontend:test -- src/components/__tests__/ReadonlyCommandTerminal.test.tsx
mise run //frontend:test -- src/__tests__/App.commandEvents.test.tsx
```

---

## Related Task

- Task doc: `docs/tasks/015_generic-live-command-execution-console-via-websocket.md`
- GitHub issue: #540
