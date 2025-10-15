# srat-frontend

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Console Error Callback Registry](#console-error-callback-registry)
  - [API](#api)
  - [Behavior](#behavior)
  - [Usage (Imperative)](#usage-imperative)
  - [Usage (React Hook)](#usage-react-hook)
  - [Frontend Component Organization Rules](#frontend-component-organization-rules)
- [Test Setup Enforcement](#test-setup-enforcement)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

To install dependencies:

```bash
bun install
```

To start the dev server with hot reload:

```bash
bun run dev
```

To build the production bundle (outputs to `../backend/src/web/static`):

```bash
bun run build
```

**Note about API code generation:** The `bun run gen` command (RTK Query codegen from OpenAPI) currently fails due to a TypeScript version mismatch issue in `@rtk-query/codegen-openapi`. This is a [documented issue](https://github.com/reduxjs/redux-toolkit/issues/2425) in the Redux Toolkit repository. For now the workaround is to install `@rtk-query/codegen-openapi` globally with npm and run with node not bun.

Note: This project uses Bun as the JavaScript runtime and package manager. See `bun.build.ts` for the build pipeline.

**Bun 1.3 Compatibility:** This project is fully compatible with Bun 1.3.0. The project has been tested with Bun 1.3.0 and all breaking changes have been reviewed. The project does not use any of the affected APIs (SQL client, YAML parser) that changed in Bun 1.3.

## Console Error Callback Registry

The frontend provides a small utility to register callbacks executed asynchronously whenever `console.error` is called, and a React hook for ergonomic usage.

### API

- `registerConsoleErrorCallback(cb: (...args: unknown[]) => void): () => void`
  - Registers a callback; returns an unsubscribe function.
- `getConsoleErrorCallbackCount(): number`
  - Returns number of currently registered callbacks (for diagnostics).
- `enableConsoleErrorPatch(): void`
  - Manually apply the patch (usually not needed; registration auto-patches).
- `disableConsoleErrorPatch(): void`
  - Restore the original `console.error` (intended for tests).

### Behavior

- The original `console.error` is always called first.
- Callbacks run asynchronously in a microtask to avoid interfering with React render/error flows.
- Callbacks receive the same arguments passed to `console.error` (compatible with `console.log` style).
- The patch is applied once and is idempotent.

### Usage (Imperative)

```ts
import { registerConsoleErrorCallback } from "./src/devtool/consoleErrorRegistry";

const unsubscribe = registerConsoleErrorCallback((...args) => {
  // Send to telemetry, show toast, collect metrics, etc.
});

// Later
unsubscribe();
```

### Usage (React Hook)

```ts
import { useConsoleErrorCallback } from "./src/hooks/useConsoleErrorCallback";

export function ErrorTelemetryBinder() {
  useConsoleErrorCallback((...args) => {
    // Runs asynchronously after the original console.error
    // e.g., send to monitoring service
  });
  return null;
}
```

### Frontend Component Organization Rules

- `src/components/` is **only for generic, reusable components** that can be used across multiple pages.
- **Page-specific components** must go in `src/pages/<pagename>/`.
- If a page has specific components, place both the page and its components in `src/pages/<pagename>/`.
  - **Example:** For the dashboard page:
    - Page: `src/pages/dashboard/Dashboard.tsx`
    - Specific components: `src/pages/dashboard/DashboardWidget.tsx`, `src/pages/dashboard/ChartPanel.tsx`, etc.
    - Do **not** place dashboard-specific components in `src/components/`.

## Test Setup Enforcement

All test files must import the shared test setup (`import '../../../../test/setup'`). This is enforced by the `test:prepare` script, which runs automatically before linting (see `package.json` lint script). If any test file is missing the setup import, lint will fail. To fix, run `bun run test:fix`.

Run tests locally:

```bash
bun test
```

### Concurrent Testing (Bun v1.3+)

Bun 1.3 introduces support for concurrent test execution, which can significantly speed up test runs. This feature is configured in `bunfig.toml` and can be enabled via CLI flag:

```bash
# Run tests with concurrent execution
bun run test:concurrent
# Or directly:
bun test --concurrent
```

**Current Status**: Concurrent mode is disabled by default because some tests share global state (localStorage, DOM) which causes race conditions when tests run in parallel. 

**To enable concurrent testing**:
1. Uncomment `concurrent = true` in `bunfig.toml`
2. Optionally adjust `maxConcurrency` to limit parallel test count (default: 20)

**Performance**: 
- Sequential mode: ~15s for 451 tests
- Concurrent mode: Currently has test failures due to shared state

**Future Work**: Tests need to be refactored to properly isolate state:
- Use per-test localStorage instances instead of shared global
- Ensure DOM cleanup between concurrent tests
- Avoid module-scoped test state

For more information, see the [Bun v1.3 concurrent testing blog post](https://bun.com/blog/bun-v1.3#concurrent-testing-with-bun-test).

> **Note**: Bun 1.2.23 rejects registering lifecycle hooks from inside a running test. We preload `@testing-library/react` in `test/setup.ts` so its automatic cleanup attaches to `afterEach` before any spec runs. If you reorganize the shared setup, keep that top-level import in place or the suite will fail with "Cannot call afterEach() inside a test" errors.

Run linter and typecheck:

```bash
bun run lint
```
