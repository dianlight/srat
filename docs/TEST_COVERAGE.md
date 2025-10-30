<!-- DOCTOC SKIP -->

# Test Coverage

This document provides an overview of the test coverage for the SRAT project, including both backend (Go) and frontend (TypeScript/React) components.

## Current Coverage

| Component             | Coverage | Status    |
| --------------------- | -------- | --------- |
| Backend (Go)          | 34.3%    | 🟠 Orange |
| Frontend (TypeScript) | 70.30%   | 🟢 Green  |
| Global (Weighted)     | 48.7%    | 🟡 Yellow |

*Last updated: 2025-10-30*

## Backend Package-Level Coverage

The following table shows coverage for all backend packages. The target is **60% minimum coverage** for each package.

| Package                    | Coverage | Status        | Priority to Improve       |
| -------------------------- | -------- | ------------- | ------------------------- |
| `api`                      | 48.1%    | 🟠 Needs Work | High - Close to target    |
| `cmd/srat-cli`             | 5.7%     | 🔴 Critical   | Low - CLI testing complex |
| `cmd/srat-openapi`         | 17.9%    | 🔴 Critical   | Low - Code gen utility    |
| `cmd/srat-server`          | 5.4%     | 🔴 Critical   | Low - Main entry point    |
| `config`                   | 71.1%    | ✅ Good       | None                      |
| `converter`                | 27.2%    | 🟠 Needs Work | Medium                    |
| `dbom`                     | 19.0%    | 🔴 Critical   | High                      |
| `dbom/migrations`          | 63.8%    | ✅ Good       | None                      |
| `dto`                      | 19.1%    | 🔴 Critical   | High                      |
| `homeassistant/addons`     | 13.2%    | 🔴 Critical   | Medium                    |
| `homeassistant/core`       | 25.8%    | 🟠 Needs Work | Medium                    |
| `homeassistant/core_api`   | 45.0%    | 🟠 Needs Work | Medium                    |
| `homeassistant/hardware`   | 16.3%    | 🔴 Critical   | Medium                    |
| `homeassistant/host`       | 27.3%    | 🟠 Needs Work | Medium                    |
| `homeassistant/ingress`    | 20.8%    | 🟠 Needs Work | Medium                    |
| `homeassistant/mount`      | 36.8%    | 🟠 Needs Work | Medium                    |
| `homeassistant/resolution` | 13.7%    | 🔴 Critical   | Medium                    |
| `homeassistant/root`       | 19.6%    | 🔴 Critical   | Medium                    |
| `homeassistant/websocket`  | 60.9%    | ✅ Good       | None                      |
| `internal`                 | 88.6%    | ✅ Excellent  | None                      |
| `internal/appsetup`        | 80.0%    | ✅ Excellent  | None                      |
| `internal/osutil`          | 77.4%    | ✅ Excellent  | None                      |
| `repository`               | 71.0%    | ✅ Good       | None                      |
| `server`                   | 33.8%    | 🟠 Needs Work | High                      |
| `service`                  | 33.6%    | 🟠 Needs Work | High                      |
| `tempio`                   | 85.7%    | ✅ Excellent  | None                      |
| `tlog`                     | 83.9%    | ✅ Excellent  | None                      |
| `unixsamba`                | 75.1%    | ✅ Excellent  | None                      |

**Summary:**

- ✅ **10 packages** already meet or exceed 60% threshold
- 🟠 **10 packages** need improvement (20-59% coverage)
- 🔴 **9 packages** critical (below 20% coverage)

## Coverage Thresholds

The project uses the following coverage thresholds for badge colors:

- **Bright Green** (>=80%): Excellent coverage
- **Green** (>=60%): Good coverage
- **Yellow** (>=40%): Acceptable coverage
- **Orange** (>=20%): Needs improvement
- **Red** (<20%): Critical - requires immediate attention

## Backend Testing

### Framework & Tools

- **Test Framework**: `testify/suite` with `mockio/v2` for mocks
- **HTTP Testing**: `humatest` for API endpoint validation
- **Coverage Tool**: Go's built-in `cover` tool
- **Minimum Coverage**: 5% (enforced in CI)

### Test Structure

Backend tests follow these patterns:

- Tests are in `{package}_test` packages
- Suite-based tests with `{HandlerName}HandlerSuite` structs
- Dependency injection using `fxtest.New()` and `fx.Populate()`
- Mock pattern: `mock.When(...).ThenReturn(...)` and `mock.Verify(...)`

