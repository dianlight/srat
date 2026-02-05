<!-- DOCTOC SKIP -->

## COPILOT: SRAT repository–quick orientation for coding agents

This file highlights the must-know, discoverable rules and workflows for productive changes in this repo. Keep it short and actionable.

- **Languages**: Go backend (Go 1.25), TypeScript React frontend (Bun runtime). See `backend/go.mod` and `frontend/package.json`/`bun.lockb`.
- **Builds**: Root `Makefile` proxies to `backend/Makefile`. Frontend uses Bun: `cd frontend && bun install && bun run build`.
- **Pre-commit**: Repository uses `prek`. Do not edit `.git/hooks` manually. See `.pre-commit-config.yaml` and run `prek install` locally.
- **Tests**: Backend uses `testify/suite` with `mockio/v2`. Frontend uses `bun:test` with `@testing-library/react`. See below for patterns.
- **Git Operations**: **NEVER** perform `git add`, `git commit`, `git push`, or any other git write operations without explicit user request. Always wait for the user to request git operations after changes are complete and verified.
- **File-Specific Rules (MANDATORY)**: **ALWAYS** read the top comment/header section of any file you modify. Many files contain important rules, constraints, or patterns specific to that file. These rules take precedence and must be followed in addition to (or instead of) general patterns. Examples: licensing headers, code generation markers, restricted modification zones, special formatting rules, or dependencies on external tools. If a file's top comment specifies different behavior than general guidelines, follow the file-specific rules.

## Architecture Overview

SRAT is a Samba administration tool with a Go REST API backend and React frontend, designed to run within Home Assistant. Key architectural patterns:

- **Backend**: Clean architecture with API handlers → Services → Generated GORM helpers → Database (GORM/SQLite). Persistence now happens through the generated DBOM helpers rather than a handwritten repository tier.
- **Frontend**: React + TypeScript + Material-UI + RTK Query for API state management
- **Communication**: REST API with Server-Sent Events (SSE) or WebSockets for real-time updates
- **Database**: SQLite with GORM ORM, embedded in production binary
- **Dependency Injection**: Uber FX throughout backend for service wiring

## Key Patterns & Where to Find Them

### Backend Patterns

- **API Handlers**: `backend/src/api/*`–Use constructor `NewXHandler` and `RegisterXHandler(api huma.API)`. Handlers use Huma framework for REST API.
- **Services**: `backend/src/service/*`–Each service has an interface and implementation wired via FX (`fx.In` param structs). Services coordinate business logic.
- **Persistence**: Services now work directly with generated GORM helpers under `backend/src/dbom/g` (see `backend/src/service/share_service.go` for a live example of `gorm.G[dbom.ExportedShare]`). The `repository` packages no longer need to mediate persistence; generated DBOM structs and their GORM helpers deliver all read/write behavior.
- **DTOs**: `backend/src/dto`–Define domain objects, error codes (see `dto/error_code.go`), and request/response shapes.
- **Converters**: `backend/src/converter/*`–Goverter-generated converters for DTO↔DBOM transformations. Run `go generate` after changes.
- **Logging**: `backend/src/tlog`–Custom logging with sensitive data masking, structured logs, and terminal color support.

#### Context-Aware Logging (MANDATORY RULE)

When writing or modifying Go backend code, prefer the context variants of logging functions whenever a `context.Context` is ALREADY in scope:

Use:
`slog.InfoContext(ctx, ...)`, `slog.WarnContext(ctx, ...)`, `slog.ErrorContext(ctx, ...)`, `slog.DebugContext(ctx, ...)`
`tlog.TraceContext(ctx, ...)`, `tlog.DebugContext(ctx, ...)`, `tlog.InfoContext(ctx, ...)`, `tlog.WarnContext(ctx, ...)`, `tlog.ErrorContext(ctx, ...)`

Rules:

