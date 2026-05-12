<!-- DOCTOC SKIP -->

---

description: "Mandatory guidance for backend command execution implementation and migration"
applyTo: "backend/src/\*_/_.go"

---

# Backend Command Execution Instructions (Mandatory)

Use these rules whenever backend code executes system commands, migrates legacy `exec` call sites, or emits command execution events.

## Required architecture

- Use `internal/commandexec.Executor` as the default execution path for backend command execution, typically provided by `commandexec.NewCommandExecutor`.
- Prefer `Execute(...)` for synchronous command needs and `Start(...)` for asynchronous/streamed flows.
- Do not introduce new direct `os/exec` usage in services that can use the shared command executor.
- Emit command lifecycle notifications through `events.EventBusInterface` as `events.CommandExecutionEvent`; `BroadcasterService` should subscribe and forward those DTOs instead of being called directly.
- Keep execution correlation by `execution_id` and preserve event ordering:
  1. `command_started`
  2. `command_output` (zero or more)
  3. `command_terminated`

## Event contract requirements

- Use DTOs from `backend/src/dto/command_execution.go` for payloads.
- Keep WebSocket event mapping aligned in `backend/src/dto/webevent_map.go`.
- Do not rename event types without updating enum/map/tests and frontend consumers.

## Safety and observability

- Never use shell interpolation for untrusted input.
- Pass command and args as explicit argv entries.
- Keep bounded output buffering (current contract: latest 500 lines).
- Preserve `stdout` vs `stderr` channel separation.
- Use context-aware logging (`slog.*Context` / `tlog.*Context`) when context is available.

## Migration rules for legacy execution

- Migrate incrementally behind existing service behavior.
- If a migration changes behavior risk, add a focused test first.
- For long-running or progress-driven operations, keep lifecycle semantics explicit and test cancellation/termination behavior.

## Validation requirements

- Backend command execution changes must include targeted tests for:
  - stream emission (`stdout`/`stderr`)
  - termination status (`success`, `exit_code`, `error`)
  - buffer boundaries (tail retention)
- Run at minimum targeted backend tests touching command execution logic before handoff.
