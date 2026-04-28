<!-- DOCTOC SKIP -->

# [FIX]: WebSocket Reconnect Resilience and Frontend Safety Guards

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in security/performance review 2026-04-28_

## 🎯 Objective

Fix four related issues in the frontend WebSocket layer and `App.tsx` command-output handling:

1. **No exponential backoff** — all tabs reconnect at exactly 1s after disconnect, causing synchronized storms on HA supervisor restarts.
2. **No error handling on `JSON.parse`** — malformed server frames silently corrupt Redux state.
3. **`globalThis`-based configuration** — `__SRAT_WS_INACTIVITY_MS` / `__SRAT_WS_RECONNECT_MS` read from the global scope; any same-origin script can set them to 0.
4. **`mergeCommandLines` GC pressure** — allocates a new `Set` and arrays on every `command_output` WebSocket event.
5. **`localStorage` JSON.parse without try/catch** in `issueHooks.ts` and `SystemMetricsAccordion.tsx` can crash tabs.

## 🛠️ Technical Specifications

- **Inputs:** `wsApi.ts` `onCacheEntryAdded` lifecycle, `App.tsx` command session state, `issueHooks.ts`, `SystemMetricsAccordion.tsx`
- **Outputs:** Resilient reconnect with jitter; safe frame parsing; no globalThis dependency; reduced GC pressure
- **Dependencies:** `frontend/src/store/wsApi.ts`, `frontend/src/App.tsx`, `frontend/src/hooks/issueHooks.ts`, `frontend/src/pages/dashboard/metrics/SystemMetricsAccordion.tsx`

## 📝 Task List

- [ ] Task 1: Add `reconnectAttempt` counter to the `onCacheEntryAdded` scope; implement exponential backoff: `delay = Math.min(base * 2 ** attempt, maxDelay) + Math.random() * 200`; cap at 30s; reset counter on successful `open`
- [ ] Task 2: Wrap `JSON.parse(data)` in `wsApi.ts` listener in try/catch; log malformed frames with `console.warn` and skip
- [ ] Task 3: Remove `globalThis.__SRAT_WS_INACTIVITY_MS` and `__SRAT_WS_RECONNECT_MS` configuration channel; expose timing via RTK Query `createApi` options or a `wsConfig` argument to the endpoint instead
- [ ] Task 4: Replace `mergeCommandLines` in `App.tsx` with a `Map<string, CommandOutputLineSnapshot>` keyed by `${timestamp}:${channel}:${line}`; update `mergeCommandSession` to use the Map; convert back to array only when rendering
- [ ] Task 5: Fix `isStopped` race in `wsApi.ts` finally block: set `isStopped = true` before `ws?.close()`, and guard `setWsConnected` with `if (!isStopped)`
- [ ] Task 6: Wrap `JSON.parse(localStorage.getItem(IGNORED_ISSUES_KEY))` in `issueHooks.ts` in try/catch returning `[]`
- [ ] Task 7: Wrap `JSON.parse(storedVisibility)` in `SystemMetricsAccordion.tsx` in try/catch returning the default value
- [ ] Task 8: Add/extend tests: verify reconnect delay increases on repeated disconnects; verify malformed frame is discarded without Redux state mutation; verify localStorage failure returns default state
- [ ] Task 9: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-SEC-04, F-SEC-05, F-PERF-04, F-PERF-05, F-REL-02 resolved

## 🧠 Implementation Notes

```typescript
// wsApi.ts — exponential backoff
let reconnectAttempt = 0;

const scheduleReconnect = (reason: string) => {
    if (isStopped || reconnectTimer) return;
    const delay = Math.min(
        reconnectDelayMs * Math.pow(2, reconnectAttempt),
        30_000,
    ) + Math.floor(Math.random() * 200);
    reconnectAttempt++;
    reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        if (isStopped) return;
        if (ws) ws.close();
        connect();
    }, delay);
};

// Reset on successful open:
ws.addEventListener("open", () => {
    reconnectAttempt = 0;
    setWsConnected(true);
    scheduleInactivityTimer();
});
```

```typescript
// wsApi.ts — safe JSON parse
try {
    draft[eventTypeEnum] = JSON.parse(data);
} catch (e) {
    console.warn("[wsApi] Malformed event frame discarded", { eventType, error: e });
}
```

## 🔗 Code References & TODOs

- [ ] `TODO: frontend/src/store/wsApi.ts:142-150` — add exponential backoff
- [ ] `TODO: frontend/src/store/wsApi.ts:197` — wrap JSON.parse in try/catch
- [ ] `TODO: frontend/src/store/wsApi.ts:44-46` — remove globalThis configuration
- [ ] `TODO: frontend/src/App.tsx:34-57` — optimize mergeCommandLines
- [ ] `TODO: frontend/src/hooks/issueHooks.ts:8-10` — safe localStorage parse
- [ ] `TODO: frontend/src/pages/dashboard/metrics/SystemMetricsAccordion.tsx:83` — safe localStorage parse
