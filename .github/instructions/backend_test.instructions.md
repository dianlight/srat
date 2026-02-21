<!-- DOCTOC SKIP -->

---

description: "Backend testing standards for the SRAT Go codebase"
applyTo: "backend/src/\**/*\_test.go"

---

# Backend Testing Instructions

## Overview

All backend tests in the SRAT project follow consistent patterns using testify/suite, mockio/v2 for mocking, and uber-go/fx for dependency injection in tests. These instructions ensure uniform, maintainable, and comprehensive test coverage.

## Core Testing Stack

- **Test Framework**: `github.com/stretchr/testify/suite` - Provides suite-based test organization
- **Mocking**: `github.com/ovechkin-dm/mockio/v2/mock` and `github.com/ovechkin-dm/mockio/v2/matchers` - Type-safe mock generation
- **DI Framework**: `go.uber.org/fx` and `go.uber.org/fx/fxtest` - Dependency injection for test setup
- **HTTP Testing**: `github.com/danielgtaylor/huma/v2/humatest` - For API handler tests
- **Assertions**: `github.com/stretchr/testify/require` and `github.com/stretchr/testify/assert`

## Test Package Organization

### External Test Package (Preferred for Service/API Layers)

Use the `_test` package suffix for service and API layer tests to enforce black-box testing:

```go
package service_test  // NOT package service

import (
    "context"
    "testing"

    "github.com/dianlight/srat/service"
    "github.com/stretchr/testify/suite"
    "go.uber.org/fx"
    "go.uber.org/fx/fxtest"
)
```

**Benefits:**

- Forces testing through public interfaces only
- Prevents access to unexported fields/methods
- Better encapsulation and maintainability

### Same Package (For Internal/Utils/Converters)

Use the same package name for converter, utils, and internal tests where access to unexported functions is needed:

```go
package converter

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

## Suite-Based Testing Pattern

### Standard Test Suite Structure

All tests MUST use testify/suite for consistency:

```go
type ServiceNameTestSuite struct {
    suite.Suite
    app            *fxtest.App
    serviceUnderTest service.ServiceInterface
    mockDependency service.DependencyInterface
    ctrl           *matchers.MockController
    ctx            context.Context
    cancel         context.CancelFunc
}

func TestServiceNameTestSuite(t *testing.T) {
    suite.Run(t, new(ServiceNameTestSuite))
}
```

### SetupTest and TearDownTest

**ALWAYS** implement both SetupTest and TearDownTest for proper resource management:

```go
func (suite *ServiceNameTestSuite) SetupTest() {
    suite.app = fxtest.New(suite.T(),
        fx.Provide(
            func() *matchers.MockController { return mock.NewMockController(suite.T()) },
            func() (context.Context, context.CancelFunc) {
                return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
            },
            service.NewServiceName,  // Real implementation
            mock.Mock[service.DependencyInterface],  // Mock
            func() *dto.ContextState {
                return &dto.ContextState{
                    // Configure test context
                }
            },
        ),
        fx.Populate(&suite.serviceUnderTest),
        fx.Populate(&suite.mockDependency),
        fx.Populate(&suite.ctx),
        fx.Populate(&suite.cancel),
        fx.Populate(&suite.ctrl),
    )
    suite.app.RequireStart()
}

