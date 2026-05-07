<!-- DOCTOC SKIP -->

# Test Setup & Teardown Patterns

This document consolidates test lifecycle patterns across SRAT (Go, TypeScript, Python) to reduce duplication and ensure consistency.

## Core Principle: Lifecycle Symmetry

Every test has a setup phase (initialization), execution phase (test logic), and teardown phase (cleanup). Proper teardown is critical—incomplete cleanup causes resource leaks, flaky tests, and CI failures.

## Go Backend Tests (testify/suite + fx)

### Setup Pattern

```go
type ServiceTestSuite struct {
    suite.Suite
    app            *fxtest.App
    serviceUnderTest service.ServiceInterface
    mockDependency service.DependencyInterface
    ctx            context.Context
    cancel         context.CancelFunc
}

func (suite *ServiceTestSuite) SetupTest() {
    suite.app = fxtest.New(suite.T(),
        fx.Provide(
            func() *matchers.MockController { return mock.NewMockController(suite.T()) },
            func() (context.Context, context.CancelFunc) {
                wg := &sync.WaitGroup{}
                ctx := context.WithValue(context.Background(), "wg", wg)
                return context.WithCancel(ctx)
            },
            service.NewServiceName,           // Real implementation
            mock.Mock[service.DependencyInterface], // Type-safe mock
            func() *dto.ContextState { return &dto.ContextState{} },
        ),
        fx.Populate(&suite.serviceUnderTest),
        fx.Populate(&suite.mockDependency),
        fx.Populate(&suite.ctx),
        fx.Populate(&suite.cancel),
    )
    suite.app.RequireStart()
}
```

**Key points:**

- Inject WaitGroup via context.WithValue (for goroutine tracking)
- Use `fx.Populate(&field)` to extract dependencies
- Call `RequireStart()` to initialize the Fx app

### Teardown Pattern

```go
func (suite *ServiceTestSuite) TearDownTest() {
    // 1. Cancel context FIRST (signals goroutines to exit)
    if suite.cancel != nil {
        suite.cancel()
    }

    // 2. Wait for goroutines to finish
    if suite.ctx != nil {
        if wg := suite.ctx.Value("wg"); wg != nil {
            wg.(*sync.WaitGroup).Wait()
        }
    }

    // 3. Stop the Fx app
    if suite.app != nil {
        suite.app.RequireStop()
    }
}
```

**Critical ordering:** Cancel context BEFORE waiting on WaitGroup. Otherwise, goroutines won't receive the cancel signal and will deadlock.

## TypeScript/JavaScript Frontend Tests (Vitest + React Testing Library)

### Setup Pattern

```typescript
import { beforeEach, afterEach, describe, test, expect } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

describe("MyComponent", () => {
  let user: ReturnType<typeof userEvent.setup>;

  beforeEach(() => {
    // Configure user event BEFORE rendering
    user = userEvent.setup();

    // Render component
    render(<MyComponent />);
  });

  afterEach(() => {
        // Shared cleanup in frontend/test/bun-setup.ts handles cleanup()
    // DO NOT add manual cleanup() calls here—it causes race conditions
  });

  test("should update when clicked", async () => {
    const button = screen.getByRole("button", { name: /click/i });
    await user.click(button);
    expect(screen.getByText(/clicked/i)).toBeInTheDocument();
  });
});
```

**Key points:**

- Initialize `userEvent.setup()` in `beforeEach` BEFORE rendering
- Shared setup in `frontend/test/bun-setup.ts` calls `cleanup()` after each test—do NOT duplicate
- Never use manual `fireEvent`; always use `userEvent`

### Setup File (Reusable)

Lifecycle hooks live in `frontend/test/bun-setup.ts` and are loaded by `frontend/test/setup.ts`:

```typescript
import { afterEach } from "vitest";
import { cleanup } from "@testing-library/react";
import { GlobalRegistrator } from "@happy-dom/global-registrator";

// Register DOM globals when the runner has not already done so
GlobalRegistrator.register();

// CRITICAL: Set act() environment AFTER registration (not before)
(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

// Shared cleanup after each test
afterEach(() => {
  cleanup();
});
```

**Important:** `IS_REACT_ACT_ENVIRONMENT` must be set AFTER `GlobalRegistrator.register()`. Setting it before has no effect.

## Python Home Assistant Tests (pytest + unittest)

### Setup Pattern

```python
import pytest
from unittest.mock import AsyncMock, patch
from homeassistant.core import HomeAssistant
from homeassistant.setup import async_setup_component


@pytest.fixture
async def hass_fixture(hass: HomeAssistant):
    """Setup Home Assistant test instance."""
    # Initialize any required services
    await async_setup_component(hass, "srat", {})
    yield hass
    # Cleanup happens automatically via pytest-homeassistant-custom-component


@pytest.mark.asyncio
async def test_something(hass_fixture):
    """Test with setup fixture."""
    # Test logic here
    pass
```

**Key points:**

- Use pytest fixtures for shared setup
- Leverage pytest-homeassistant-custom-component for HA-specific setup
- `yield` in fixtures enables automatic cleanup after test

### Cleanup Pattern

```python
@pytest.fixture
def mock_service(monkeypatch):
    """Mock external service."""
    mock = AsyncMock()
    monkeypatch.setattr("module.external_service", mock)
    yield mock
    # Monkeypatch automatically resets after test
```

## Common Anti-Patterns

### ❌ Incomplete Teardown

```go
// WRONG: No TearDownTest
func (suite *ServiceTestSuite) SetupTest() {
    suite.app = fxtest.New(...)
    suite.app.RequireStart()
    // App never stops—resource leak!
}
```

**Fix:** Always implement `TearDownTest()` with proper cleanup.

### ❌ Wrong Teardown Order (Go)

```go
// WRONG: Wait before cancel
func (suite *ServiceTestSuite) TearDownTest() {
    if wg := suite.ctx.Value("wg"); wg != nil {
        wg.(*sync.WaitGroup).Wait() // Deadlock! Goroutines won't exit
    }
    suite.cancel() // Too late
}
```

**Fix:** Cancel BEFORE waiting on WaitGroup.

### ❌ Duplicate Cleanup (TypeScript)

```typescript
// WRONG: Manual cleanup in test file
afterEach(() => {
  cleanup(); // Already called by bun-setup.ts!
  document.body.innerHTML = ""; // Also wrong
});
```

**Fix:** Trust the shared setup; don't duplicate.

### ❌ Setup & Teardown Not Paired (Python)

```python
# WRONG: Setup but no cleanup
@pytest.fixture
def mock_file(tmp_path):
    file = tmp_path / "test.txt"
    file.write_text("data")
    return file
    # Doesn't yield—no cleanup context
```

**Fix:** Use `yield` to establish cleanup point.

## Checklist: Verify Your Test Lifecycle

- [ ] **Setup initializes all dependencies** (mocks, fixtures, context)
- [ ] **Teardown cancels/stops in correct order** (context cancel BEFORE wait in Go)
- [ ] **No shared state between tests** (each test is isolated)
- [ ] **Resources explicitly released** (app.Stop(), cleanup(), monkeypatch reset)
- [ ] **No duplicate teardown** (no manual cleanup if shared setup already does it)
- [ ] **Test runs consistently** (no flakiness, no "sometimes passes/fails")

## References

- **Go details:** `.github/instructions/backend_test.instructions.md`
- **TypeScript details:** `.github/instructions/fontend_test.instructions.md`
- **Python details:** `.github/instructions/python.instructions.md`
- **Shared principles:** `docs/shared-principles.md`
