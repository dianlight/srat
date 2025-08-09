# srat-frontend

To install dependencies:

```bash
bun install
```

To run:

```bash
bun run index.ts
```

This project was created using `bun init` in bun v1.1.34. [Bun](https://bun.sh) is a fast all-in-one JavaScript runtime.

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
