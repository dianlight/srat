<!-- DOCTOC SKIP -->

# [FEATURE]: Generic Live Command Execution Console via WebSocket

**Target Repo:** `srat`  **Status:** 🔄 In Progress  **Issue Link:** https://github.com/dianlight/srat/issues/540

## 🎯 Objective

Implement a generic backend/frontend system that executes commands on the backend and streams live command output to the UI through the existing WebSocket channel, while preserving current UI behavior and replacing direct backend exec usage behind the new execution abstraction.

> *Context for Copilot: Provide a reusable command-execution pipeline with real-time stdout/stderr streaming, readonly terminal visualization, stderr-driven notifications, and a popup to inspect the latest output/exit status.*

## 🛠️ Technical Specifications

- **Inputs:**
  - Command execution request (command id/type + args/options)
  - Backend command output streams (`stdout`, `stderr`)
  - Existing WebSocket event channel messages
- **Outputs:**
  - Live stream events to frontend with separated `stdout`/`stderr` lines
  - Readonly terminal UI component with color-separated output
  - Notification Center entry when `stderr` arrives and no terminal instance is open
  - Popup showing last 500 lines and exit code when process has terminated
  - The Popup should also be accessible via a notification action button when a `stderr` line is received and no terminal viewer is open.
  - The Popup has a Download button to download the full output as a text file.
- **Dependencies:**
  - Backend command execution services in `backend/src/service/` and existing exec call sites (`_TBD_` exact files)
  - Existing backend WebSocket event pipeline (`backend/src/api/ws.go`, `backend/src/events/`)
  - Frontend WebSocket/RTK event handling and notification center components (`frontend/src/`)

## 📝 Task List

- [x] Task 1: Audit and catalog all backend exec call sites (completed — see code references for migration matrix)
- [x] Task 2: Define WebSocket event names and payload schema for streaming command output and process lifecycle events (e.g., `command_output`, `command_started`, `command_terminated`) with necessary metadata (execution id, timestamp, channel, exit code).
- [x] Task 3: Find the right MUI component pattern (lightweight in-memory buffering + readonly pre tag + MUI Dialog MVP)
- [x] Task 4: Design the frontend readonly terminal component pattern (implemented in App.tsx with 500-line buffer per execution)
- [x] Task 5: Implement backend generic command execution service with line-by-line stdout/stderr streaming, bounded ring buffer (500 lines), and exit code/state tracking
- [x] Task 6: Integrate the command execution service with the WebSocket event system to emit real-time updates to the frontend, ensuring that existing event consumers are not disrupted.
- [x] Task 7: Design the frontend readonly terminal component pattern that can efficiently append lines and visually distinguish `stdout` vs `stderr` (e.g., color coding). Define the component API for receiving streamed events and updating the display.
- [x] Task 8: Rerun all existing backend and frontend tests (backend ✓ all suites, frontend ✓ 613 pass)
- [x] Task 9: Add Notification Center logic to show stderr alert when no command-output component instance is open
- [x] Task 10: Implement notification action to open a generic popup showing last 500 lines and exit code/termination status
- [x] Task 11: Add unit/integration tests for backend command streaming, buffer trimming, and frontend notification/popup behavior
- [x] Task 12: Integration and documentation updates for architecture, event contract, and operator usage
- [x] Task 13: Create a Github copilot instruction that guide how use the execution system and how implement. Make it a required instruction to follow always when there is an execution on the backend.
- [ ] Task 14: Migrate any remaining exec usage in backend to the new command runner abstraction, ensuring that all command executions benefit from the new streaming and event capabilities. If the actual exec use cases don't have a specific test, create the test for it and then migrate to the new command runner abstraction. Perform a final code review and security audit to ensure that there are no risks of command injection or sensitive data exposure in the new implementation.
- [ ] Task 15: Final code review code cleanup, and documentation updates to ensure that the new feature is well-documented for future maintainers and users.
- [ ] Task 16: Mark the task as complete and prepare for release. 
- [ ] Task 17: Create a PR with the implementation and link it to this task for tracking. Ensure that the PR description clearly outlines the changes made, the new features added, and any important notes for reviewers. 

## 🧠 Implementation Notes (Copilot Context)

- Agreed implementation sequence:
  - Define backend event contract first (`command_started`, `command_output`, `command_terminated`) in DTO enums/map before wiring services.
  - Introduce a reusable command runner service with execution id, line-by-line streaming, bounded ring buffer (500 lines), and termination metadata.
  - Integrate command events into existing event bus → broadcaster → WebSocket pipeline without creating a parallel transport.
  - Migrate initial high-value call sites in `server_process_service.go` (test/restart/start/stop commands) behind the new abstraction, preserving current behavior.
  - Add frontend event consumption and readonly terminal/popup/notification flow after backend contract and emission are stable.
  - Validate with focused backend + frontend tests, then broaden to full regression.
