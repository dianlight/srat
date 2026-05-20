# [REFACTOR]: Migrate Rollbar to Sentry

**Target Repo:** `srat`
**Status:** � In Progress
**Issue Link:** https://github.com/dianlight/srat/issues/647

## 🎯 Objective

Replace the current Rollbar error-tracking integration (backend Go + React frontend) with [Sentry](https://sentry.io). Sentry provides richer error context, performance tracing, release tracking, and a more developer-friendly dashboard. The migration must preserve the existing four-mode telemetry consent model (`disabled`, `errorsOnly`, `full`, `ask`) and update all related documentation and build infrastructure.

> *Context for Copilot: All references to Rollbar — Go packages, React packages, build-time linker variables, CI secrets, environment configuration, MCP tooling, and documentation — must be replaced with Sentry equivalents without changing the public telemetry behaviour visible to users.*

## 🛠️ Technical Specifications

- **Inputs:**
  - Existing `TelemetryService` backed by `github.com/rollbar/rollbar-go`
  - Frontend `useRollbarTelemetry` hook and `@rollbar/react` provider
  - Build-time vars `config.RollbarToken` / `config.RollbarEnvironment`
  - CI secret `ROLLBAR_CLIENT_ACCESS_TOKEN`
- **Outputs:**
  - `TelemetryService` backed by `github.com/getsentry/sentry-go`
  - Frontend `useSentryTelemetry` hook and `@sentry/react` provider/error-boundary
  - Build-time vars `config.SentryDSN` (single DSN replaces token+env pair)
  - CI secret `SENTRY_DSN` (backend) + `VITE_SENTRY_DSN` (frontend)
- **Dependencies:**
  - Go: `github.com/getsentry/sentry-go` (latest stable)
  - Frontend: `@sentry/react` + `@sentry/vite-plugin` (latest stable)
  - Sentry project: one project per component (`srat-backend`, `srat-frontend`) or a single project with separate environments — _decide before coding_
  - Sentry MCP server: `@sentry/mcp-server` — must be added to the VS Code MCP configuration before implementation starts so Copilot can query Sentry projects, releases, and issues during development

## 📝 Task List

### Setup & Configuration
- [x] Task 1: Add Sentry MCP server (`@sentry/mcp-server`) to `.vscode/mcp.json` (or workspace MCP config) so Copilot can use Sentry tooling during development
- [ ] Task 2: Create Sentry organisation project(s) (`srat-backend`, `srat-frontend`), obtain DSNs for `development`, `prerelease`, and `production` environments; store them as repository secrets (`SENTRY_DSN`, `VITE_SENTRY_DSN`)

### Backend Migration
- [x] Task 3: Add `github.com/getsentry/sentry-go` to `backend/src/go.mod`; vendor it; remove `github.com/rollbar/rollbar-go`
- [x] Task 4: Rename `config.RollbarToken` / `config.RollbarEnvironment` → `config.SentryDSN` in `backend/src/config/version.go`; update all consumers
- [x] Task 5: Rewrite `backend/src/service/telemetry_service.go` — replace `rollbar.*` calls with `sentry.CaptureException`, `sentry.CaptureEvent`, `sentry.Flush`; keep the four-mode consent logic intact
- [x] Task 6: Update `backend/.mise.toml` build flags — replace `--rollbar_env` / `--rollbar_token` args and `-X …RollbarToken` / `-X …RollbarEnvironment` linker vars with `--sentry_dsn` / `-X …SentryDSN`
- [x] Task 7: Update root `.mise.toml` `rollbar_env` task → `sentry_env` or remove if no longer needed; update the `//backend:build` calls

### Frontend Migration
- [x] Task 8: Remove `@rollbar/react` and `rollbar` packages; install `@sentry/react` and `@sentry/vite-plugin`; update `frontend/package.json`
- [x] Task 9: Rename / rewrite `frontend/src/macro/Environment.ts` — replace `getRollbarClientAccessToken()` with `getSentryDsn()`; read `VITE_SENTRY_DSN` env var
- [x] Task 10: Rewrite `frontend/src/hooks/useRollbarTelemetry.ts` → `useSentryTelemetry.ts` — use Sentry `captureException`, `captureEvent`, `addBreadcrumb`; preserve the telemetry-mode guard logic
- [x] Task 11: Rename `frontend/src/components/ConsoleErrorToRollbar.tsx` → `ConsoleErrorToSentry.tsx`; update internals to use the new hook
- [x] Task 12: Update `frontend/src/components/ErrorBoundaryWrapper.tsx` — replace `@rollbar/react`'s `ErrorBoundary` with `@sentry/react`'s `ErrorBoundary`
- [x] Task 13: Update `frontend/src/index.tsx` — remove `RollbarProvider`; initialise Sentry with `Sentry.init(...)` before `ReactDOM.render`; remove `<ConsoleErrorToRollbar />`; add `<ConsoleErrorToSentry />`
- [x] Task 14: Update `frontend/src/components/TelemetryModal.tsx` — replace UI text that references Rollbar with Sentry; update privacy links

### CI / GitHub Actions
- [ ] Task 15: Update `.github/workflows/build.yaml` — replace `ROLLBAR_CLIENT_ACCESS_TOKEN` env var with `SENTRY_DSN` (backend build) and `VITE_SENTRY_DSN` (frontend build); wire Sentry release/version tagging with `@sentry/cli` or the Vite plugin

### Testing
- [x] Task 16: Add / update backend unit tests in `service/telemetry_service_test.go` to cover Sentry integration (mock `sentry-go` transport to assert events are captured)
- [x] Task 17: Update `frontend/src/components/__tests__/DonationButton.test.tsx` and any other test that imports `@rollbar/react` or mocks Rollbar — replace with Sentry mocks
- [x] Task 18: Run full backend test suite (`mise run //backend:test`) and full frontend test suite (`mise run //frontend:test`) — all must pass

### Documentation
- [x] Task 19: Update `PRIVACY.md` — replace all Rollbar references with Sentry; update data-retention and privacy-policy links
- [x] Task 20: Update `docs/TELEMETRY_CONFIGURATION.md` and `docs/TELEMETRY_IMPLEMENTATION.md` — describe Sentry DSN configuration, environments, and release tracking
- [x] Task 21: Update `CHANGELOG.md` — add entry under `[🚧 Unreleased]` describing the migration
- [x] Task 22: Run `mise run docs-validate` (and `mise run docs-fix` if needed) to ensure all documentation passes validation

### Cleanup & Close
- [x] Task 23: Remove any residual Rollbar references (`grep -r rollbar .` should return zero matches outside `CHANGELOG.md` historical entries and this task file)
- [x] Task 24: Run pre-commit checks (`hk check`) — all linters and security scanners must pass
- [x] Task 25: Capture the lessons learned and update documentation / GitHub instructions as appropriate
- [ ] Task 26: Ask to create a PR with the task implementation and link it here for tracking (PR form submitted, awaiting user confirmation)

## 🧠 Agreed Implementation Plan

### Phases
1. **Backend Go** — swap `sentry-go` for `rollbar-go`; rewrite `TelemetryService`; single `SentryDSN` config var; environment inferred at runtime from version string; custom `sentry.EventProcessor` for `tozd/go/errors` stack traces; remove global mutex (Sentry re-init is safe).
2. **Backend tests** — replace `httpmock`+`rollbar.*` with a channel-based custom `sentry.Transport` mock; rename test helpers.
3. **Frontend** — `bun remove rollbar @rollbar/react`; `bun add @sentry/react @sentry/vite-plugin`; `Sentry.init()` at module level (no React Provider); rewrite hook/components/entry.
4. **Build / CI** — `--sentry_dsn` linker flag (single var, no env flag); remove `rollbar_env` task; update CI secret names.
5. **Docs + cleanup** — PRIVACY.md, telemetry docs, CHANGELOG; docs-validate; hk check; grep audit.

### Key Decisions
- Environment **not** injected at build time — inferred at runtime from `config.Version` (already present).
- Sentry Session Replay **deferred** — not included in this migration scope.
- `VITE_SENTRY_DSN` env var (Vite convention) for frontend; `SENTRY_DSN` for backend build.
- DSN placeholder `"disabled"` (empty string disables Sentry natively).

## 🧠 Implementation Notes (Copilot Context)

### Sentry MCP (Task 1)
The Sentry MCP server (`@sentry/mcp-server`) exposes tools for listing projects, querying issues, and posting releases. Add it to the workspace MCP configuration so subsequent tasks can query Sentry during development:

```json
// .vscode/mcp.json (example entry)
{
  "servers": {
    "sentry": {
      "command": "npx",
      "args": ["-y", "@sentry/mcp-server@latest"],
      "env": { "SENTRY_AUTH_TOKEN": "${input:sentryAuthToken}" }
    }
  }
}
```

### Backend (Go) — `sentry-go`
- Initialise with `sentry.Init(sentry.ClientOptions{Dsn: config.SentryDSN, Environment: …, Release: …})`
- `Configure(mode)`: call `sentry.Init(…, {Dsn: ""})` (blank DSN = disabled) for disabled/ask modes; otherwise re-init with real DSN
- Use `sentry.CaptureException(err)` instead of `rollbar.Error(err)`
- Use `sentry.CaptureEvent(&sentry.Event{…})` instead of `rollbar.Info(…)`
- Replace `rollbar.Close()` / `rollbar.Wait()` with `sentry.Flush(2 * time.Second)`
- Remove global `rollbarGlobalMu` / `rollbarGlobalClosed` — Sentry handles reinitialisation safely
- Keep `skipRollbarCloseForTest` pattern but rename to `skipSentryFlushForTest`

### Frontend (React) — `@sentry/react`
- Use `Sentry.init({dsn, environment, release, integrations: [Sentry.browserTracingIntegration()]})` in `index.tsx` before `ReactDOM.render`
- Replace `useRollbar()` / `useRollbarConfiguration()` with direct `Sentry.captureException` / `Sentry.captureMessage`
- Replace `<ErrorBoundary>` from `@rollbar/react` with `<Sentry.ErrorBoundary>`
- The telemetry-mode guard in `useSentryTelemetry` should call `Sentry.getCurrentHub().getClient()?.getOptions().enabled` to check whether Sentry is active
- Use `import.meta.env.VITE_SENTRY_DSN` (Vite convention) instead of the Bun macro; keep macro wrapper for testability

### Build-time DSN injection
Backend: single `-X github.com/dianlight/srat/config.SentryDSN=${SENTRY_DSN}` linker flag — environment is inferred from version at runtime (same logic as current Rollbar environment detection).

Frontend: Vite automatically exposes `VITE_SENTRY_DSN` env var at build time; no macro change needed if the macro simply reads the env.

### Sentry Environments
Keep same three-tier classification:
| Version pattern | Sentry environment |
|---|---|
| `*-dev.*` or `0.0.0-*` | `development` |
| `*-rc.*` | `prerelease` |
| everything else | `production` |

### Testing Strategy
- Backend: use `sentry-go`'s `sentry.NewTestTransport()` (or a custom `sentry.Transport` mock) to capture events without network calls
- Frontend: use `vi.mock("@sentry/react", …)` in Vitest; replace `RollbarProvider` wrappers in existing tests with a no-op Sentry mock

## 🔗 Code References & TODOs

- `backend/src/service/telemetry_service.go` — core service to rewrite
- `backend/src/config/version.go:10-11` — `RollbarToken`, `RollbarEnvironment` vars to rename
- `backend/.mise.toml:33,39,70-71` — build flag definitions to update
- `.mise.toml:99-101,117-130` — `rollbar_env` task to update
- `.github/workflows/build.yaml:390,392` — CI secrets to update
- `frontend/src/hooks/useRollbarTelemetry.ts` — hook to rewrite
- `frontend/src/components/ConsoleErrorToRollbar.tsx` — component to rename/rewrite
- `frontend/src/components/ErrorBoundaryWrapper.tsx` — ErrorBoundary import to swap
- `frontend/src/index.tsx:16,19,104,110,151` — provider wiring to update
- `frontend/src/macro/Environment.ts:13-14` — access token helper to replace
- `frontend/src/components/TelemetryModal.tsx:112` — UI text to update
- `PRIVACY.md:19,44,85,96,114` — Rollbar mentions to replace
