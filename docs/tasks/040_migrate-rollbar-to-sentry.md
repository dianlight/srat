# [REFACTOR]: Migrate Rollbar to Sentry

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _TBD_

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
- [ ] Task 1: Add Sentry MCP server (`@sentry/mcp-server`) to `.vscode/mcp.json` (or workspace MCP config) so Copilot can use Sentry tooling during development
- [ ] Task 2: Create Sentry organisation project(s) (`srat-backend`, `srat-frontend`), obtain DSNs for `development`, `prerelease`, and `production` environments; store them as repository secrets (`SENTRY_DSN`, `VITE_SENTRY_DSN`)

### Backend Migration
- [ ] Task 3: Add `github.com/getsentry/sentry-go` to `backend/src/go.mod`; vendor it; remove `github.com/rollbar/rollbar-go`
- [ ] Task 4: Rename `config.RollbarToken` / `config.RollbarEnvironment` → `config.SentryDSN` in `backend/src/config/version.go`; update all consumers
- [ ] Task 5: Rewrite `backend/src/service/telemetry_service.go` — replace `rollbar.*` calls with `sentry.CaptureException`, `sentry.CaptureEvent`, `sentry.Flush`; keep the four-mode consent logic intact
- [ ] Task 6: Update `backend/.mise.toml` build flags — replace `--rollbar_env` / `--rollbar_token` args and `-X …RollbarToken` / `-X …RollbarEnvironment` linker vars with `--sentry_dsn` / `-X …SentryDSN`
- [ ] Task 7: Update root `.mise.toml` `rollbar_env` task → `sentry_env` or remove if no longer needed; update the `//backend:build` calls

### Frontend Migration
- [ ] Task 8: Remove `@rollbar/react` and `rollbar` packages; install `@sentry/react` and `@sentry/vite-plugin`; update `frontend/package.json`
- [ ] Task 9: Rename / rewrite `frontend/src/macro/Environment.ts` — replace `getRollbarClientAccessToken()` with `getSentryDsn()`; read `VITE_SENTRY_DSN` env var
- [ ] Task 10: Rewrite `frontend/src/hooks/useRollbarTelemetry.ts` → `useSentryTelemetry.ts` — use Sentry `captureException`, `captureEvent`, `addBreadcrumb`; preserve the telemetry-mode guard logic
- [ ] Task 11: Rename `frontend/src/components/ConsoleErrorToRollbar.tsx` → `ConsoleErrorToSentry.tsx`; update internals to use the new hook
- [ ] Task 12: Update `frontend/src/components/ErrorBoundaryWrapper.tsx` — replace `@rollbar/react`'s `ErrorBoundary` with `@sentry/react`'s `ErrorBoundary`
- [ ] Task 13: Update `frontend/src/index.tsx` — remove `RollbarProvider`; initialise Sentry with `Sentry.init(...)` before `ReactDOM.render`; remove `<ConsoleErrorToRollbar />`; add `<ConsoleErrorToSentry />`
- [ ] Task 14: Update `frontend/src/components/TelemetryModal.tsx` — replace UI text that references Rollbar with Sentry; update privacy links

### CI / GitHub Actions
- [ ] Task 15: Update `.github/workflows/build.yaml` — replace `ROLLBAR_CLIENT_ACCESS_TOKEN` env var with `SENTRY_DSN` (backend build) and `VITE_SENTRY_DSN` (frontend build); wire Sentry release/version tagging with `@sentry/cli` or the Vite plugin

### Testing
- [ ] Task 16: Add / update backend unit tests in `service/telemetry_service_test.go` to cover Sentry integration (mock `sentry-go` transport to assert events are captured)
- [ ] Task 17: Update `frontend/src/components/__tests__/DonationButton.test.tsx` and any other test that imports `@rollbar/react` or mocks Rollbar — replace with Sentry mocks
- [ ] Task 18: Run full backend test suite (`mise run //backend:test`) and full frontend test suite (`mise run //frontend:test`) — all must pass

### Documentation
- [ ] Task 19: Update `PRIVACY.md` — replace all Rollbar references with Sentry; update data-retention and privacy-policy links
- [ ] Task 20: Update `docs/TELEMETRY_CONFIGURATION.md` and `docs/TELEMETRY_IMPLEMENTATION.md` — describe Sentry DSN configuration, environments, and release tracking
- [ ] Task 21: Update `CHANGELOG.md` — add entry under `[🚧 Unreleased]` describing the migration
- [ ] Task 22: Run `mise run docs-validate` (and `mise run docs-fix` if needed) to ensure all documentation passes validation

### Cleanup & Close
- [ ] Task 23: Remove any residual Rollbar references (`grep -r rollbar .` should return zero matches outside `CHANGELOG.md` historical entries and this task file)
- [ ] Task 24: Run pre-commit checks (`hk check`) — all linters and security scanners must pass
- [ ] Task 25: Capture the lessons learned and update documentation / GitHub instructions as appropriate
- [ ] Task 26: Ask to create a PR with the task implementation and link it here for tracking

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