### Running Backend Tests

```bash
cd backend
make test
```

### Coverage Details

The backend test coverage is calculated as the total percentage of statements covered across all packages. Key areas:

- **API Handlers** (`src/api/*`): Test HTTP endpoints with humatest
- **Services** (`src/service/*`): Business logic with mocked dependencies
- **Repositories** (`src/repository/*`): Data access layer testing
- **Converters** (`src/converter/*`): Auto-generated, typically high coverage
- **Utilities** (`src/tlog`, `src/internal/*`): Helper functions

## Frontend Testing

### Framework & Tools

- **Test Framework**: `bun:test` with `happy-dom` for DOM simulation
- **Component Testing**: `@testing-library/react` for component testing
- **User Interactions**: `@testing-library/user-event` for simulating user actions (REQUIRED)
- **Assertions**: `@testing-library/jest-dom` for DOM assertions
- **Coverage Tool**: Bun's built-in coverage tool
- **Minimum Coverage**: 80% functions coverage

**IMPORTANT**: All user interactions MUST use `@testing-library/user-event`. The deprecated `fireEvent` API is strictly prohibited in all new and modified tests.

### Test Structure

Frontend tests follow these mandatory patterns:

- Tests are in `__tests__` directories alongside components
- Test files use `.test.tsx` extension
- Dynamic imports for React components to avoid module loading issues
- Redux store integration using `createTestStore()` helper

### Running Frontend Tests

```bash
cd frontend
bun test --coverage
```

### Coverage Details

The frontend test coverage includes:

- **Components** (`src/components/*`): Reusable UI components
- **Pages** (`src/pages/*`): Route-based page components
- **Store** (`src/store/*`): Redux slices and RTK Query (excluding auto-generated API)
- **Hooks** (`src/hooks/*`): Custom React hooks
- **Utilities** (`src/utils/*`): Helper functions

## Global Coverage

The global coverage is calculated as a **weighted average**:

- **60% Backend** weight
- **40% Frontend** weight

This weighting reflects the relative importance and complexity of backend business logic versus frontend presentation logic.

**Formula**: `Global = (Backend × 0.6) + (Frontend × 0.4)`

## Coverage History

The following graphs show the evolution of test coverage over time.

### Backend Coverage Over Time

```mermaid
%%{init: {'theme':'dark', 'themeVariables': { 'primaryColor':'#222','primaryTextColor':'#fff','primaryBorderColor':'#388E3C','lineColor':'#2196F3','secondaryColor':'#FFC107','tertiaryColor':'#fff','background':'#181818'}}}%%
xychart-beta
  title "Backend Test Coverage Over Time"
  x-axis [2025-10-06, 2025-10-07, 2025-10-11, 2025-10-28, 2025-10-29, 2025-10-30]
  y-axis "Coverage %" 0 --> 100
  line [39.9, , , , , 34.3]
```

### Frontend Coverage Over Time

```mermaid
%%{init: {'theme':'dark', 'themeVariables': { 'primaryColor':'#222','primaryTextColor':'#fff','primaryBorderColor':'#388E3C','lineColor':'#2196F3','secondaryColor':'#FFC107','tertiaryColor':'#fff','background':'#181818'}}}%%
xychart-beta
  title "Frontend Test Coverage Over Time"
  x-axis [2025-10-06, 2025-10-07, 2025-10-11, 2025-10-28, 2025-10-29, 2025-10-30]
  y-axis "Coverage %" 0 --> 100
  line [73.11, , , , , 70.30]
```

### Global Coverage Over Time

```mermaid
%%{init: {'theme':'dark', 'themeVariables': { 'primaryColor':'#222','primaryTextColor':'#fff','primaryBorderColor':'#388E3C','lineColor':'#2196F3','secondaryColor':'#FFC107','tertiaryColor':'#fff','background':'#181818'}}}%%
xychart-beta
  title "Global Test Coverage Over Time"
  x-axis [2025-10-06, 2025-10-07, 2025-10-11, 2025-10-28, 2025-10-29, 2025-10-30]
  y-axis "Coverage %" 0 --> 100
  line [53.2, , , , , 48.7]
```

## Coverage Improvement Goals

### Short-term Goals (Next Release)

- **Backend**: Increase to 40% (+5.7%)
- **Frontend**: Maintain above 70%
- **Global**: Reach 50% (+1.3%)

