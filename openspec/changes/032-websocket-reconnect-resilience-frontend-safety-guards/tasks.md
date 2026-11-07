# Tasks

## 1. Implementation

- [ ] 1.1 Add `reconnectAttempt` counter to the `onCacheEntryAdded` scope; implement exponential backoff: `delay = Math.min(base * 2 ** attempt, maxDelay) + Math.random() * 200`; cap at 30s; reset counter on successful `open` — verified by targeted tests
- [ ] 1.2 Wrap `JSON.parse(data)` in `wsApi.ts` listener in try/catch; log malformed frames with `console.warn` and skip — verified by targeted tests
- [ ] 1.3 Remove `globalThis.__SRAT_WS_INACTIVITY_MS` and `__SRAT_WS_RECONNECT_MS` configuration channel; expose timing via RTK Query `createApi` options or a `wsConfig` argument to the endpoint instead — verified by targeted tests
- [ ] 1.4 Replace `mergeCommandLines` in `App.tsx` with a `Map<string, CommandOutputLineSnapshot>` keyed by `${timestamp}:${channel}:${line}`; update `mergeCommandSession` to use the Map; convert back to array only when rendering — verified by targeted tests
- [ ] 1.5 Fix `isStopped` race in `wsApi.ts` finally block: set `isStopped = true` before `ws?.close()`, and guard `setWsConnected` with `if (!isStopped)` — verified by targeted tests
- [ ] 1.6 Wrap `JSON.parse(localStorage.getItem(IGNORED_ISSUES_KEY))` in `issueHooks.ts` in try/catch returning `[]` — verified by targeted tests
- [ ] 1.7 Wrap `JSON.parse(storedVisibility)` in `SystemMetricsAccordion.tsx` in try/catch returning the default value — verified by targeted tests
- [ ] 1.8 Add/extend tests: verify reconnect delay increases on repeated disconnects; verify malformed frame is discarded without Redux state mutation; verify localStorage failure returns default state — verified by targeted tests
- [ ] 1.9 Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-SEC-04, F-SEC-05, F-PERF-04, F-PERF-05, F-REL-02 resolved — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
