## COPILOT: SRAT repository — quick orientation for coding agents

This file highlights the must-know, discoverable rules and workflows for productive changes in this repo. Keep it short and actionable.

- **Languages**: Go backend (Go 1.25), TypeScript React frontend (Bun runtime). See `backend/go.mod` and `frontend/package.json`/`bun.lockb`.
- **Builds**: Root `Makefile` proxies to `backend/Makefile`. Frontend uses Bun: `cd frontend && bun install && bun run build`.
- **Pre-commit**: Repository uses `pre-commit`. Do not edit `.git/hooks` manually. See `.pre-commit-config.yaml` and run `pre-commit install` locally.
- **Tests**: Backend uses `testify/suite` with `mockio/v2`. Frontend uses `bun:test` with `@testing-library/react`. See below for patterns.

## Architecture Overview

SRAT is a Samba administration tool with a Go REST API backend and React frontend, designed to run within Home Assistant. Key architectural patterns:

- **Backend**: Clean architecture with API handlers → Services → Repositories → Database (GORM/SQLite)
- **Frontend**: React + TypeScript + Material-UI + RTK Query for API state management
- **Communication**: REST API with Server-Sent Events (SSE) or WebSockets for real-time updates
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
- **API Integration**: Auto-generated RTK Query hooks from OpenAPI spec (see `frontend/src/store/sratApi.ts`). Never make change on `frontend/src/store/sratApi.ts` or on `backend/docs/openapi.*` directly; update Go code and run `cd frontend && bun run gen`.
- **MUI Grid**: Use modern Grid syntax with `size` prop (e.g., `<Grid size={{ xs: 12, sm: 6 }}>`) — Grid2 is now promoted as the default Grid in MUI v7.3.2+

## Development Workflows

### Backend Development

- **Start dev server**: `cd backend && make dev` (uses Air for hot reload)
- **Build**: `cd backend && make build` (production) or `make test_build` (debug symbols)
- **Test**: `cd backend && make test` (runs with `-p 1` for deterministic output)
- **Format**: `cd backend && make format` (includes gofmt, testifylint, govet)
- **Generate**: `cd backend && make gen` (goverter converters + OpenAPI docs)

### Frontend Development

- **Start dev server**: `cd frontend && bun run dev` (hot reload with live reload)
- **Start remote dev server**: `cd frontend && bun run dev:remote` (for testing with remote backend)
- **Build**: `cd frontend && bun run build` (outputs to `../backend/src/web/static`)
- **Watch mode**: `cd frontend && bun run gowatch` (builds directly to backend static dir)
- **Generate API**: `cd frontend && bun run gen` (RTK Query from OpenAPI spec)
- **Lint**: `cd frontend && bun run lint` (Biome formatter/linter)
- **Test**: `cd frontend && bun test` (runs all tests with bun:test)

### Full Stack Development

- **Prepare environment**: `make prepare` (installs pre-commit + dependencies)
- **Build all**: `make ALL` (multi-arch: amd64, armv7, aarch64)
- **Clean**: `make clean`

## Testing Patterns

- **Code coverage**: Backend uses `cd backend && make test`. Frontend uses `cd frontend && bun test --coverage`.
- **Test data**: Use `backend/test/data/` dirs for static test files
- **Minimal coverage**: Backend enforces 5% coverage. Frontend enforces 80% functions coverage.
- **New tests**: All new features/bug fixes must include tests covering positive and negative cases. Minimal functions coverage is 90% for frontend tests and 6% for backend tests.

### Backend Testing

- **Framework**: `testify/suite` with `mockio/v2` for mocks
- **Test Structure**: `{package}_test` with `{HandlerName}HandlerSuite` structs
- **Setup**: Use `fxtest.New()` to build dependency graph, `fx.Populate()` to inject mocks
- **HTTP Tests**: `humatest.New()` for API testing, `autopatch.AutoPatch(api)` for PATCH endpoints
- **Mock Pattern**: `mock.When(service.Method(...)).ThenReturn(...)` then `mock.Verify(..., matchers.Times(1)).Method()`
- **State Verification**: Always check `dirtyService.SetDirty*()` calls when data is modified

### Frontend Testing

- **Framework**: `bun:test` with `happy-dom` for DOM simulation
- **Testing Libraries**: `@testing-library/react` for component testing, `@testing-library/jest-dom` for assertions
- **Test Structure**: Place tests in `__tests__` directories alongside components/pages
- **File Naming**: Use `.test.tsx` extension for test files
- **Setup**: Import test utilities from `bun:test`: `describe`, `it`, `expect`, `beforeEach`
- **DOM Setup**: Use `happy-dom` for DOM globals, custom localStorage shim for storage tests
- **Store Testing**: Use `createTestStore()` helper from `frontend/test/setup.ts` for Redux store
- **Component Testing**: Dynamic imports for React components to avoid module loading issues
- **Async Testing**: Use `screen.findByText()` for waiting on async renders

