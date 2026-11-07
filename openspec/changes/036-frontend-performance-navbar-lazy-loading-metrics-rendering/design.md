# Design

## Context

Improve frontend rendering performance in three high-impact areas: 1. **NavBar eager loading** — all 7 page components (including Swagger with its large `openapi-explorer` dependency) are instantiated at module load. All their hooks, RTK Query subscriptions, and `useEffect` calls fire on first render regardless of the active tab.

## Goals / Non-Goals

**Goals:**

- Deliver: Frontend Performance — NavBar Lazy Loading and Metrics Rendering

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Convert all tab page components in `ALL_TAB_CONFIGS` (NavBar.tsx) to `React.lazy(() => import('./path'))` with a `<Suspense fallback={<CircularProgress />}>` wrapper per `TabPanel`
- Extract `SSEDebugOverlay` (SSE message list, JSON dialog, heartbeat filter toggle) from `NavBar.tsx` into a standalone `<SSEDebugOverlay>` memoized component that only receives `lastMessages` and `filteredMessages` as props
- Extract `UpdateButton` (update available badge, restart dialog) from `NavBar.tsx` into a standalone `<UpdateButton>` memoized component
- Wrap all event handlers in `NavBar.tsx` (`handleOpenNavMenu`, `handleCloseNavMenu`, `handleMenuItemClick`, `handleChange`, `handleDoUpdate`, `handleShowJson`, `handleCopyToClipboard`) in `useCallback` with appropriate dependency arrays
- Consolidate `cpuHistory`, `connectionsHistory`, and `memoryHistory` in `DashboardMetrics.tsx` into a single `useReducer` action `UPDATE_HISTORY` that batches all three updates; update the single `useEffect` to dispatch one action
- Apply the same consolidation in `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, and `SystemMetricsAccordion.tsx`
- Remove `import.meta.hot.accept(() => window.location.reload())` blocks from `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, `ProcessMetrics.tsx`, and `MetricCard.tsx` — these defeat HMR for the entire page during development
- Add a `React.memo` wrapper to `MetricCard`, `MetricDetails`, and `SafeSparkLineChart` to prevent re-renders when their props have not changed

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