1. Only add the context form if a context variable is naturally available (for example `ctx`, `self.ctx`, `s.ctx`, `handler.ctx`, `apiContext`, `r.Context()`, constructor-local `Ctx`).
2. DO NOT create a new context just for logging (no `context.Background()`, `context.TODO()`, `context.WithTimeout(...)` solely to pass to log).
3. Preserve original argument order; only insert the context as the first argument.
4. Do not refactor method signatures to add context purely for logging.
5. In goroutines: use an existing captured context if present; do not capture a new one solely for logging.
6. Leave the original non-context call if no appropriate context exists (this is acceptable and preferred over manufacturing one).
7. Avoid changing vendor or third-party code for this; skip files under `backend/src/vendor/` unless explicitly patching via the established patch workflow.
8. Tests may keep simple non-context logging unless the test specifically exercises context logging behavior.
9. Never pass a nil context or a fabricated stand‑in (for example a struct field that is not a `context.Context`).

Acceptable context identifiers (examples, not exhaustive): `ctx`, `self.ctx`, `s.ctx`, `ms.ctx`, `ts.ctx`, `handler.ctx`, `apiContext`, `self.apiContext`, `r.Context()`, locally declared `Ctx` in constructors.

Examples:

Before:

```go
slog.Info("Reloading config", "component", comp)
tlog.Trace("Starting scan", "disk", d)
```

After (if `ctx` available):

```go
slog.InfoContext(ctx, "Reloading config", "component", comp)
tlog.TraceContext(ctx, "Starting scan", "disk", d)
```

Before (goroutine without captured context):

```go
go func() {
  slog.Debug("Background task running", "id", id)
}()
```

Leave as-is unless the goroutine already captures a legitimate context for other reasons.

Incorrect (manufactures context only for logging):

```go
slog.WarnContext(context.Background(), "Will retry", "attempt", n) // NOT ALLOWED
```

Correct alternative:

```go
slog.Warn("Will retry", "attempt", n)
```

Rationale: Using context-aware logging lets structured handlers attach trace/span, cancellation, and request lineage automatically. Avoid polluting code with artificial contexts—only leverage what is organically available.

### Frontend Patterns

- **Components**: `frontend/src/components/`–Reusable React components with Material-UI
- **Pages**: `frontend/src/pages/`–Route-based page components
- **Store**: `frontend/src/store/`–RTK Query for API calls, Redux slices for local state
- **Hooks**: `frontend/src/hooks/`–Custom React hooks for shared logic
- **API Integration**: Auto-generated RTK Query hooks from OpenAPI spec (see `frontend/src/store/sratApi.ts`). Never make change on `frontend/src/store/sratApi.ts` or on `backend/docs/openapi.*` directly; update Go code and run `cd frontend && bun run gen`.
- **MUI Grid**: Use modern Grid syntax with `size` prop (for example, `<Grid size={{ xs: 12, sm: 6 }}>`)–Grid2 is now promoted as the default Grid in MUI v7.3.2+

## Development Workflows

### Backend Development

- **Start dev server**: `cd backend && make dev` (uses Air for hot reload)
- **Build**: `cd backend && make build` (production) or `make test_build` (debug symbols)
- **Test**: `cd backend && make test` (runs with `-p 1` for deterministic output)
- **Format**: `cd backend && make format` (includes gofmt, testifylint, govet)
- **Generate**: `cd backend && make gen` (goverter converters + OpenAPI docs)
- **Patch Libraries**: `cd backend && make patch` (applies patches to vendored dependencies)

### Library Patching with Go Vendor

SRAT uses **patched versions** of certain external Go libraries to enable additional functionality not available in the original packages. The patching is managed via Go's vendor mechanism with patch files stored in `backend/patches/`.

**Patched Libraries:**

- `github.com/zarldev/goenums`–Custom enum generation improvements
- `github.com/jpillora/overseer`–Process management enhancements

**Patch Workflow:**

```bash
cd backend
make patch          # Apply patches to vendored dependencies
make gen_patch      # Instructions for generating new patches
go mod vendor       # Vendor all dependencies (done automatically by make)
```

**How it works:**

1. Libraries are stored in `backend/src/vendor/` via `go mod vendor`
2. Patch files from `backend/patches/*.patch` are applied to vendored copies using the `patch` utility
3. For smart.go, multiple patches are applied in alphabetical order: `smart.go-*`
4. Patches are applied automatically during build via `make patch` target
5. The vendor directory is committed to the repository with patches already applied

**Adding a new patch:**

1. Edit the library code directly in `backend/src/vendor/github.com/{library}/`
2. Test your changes with `make build` or `make test`
3. Generate a patch file using: `diff -Naur original_version patched_version > backend/patches/{name}.patch`
4. For smart.go, use naming pattern `smart.go-<priority>.patch` (for example, `smart.go-#010.patch`, `smart.go-srat#999.patch`)
5. Commit both the patch file and any changes to vendor to the repository
6. Future developers can regenerate vendor with: `cd backend/src && go mod vendor && cd .. && make patch`

**Important notes:**

- Patches are **required** for SMART service operations (enable/disable, test execution)
- Multiple patches can be applied to the same library (for example, smart.go has multiple patches)
- Patches are applied in alphabetical order by filename
- The vendor directory contains the complete dependency tree with patches pre-applied
- Run `make patch` after `go mod vendor` to ensure patches are applied
- To update a library version: `cd backend/src && go get -u github.com/library/name && go mod vendor && cd .. && make patch`

### Frontend Development

- **Start dev server**: `cd frontend && bun run dev` (hot reload with live reload)
- **Start remote dev server**: `cd frontend && bun run dev:remote` (for testing with remote backend)
- **Build**: `cd frontend && bun run build` (outputs to `../backend/src/web/static`)
- **Watch mode**: `cd frontend && bun run gowatch` (builds directly to backend static dir)
- **Generate API**: `cd frontend && bun run gen` (RTK Query from OpenAPI spec)
- **Lint**: `cd frontend && bun run lint` (Biome formatter/linter)
- **Test**: `cd frontend && bun test` (runs all tests with bun:test)

### Full Stack Development

- **Prepare environment**: `make prepare` (installs prek + dependencies)
- **Build all**: `make ALL` (multi-arch: amd64, aarch64)
- **Clean**: `make clean`

## Testing Patterns

- **Code coverage**: Backend uses `cd backend && make test`. Frontend uses `cd frontend && bun test --coverage`.
- **Test data**: Use `backend/test/data/` dirs for static test files
- **Minimal coverage**: Backend enforces 5% coverage. Frontend enforces 80% functions coverage.
- **New tests**: All new features/bug fixes must include tests covering positive and negative cases. Minimal functions coverage is 90% for frontend tests and 6% for backend tests.
- **Local CI Testing with act**: When testing GitHub Actions workflows locally using `act`, always use `ghcr.io/catthehacker/ubuntu:act-latest` image instead of `full-latest` to reduce resource usage and speed up testing. Also always use -rm flag to avoid vm pollution. Example: `act --rm -W .github/workflows/build.yaml -j test-frontend --container-architecture linux/amd64 -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:act-latest`

### Backend Testing

- **Framework**: `testify/suite` with `mockio/v2` for mocks
- **Test Structure**: `{package}_test` with `{HandlerName}HandlerSuite` structs
- **Setup**: Use `fxtest.New()` to build dependency graph, `fx.Populate()` to inject mocks
- **HTTP Tests**: `humatest.New()` for API testing, `autopatch.AutoPatch(api)` for PATCH endpoints
- **Mock Pattern**: `mock.When(service.Method(...)).ThenReturn(...)` then `mock.Verify(..., matchers.Times(1)).Method()`
- **State Verification**: Always check `dirtyService.SetDirty*()` calls when data is modified

