## COPILOT: SRAT repository — quick orientation for coding agents

This file highlights the must-know, discoverable rules and workflows for productive changes in this repo. Keep it short and actionable.

- **Languages**: Go backend (Go 1.25), TypeScript React frontend (Bun runtime). See `backend/go.mod` and `frontend/package.json`/`bun.lockb`.
- **Builds**: Root `Makefile` proxies to `backend/Makefile`. Frontend uses Bun: `cd frontend && bun install && bun run build`.
- **Pre-commit**: Repository uses `pre-commit`. Do not edit `.git/hooks` manually. See `.pre-commit-config.yaml` and run `pre-commit install` locally.

## Architecture Overview

SRAT is a Samba administration tool with a Go REST API backend and React frontend, designed to run within Home Assistant. Key architectural patterns:

- **Backend**: Clean architecture with API handlers → Services → Repositories → Database (GORM/SQLite)
- **Frontend**: React + TypeScript + Material-UI + RTK Query for API state management
- **Communication**: REST API with Server-Sent Events (SSE) for real-time updates
- **Database**: SQLite with GORM ORM, embedded in production binary
- **Dependency Injection**: Uber FX throughout backend for service wiring

## Key Patterns & Where to Find Them

### Backend Patterns

- **API Handlers**: `backend/src/api/*` — Use constructor `NewXHandler` and `RegisterXHandler(api huma.API)`. Handlers use Huma framework for REST API.
- **Services**: `backend/src/service/*` — Each service has an interface and implementation wired via FX (`fx.In` param structs). Services coordinate business logic.
- **Repositories**: `backend/src/repository/*` and `backend/src/dbom/*` — GORM models in `dbom`, repositories handle data access with mutex protection.
- **DTOs**: `backend/src/dto` — Define domain objects, error codes (see `dto/error_code.go`), and request/response shapes.
- **Converters**: `backend/src/converter/*` — Goverter-generated converters for DTO↔DBOM transformations. Run `go generate` after changes.
- **Logging**: `backend/src/tlog` — Custom logging with sensitive data masking, structured logs, and terminal color support.

### Frontend Patterns

- **Components**: `frontend/src/components/` — Reusable React components with Material-UI
- **Pages**: `frontend/src/pages/` — Route-based page components
- **Store**: `frontend/src/store/` — RTK Query for API calls, Redux slices for local state
- **Hooks**: `frontend/src/hooks/` — Custom React hooks for shared logic
- **API Integration**: Auto-generated RTK Query hooks from OpenAPI spec (see `frontend/src/store/sratApi.ts`)

## Development Workflows

### Backend Development
- **Start dev server**: `cd backend && make dev` (uses Air for hot reload)
- **Build**: `cd backend && make build` (production) or `make test_build` (debug symbols)
- **Test**: `cd backend && make test` (runs with `-p 1` for deterministic output)
- **Format**: `cd backend && make format` (includes gofmt, testifylint, govet)
- **Generate**: `cd backend && make gen` (goverter converters + OpenAPI docs)

### Frontend Development
- **Start dev server**: `cd frontend && bun run dev` (hot reload with live reload)
- **Build**: `cd frontend && bun run build` (outputs to `../backend/src/web/static`)
- **Watch mode**: `cd frontend && bun run gowatch` (builds directly to backend static dir)
- **Generate API**: `cd frontend && bun run gen` (RTK Query from OpenAPI spec)
- **Lint**: `cd frontend && bun run lint` (Biome formatter/linter)

### Full Stack Development
- **Prepare environment**: `make prepare` (installs pre-commit + dependencies)
- **Build all**: `make ALL` (multi-arch: amd64, armv7, aarch64)
- **Clean**: `make clean`

## Testing Patterns

### Backend Testing
- **Framework**: `testify/suite` with `mockio/v2` for mocks
- **Test Structure**: `{package}_test` with `{HandlerName}HandlerSuite` structs
- **Setup**: Use `fxtest.New()` to build dependency graph, `fx.Populate()` to inject mocks
- **HTTP Tests**: `humatest.New()` for API testing, `autopatch.AutoPatch(api)` for PATCH endpoints
- **Mock Pattern**: `mock.When(service.Method(...)).ThenReturn(...)` then `mock.Verify(..., matchers.Times(1)).Method()`
- **State Verification**: Always check `dirtyService.SetDirty*()` calls when data is modified

### Test Examples
```go
// HTTP handler test
func (suite *ShareHandlerSuite) TestCreateShareSuccess() {
    mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(expectedShare, nil)
    _, api := humatest.New(suite.T())
    suite.handler.RegisterShareHandler(api)
    resp := api.Post("/share", input)
    suite.Equal(http.StatusCreated, resp.Code)
    mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}
```

## Quality Gates & Validation

### Pre-commit Hooks
- **Security**: `gosec` scans Go code (high severity/confidence only)
- **Dependencies**: Remove/restore `go.mod` replace directives
- **Documentation**: Link format validation, CHANGELOG format checks
- **Install**: `pre-commit install && pre-commit install --hook-type pre-push`

### Documentation
- **Validation**: `make docs-validate` (markdownlint, link checks, spellcheck)
- **Auto-fix**: `make docs-fix` (formatting fixes)
- **OpenAPI**: Auto-generated from Go code, served at `/docs`

### Security
- **Backend**: `make security` runs `gosec` (exclude generated code)
- **Frontend**: Bun handles dependency security
- **Include in PR**: Security scan results in "Quality Gates" section

## Integration Points

### External Dependencies
- **Database**: SQLite with WAL mode, busy timeout, foreign keys
- **Home Assistant**: Integration via supervisor API, addon configuration
- **Samba**: Configuration generation and service management
- **Telemetry**: Optional Rollbar integration with user consent

### Cross-Component Communication
- **Dirty State**: `dirtyService.SetDirty*()` methods mark data as changed
- **Notifications**: Services call `NotifyClient()` for real-time updates via SSE
- **Error Handling**: Custom error codes in `dto/error_code.go`, wrapped with `errors.Wrap()`
- **Context**: Request context passed through all layers for cancellation/tracing

## Files to Open First

- **Backend entry**: `backend/Makefile`, `backend/src/api/*`, `backend/src/service/*`
- **Frontend entry**: `frontend/src/App.tsx`, `frontend/src/store/sratApi.ts`
- **Architecture**: `backend/src/dto/error_code.go`, `backend/src/converter/*`
- **Build system**: Root `Makefile`, `frontend/bun.build.ts`

## Common Gotchas

- **FX Wiring**: When changing service interfaces, update all `fx.Provide()` calls and test `fx.Populate()`
- **Converters**: After DTO/DBOM changes, run `go generate` to regenerate goverter code
- **Patches**: External dependencies are patched via `gohack` — see `backend/Makefile` patch targets
- **Multi-arch**: Always test builds on target architectures, especially ARM variants
- **Embedded Assets**: Frontend builds to `backend/src/web/static` for embedding in binary
- **Database Paths**: Use `--db` flag; app validates filesystem permissions

If uncertain, run: `pre-commit run --all-files`, `make docs-validate`, `make security`

If this file misses anything important, tell me which area (build, tests, DI, logging, frontend) and I will expand with concrete examples.