- Initial risks to handle explicitly:
  - High-output backpressure and event ordering between output lines and termination events.
  - Multi-execution concurrency isolation by execution id.
  - Command safety (no shell interpolation), error redaction, and bounded in-memory retention.

- Design a reusable backend command runner interface (for example: `Start`, `Subscribe`, `GetSnapshot`, `Stop`) and adapt existing exec usage to it incrementally.
- Keep backward compatibility by preserving current functionality and event semantics where possible.
- Stream output line-by-line and tag each line with source channel (`stdout` or `stderr`), timestamp, and execution id.
- Maintain an in-memory bounded buffer of the latest 500 lines per execution for popup retrieval.
- Include lifecycle events (`started`, `running`, `terminated`) with process metadata and final `exit_code`.
- Use existing WebSocket infrastructure/events to deliver command updates rather than introducing a parallel transport.
- Frontend terminal component must be readonly and optimized for high-frequency line append.
- Notification rule (first implementation): if a `stderr` line arrives and no terminal viewer is open, create a notification with an action button to open the popup viewer.
- Popup viewer should load/show: execution id, command label, latest 500 lines, and final/ongoing exit status.
- Define and document a stable event payload contract shared by backend and frontend (`_TBD_` final schema).
- Implemented milestone:
  - Added DTO payloads for `command_started`, `command_output`, `command_terminated` in `backend/src/dto/command_execution.go`.
  - Extended websocket event enum/map registration in `backend/src/dto/webevent_type.go`, generated `backend/src/dto/webeventtypes_enums.go`, and `backend/src/dto/webevent_map.go`.
  - Added/updated backend tests in `backend/src/dto/webevent_type_test.go` and `backend/src/dto/enums_test.go`.
  - Added `backend/src/service/command_execution_service.go` with:
    - async `Start(...)` execution,
    - sync `Execute(...)` wrapper for incremental migration,
    - line-by-line stdout/stderr streaming,
    - 500-line bounded in-memory snapshot buffering,
    - termination status/exit-code tracking.
  - Wired service in FX (`backend/src/internal/appsetup/appsetup.go`) and migrated first `server_process_service.go` call sites (`GetSambaStatus`, `testSambaConfig`, restart/start/stop commands) to use the runner with a fallback path.
  - Added focused service tests in `backend/src/service/command_execution_service_test.go`.
  - Updated frontend websocket parsing in `frontend/src/store/wsApi.ts` to accept `command_started`, `command_output`, and `command_terminated` events without modifying generated `sratApi.ts`.
  - Added first-pass command output UX in `frontend/src/App.tsx`:
    - local 500-line per-execution buffering in UI state,
    - stderr toast when popup is closed,
    - toast action button opening command popup,
    - popup showing execution id/status/exit code and readonly output,
    - download button exporting captured output as `.txt`.
  - Validated frontend ws layer and app integration via `bun test src/store/__tests__/wsApi.test.tsx src/__tests__/App.test.tsx`.
  - Implemented Task 7 terminal component API in `frontend/src/components/ReadonlyCommandTerminal.tsx`:
    - Props: `lines`, `maxHeight`, `emptyText`
    - Visual distinction: `stderr` lines render with `error.main`, `stdout` with `text.primary`
    - Integrated into popup in `frontend/src/App.tsx`
    - Validated with `bun test src/components/__tests__/ReadonlyCommandTerminal.test.tsx src/__tests__/App.test.tsx`
  - Implemented Task 11 focused test coverage updates:
    - Backend: strengthened `backend/src/service/command_execution_service_test.go` to assert `CommandOutputNotification` broadcast channels include both `stdout` and `stderr`.
    - Frontend: added `frontend/src/__tests__/App.commandEvents.test.tsx` to verify:
      - stderr output triggers toast notification when dialog is closed,
      - toast action opens command-output popup,
      - popup shows execution id, stderr line, and terminated failed status.
    - Stabilized popup state fields in `frontend/src/App.tsx` (`execution_id`, `command_id`, `exit_code`, `finished_at`) for correct UI rendering.
    - Validation:
      - `go test ./service -run TestCommandExecutionServiceTestSuite` ✓
      - `bun test src/__tests__/App.commandEvents.test.tsx src/components/__tests__/ReadonlyCommandTerminal.test.tsx` ✓
  - Implemented Task 12 documentation integration updates:
    - Updated `docs/EVENT_DRIVEN_ARCHITECTURE.md` with command execution architecture notes, lifecycle ordering (`command_started` → `command_output` → `command_terminated`), and explicit WebSocket contract field references.
    - Updated `docs/EVENT_SYSTEM_QUICK_REFERENCE.md` with WebSocket-only event flow wording and operator-facing command console usage/troubleshooting notes.
    - Validation:
      - `npx --yes markdownlint-cli2 /workspaces/srat/docs/EVENT_DRIVEN_ARCHITECTURE.md /workspaces/srat/docs/EVENT_SYSTEM_QUICK_REFERENCE.md` ✓
  - Implemented Task 13 mandatory Copilot instruction for backend execution:
    - Added `.github/instructions/backend-command-execution.instructions.md` with required architecture, event contract, safety, migration, and validation rules for backend command execution work.
    - Updated `.github/copilot-instructions.md` to explicitly require this instruction whenever backend execution is implemented/migrated.
    - Updated `AGENTS.md` required-reading list to include the new instruction file.

  **Phase 1 status (current as of 2026-04-02):**
  ✅ All core infrastructure implemented and validated:
  - Backend event contract (command_started, command_output, command_terminated) fully registered with WS pipeline.
  - CommandExecutionService with async/sync API, line-by-line streaming, 500-line ring buffer, complete test coverage.
  - Initial migration: ServerProcessService (GetSambaStatus, testSambaConfig, service start/restart/stop, lifecycle OnStop).
  - Frontend: WS event parsing (wsApi.ts), command session state management, stderr notification toast + popup dialog with download.
  - Full regression: backend go test ./... ✓, frontend bun test ✓ (613 pass).
  - Branch: feature/generic-live-command-execution-console-via-websocket, Issue: #540.

  **Next steps (Phase 2):**
  - [ ] Task 10: Edge case tests (buffer trimming under load, concurrent executions, long-running streams).
  - [ ] Task 11: Enhanced integration tests + documentation of event schema and architecture.
  - [ ] Task 12: Remaining exec migrations (osutil version probing as quick win; filesystem adapters deferred to Phase 3).
  - [ ] Task 13-15: Code review, changelog, PR, release prep.