### Frontend Testing

- **Timeout**: Always run with timeout at last of 10sec `bun test --timeout 10000`. If a test need more of 10sec split the test in smaller tests
- **Framework**: `bun:test` with `happy-dom` for DOM simulation
- **Testing Libraries**: `@testing-library/react` for component testing, `@testing-library/jest-dom` for assertions
- **Test Structure**: Place tests in `__tests__` directories alongside components/pages
- **File Naming**: Use `.test.tsx` extension for test files
- **Setup**: Import test utilities from `bun:test`: `describe`, `it`, `expect`, `beforeEach`
- **DOM Setup**: Use `happy-dom` for DOM globals, custom localStorage shim for storage tests
- **Store Testing**: Use `createTestStore()` helper from `frontend/test/setup.ts` for Redux store
- **Component Testing**: Dynamic imports for React components to avoid module loading issues
- **Async Testing**: Use `screen.findByText()` for waiting on async renders
- **Bun Test Cli Commands**: Always use only valid cli command with bun. Use `bun test --help` to check the command avability before use it.

### Test Examples

```go
// Backend HTTP handler test
func (suite *ShareHandlerSuite) TestCreateShareSuccess() {
    mock.When(suite.mockShareService.CreateShare(mock.Any[dto.SharedResource])).ThenReturn(expectedShare, nil)
    _, api := humatest.New(suite.T())
    suite.handler.RegisterShareHandler(api)
    resp := api.Post("/share", input)
    suite.Equal(http.StatusCreated, resp.Code)
    mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
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

    // Setup userEvent for all interactions
    const userEvent = (await import("@testing-library/user-event")).default;
    const user = userEvent.setup();

    render(
      React.createElement(
        Provider,
        { store },
        React.createElement(MyComponent, { prop: "value" }),
      ),
    );

    const element = await screen.findByText("Expected Text");
    expect(element).toBeTruthy();

    // Always await user interactions
    const button = await screen.findByRole("button");
    await user.click(button);
  });
});
```

### Test-Driven Debugging (MANDATORY RULE FOR BUG FIXES)

When addressing any issue or bug, **ALWAYS follow this workflow**:

1. **Understand the issue**: Read the issue description, error messages, or reproduction steps carefully
2. **Create a failing test FIRST**: Before writing any fix, create a test that reproduces the issue
   - For backend bugs: Add a test case to the relevant handler or service test suite that fails with the current code
   - For frontend bugs: Add a test case to the relevant component test that demonstrates the broken behavior
   - The test should fail with the current code and pass after the fix is applied
3. **Verify the test fails**: Run the test suite to confirm your test fails before any code changes
   - Backend: `cd backend && make test` (run specific test if possible)
   - Frontend: `cd frontend && bun test [ComponentName]` (run specific test)
4. **Implement the fix**: Make minimal changes to fix the failing test
5. **Verify the test passes**: Run the test again to confirm it now passes
6. **Verify no regressions**: Run the full test suite for the affected module
   - Backend: `cd backend && make test`
   - Frontend: `cd frontend && bun test` with `--rerun-each 10` to detect flakiness

**Benefits of this approach:**

- Ensures the bug is actually fixed (not masked by incomplete changes)
- Prevents regressions when the same issue reappears later
- Creates documentation of the expected behavior for future developers
- Speeds up debugging by having automated verification of the fix

**Example: Bug fix workflow**

```plaintext
1. Issue: Share name validation not working in create dialog
2. Test: Add frontend component test that verifies validation message appears when empty name submitted
3. Run: bun test ShareDialog --rerun-each 10 → Test FAILS ✗
4. Fix: Update ShareDialog validation logic
5. Run: bun test ShareDialog --rerun-each 10 → Test PASSES ✓
6. Run: bun test → Verify no other tests broken
7. Result: Bug fixed with test coverage preventing future regression
```

## Quality Gates & Validation

