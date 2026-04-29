<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Quick Reference: High-Impact Patterns](#quick-reference-high-impact-patterns)
  - [1. MSW Body Consumption (Frontend Testing)](#1-msw-body-consumption-frontend-testing)
  - [2. ProblemSeverity Enum Family (Backend Testing)](#2-problemseverity-enum-family-backend-testing)
  - [3. CanUpgrade Semver Comparison (Backend Services)](#3-canupgrade-semver-comparison-backend-services)
  - [4. Service Architecture Diagram (Backend Services)](#4-service-architecture-diagram-backend-services)
  - [5. HA Supervisor API vs REST Proxy (Backend Services)](#5-ha-supervisor-api-vs-rest-proxy-backend-services)
  - [6. Test Cleanup Pattern (Frontend & Backend Testing)](#6-test-cleanup-pattern-frontend--backend-testing)
  - [7. RTK Query Lazy Hooks (Frontend)](#7-rtk-query-lazy-hooks-frontend)
  - [8. IssueCard Ignored-State Keying (Frontend)](#8-issuecard-ignored-state-keying-frontend)
  - [Links to Full Documentation](#links-to-full-documentation)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Quick Reference: High-Impact Patterns

Fast lookup for the 8 highest-value patterns discovered across optimization rounds. Copy-paste snippets to avoid rediscovering these patterns.

---

## 1. MSW Body Consumption (Frontend Testing)

**Problem**: `InvalidStateError: Body has already been used` during test reruns with `--rerun-each`  
**Solution**: Clone request before reading

```typescript
// ✅ CORRECT - Works with test reruns
import { http, HttpResponse } from "msw";

http.post("/api/filesystem/format", async ({ request }) => {
  const body = await request.clone().json(); // Clone first!
  const { filesystem, label } = body;

  return HttpResponse.json({ success: true });
}),

// ❌ WRONG - Fails on reruns
http.post("/api/filesystem/format", async ({ request }) => {
  const body = await request.json(); // Body consumed; can't reuse
  // ...
}),
```

**Why**: Request body is a stream (one-time read). Cloning preserves it for inspection across test reruns.

---

## 2. ProblemSeverity Enum Family (Backend Testing)

**Problem**: Using wrong enum family (`IssueSeverity` instead of `ProblemSeverity`) compiles but causes silent type mismatches  
**Solution**: Use correct enum family for `dto.Problem`

```go
// ✅ CORRECT - ProblemSeverity for dto.Problem
problem := &dto.Problem{
    ID:            "my-issue",
    Severity:      dto.ProblemSeverities.PROBLEMSEVERITYWARNING,  // Correct enum
    Title:         "Something needs attention",
    Message:       "Details here",
    Category:      dto.ProblemCategories.PROBLEMCATEGORYADDON,
}

// ❌ WRONG - IssueSeverity is different enum family
problem := &dto.Problem{
    Severity: dto.IssueSeverities.WARNING,  // Type mismatch!
}
```

**Why**: `dto.IssueSeverity` and `dto.ProblemSeverity` are distinct types generated from separate enums. Mixing them is silent but wrong.

**Reference**: `backend/src/dto/problem.go` (Problem definition), `backend/src/dto/issue_severity.go` (IssueSeverity), `backend/src/dto/problem_severity.go` (ProblemSeverity)

---

## 3. CanUpgrade Semver Comparison (Backend Services)

**Problem**: Setting `CanUpgrade` as simple boolean or version check fails; silent production bugs  
**Solution**: Use semver comparison via `Masterminds/semver`

```go
import "github.com/Masterminds/semver/v3"

// ✅ CORRECT - Semver comparison
func canUpgrade(installed, latest *string) bool {
    if installed == nil || latest == nil {
        return false
    }
    v1, err := semver.NewVersion(*installed)
    if err != nil {
        return false
    }
    v2, err := semver.NewVersion(*latest)
    if err != nil {
        return false
    }
    return v1.LessThan(v2)  // Proper version comparison
}

// Then in your DTO/service:
status.CanUpgrade = canUpgrade(status.Installed, status.Latest)

// ❌ WRONG - Simple string or flag check
status.CanUpgrade = status.Installed != ""  // Ignores actual versions!
status.CanUpgrade = status.Latest != ""     // Just checks if field exists
```

**Why**: `1.9 < 1.10` lexically but not numerically. Semver handles pre-release, patch, build metadata correctly.

**Reference**: `backend/src/service/homeassistant_component_service.go` (canonical implementation)

---

## 4. Service Architecture Diagram (Backend Services)

**Problem**: Duplicate service responsibilities; confusion about where to call what  
**Solution**: Understand the service layer organization

```text
┌─────────────────────────────────────────────────────────┐
│ API Layer (api/)                                        │
│ - Receives HTTP requests                                │
│ - Delegates to services                                 │
└─────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────┐
│ Service Layer (service/)                                │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ ProblemService                                      │ │
│ │ - Centralized issue tracking                        │ │
│ │ - Emits problem_* WebSocket events                  │ │
│ │ - Used by: AddonConfigWatcherService,              │ │
│ │   HomeAssistantComponentService, RepairService      │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ RepairService                                       │ │
│ │ - Legacy repair command handling                    │ │
│ │ - Mirrors operations → ProblemService (best-effort) │ │
│ │ - Used by: API handlers, automation flows           │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ HomeAssistantComponentService                       │ │
│ │ - Tracks custom component lifecycle                 │ │
│ │ - Calls: ProblemService.Create/Dismiss              │ │
│ │ - Calls: RepairService.Create for restart flow      │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ AddonConfigWatcherService                           │ │
│ │ - Watches addon config changes                      │ │
│ │ - Calls: ProblemService.Create/Dismiss              │ │
│ │ - Fallback: HA persistent notifications             │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
└─────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────┐
│ Event Bus (events/)                                     │
│ - Broadcasts events to WebSocket subscribers            │
│ - BroadcasterService listens to CommandExecutionEvent   │
└─────────────────────────────────────────────────────────┘
```

**Key Insight**: `RepairService` mirrors to `ProblemService` with best-effort (non-fatal failures). New code should prefer `ProblemService` directly.

---

## 5. HA Supervisor API vs REST Proxy (Backend Services)

**Problem**: Using HA REST proxy for lifecycle ops causes 504 timeouts because HA drops connection during restart  
**Solution**: Use Supervisor API for lifecycle operations

```go
import "github.com/dianlight/srat/homeassistant/core"

// ✅ CORRECT - Use Supervisor API for lifecycle ops
func (s *HomeAssistantService) Restart(ctx context.Context) error {
    response, err := s.supervisorCoreClient.RestartCoreWithResponse(ctx)
    if err != nil {
        return fmt.Errorf("restart failed: %w", err)
    }
    // Handle response...
    return nil
}

// ❌ WRONG - REST proxy for lifecycle ops
func (s *HomeAssistantService) Restart(ctx context.Context) error {
    _, err := s.coreClient.CallServiceWithResponse(
        ctx,
        "homeassistant",
        "restart",
        nil,
    )
    // This hangs/times out because HA drops the connection!
    return err
}
```

**Why**: HA Core API is for entity states and service calls (stateless). Supervisor API is for addon/core lifecycle (manages connectivity).

**Reference**: `backend/src/service/homeassistant_service.go` (implementation), `backend/src/internal/appsetup/appsetup.go` (setup)

---

## 6. Test Cleanup Pattern (Frontend & Backend Testing)

**Frontend - Avoid Duplicate Cleanup**:

```typescript
// ✅ CORRECT - Let bun-setup.ts handle cleanup
describe("MyComponent", () => {
  test("should render", () => {
    render(<MyComponent />);
    expect(screen.getByText(/test/i)).toBeInTheDocument();
    // No manual cleanup() call
  });
});

// ❌ WRONG - Duplicate cleanup causes MUI timing issues
describe("MyComponent", () => {
  afterEach(() => {
    cleanup();  // bun-setup.ts already does this!
    document.body.innerHTML = "";  // And this!
  });
});
```

**Go - Cancel Before Wait**:

```go
// ✅ CORRECT - Cancel first, wait second
func (suite *ServiceTestSuite) TearDownTest() {
    if suite.cancel != nil {
        suite.cancel()  // Signal goroutines to exit
    }
    if suite.ctx != nil {
        if wg := suite.ctx.Value("wg"); wg != nil {
            wg.(*sync.WaitGroup).Wait()  // Now wait for exit
        }
    }
}

// ❌ WRONG - Wait without cancel causes deadlock
func (suite *ServiceTestSuite) TearDownTest() {
    if suite.ctx != nil {
        if wg := suite.ctx.Value("wg"); wg != nil {
            wg.(*sync.WaitGroup).Wait()  // Goroutines never exit!
        }
    }
    if suite.cancel != nil {
        suite.cancel()  // Too late
    }
}
```

---

## 7. RTK Query Lazy Hooks (Frontend)

**Problem**: Lazy hooks aren't codegen'd into `sratApi.ts`; imperative calls fail  
**Solution**: Use endpoint-specific lazy hook

```typescript
// ✅ CORRECT - Use sratApi.endpoints.<name>.useLazyQuery()
import { sratApi } from "@/store/sratApi";

export function useCommandOutput() {
  const [trigger, result] =
    sratApi.endpoints.getApiCommandOutput.useLazyQuery();

  const fetchOutput = useCallback(
    async (id: string) => {
      const response = await trigger({ id }).unwrap();
      // Use response...
    },
    [trigger],
  );

  return { fetchOutput, result };
}

// ❌ WRONG - useLazyGetApiCommandOutput doesn't exist
import { useLazyGetApiCommandOutputQuery } from "@/store/sratApi";
// TS2554: This function doesn't exist!
```

**Why**: Huma codegen creates query hooks but not lazy variants. Access lazy via endpoint object instead.

**Reference**: `frontend/src/store/sratApi.ts` (generated types), `frontend/src/App.tsx` (example usage)

---

## 8. IssueCard Ignored-State Keying (Frontend)

**Problem**: Ignored issues stored by string `problem_key`, but tests use numeric ids; cards never hide  
**Solution**: Key localStorage by problem_key, match fixture

```typescript
// ✅ CORRECT - Use problem_key string for both storage and fixture
const mockIssue: dto.Problem = {
  problem_key: "addon_config_changed",  // String key
  id: 9,
  severity: dto.ProblemSeverities.PROBLEMSEVERITYINFO,
  // ...
};

localStorage.setItem(
  "srat_ignored_issues",
  JSON.stringify(["addon_config_changed"])  // Match problem_key
);

render(<IssueCard issue={mockIssue} />);
expect(screen.queryByText(/config/i)).not.toBeInTheDocument();  // Hides correctly

// ❌ WRONG - Numeric id doesn't match storage key logic
const mockIssue: dto.Problem = {
  problem_key: "addon_config_changed",
  id: 9,
  // ...
};

localStorage.setItem(
  "srat_ignored_issues",
  JSON.stringify([9])  // Numeric id!
);

render(<IssueCard issue={mockIssue} />);
expect(screen.queryByText(/config/i)).toBeInTheDocument();  // NOT hidden; mismatch!
```

**Why**: `useIgnoredIssues` hook checks `isIssueIgnored(issue.problem_key)`, not numeric id. Storage and fixture must align on string key.

**Reference**: `frontend/src/hooks/issueHooks.ts` (storage logic), `frontend/src/components/IssueCard.tsx` (usage), `frontend/src/components/__tests__/IssueCard.test.tsx` (example test)

---

## Links to Full Documentation

| Pattern              | Full Doc                                                                                                                            |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| MSW patterns         | `.github/instructions/fontend_test.instructions.md` lines 106–155                                                                   |
| DTO type safety      | `.github/instructions/backend_test.instructions.md` lines 709–732                                                                   |
| Service architecture | `.github/copilot-instructions.md` lines 93–110                                                                                      |
| Test cleanup         | `.github/instructions/fontend_test.instructions.md` lines 12–19 + `.github/instructions/backend_test.instructions.md` lines 289–321 |
| RTK lazy hooks       | `.github/instructions/reactjs.instructions.md` lines 180–186                                                                        |
| IssueCard state      | `frontend/src/hooks/issueHooks.ts` (source of truth)                                                                                |

---

**Last Updated**: 2026-04-25  
**Patterns**: 8 high-impact  
**Use**: Copy snippets, refer to links for context, run tests to verify