func (suite *ServiceNameTestSuite) TearDownTest() {
    if suite.cancel != nil {
        suite.cancel()
    }
    if suite.ctx != nil {
        if wg := suite.ctx.Value("wg"); wg != nil {
            wg.(*sync.WaitGroup).Wait()
        }
    }
    if suite.app != nil {
        suite.app.RequireStop()
    }
}
```

**Key Points:**

- Use `fxtest.New(suite.T(), ...)` - never `fxtest.New(t, ...)` inside SetupTest
- Always cancel context before waiting on WaitGroup
- Use `fx.Populate` to extract dependencies into suite fields
- Call `RequireStart()` after setup, `RequireStop()` in teardown

## Dependency Injection with Fx

### Providing Real Implementations

```go
fx.Provide(
    service.NewServiceName,          // Constructor function
    events.NewEventBus,              // Real event bus
    dbom.NewDB,                      // Real database (in-memory for tests)
)
```

### Providing Mocks

```go
fx.Provide(
    func() *matchers.MockController { return mock.NewMockController(suite.T()) },
    mock.Mock[service.DependencyInterface],  // Type-safe mock
)
```

### Providing Test Fixtures

```go
fx.Provide(
    func() *dto.ContextState {
        return &dto.ContextState{
            ReadOnlyMode:    false,
            Heartbeat:       1,
            DockerInterface: "hassio",
            DockerNet:       "172.30.32.0/23",
            DatabasePath:    "file::memory:?cache=shared&_pragma=foreign_keys(1)",
        }
    },
    func() (context.Context, context.CancelFunc) {
        wg := &sync.WaitGroup{}
        ctx := context.WithValue(context.Background(), "wg", wg)
        return context.WithCancel(ctx)
    },
)
```

### Populating Suite Fields

```go
fx.Populate(&suite.serviceUnderTest),
fx.Populate(&suite.mockDependency),
fx.Populate(&suite.ctx),
fx.Populate(&suite.cancel),
```

**IMPORTANT:** Populate extracts the dependencies from the Fx container into your suite struct fields for use in tests.

## Mocking with Mockio

Always use mockio for type-safe mocks instead of manual mock implementations. This ensures compile-time safety and reduces boilerplate.

### General Role of Mocks

- Mocks should be used for external dependencies (e.g., databases, external services) **do always** mock dependencies, never the service under test itself.
- Use mocks to simulate various scenarios (success, errors, edge cases)
- Verify interactions with dependencies (e.g., method calls, arguments)

### Creating Mocks

Mockio v2 provides type-safe mock generation:

```go
// In SetupTest, provide mock in fx container
fx.Provide(
    mock.Mock[service.DependencyInterface],
)

// Populate into suite field
fx.Populate(&suite.mockDependency),
```

### Setting Expectations

```go
// When a method is called, return specific values
mock.When(suite.mockDependency.GetData(mock.AnyContext(), mock.Exact("key"))).
    ThenReturn(expectedData, nil)

// Match any argument
mock.When(suite.mockDependency.DoSomething(mock.Any[dto.Input]())).
    ThenReturn(nil)

// Use exact matcher for specific values
mock.When(suite.mockDependency.GetByID(mock.Exact("disk-123"))).
    ThenReturn(expectedDisk, nil)
```

### Verifying Calls

```go
// Verify exact number of calls
mock.Verify(suite.mockDependency, matchers.Times(1)).GetData(mock.AnyContext(), mock.Any[string]())

// Verify never called
mock.Verify(suite.mockDependency, matchers.Times(0)).GetData(mock.AnyContext(), mock.Any[string]())
```

**Matchers:**

- `mock.AnyContext()` - matches any context.Context
- `mock.Any[T]()` - matches any value of type T
- `mock.Exact(value)` - matches exact value
- `matchers.Times(n)` - verify called exactly n times

## Table-Driven Tests

Use table-driven tests for testing multiple scenarios:

```go
func (suite *ServiceTestSuite) TestMethodName() {
    testCases := []struct {
        name           string
        input          dto.Input
        setupMock      func()
        expectedResult *dto.Result
        expectedError  bool
        errorContains  string
    }{
        {
            name:  "Successful case",
            input: dto.Input{Value: "test"},
            setupMock: func() {
                mock.When(suite.mockDep.GetData()).ThenReturn("data", nil)
            },
            expectedResult: &dto.Result{Value: "expected"},
            expectedError:  false,
        },
        {
            name:  "Error case",
            input: dto.Input{Value: "bad"},
            setupMock: func() {
                mock.When(suite.mockDep.GetData()).ThenReturn(nil, errors.New("error"))
            },
            expectedError: true,
            errorContains: "error",
        },
    }

    for _, tc := range testCases {
        suite.T().Run(tc.name, func(t *testing.T) {
            // Setup mocks if needed
            if tc.setupMock != nil {
                tc.setupMock()
            }

            // Execute
            result, err := suite.serviceUnderTest.MethodName(suite.ctx, tc.input)

            // Assert
            if tc.expectedError {
                require.Error(t, err)
                if tc.errorContains != "" {
                    assert.Contains(t, err.Error(), tc.errorContains)
                }
            } else {
                require.NoError(t, err)
                assert.Equal(t, tc.expectedResult, result)
            }
        })
    }
}
```

**Key Points:**

- Use `suite.T().Run(tc.name, ...)` for subtests within a suite
- Use `require` for critical assertions (stops test on failure)
- Use `assert` for non-critical assertions (continues test)
- Always test both success and error paths

## API Handler Testing

For API handlers, combine suite + fx + humatest:

```go
type HandlerTestSuite struct {
    suite.Suite
    app          *fxtest.App
    handler      *api.HandlerName
    mockService  service.ServiceInterface
    ctx          context.Context
    cancel       context.CancelFunc
}