### Test Examples

```go
// Backend HTTP handler test
func (suite *ShareHandlerSuite) TestCreateShareSuccess() {
    mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource]())).ThenReturn(expectedShare, nil)
    _, api := humatest.New(suite.T())
    suite.handler.RegisterShareHandler(api)
    resp := api.Post("/share", input)
    suite.Equal(http.StatusCreated, resp.Code)
        mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
}
```

```tsx
}
```

```tsx
// Frontend localStorage test
import { describe, it, expect, beforeEach } from "bun:test";

describe("Component localStorage functionality", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    it("saves and restores data to localStorage", () => {
        const testData = "test-value";
        localStorage.setItem("component.data", testData);
        expect(localStorage.getItem("component.data")).toBe(testData);
    });
}
```

```tsx
});
```

```tsx
// Frontend component test with React Testing Library
import { describe, it, expect, beforeEach } from "bun:test";
import { createTestStore } from "../../../test/setup";

describe("Component rendering", () => {
    it("renders component with initial data", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MyComponent } = await import("../MyComponent");
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                { store },
                React.createElement(MyComponent, { prop: "value" })
            )
        );

        const element = await screen.findByText("Expected Text");
        expect(element).toBeTruthy();
    });
});
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

## Frontend Testing Rules

**MANDATORY patterns for all frontend tests:**

### File Structure & Naming

- Tests MUST be in `__tests__` directories alongside the component/page being tested
- Test files MUST use `.test.tsx` extension
- Component tests go in `src/pages/[page]/__tests__/` or `src/components/[component]/__tests__/`

### Required Imports & Setup

- ALWAYS import from `bun:test`: `import { describe, it, expect, beforeEach } from "bun:test";`
- For localStorage tests: Include the minimal localStorage shim (see existing tests for exact code)
- For component tests: Use `createTestStore()` helper from `../../../test/setup` (adjust path as needed)
- For React components: Use dynamic imports to avoid module loading issues

### Testing Library Standards

- Use `@testing-library/react` for component rendering: `const { render, screen } = await import("@testing-library/react");`
- Use `@testing-library/jest-dom` assertions: `expect(element).toBeTruthy();`
- For async rendering: Use `await screen.findByText()` not `getByText()`
- Always use `React.createElement()` syntax, not JSX, in test files

### localStorage Testing Pattern

```tsx
// REQUIRED localStorage shim for every localStorage test
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("Component localStorage functionality", () => {
    beforeEach(() => {
        localStorage.clear(); // ALWAYS clear before each test
    });
    // ... tests
});
```

### Component Testing Pattern

```tsx
describe("Component rendering", () => {
    it("renders component with data", async () => {
        // REQUIRED: Dynamic imports after globals are set
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ComponentName } = await import("../ComponentName");
        const store = await createTestStore();

        // REQUIRED: Use React.createElement, not JSX
        render(
            React.createElement(
                Provider,
                { store },
                React.createElement(ComponentName as any, { props })
            )
        );

        // REQUIRED: Use findByText for async, toBeTruthy() for assertions
        const element = await screen.findByText("Expected Text");
        expect(element).toBeTruthy();
    });
});
```

### Redux Store Integration

- ALWAYS use `createTestStore()` for tests that need Redux state
- Import store helper: `import { createTestStore } from "../../../test/setup";` (adjust path)
- Wrap components with Redux Provider using `React.createElement(Provider, { store }, ...)`

### Async Testing Requirements

- Use `await screen.findByText()` for elements that appear after rendering
- Use `beforeEach(() => { localStorage.clear(); })` for localStorage tests
- Dynamic imports MUST be used for React components and testing utilities

**NON-NEGOTIABLE:** All frontend tests must follow these exact patterns. No exceptions for import style, file structure, or testing utilities.

## Final Checklist Before Consider a Changes as Done

Ensure all relevant pre-commit hooks pass locally before pushing changes. This includes formatting, linting, security scans, and documentation validation.

If uncertain, run: `pre-commit run --all-files`, `make docs-validate`, `make security`

If this file misses anything important, tell me which area (build, tests, DI, logging, frontend) and I will expand with concrete examples.

All backend and frontend changes must also follow the established patterns in existing tests or introduce new patterns that are well-documented, covered by tests, and justified.

Always prioritize maintainability and clarity in tests.

Always ensure tests are deterministic and can run in CI environments without special setup.

Update documentation to reflect any new patterns, changes in workflows or architecture.

The goal is to maintain high code quality, consistency, and ease of onboarding for future contributors.

Dead code or commented-out code should be removed, not left in the codebase.

When in doubt, ask for clarification on the intended pattern or best practice before proceeding with changes.

Update CHANGELOG.md and relevant documentation files for any new features, bug fixes, or breaking changes.

Check for open issues related to your changes and reference them in your commit messages or PR descriptions.

## END OF COPILOT INSTRUCTIONS
