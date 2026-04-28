<!-- DOCTOC SKIP -->

# [REFACTOR]: Frontend Performance — NavBar Lazy Loading and Metrics Rendering

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in performance review 2026-04-28_

## 🎯 Objective

Improve frontend rendering performance in three high-impact areas:

1. **NavBar eager loading** — all 7 page components (including Swagger with its large `openapi-explorer` dependency) are instantiated at module load. All their hooks, RTK Query subscriptions, and `useEffect` calls fire on first render regardless of the active tab.
2. **NavBar re-renders at heartbeat frequency** — 848 lines in a single component with 15+ state variables and unmoistened handlers re-render the entire navigation chrome on every WebSocket heartbeat (~every few seconds).
3. **Metrics history triple-state anti-pattern** — `DashboardMetrics`, `NetworkHealthMetrics`, `DiskHealthMetrics`, and `SystemMetricsAccordion` each maintain 3+ separate `useState` history arrays updated in separate `useEffect` blocks, causing multiple re-renders per heartbeat.

> *These changes are pure performance refactors: no behavior changes, no new features.*

## 🛠️ Technical Specifications

- **Inputs:** `frontend/src/components/NavBar.tsx`, `frontend/src/pages/dashboard/DashboardMetrics.tsx`, `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, `SystemMetricsAccordion.tsx`
- **Outputs:** Pages lazy-loaded; NavBar stable across heartbeats; metrics update in single state transitions
- **Dependencies:** `React.lazy`, `React.Suspense`, `useCallback`, `useReducer`

## 📝 Task List

- [ ] Task 1: Convert all tab page components in `ALL_TAB_CONFIGS` (NavBar.tsx) to `React.lazy(() => import('./path'))` with a `<Suspense fallback={<CircularProgress />}>` wrapper per `TabPanel`
- [ ] Task 2: Extract `SSEDebugOverlay` (SSE message list, JSON dialog, heartbeat filter toggle) from `NavBar.tsx` into a standalone `<SSEDebugOverlay>` memoized component that only receives `lastMessages` and `filteredMessages` as props
- [ ] Task 3: Extract `UpdateButton` (update available badge, restart dialog) from `NavBar.tsx` into a standalone `<UpdateButton>` memoized component
- [ ] Task 4: Wrap all event handlers in `NavBar.tsx` (`handleOpenNavMenu`, `handleCloseNavMenu`, `handleMenuItemClick`, `handleChange`, `handleDoUpdate`, `handleShowJson`, `handleCopyToClipboard`) in `useCallback` with appropriate dependency arrays
- [ ] Task 5: Consolidate `cpuHistory`, `connectionsHistory`, and `memoryHistory` in `DashboardMetrics.tsx` into a single `useReducer` action `UPDATE_HISTORY` that batches all three updates; update the single `useEffect` to dispatch one action
- [ ] Task 6: Apply the same consolidation in `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, and `SystemMetricsAccordion.tsx`
- [ ] Task 7: Remove `import.meta.hot.accept(() => window.location.reload())` blocks from `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, `ProcessMetrics.tsx`, and `MetricCard.tsx` — these defeat HMR for the entire page during development
- [ ] Task 8: Add a `React.memo` wrapper to `MetricCard`, `MetricDetails`, and `SafeSparkLineChart` to prevent re-renders when their props have not changed
- [ ] Task 9: Write/update tests to verify lazy-loaded pages render correctly with `Suspense`; confirm `DashboardMetrics` only triggers one state update per heartbeat (use render count assertion)
- [ ] Task 10: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-PERF-01, F-PERF-02, F-PERF-03, F-QUAL-02 resolved

## 🧠 Implementation Notes

```tsx
// NavBar.tsx — lazy tab pages
const Dashboard = React.lazy(() => import('../pages/dashboard/Dashboard'));
const Volumes = React.lazy(() => import('../pages/volumes/Volumes'));
// ... etc.

// In ALL_TAB_CONFIGS or TabPanel render:
<Suspense fallback={<Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}><CircularProgress /></Box>}>
    {activeTab === TabIDs.Dashboard && <Dashboard />}
</Suspense>
```

```typescript
// DashboardMetrics.tsx — single useReducer for history
type HistoryState = {
    cpuHistory: Record<string, number[]>;
    connectionsHistory: Record<string, number[]>;
    memoryHistory: Record<string, number[]>;
};
type HistoryAction = { type: 'UPDATE'; payload: Partial<HistoryState> };

const [history, dispatchHistory] = useReducer(
    (state: HistoryState, action: HistoryAction): HistoryState => ({
        ...state, ...action.payload
    }),
    { cpuHistory: {}, connectionsHistory: {}, memoryHistory: {} }
);

useEffect(() => {
    if (!health?.samba_process_status) return;
    // compute all three updates
    dispatchHistory({ type: 'UPDATE', payload: { cpuHistory: ..., connectionsHistory: ..., memoryHistory: ... } });
}, [health]);
```

## 🔗 Code References & TODOs

- [ ] `TODO: frontend/src/components/NavBar.tsx:100-145` — convert to React.lazy
- [ ] `TODO: frontend/src/components/NavBar.tsx:400-435` — wrap handlers in useCallback
- [ ] `TODO: frontend/src/pages/dashboard/DashboardMetrics.tsx:13-19` — consolidate history state
- [ ] `TODO: frontend/src/pages/dashboard/metrics/NetworkHealthMetrics.tsx` — same consolidation
- [ ] `TODO: frontend/src/pages/dashboard/metrics/DiskHealthMetrics.tsx` — same consolidation
