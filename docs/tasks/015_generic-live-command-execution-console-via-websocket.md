<!-- DOCTOC SKIP -->

# [FEATURE]: Generic Live Command Execution Console via WebSocket

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** [Optional GH Issue URL]

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
- **Dependencies:**
  - Backend command execution services in `backend/src/service/` and existing exec call sites (`_TBD_` exact files)
  - Existing backend WebSocket event pipeline (`backend/src/api/ws.go`, `backend/src/events/`)
  - Frontend WebSocket/RTK event handling and notification center components (`frontend/src/`)

## 📝 Task List

- [ ] Task 1: Audit and catalog all backend exec call sites to replace with a generic command runner abstraction
- [ ] Task 2: Implement backend generic command execution service with line-by-line stdout/stderr streaming, bounded ring buffer (500 lines), and exit code/state tracking
- [ ] Task 3: Integrate command stream events into existing WebSocket channel without breaking current event consumers
- [ ] Task 4: Identify the right MUI component pattern for a readonly terminal that can efficiently append lines and visually distinguish stdout vs stderr (e.g., color coding) and also check for existing libraries that can be adapted
- [ ] Task 5: Define and implement a frontend readonly terminal component that can display live command output with distinct styling for stdout vs stderr and supports efficient appending of new lines
- [ ] Task 6: Add Notification Center logic to show stderr alert when no command-output component instance is open
- [ ] Task 7: Implement notification action to open a generic popup showing last 500 lines and exit code/termination status
- [ ] Task 8: Add unit/integration tests for backend command streaming, buffer trimming, and frontend notification/popup behavior
- [ ] Task 9: Integration and documentation updates for architecture, event contract, and operator usage

## 🧠 Implementation Notes (Copilot Context)

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

## 🔗 Code References & TODOs

- [ ] `TODO: audit` - Find and list all `exec` usage in backend to migrate behind generic runner.
- [ ] `TODO: websocket-contract` - Define event names/payload shape for command output and process state.
- [ ] `TODO: ui-component` - Add readonly terminal component in frontend UI library/features.
- [ ] `TODO: notification-action` - Add Notification Center action that opens command-output popup.
- [ ] `FIXME: _TBD_` - Confirm concurrency/backpressure behavior for long-running/high-output commands.