func (suite *HandlerTestSuite) SetupTest() {
    suite.app = fxtest.New(suite.T(),
        fx.Provide(
            func() *matchers.MockController { return mock.NewMockController(suite.T()) },
            func() (context.Context, context.CancelFunc) {
                return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
            },
            api.NewHandlerName,
            mock.Mock[service.ServiceInterface],
            func() *dto.ContextState { return &dto.ContextState{} },
        ),
        fx.Populate(&suite.handler),
        fx.Populate(&suite.mockService),
        fx.Populate(&suite.ctx),
        fx.Populate(&suite.cancel),
    )
    suite.app.RequireStart()
}

func (suite *HandlerTestSuite) TestGetResource() {
    // Setup mock expectations
    mock.When(suite.mockService.GetResource()).ThenReturn(&dto.Resource{ID: "123"}, nil)

    // Create test API
    _, testAPI := humatest.New(suite.T())
    suite.handler.RegisterHandlers(testAPI)

    // Make request
    resp := testAPI.Get("/resource/123")

    // Assert HTTP response
    suite.Equal(http.StatusOK, resp.Code)

    // Parse and verify response body
    var result dto.Resource
    err := json.Unmarshal(resp.Body.Bytes(), &result)
    suite.Require().NoError(err)
    suite.Equal("123", result.ID)
}
```

**Humatest Patterns:**

- `_, testAPI := humatest.New(suite.T())` - creates test API instance
- Register handlers before making requests
- Use `testAPI.Get()`, `testAPI.Post()`, etc. for HTTP methods
- Verify `resp.Code` for status codes
- Unmarshal `resp.Body.Bytes()` for response validation

## Context and Cancellation

Always use context for lifecycle management:

```go
// In SetupTest
func() (context.Context, context.CancelFunc) {
    wg := &sync.WaitGroup{}
    ctx := context.WithValue(context.Background(), "wg", wg)
    return context.WithCancel(ctx)
}

// In TearDownTest - ALWAYS in this order
if suite.cancel != nil {
    suite.cancel()  // Cancel context first
}
if suite.ctx != nil {
    if wg := suite.ctx.Value("wg"); wg != nil {
        wg.(*sync.WaitGroup).Wait()  // Wait for goroutines to finish
    }
}
```

**CRITICAL:** Cancel context before waiting on WaitGroup to allow goroutines to exit.

## Assertions

### Require vs Assert

- **require**: Stops test execution on failure (use for critical preconditions)
- **assert**: Continues test execution on failure (use for multiple checks)

```go
// Require - test stops if this fails
suite.Require().NoError(err)
suite.Require().NotNil(result)

// Assert - test continues if this fails
suite.Equal(expected, actual)
suite.True(condition)
suite.Contains(slice, element)
suite.Len(collection, expectedLen)
```

### Common Assertions

```go
// Equality
suite.Equal(expected, actual)
suite.NotEqual(notExpected, actual)

// Nil checks
suite.Nil(value)
suite.NotNil(value)