## 🔗 Code References & TODOs

- [ ] `TODO: audit` - Backend `exec` migration inventory:
  - `backend/src/service/server_process_service.go` (`GetSambaStatus`, `testSambaConfig`, `restartServerServices`, `OnStop` shutdown flow)
  - `backend/src/service/filesystem/base_adapter.go` (`runCommand`, `executeCommandWithProgress`) and adapters calling it
  - `backend/src/unixsamba/unixsamba.go` (`defaultCommandExecutor.RunCommand` + user-management call sites)
  - `backend/src/internal/osutil/osutil.go` (Samba version probing)
- [x] `TODO: websocket-contract` - Contract definition/wiring touchpoints:
  - `backend/src/dto/webevent_type.go`
  - `backend/src/dto/webeventtypes_enums.go` (generated)
  - `backend/src/dto/webevent_map.go`
  - `backend/src/service/broadcaster_service.go`
  - `backend/src/api/ws.go`
- [ ] `TODO: ui-component` - Frontend consumer + presentation touchpoints:
  - `frontend/src/store/wsApi.ts`
  - `frontend/src/components/ReadonlyCommandTerminal.tsx`
  - `frontend/src/__tests__/App.commandEvents.test.tsx`
  - `frontend/src/components/__tests__/ReadonlyCommandTerminal.test.tsx`
  - `frontend/src/components/NotificationCenter.tsx`
  - `frontend/src/components/PreviewDialog.tsx`
  - `frontend/src/pages/**` (placement decision for terminal viewer and popup trigger)
- [x] `TODO: docs` - Architecture/event-contract/operator docs touchpoints:
  - `docs/EVENT_DRIVEN_ARCHITECTURE.md`
  - `docs/EVENT_SYSTEM_QUICK_REFERENCE.md`
- [x] `TODO: instructions` - Backend execution guidance touchpoints:
  - `.github/instructions/backend-command-execution.instructions.md`
  - `.github/copilot-instructions.md`
  - `AGENTS.md`
- [ ] `TODO: notification-action` - Add action workflow from stderr notification to popup open state + selected execution id.
- [ ] `FIXME: backpressure` - Confirm queue/ring-buffer strategy for long-running or high-output command streams and define drop/truncate policy.

- [x] AUDIT COMPLETE - Backend exec migration inventory by phase:
