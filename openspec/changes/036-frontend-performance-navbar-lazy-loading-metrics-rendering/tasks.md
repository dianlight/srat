# Tasks

## 1. Implementation

- [ ] 1.1 Convert all tab page components in `ALL_TAB_CONFIGS` (NavBar.tsx) to `React.lazy(() => import('./path'))` with a `<Suspense fallback={<CircularProgress />}>` wrapper per `TabPanel` — verified by targeted tests
- [ ] 1.2 Extract `SSEDebugOverlay` (SSE message list, JSON dialog, heartbeat filter toggle) from `NavBar.tsx` into a standalone `<SSEDebugOverlay>` memoized component that only receives `lastMessages` and `filteredMessages` as props — verified by targeted tests
- [ ] 1.3 Extract `UpdateButton` (update available badge, restart dialog) from `NavBar.tsx` into a standalone `<UpdateButton>` memoized component — verified by targeted tests
- [ ] 1.4 Wrap all event handlers in `NavBar.tsx` (`handleOpenNavMenu`, `handleCloseNavMenu`, `handleMenuItemClick`, `handleChange`, `handleDoUpdate`, `handleShowJson`, `handleCopyToClipboard`) in `useCallback` with appropriate dependency arrays — verified by targeted tests
- [ ] 1.5 Consolidate `cpuHistory`, `connectionsHistory`, and `memoryHistory` in `DashboardMetrics.tsx` into a single `useReducer` action `UPDATE_HISTORY` that batches all three updates; update the single `useEffect` to dispatch one action — verified by targeted tests
- [ ] 1.6 Apply the same consolidation in `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, and `SystemMetricsAccordion.tsx` — verified by targeted tests
- [ ] 1.7 Remove `import.meta.hot.accept(() => window.location.reload())` blocks from `NetworkHealthMetrics.tsx`, `DiskHealthMetrics.tsx`, `ProcessMetrics.tsx`, and `MetricCard.tsx` — these defeat HMR for the entire page during development — verified by targeted tests
- [ ] 1.8 Add a `React.memo` wrapper to `MetricCard`, `MetricDetails`, and `SafeSparkLineChart` to prevent re-renders when their props have not changed — verified by targeted tests
- [ ] 1.9 Write/update tests to verify lazy-loaded pages render correctly with `Suspense`; confirm `DashboardMetrics` only triggers one state update per heartbeat (use render count assertion) — verified by targeted tests
- [ ] 1.10 Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark F-PERF-01, F-PERF-02, F-PERF-03, F-QUAL-02 resolved — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