// Boolean
suite.True(condition)
suite.False(condition)

// Errors
suite.NoError(err)
suite.Error(err)
suite.ErrorIs(err, targetErr)
suite.ErrorContains(err, "substring")

// Collections
suite.Empty(collection)
suite.NotEmpty(collection)
suite.Len(collection, expectedLen)
suite.Contains(slice, element)
```

## Error Testing

When testing errors, use tozd/go/errors conventions:

```go
// Test error type
suite.Error(err)
suite.True(errors.Is(err, dto.ErrorNotFound))

// Test error details
details := errors.Details(err)
suite.NotNil(details)
suite.Contains(details["Message"], "expected substring")
```

## Test Data Management

### In-Memory Database

For tests requiring database:

```go
fx.Provide(
    func() *dto.ContextState {
        return &dto.ContextState{
            DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)",
        }
    },
    dbom.NewDB,
)
```

### Mock Data Files

Load test fixtures from files:

```go
data, err := os.ReadFile("../../test/data/mount_info.txt")
suite.Require().NoError(err)
// Use data in test
```

### Temporary Directories

Use `suite.T().TempDir()` for temporary file operations:

```go
tempDir := suite.T().TempDir()
testFile := filepath.Join(tempDir, "test.txt")
suite.Require().NoError(os.WriteFile(testFile, []byte("test"), 0o644))
// TempDir is automatically cleaned up after test
```

## Testing Best Practices

### DO

- ✅ Use suite.Suite for all service/API layer tests
- ✅ Use fx/fxtest for dependency injection
- ✅ Use mockio for type-safe mocking
- ✅ Always implement SetupTest and TearDownTest
- ✅ Test both success and error paths
- ✅ Use table-driven tests for multiple scenarios
- ✅ Use descriptive test names: `TestMethodName_Scenario`
- ✅ Use subtests with `suite.T().Run()` in table-driven tests
- ✅ Verify mock calls with `mock.Verify()`
- ✅ Cancel context before waiting on WaitGroup
- ✅ Use external test package (`_test` suffix) for service/API layers

### DON'T

- ❌ Don't use naked `testing.T` without suite for service tests
- ❌ Don't manually create mocks when mockio can generate them
- ❌ Don't skip TearDownTest (causes resource leaks)
- ❌ Don't ignore errors in test setup
- ❌ Don't use global state or shared variables between tests
- ❌ Don't hardcode paths; use `filepath.Join()` or `suite.T().TempDir()`
- ❌ Don't test implementation details; test behavior
- ❌ Don't wait on WaitGroup before cancelling context (causes deadlock)

## Example: Complete Service Test

```go
package service_test

import (
    "context"
    "sync"
    "testing"

    "github.com/dianlight/srat/dto"
    "github.com/dianlight/srat/service"
    "github.com/ovechkin-dm/mockio/v2/matchers"
    "github.com/ovechkin-dm/mockio/v2/mock"
    "github.com/stretchr/testify/suite"
    "go.uber.org/fx"
    "go.uber.org/fx/fxtest"
)

type ExampleServiceTestSuite struct {
    suite.Suite
    app                *fxtest.App
    exampleService     service.ExampleServiceInterface
    mockDependency     service.DependencyInterface
    ctrl               *matchers.MockController
    ctx                context.Context
    cancel             context.CancelFunc
}

func TestExampleServiceTestSuite(t *testing.T) {
    suite.Run(t, new(ExampleServiceTestSuite))
}

func (suite *ExampleServiceTestSuite) SetupTest() {
    suite.app = fxtest.New(suite.T(),
        fx.Provide(
            func() *matchers.MockController { return mock.NewMockController(suite.T()) },
            func() (context.Context, context.CancelFunc) {
                return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
            },
            service.NewExampleService,
            mock.Mock[service.DependencyInterface],
            func() *dto.ContextState {
                return &dto.ContextState{}
            },
        ),
        fx.Populate(&suite.exampleService),
        fx.Populate(&suite.mockDependency),
        fx.Populate(&suite.ctx),
        fx.Populate(&suite.cancel),
        fx.Populate(&suite.ctrl),
    )
    suite.app.RequireStart()
}

