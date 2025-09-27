# srat-frontend

<!-- START doctoc -->
<!-- END doctoc -->

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

Note: This project uses Bun as the JavaScript runtime and package manager. See `bun.build.ts` for the build pipeline.

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

Run linter and typecheck:

```bash
bun run lint
```
