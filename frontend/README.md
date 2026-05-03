# SRAT Frontend

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Mise-based Frontend Workflows](#mise-based-frontend-workflows)
- [Getting Started](#getting-started)
  - [Setup Wizard](#setup-wizard)
  - [Guided Tour](#guided-tour)
- [Console Error Callback Registry](#console-error-callback-registry)
  - [API](#api)
  - [Behavior](#behavior)
  - [Usage (Imperative)](#usage-imperative)
  - [Usage (React Hook)](#usage-react-hook)
  - [Frontend Component Organization Rules](#frontend-component-organization-rules)
- [Test Setup Enforcement](#test-setup-enforcement)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Mise-based Frontend Workflows

All frontend build, test, and lint workflows are now managed by [mise](https://mise.jdx.dev).

**Common commands:**

```sh
# Build frontend
mise run //frontend:build
# Run frontend tests
mise run //frontend:test
# Lint frontend
mise run //frontend:lint
```

See `.mise.toml` for all available tasks.

To start the dev server with hot reload:

```sh
mise run //frontend:dev
```

To build the production bundle (outputs to `../backend/src/web/static`):

```sh
mise run //frontend:build
```

**Note about API code generation:** The `bun run gen` command (RTK Query codegen from OpenAPI) currently fails due to a TypeScript version mismatch issue in `@rtk-query/codegen-openapi`. This is a [documented issue](https://github.com/reduxjs/redux-toolkit/issues/2425) in the Redux Toolkit repository. For now the workaround is to install `@rtk-query/codegen-openapi` globally with npm and run with node not bun.

**TypeScript 6.0/7.0:** This project uses TypeScript 6.0 final / 7.0 Preview (tsgo) with ES2022 target. Type checking uses `bun tsgo --noEmit` instead of regular `tsc`. See `TYPESCRIPT_MIGRATION.md` for migration details and configuration guidelines.

Note: This project uses Bun as the JavaScript runtime and package manager. See `bun.build.ts` for the build pipeline.

**Bun 1.3 Compatibility:** This project is fully compatible with Bun 1.3.0. The project has been tested with Bun 1.3.0 and all breaking changes have been reviewed. The project does not use any of the affected APIs (SQL client, YAML parser) that changed in Bun 1.3.

## Getting Started

SRAT includes two onboarding helpers to make first-time setup faster:

- **Setup Wizard** (first run + manual trigger)
- **Guided Tour** (contextual in-app help)

### Setup Wizard

- The wizard opens automatically on first run when core setup is still required (for example default admin password, missing hostname/workgroup, or telemetry decision pending with internet available).
- You can reopen it anytime from **Settings → Setup Wizard**.
- Wizard flow:
  1. **Security** — hostname, workgroup, optional admin password update
  2. **Network** — bind all interfaces or select specific interfaces
  3. **First Share** — optional first share name
  4. **Telemetry** — choose usage/error reporting mode
- Selecting **Skip Setup** closes the wizard and marks onboarding as seen.

### Guided Tour

- Use the **Help (?)** button in the top bar to start or stop the tour.
- Tour steps adapt to the current tab (Dashboard, Volumes, Shares, Users, Settings).
- Tour anchors are implemented via `data-tutor="reactour__tab...__step..."` attributes in UI components.

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
    // for example, send to monitoring service
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

All test files must import the shared test setup (`import '../../../../test/setup'`). This is enforced by the `bun ./scripts/add-test-setup.js` script, which runs automatically before linting and testing. If the setup import is missing, the script will add it to the top of the file.

Run tests locally:

```bash
mise run //frontend:test
```

**Testing Standards:**

- All user interactions MUST use `@testing-library/user-event` - the deprecated `fireEvent` API is strictly prohibited
- Import: `const userEvent = (await import("@testing-library/user-event")).default;`
- Setup: `const user = userEvent.setup();`
- Always await interactions: `await user.click(button)`, `await user.type(input, "text")`
- See `.github/copilot-instructions.md` for complete testing patterns

> **Note**: Bun 1.2.23 rejects registering lifecycle hooks from inside a running test. We preload `@testing-library/react` in `test/setup.ts` so its automatic cleanup attaches to `afterEach` before any spec runs. If you reorganize the shared setup, keep that top-level import in place or the suite will fail with "Cannot call afterEach() inside a test" errors.

Run linter and typecheck:

```bash
mise run //frontend:lint
```