### Long-term Goals (6 months)

- **Backend**: Reach 60% (+25.7%)
- **Frontend**: Reach 80% (+9.7%)
- **Global**: Reach 65% (+16.3%)

## Package-Specific Improvement Strategies

### High Priority Packages (Close to 60%)

#### `api` Package (48.1% → 60% target)

**Gap**: 12% coverage needed  
**Files needing tests**:

- `issue.go` - CRUD operations for issues (no tests currently)
- `telemetry.go` - Telemetry modes and internet connection checks (no tests)
- `upgrade.go` - System upgrade operations (no tests)
- `volumes.go` - Volume mount/unmount operations (no tests)
- `sse.go` - Server-Sent Events implementation (no tests)
- `ws.go` - WebSocket handlers (no tests)

**Strategy**: Add test suites following the pattern in `shares_test.go`:

1. Use `fxtest.New()` for dependency injection
2. Mock services with `mock.Mock[ServiceInterface]`
3. Test both success and error paths for each endpoint
4. Verify `SetDirty*()` calls for state-changing operations

#### `server` Package (33.8% → 60% target)

**Gap**: 26% coverage needed  
**Strategy**:

- Test server initialization with various configurations
- Test middleware chains (CORS, logging, etc.)
- Test WebSocket connection lifecycle
- Test SSE connection management
- Test error handling and recovery

#### `service` Package (33.6% → 60% target)

**Gap**: 26% coverage needed  
**Files with low coverage**:

- Filesystem service methods
- Share service edge cases
- User management error paths
- System service operations

**Strategy**:

- Add tests for untested service methods
- Test error conditions and edge cases
- Test service interactions and state management
- Use repository mocks to isolate service logic

### Medium Priority Packages

#### `converter` Package (27.2% → 60% target)

**Gap**: 33% coverage needed  
**Strategy**:

- Test all converter functions with various input types
- Test null/empty value handling
- Test type conversion edge cases
- Converters are auto-generated by goverter but still need edge case tests

#### `dbom` Package (19.0% → 60% target)

**Gap**: 41% coverage needed  
**Strategy**:

- Test model methods and validations
- Test model relationships and foreign keys
- Test custom GORM callbacks (BeforeCreate, AfterFind, etc.)
- Test JSON marshaling/unmarshaling for complex types

#### `dto` Package (19.1% → 60% target)

**Gap**: 41% coverage needed  
**Strategy**:

- Test DTO validation rules
- Test error code definitions and usage
- Test serialization/deserialization
- Test DTO helper methods and constructors

#### `homeassistant/*` Packages (various → 60% target)

Multiple sub-packages need improvement:

- `addons` (13.2%) - Test addon discovery and management
- `core` (25.8%) - Test Home Assistant core API interactions
- `core_api` (45.0%) - Close to target, add edge case tests
- `hardware` (16.3%) - Test hardware detection and enumeration
- `host` (27.3%) - Test host system interactions
- `ingress` (20.8%) - Test ingress configuration
- `mount` (36.8%) - Test mount point operations
- `resolution` (13.7%) - Test resolution checks
- `root` (19.6%) - Test root API client initialization

**Common Strategy for homeassistant packages**:

- Mock HTTP responses from Home Assistant supervisor
- Test error handling for network failures
- Test response parsing and data transformation
- Test authentication and authorization flows

### Low Priority Packages (defer improvement)

#### `cmd/*` Packages (5-18% coverage)

These are command-line entry points and are difficult to test comprehensively:

- `cmd/srat-cli` (5.7%) - CLI flag parsing and command execution
- `cmd/srat-server` (5.4%) - Main server entry point
- `cmd/srat-openapi` (17.9%) - OpenAPI document generation

**Strategy**:

- Focus on testing core business logic in service/repository layers instead
- Add integration tests for critical CLI commands
- Test flag validation and error messages
- Current low coverage is acceptable for these packages

## Test Implementation Guidelines

### API Handler Test Pattern

All API handler tests should follow this pattern (example from `shares_test.go`):

