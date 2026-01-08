# MSW Test Migration Tracking

This document tracks the migration of existing frontend tests to use MSW (Mock Service Worker) for API mocking.

## Migration Status Overview

- **Total Tests**: 60 test files
- **Already Using MSW**: 2 (MSW infrastructure tests)
- **Verified Working with MSW**: 10 (tests #1-10 verified - 97 tests passing)
- **Require Analysis**: 48
- **Migrated**: 10
- **In Progress**: 0

## Latest Test Results

### ✅ Verified Working (2026-01-08)
- `test/__tests__/msw-smoke.test.ts` - 2/2 tests passing
- `src/hooks/__tests__/healthHook.test.ts` - 10/10 tests passing
- **Tests #1-10 Batch Verification (97 total tests passing)**:
  - `src/components/__tests__/BaseConfigModal.test.tsx` - 19/19 ✅
  - `src/components/__tests__/ErrorBoundary.test.tsx` - 4/4 ✅
  - `src/components/__tests__/FontAwesomeSvgIcon.test.tsx` - 1/1 ✅
  - `src/components/__tests__/Footer.test.tsx` - 6/6 ✅
  - `src/components/__tests__/IssueCard.test.tsx` - 6/6 ✅
  - `src/components/__tests__/NavBar.test.tsx` - 42/42 ✅
  - `src/components/__tests__/NotificationCenter.test.tsx` - 3/3 ✅
  - `src/components/__tests__/PreviewDialog.test.tsx` - 4/4 ✅
  - `src/hooks/__tests__/githubNewsHook.test.ts` - 6/6 ✅
  - `src/hooks/__tests__/healthHook.test.ts` - 10/10 ✅ (re-verified)

**Note**: All tests work seamlessly with MSW! No code changes were needed.

## Migration Categories

### Category 1: Hook Tests with API Calls (High Priority)
These tests use RTK Query hooks and will benefit most from MSW mocking.

| File | Status | API Endpoints Used | Notes |
|------|--------|-------------------|-------|
| `src/hooks/__tests__/githubNewsHook.test.ts` | ✅ Verified | GitHub API | 6/6 tests passing - External API mocked |
| `src/hooks/__tests__/healthHook.test.ts` | ✅ Verified | `/api/health` | 10/10 tests passing with MSW |
| `src/hooks/__tests__/issueHooks.test.ts` | ⏳ Not Started | Issue endpoints | Check which endpoints |
| `src/hooks/__tests__/shareHook.test.tsx` | ⏳ Not Started | `/api/shares` | Already has MSW handler |
| `src/hooks/__tests__/volumeHook.test.ts` | ⏳ Not Started | `/api/volumes` | Already has MSW handler |

### Category 2: Component Tests with Data Fetching (Medium Priority)
Components that fetch data and would benefit from realistic API responses.

| File | Status | Components Tested | Notes |
|------|--------|------------------|-------|
| `src/pages/dashboard/__tests__/Dashboard.test.tsx` | ⏳ Not Started | Dashboard | Main dashboard with health data |
| `src/pages/shares/__tests__/Shares.test.tsx` | ⏳ Not Started | Shares list | Uses share hooks |
| `src/pages/volumes/__tests__/Volumes.test.tsx` | ⏳ Not Started | Volumes list | Uses volume hooks |
| `src/pages/users/__tests__/Users.test.tsx` | ⏳ Not Started | Users list | User management |
| `src/pages/settings/__tests__/Settings.test.tsx` | ⏳ Not Started | Settings page | Settings API |

### Category 3: UI Component Tests (Low Priority)
Pure UI components without API calls - may not need changes.

| File | Status | Type | Notes |
|------|--------|------|-------|
| `src/components/__tests__/BaseConfigModal.test.tsx` | ✅ Verified | UI Component | 19/19 tests passing |
| `src/components/__tests__/ErrorBoundary.test.tsx` | ✅ Verified | UI Component | 4/4 tests passing |
| `src/components/__tests__/FontAwesomeSvgIcon.test.tsx` | ✅ Verified | UI Component | 1/1 test passing |
| `src/components/__tests__/Footer.test.tsx` | ✅ Verified | UI Component | 6/6 tests passing |
| `src/components/__tests__/IssueCard.test.tsx` | ✅ Verified | UI Component | 6/6 tests passing |
| `src/components/__tests__/NavBar.test.tsx` | ✅ Verified | UI Component | 42/42 tests passing |
| `src/components/__tests__/NotificationCenter.test.tsx` | ✅ Verified | UI Component | 3/3 tests passing |
| `src/components/__tests__/PreviewDialog.test.tsx` | ✅ Verified | UI Component | 4/4 tests passing |

### Category 4: Store/State Tests (Low Priority)
Redux store and middleware tests - may not need changes.

| File | Status | Type | Notes |
|------|--------|------|-------|
| `src/store/__tests__/errorSlice.test.ts` | ⏳ Not Started | Redux Slice | No API calls |
| `src/store/__tests__/mdcMiddleware.test.ts` | ⏳ Not Started | Middleware | No API calls |
| `src/store/__tests__/mdcSlice.test.ts` | ⏳ Not Started | Redux Slice | No API calls |
| `src/store/__tests__/store.test.ts` | ⏳ Not Started | Store | No API calls |

### Category 5: Utility and Helper Tests (Low Priority)
Pure utility functions - should not need changes.

| File | Status | Type | Notes |
|------|--------|------|-------|
| `src/pages/dashboard/metrics/__tests__/utils.test.ts` | ⏳ Not Started | Utils | No API calls |
| `src/pages/shares/__tests__/utils.test.ts` | ⏳ Not Started | Utils | No API calls |
| `src/pages/volumes/__tests__/utils.test.ts` | ⏳ Not Started | Utils | No API calls |
| `src/macro/__tests__/Environment.test.ts` | ⏳ Not Started | Macro | No API calls |

### Category 6: Already Using MSW (Complete)
Tests that are part of the MSW infrastructure.

| File | Status | Type | Notes |
|------|--------|------|-------|
| `test/__tests__/msw-smoke.test.ts` | ✅ Complete | MSW Infrastructure | MSW smoke tests |
| `test/__tests__/msw-integration.test.tsx` | ✅ Complete | MSW Infrastructure | MSW integration example |

## Complete Test File List

All 60 test files discovered in the frontend:

1. ✅ `src/components/__tests__/BaseConfigModal.test.tsx` - 19/19 tests passing
2. ✅ `src/components/__tests__/ErrorBoundary.test.tsx` - 4/4 tests passing
3. ✅ `src/components/__tests__/FontAwesomeSvgIcon.test.tsx` - 1/1 test passing
4. ✅ `src/components/__tests__/Footer.test.tsx` - 6/6 tests passing
5. ✅ `src/components/__tests__/IssueCard.test.tsx` - 6/6 tests passing
6. ✅ `src/components/__tests__/NavBar.test.tsx` - 42/42 tests passing
7. ✅ `src/components/__tests__/NotificationCenter.test.tsx` - 3/3 tests passing
8. ✅ `src/components/__tests__/PreviewDialog.test.tsx` - 4/4 tests passing
9. ✅ `src/hooks/__tests__/githubNewsHook.test.ts` - 6/6 tests passing
10. ✅ `src/hooks/__tests__/healthHook.test.ts` - 10/10 tests passing
11. ⏳ `src/hooks/__tests__/issueHooks.test.ts`
12. ⏳ `src/hooks/__tests__/shareHook.test.tsx`
13. ⏳ `src/hooks/__tests__/useConsoleErrorCallback.test.ts`
14. ⏳ `src/hooks/__tests__/useTelemetryModal.test.ts`
15. ⏳ `src/hooks/__tests__/volumeHook.test.ts`
16. ⏳ `src/macro/__tests__/Environment.test.ts`
17. ⏳ `src/pages/__tests__/SmbConf.test.tsx`
18. ⏳ `src/pages/__tests__/Swagger.test.tsx`
19. ⏳ `src/pages/dashboard/__tests__/ActionableItems.test.tsx`
20. ⏳ `src/pages/dashboard/__tests__/BasicDashboard.test.tsx`
21. ⏳ `src/pages/dashboard/__tests__/CollapsibleSections.test.tsx`
22. ⏳ `src/pages/dashboard/__tests__/Dashboard.test.tsx`
23. ⏳ `src/pages/dashboard/__tests__/DashboardActions.test.tsx`
24. ⏳ `src/pages/dashboard/__tests__/DashboardMetrics.test.tsx`
25. ⏳ `src/pages/dashboard/__tests__/DashboardTourStep.test.tsx`
26. ⏳ `src/pages/dashboard/__tests__/SystemMetrics.test.tsx`
27. ⏳ `src/pages/dashboard/metrics/__tests__/DiskHealthMetrics.test.tsx`
28. ⏳ `src/pages/dashboard/metrics/__tests__/utils.test.ts`
29. ⏳ `src/pages/settings/__tests__/Settings.test.tsx`
30. ⏳ `src/pages/settings/__tests__/SettingsTourStep.test.tsx`
31. ⏳ `src/pages/shares/__tests__/ShareActions.test.tsx`
32. ⏳ `src/pages/shares/__tests__/ShareDetailsPanel.test.tsx`
33. ⏳ `src/pages/shares/__tests__/ShareEditDialog.test.tsx`
34. ⏳ `src/pages/shares/__tests__/Shares.localStorage.test.tsx`
35. ⏳ `src/pages/shares/__tests__/Shares.test.tsx`
36. ⏳ `src/pages/shares/__tests__/SharesTourStep.test.tsx`
37. ⏳ `src/pages/shares/__tests__/utils.test.ts`
38. ⏳ `src/pages/shares/components/__tests__/ShareEditForm.test.tsx`
39. ⏳ `src/pages/shares/components/__tests__/SharesTreeView.test.tsx`
40. ⏳ `src/pages/users/__tests__/UserActions.test.tsx`
41. ⏳ `src/pages/users/__tests__/UserEditDialog.test.tsx`
42. ⏳ `src/pages/users/__tests__/Users.test.tsx`
43. ⏳ `src/pages/users/__tests__/UsersSteps.test.tsx`
44. ⏳ `src/pages/users/components/__tests__/UserDetailsPanel.test.tsx`
45. ⏳ `src/pages/users/components/__tests__/UserEditForm.test.tsx`
46. ⏳ `src/pages/users/components/__tests__/UsersTreeView.test.tsx`
47. ⏳ `src/pages/volumes/__tests__/Volumes.restore.test.tsx`
48. ⏳ `src/pages/volumes/__tests__/Volumes.test.tsx`
49. ⏳ `src/pages/volumes/__tests__/VolumesTourStep.test.tsx`
50. ⏳ `src/pages/volumes/__tests__/utils.test.ts`
51. ⏳ `src/pages/volumes/components/__tests__/HDIdleDiskSettings.applyCancel.test.tsx`
52. ⏳ `src/pages/volumes/components/__tests__/HDIdleDiskSettings.test.tsx`
53. ⏳ `src/pages/volumes/components/__tests__/PartitionActions.test.tsx`
54. ⏳ `src/pages/volumes/components/__tests__/SmartStatusPanel.test.tsx`
55. ⏳ `src/pages/volumes/components/__tests__/VolumeDetailsPanel.test.tsx`
56. ⏳ `src/pages/volumes/components/__tests__/VolumeMountDialog.test.tsx`
57. ⏳ `src/pages/volumes/components/__tests__/VolumesTreeView.test.tsx`
58. ⏳ `src/pages/volumes/components/__tests__/SmartStatusPanel.test.tsx`
59. ✅ `test/__tests__/msw-smoke.test.ts`
60. ✅ `test/__tests__/msw-integration.test.tsx`

## Migration Progress

### Current Sprint
- [ ] Verify all existing tests still pass with MSW infrastructure
- [ ] Identify tests that make API calls
- [ ] Create migration plan for high-priority tests
- [ ] Migrate first batch of hook tests

### Next Steps
1. Run all tests with current MSW setup to establish baseline
2. Migrate Category 1 (Hook tests) first
3. Migrate Category 2 (Component tests with data fetching)
4. Verify Category 3-5 don't need changes
5. Update documentation with migration patterns

## Migration Guidelines

### For Tests That Don't Need Changes
Most tests will continue to work as-is since they already import `test/setup.ts` which now includes MSW. These tests just benefit from having MSW available.

### For Tests That Need MSW Handlers
1. Identify which API endpoints the test uses
2. Verify MSW handlers exist in `src/mocks/generatedHandlers.ts` or `src/mocks/streamingHandlers.ts`
3. If custom responses needed, use `getMswServer()` to add runtime handlers
4. Test with `--rerun-each 10` to ensure no flakiness

### Example Migration Pattern
```typescript
// Before: Test relies on global fetch mock
it("fetches health data", async () => {
  // Test code - uses global fetch mock
});

// After: Test uses MSW (no code changes if using default handlers)
it("fetches health data", async () => {
  // Same test code - MSW automatically mocks /api/health
});

// After: Test needs custom response
it("handles health error", async () => {
  const { getMswServer } = await import("../../../test/bun-setup");
  const { http, HttpResponse } = await import("msw");
  
  const server = await getMswServer();
  server.use(
    http.get('/api/health', () => new HttpResponse(null, { status: 500 }))
  );
  
  // Test error handling
});
```

## CI/CD Integration

The migration is validated through GitHub Actions workflow:
- **Workflow**: `.github/workflows/build.yaml`
- **Job**: `test-frontend`
- **Command**: `bun test:ci --coverage-reporter=lcov`
- **Success Criteria**: All tests pass, coverage maintained

## Notes

- MSW is already integrated into `test/setup.ts` which is imported by all tests
- Most tests won't need code changes - they automatically get MSW benefits
- Only tests needing custom API responses will require modifications
- Focus migration effort on tests that currently use fetch mocks or would benefit from realistic API responses
