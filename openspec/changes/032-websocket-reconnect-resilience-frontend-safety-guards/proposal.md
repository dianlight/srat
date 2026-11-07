# Proposal

## Why

Fix four related issues in the frontend WebSocket layer and `App.tsx` command-output handling: 1. **No exponential backoff** — all tabs reconnect at exactly 1s after disconnect, causing synchronized storms on HA supervisor restarts.

## What Changes

- Add `reconnectAttempt` counter to the `onCacheEntryAdded` scope; implement exponential backoff: `delay = Math.min(base * 2 ** attempt, maxDelay) + Math.random() * 200`; cap at 30s; reset counter on successful `open`
- Wrap `JSON.parse(data)` in `wsApi.ts` listener in try/catch; log malformed frames with `console.warn` and skip
- Remove `globalThis.__SRAT_WS_INACTIVITY_MS` and `__SRAT_WS_RECONNECT_MS` configuration channel; expose timing via RTK Query `createApi` options or a `wsConfig` argument to the endpoint instead
- Replace `mergeCommandLines` in `App.tsx` with a `Map<string, CommandOutputLineSnapshot>` keyed by `${timestamp}:${channel}:${line}`; update `mergeCommandSession` to use the Map; convert back to array only when rendering
- Fix `isStopped` race in `wsApi.ts` finally block: set `isStopped = true` before `ws?.close()`, and guard `setWsConnected` with `if (!isStopped)`
- Wrap `JSON.parse(localStorage.getItem(IGNORED_ISSUES_KEY))` in `issueHooks.ts` in try/catch returning `[]`
- Wrap `JSON.parse(storedVisibility)` in `SystemMetricsAccordion.tsx` in try/catch returning the default value
- Add/extend tests: verify reconnect delay increases on repeated disconnects; verify malformed frame is discarded without Redux state mutation; verify localStorage failure returns default state
- Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-SEC-04, F-SEC-05, F-PERF-04, F-PERF-05, F-REL-02 resolved

## Capabilities

### New Capabilities

- `websocket-reconnect-resilience-frontend-safety-guards`: WebSocket Reconnect Resilience and Frontend Safety Guards

### Modified Capabilities

(none)

## Impact

- **Inputs:** `wsApi.ts` `onCacheEntryAdded` lifecycle, `App.tsx` command session state, `issueHooks.ts`, `SystemMetricsAccordion.tsx`
- **Outputs:** Resilient reconnect with jitter; safe frame parsing; no globalThis dependency; reduced GC pressure
- **Dependencies:** `frontend/src/store/wsApi.ts`, `frontend/src/App.tsx`, `frontend/src/hooks/issueHooks.ts`, `frontend/src/pages/dashboard/metrics/SystemMetricsAccordion.tsx`
- Legacy task: `docs/tasks/032_websocket-reconnect-resilience.md`
