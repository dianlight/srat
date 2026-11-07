# Proposal

## Why

Improve frontend rendering performance in three high-impact areas: 1. **NavBar eager loading** — all 7 page components (including Swagger with its large `openapi-explorer` dependency) are instantiated at module load. All their hooks, RTK Query subscriptions, and `useEffect` calls fire on first render regardless of the active tab.

## What Changes

- Convert all tab page components in `ALL_TAB_CONFIGS` (NavBar.tsx) to `React.lazy(() => import('./path'))` with a `<Suspense fallback={<CircularProgress />}>` wrapper per `TabPanel`
- Extract `SSEDebugOverlay` (SSE message list, JSON dialog, heartbeat filter toggle) from `NavBar.tsx` into a standalone `<SSEDebugOverlay>` memoized component that only receives `lastMessages` and `filteredMessages` as props
- Extract `UpdateButton` (update available badge, restart dialog) from `NavBar.tsx` into a standalone `<UpdateButton>` memoized component
- Wrap all event handlers in `NavBar.tsx` (`handleOpenNavMenu`, `handleCloseNavMenu`, `handleMenuItemClick`, `handleChange`, `handleDoUpdate`, `handleShowJson`, `handleCopyToClipboard`) in `useCallback` with appropriate dependency arrays
- Consolidate `cpuHistory`, `connectionsHistory`, and `memoryHistory` in `DashboardMetrics.tsx` into a single `useReducer` action `UPDATE_HISTORY` that batches all three updates; update the single `useEffect` to dispatch one action
- Apply the same consolidation in `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, and `SystemMetricsAccordion.tsx`
- Remove `import.meta.hot.accept(() => window.location.reload())` blocks from `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, `ProcessMetrics.tsx`, and `MetricCard.tsx` — these defeat HMR for the entire page during development
- Add a `React.memo` wrapper to `MetricCard`, `MetricDetails`, and `SafeSparkLineChart` to prevent re-renders when their props have not changed
- Write/update tests to verify lazy-loaded pages render correctly with `Suspense`; confirm `DashboardMetrics` only triggers one state update per heartbeat (use render count assertion)
- Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-PERF-01, F-PERF-02, F-PERF-03, F-QUAL-02 resolved

## Capabilities

### New Capabilities

- `frontend-performance-navbar-lazy-loading-metrics-rendering`: Frontend Performance — NavBar Lazy Loading and Metrics Rendering

### Modified Capabilities

(none)

## Impact

- **Inputs:** `frontend/src/components/NavBar.tsx`, `frontend/src/pages/dashboard/DashboardMetrics.tsx`, `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, `SystemMetricsAccordion.tsx`
- **Outputs:** Pages lazy-loaded; NavBar stable across heartbeats; metrics update in single state transitions
- **Dependencies:** `React.lazy`, `React.Suspense`, `useCallback`, `useReducer`
- Legacy task: `docs/tasks/036_frontend-performance-navbar-metrics-rendering.md`