```go
package api_test

import (
    "context"
    "net/http"
    "sync"
    "testing"

    "github.com/danielgtaylor/huma/v2/humatest"
    "github.com/dianlight/srat/api"
    "github.com/dianlight/srat/dto"
    "github.com/dianlight/srat/service"
    "github.com/ovechkin-dm/mockio/v2/matchers"
    "github.com/ovechkin-dm/mockio/v2/mock"
    "github.com/stretchr/testify/suite"
    "go.uber.org/fx"
    "go.uber.org/fx/fxtest"
)

type HandlerSuite struct {
    suite.Suite
    app            *fxtest.App
    handler        *api.YourHandler
    mockService    service.YourServiceInterface
    ctx            context.Context
    cancel         context.CancelFunc
}

func (suite *HandlerSuite) SetupTest() {
    suite.app = fxtest.New(suite.T(),
        fx.Provide(
            func() *matchers.MockController { return mock.NewMockController(suite.T()) },
            func() (context.Context, context.CancelFunc) {
                return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
            },
            api.NewYourHandler,
            mock.Mock[service.YourServiceInterface],
            func() *dto.ContextState { return &dto.ContextState{ReadOnlyMode: false} },
        ),
        fx.Populate(&suite.handler),
        fx.Populate(&suite.mockService),
        fx.Populate(&suite.ctx),
        fx.Populate(&suite.cancel),
    )
    suite.app.RequireStart()
}

func (suite *HandlerSuite) TearDownTest() {
    if suite.cancel != nil {
        suite.cancel()
    }
    if suite.app != nil {
        suite.app.RequireStop()
    }
}

func (suite *HandlerSuite) TestEndpointSuccess() {
    // Arrange
    mock.When(suite.mockService.MethodCall(matchers.Any[ArgType]())).ThenReturn(expectedResult, nil)
    _, api := humatest.New(suite.T())
    suite.handler.RegisterHandlers(api)

    // Act
    resp := api.Get("/endpoint")

    // Assert
    suite.Equal(http.StatusOK, resp.Code)
    mock.Verify(suite.mockService, matchers.Times(1)).MethodCall(matchers.Any[ArgType]())
}
```

### Service Test Pattern

Service tests should mock repositories and test business logic:

```go
func (suite *ServiceSuite) SetupTest() {
    suite.app = fxtest.New(suite.T(),
        fx.Provide(
            func() *matchers.MockController { return mock.NewMockController(suite.T()) },
            service.NewYourService,
            mock.Mock[repository.YourRepositoryInterface],
        ),
        fx.Populate(&suite.service),
        fx.Populate(&suite.mockRepo),
    )
    suite.app.RequireStart()
}
```

### Key Testing Principles

1. **Use FX for Dependency Injection**: Always use `fxtest.New()` to build the dependency graph
2. **Mock External Dependencies**: Use `mock.Mock[Interface]` for all external dependencies
3. **Test Both Paths**: Always test success and error scenarios
4. **Verify State Changes**: Check `SetDirty*()` calls for operations that modify data
5. **Use Matchers**: Use `matchers.Any`, `matchers.Times`, etc. for flexible assertions
6. **Clean Up**: Always implement `TearDownTest()` to properly shut down FX app and cancel contexts

## Best Practices

### Backend Testing Best Practices

1. Always test both success and error paths
2. Use mocks for external dependencies (database, HTTP clients, etc.)
3. Verify state changes with `dirtyService.SetDirty*()` calls
4. Test with realistic data from `backend/test/data/`
5. Ensure tests are deterministic and can run in parallel

### Frontend Testing Best Practices

1. **Use `@testing-library/user-event` for ALL interactions** - NEVER use `fireEvent`
   - Import: `const userEvent = (await import("@testing-library/user-event")).default;`
   - Setup: `const user = userEvent.setup();`
   - Always await: `await user.click(element)`, `await user.type(input, "text")`
2. Use dynamic imports for React components
3. Clear localStorage before each test with `beforeEach()`
4. Use `screen.findByText()` for async rendering, not `getByText()`
5. Always use `React.createElement()` syntax in test files
6. Test user interactions, not implementation details
7. Wrap stateful UI transitions in `act()` when needed for happy-dom compatibility

## Updating Coverage Data

Coverage data is automatically updated by running:

```bash
./scripts/update-coverage-badges.sh
```

This script:

1. Runs backend tests and extracts total coverage
2. Runs frontend tests and extracts coverage
3. Calculates global weighted coverage
4. Updates README.md badges
5. Updates this document with new data points and graphs

## Resources

- [Backend Testing Patterns](../backend/README.md#testing)
- [Frontend Testing Setup](../frontend/README.md#testing)
- [Copilot Instructions](../.github/copilot-instructions.md)
- [Pre-commit Hooks](../.pre-commit-config.yaml)