### Pre-commit Hooks

- **Security**: `gosec` scans Go code (high severity/confidence only)
- **Dependencies**: Remove/restore `go.mod` replace directives
- **Documentation**: Link format validation, CHANGELOG format checks
- **Install**: `prek install && prek install --hook-type pre-push`

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
- **Patches**: External dependencies are patched via `gohack`–see `backend/Makefile` patch targets. Run `make patch` after clean checkout or when patches are updated
- **Patched Libraries**: Changes to `github.com/anatol/smart.go`, `github.com/zarldev/goenums`, or `github.com/jpillora/overseer` require updating patch files via `make gen_patch`
- **Replace Directives**: **NEVER** commit `replace` directives in `go.mod`–pre-commit hooks remove them automatically. Patches are applied via `make patch` instead
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
- **CRITICAL**: Use `@testing-library/user-event` for ALL user interactions–NEVER use `fireEvent`
  - Import: `const userEvent = (await import("@testing-library/user-event")).default;`
  - Setup: `const user = userEvent.setup();`
  - All interactions MUST be awaited: `await user.click(element)`, `await user.type(input, "text")`, `await user.clear(input)`
  - Wrap stateful UI transitions in `act()` when needed for happy-dom compatibility
  - `fireEvent` is deprecated and must not be used in any new or modified tests

### localStorage Testing Pattern