func (suite *ExampleServiceTestSuite) TearDownTest() {
    if suite.cancel != nil {
        suite.cancel()
    }
    if suite.ctx != nil {
        if wg := suite.ctx.Value("wg"); wg != nil {
            wg.(*sync.WaitGroup).Wait()
        }
    }
    if suite.app != nil {
        suite.app.RequireStop()
    }
}

func (suite *ExampleServiceTestSuite) TestProcessData_Success() {
    // Arrange
    input := dto.Input{Value: "test"}
    expectedOutput := &dto.Output{Result: "processed"}

    mock.When(suite.mockDependency.FetchData(mock.AnyContext(), mock.Exact("test"))).
        ThenReturn("data", nil)

    // Act
    result, err := suite.exampleService.ProcessData(suite.ctx, input)

    // Assert
    suite.Require().NoError(err)
    suite.Equal(expectedOutput, result)
    mock.Verify(suite.mockDependency, matchers.Times(1)).FetchData(mock.AnyContext(), mock.Exact("test"))
}

func (suite *ExampleServiceTestSuite) TestProcessData_Error() {
    // Arrange
    input := dto.Input{Value: "bad"}

    mock.When(suite.mockDependency.FetchData(mock.AnyContext(), mock.Any[string]())).
        ThenReturn(nil, errors.New("fetch failed"))

    // Act
    result, err := suite.exampleService.ProcessData(suite.ctx, input)

    // Assert
    suite.Error(err)
    suite.Nil(result)
    suite.Contains(err.Error(), "fetch failed")
}
```

## Example: Simple Converter/Utils Test

For simpler tests (converters, utils) without dependencies:

```go
package converter

import (
    "testing"

    "github.com/dianlight/srat/dto"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestConvertToDTO(t *testing.T) {
    // Simple test without suite for utility functions
    input := "test-value"

    result := ConvertToDTO(input)

    require.NotNil(t, result)
    assert.Equal(t, "test-value", result.Value)
}

func TestConvertToDTO_TableDriven(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty", "", ""},
        {"simple", "test", "test"},
        {"with spaces", "hello world", "hello world"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ConvertToDTO(tt.input)
            assert.Equal(t, tt.expected, result.Value)
        })
    }
}
```

## Common Patterns Summary

| Test Type     | Package  | Suite  | Fx/Fxtest | Mockio   | Humatest |
| ------------- | -------- | ------ | --------- | -------- | -------- |
| Service Layer | `_test`  | ✅ Yes | ✅ Yes    | ✅ Yes   | ❌ No    |
| API Handlers  | `_test`  | ✅ Yes | ✅ Yes    | ✅ Yes   | ✅ Yes   |
| Repository    | `_test`  | ✅ Yes | ⚠️ Maybe  | ⚠️ Maybe | ❌ No    |
| Converters    | Same pkg | ❌ No  | ❌ No     | ❌ No    | ❌ No    |
| Utils/Helpers | Same pkg | ❌ No  | ❌ No     | ❌ No    | ❌ No    |

**Legend:**

- ✅ Yes: Always use
- ❌ No: Never use
- ⚠️ Maybe: Use if needed

## Migration Guide

If you encounter tests not following these patterns:

1. **Add Suite Structure**: Wrap test in suite.Suite struct
2. **Add Fx DI**: Use fxtest.New for dependency injection
3. **Convert Mocks**: Replace manual mocks with mockio
4. **Add Setup/Teardown**: Implement SetupTest and TearDownTest
5. **Use External Package**: Change to `package xxx_test` if testing services/API

## References

- [Testify Suite Documentation](https://pkg.go.dev/github.com/stretchr/testify/suite)
- [Mockio v2 Documentation](https://github.com/ovechkin-dm/mockio)
- [Fx Testing Documentation](https://pkg.go.dev/go.uber.org/fx/fxtest)
- [Huma Testing Documentation](https://huma.rocks/tutorial/writing-tests/?h=testing)
