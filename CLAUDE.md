<!-- DOCTOC SKIP -->

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SRAT (SambaNAS REST Administration Tool) simplifies SAMBA configuration with a REST API and React UI, designed primarily for Home Assistant addons. It is a monorepo with three workspaces managed by [mise](https://mise.jdx.dev):

- **`backend/`** - Go 1.26 REST API server + CLI
- **`frontend/`** - TypeScript React UI (Bun toolchain)
- **`custom_components/`** - Python 3.14 Home Assistant integration

## Instruction and Skill Files

Always consult the specialized files before modifying code:

- **`.github/copilot-instructions.md`** - top-level non-negotiable rules
- **`.github/instructions/*.instructions.md`** - per-language/scenario rules (Go, TypeScript, React, testing, back end command execution, HA)
- **`.github/skills/*.SKILL.md`** - task workflows (start-task, create-pr, prepare-refactor, update-changelog, etc.)

**Critical rule**: Read the top comment/header of any file you modify first - file-specific overrides everything else.

## Build, Test, and Lint Commands

All workflows run via `mise`:

```sh
# Full test suite
mise run test

# Backend
mise run //backend:build
mise run //backend:test
mise run //backend:lint
mise run //backend:format
mise run //backend:gen          # regenerate converters + openapi

# Frontend
mise run //frontend:build
mise run //frontend:dev
mise run //frontend:test
mise run //frontend:lint
mise run //frontend:gen         # regenerate sratApi.ts from openapi

# Custom component
mise run //custom_components:check
mise run //custom_components:test
mise run //custom_components:lint
mise run //custom_components:format
mise run //custom_components:typecheck
mise run //custom_components:hassfest   # validates HA integration structure

# Quality gates (run before finalizing changes)
hk check          # pre-commit checks (linters, formatters)
hk fix            # auto-fix pre-commit issues
mise run security # gosec backend security scan
mise run docs-validate
mise run docs-fix
```

### Running a single backend test

```sh
cd backend/src && go test ./path/to/package -run TestFunctionName
# Escalation: specific subtest → full package → go test ./...
```

### Running frontend tests with stability check

```sh
cd frontend && bun run test
# For flaky tests: bun run test --rerun-each 10
```

## Architecture

### Backend (Go 1.26)

**Request flow**: API handler (Huma v2) → Service → GORM + generated query helpers (`dbom/g/`) → SQLite

Key packages under `backend/src/`:

- `cmd/` - three binaries: `srat-server`, `srat-cli`, `srat-openapi`
- `api/` - Huma HTTP handlers; each file has paired `*_test.go`
- `service/` - business logic; injected via `go.uber.org/fx`
- `dto/` - data transfer objects shared between API and services
- `dbom/` - GORM database models + migrations (`dbom/migrations/`)
- `dbom/g/` - generated GORM query helpers (do not hand-edit)
- `converter/` - generated DTO↔DBOM converters via `jmattheis/goverter`
- `events/` - `EventBusInterface` for internal pub/sub
- `internal/commandexec/` - shared command execution abstraction (mandatory for all shell commands)

**Non-negotiable patterns**:

- Use `converter.<Type>ToDtoConverterImpl{}` for all DTO↔DBOM mapping - never write manual `toDTO`/`toDBOM` helpers.
- Backend command execution must use `commandexec.Executor`; emit lifecycle events (`command_started` → `command_output` → `command_terminated`) via `EventBusInterface`. See `.github/instructions/backend-command-execution.instructions.md`.
- Use `slog.*Context` / `tlog.*Context` for logging when a real `context.Context` is in scope.
- Go 1.26 idioms: `new(expr)` for pointer values, `any` not `interface{}`, `WaitGroup.Go`, `errors.AsType[T]`.
- Do not edit `backend/src/vendor/` directly - use patch workflow (`backend/patches/` + `mise run //backend:patch`).

### Frontend (TypeScript 6 / React 19 / Bun)

**Data flow**: Components → RTK Query via `sratApi` (codegen) → REST API; real-time events via `wsApi` WebSocket.

Key rules:

- **Never** edit `frontend/src/store/sratApi.ts` or `backend/docs/openapi.*` directly - update Go source then run `cd frontend && bun run gen`.
- **Never** manually add types to `frontend/src/store/wsApi.ts` - all types come from `sratApi.ts`. WS-only payload types need a doc-stub handler in `backend/src/api/system.go` (tagged `"system","internal"`); see `HandleWelcome`/`HandleCommandEvents` for the pattern.
- **Never** use raw `fetch()` for internal API calls - always use RTK Query via `sratApi`.
- Forms **must** use `react-hook-form` + `react-hook-form-mui` (`<FormContainer>`, typed field elements). No raw `useState` for form field values, validation errors, or submit-loading state.
- MUI Grid: use the `size` prop (Grid2 default).
- Frontend test isolation: use `msw` for API mocking; shared recurring handlers go in `frontend/src/mocks/customHandlers.ts`.

### Custom Component (Python / Home Assistant)

- Runtime data in `entry.runtime_data` (not `hass.data`).
- WebSocket-only coordinator: `update_interval=None`.
- Sensors return `None` when unavailable.
- `strings.json` / `translations/en.json` issue entries use **either** `"description"` **or** `"fix_flow"` - never both (hassfest enforces this).

## Testing Conventions

### Backend tests

All service/API-layer tests use `testify/suite` + `uber-go/fx/fxtest` + `mockio/v2`. Pattern:

```go
type MyServiceTestSuite struct { suite.Suite; app *fxtest.App; ... }
func TestMyServiceTestSuite(t *testing.T) { suite.Run(t, new(MyServiceTestSuite)) }
// SetupTest: fxtest.New(suite.T(), fx.Provide(...)), app.RequireStart()
// TearDownTest: cancel() → WaitGroup.Wait() → app.RequireStop()
```

- External test package (`package foo_test`) for service/API layers.
- Same package for converter/utils.
- In-memory DB: `DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)"`.
- Bug fixes require a failing test first.

### Frontend tests

- `bun:test` + `@testing-library/react` + `user-event` (never `fireEvent`).
- Query priority: `getByRole` → `getByLabelText` → `getByPlaceholderText` → `getByText` → `getByTestId` (last resort).
- No `container.querySelector()` or CSS class selectors.

## Task and Workflow Conventions

### Tasks live in `docs/tasks/NNN_<slug>.md`

Use the `start-task-work` skill when beginning any task from `docs/tasks/`. It handles GitHub issue linking, branch creation, and pre-implementation planning.

### Refactors

**Always invoke the `prepare-refactor` skill** for any `[REFACTOR]` task. It creates a tracking doc at `docs/refactors/<slug>.md`, establishes a test baseline before changes, and verifies nothing regressed after.

### Pull Requests

Use the `create-pr` skill. PRs always target `main`. Branch naming: `feature/<kebab>`, `fix/<kebab>`, `docs/<kebab>`, `refactor/<kebab>`.

### Changelog

Use the `update-changelog` skill after completing tasks. Entries go under `[ 🚧 Unreleased ]` in `CHANGELOG.md`, mapped by task type: `[FEATURE]` → `### ✨ Features`, `[FIX]` → `### 🐛 Bug Fixes`, `[REFACTOR]` → `### 🔧 Maintenance`, `[DOCS]` → `### 🧑‍🏫 Documentation`.

## Code Generation

- `backend/src/converter/*_conv_gen.go` - generated by `goverter`; run `mise run //backend:gen`
- `backend/src/dbom/g/` - generated GORM helpers; run `mise run //backend:gen`
- `backend/docs/openapi.*` - generated from Go source annotations; run `mise run //backend:gen`
- `frontend/src/store/sratApi.ts` - generated from OpenAPI; run `mise run //frontend:gen`
- Enum types in `backend/src/dto/` - generated by `goenums`; keep enum source files small (type + iota + `//go:generate` only)

## Security and Quality Gates

```sh
mise run //backend:gosec      # high-severity issues only (CI gate)
mise run //backend:gosec_all  # all severities
```

File permission defaults: `0750` for directories, `0600` for sensitive files. Never use shell interpolation for untrusted input - pass command args as explicit argv.