```tsx
// REQUIRED localStorage shim for every localStorage test
if (!(globalThis as any).localStorage) {
  const _store: Record<string, string> = {};
  (globalThis as any).localStorage = {
    getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
    setItem: (k: string, v: string) => {
      _store[k] = String(v);
    },
    removeItem: (k: string) => {
      delete _store[k];
    },
    clear: () => {
      for (const k of Object.keys(_store)) delete _store[k];
    },
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

    // REQUIRED: Setup userEvent for interactions
    const userEvent = (await import("@testing-library/user-event")).default;
    const user = userEvent.setup();

    // REQUIRED: Use React.createElement, not JSX
    render(
      React.createElement(
        Provider,
        { store },
        React.createElement(ComponentName as any, { props }),
      ),
    );

    // REQUIRED: Use findByText for async, toBeTruthy() for assertions
    const element = await screen.findByText("Expected Text");
    expect(element).toBeTruthy();

    // REQUIRED: Await all user interactions
    const button = await screen.findByRole("button");
    await user.click(button);
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
- **ALWAYS** await userEvent interactions: `await user.click()`, `await user.type()`, `await user.clear()`
- Use `act()` wrapper when userEvent triggers state updates that cause re-renders in happy-dom

**NON-NEGOTIABLE:** All frontend tests must follow these exact patterns. No exceptions for import style, file structure, or testing utilities. `fireEvent` is strictly prohibited–use `userEvent` for all interactions.

### Frontend Test Changes & Coverage (MANDATORY RULE)

**CRITICAL: Every frontend component change requires test updates and validation.**

When making ANY changes to frontend components or tests:

0. **BEFORE making component changes**: Check if tests exist. If not, create them immediately (see "Create tests if missing" below).

1. **Component modification workflow**:
   - Identify all affected component behavior (rendering, interactions, state changes)
   - Make your component changes
   - Update or create tests to cover the new/modified behavior
   - Run the specific component test: `cd frontend && bun test [ComponentName].test.tsx`
   - Run full test suite: `cd frontend && bun test`
   - Validate no flakiness: `cd frontend && bun test --rerun-each 10 [ComponentName]` (must show 100% pass rate)

2. **Every test modification requires validation**:
   - After modifying any existing test, run: `cd frontend && bun test --rerun-each 10 [TestName]`
   - Verify 100% pass rate across all 10 re-runs (zero failures)
   - Document any flakiness issues and fix root causes before considering the change complete

3. **Create tests if missing**:
   - If a component exists WITHOUT tests, it is **incomplete and requires tests before merge**
   - Create a `__tests__` directory alongside the component (for example, `src/components/[name]/__tests__/`)
   - Create `[ComponentName].test.tsx` with coverage for:
     - Component renders without errors
     - All user interactions work (button clicks, form inputs, toggles, selections)
     - Props are respected (readOnly, disabled, onChange, etc.)
     - State changes trigger expected behavior
     - Default values are applied correctly
   - Minimum test count: 5-8 tests covering main functionality
   - Coverage: All interactive elements and state-dependent UI must be tested

4. **New pages or major sections MUST include tests**:
   - Create a `__tests__` directory alongside the new page/section (for example, `src/pages/[pageName]/__tests__/`)
   - Create at least a basic `.test.tsx` file that covers:
     - Component renders without errors
     - Key UI elements are present
     - Major user interactions work as expected
   - Minimum coverage: Test happy path flows for all major features in the page/section
   - If a page/section is created without tests, it is considered **incomplete** and must not be merged

5. **Test verification checklist**:
   - [ ] Component logic change → Tests updated to reflect new behavior
   - [ ] New interactive element added → Test added for that interaction
   - [ ] Props changed → Tests updated to validate new props
   - [ ] All new/modified tests follow the established patterns
   - [ ] Tests pass locally: `cd frontend && bun test`
   - [ ] No flakiness detected: `cd frontend && bun test --rerun-each 10 [ComponentName]`
   - [ ] Tests verify actual user behavior, not implementation details
   - [ ] localStorage/Redux state is properly cleaned between tests
   - [ ] All async operations are properly awaited
   - [ ] `userEvent` is used for all interactions (never `fireEvent`)
   - [ ] Dynamic imports are used for React components in test files

6. **When modifying components that already have tests**:
   - Update tests to reflect new behavior
   - Add tests for new features or changed UI
   - Add tests for new props or state-dependent rendering
   - Re-validate with `--rerun-each 10` after each modification
   - If test count is <5, add more tests for uncovered behavior

7. **Common pitfalls to avoid**:
   - Don't skip tests for UI refactoring–update tests alongside UI changes
   - Don't create pages without test structure–add `__tests__/` directory from the start
   - Don't rely on implementation details in tests–test user-facing behavior
   - Don't leave commented-out tests or TODOs in test files
   - Don't merge components without tests–incomplete code is blocker
   - Don't assume old tests still work–rerun with `--rerun-each 10` after changes

## Final Checklist Before Consider a Changes as Done

Ensure all relevant pre-commit hooks pass locally before pushing changes. This includes formatting, linting, security scans, and documentation validation.

If uncertain, run: `prek run --all-files`, `make docs-validate`, `make security`

If this file misses anything important, tell me which area (build, tests, DI, logging, frontend) and I will expand with concrete examples.

All backend and frontend changes must also follow the established patterns in existing tests or introduce new patterns that are well-documented, covered by tests, and justified.

Always prioritize maintainability and clarity in tests.

Always ensure tests are deterministic and can run in CI environments without special setup.

**CRITICAL for frontend tests**: Before considering any modified or new frontend test as correct, execute it with `bun test --rerun-each 10` at least one time and verify 100% pass rate (0 failures). This ensures tests are not flaky and do not have mock state or component state bleed issues. Example: `cd frontend && bun test --rerun-each 10 MyComponentName` must show all 10 test runs passing.

Update documentation to reflect any new patterns, changes in workflows or architecture.

The goal is to maintain high code quality, consistency, and ease of onboarding for future contributors.

Dead code or commented-out code should be removed, not left in the codebase.

When in doubt, ask for clarification on the intended pattern or best practice before proceeding with changes.

Update CHANGELOG.md and relevant documentation files for any new features, bug fixes, or breaking changes.

Check for open issues related to your changes and reference them in your commit messages or PR descriptions.

## END OF COPILOT INSTRUCTIONS
