# GitHub Copilot Instructions for SRAT Project

## Test Creation Guidelines

When creating Go tests for API handlers in this project, follow these patterns based on `setting_test.go`:

### Test Suite Structure

1. **Use testify/suite**: All test files should use the `testify/suite` package for structured testing
2. **Package naming**: Test files should use the `package api_test` naming convention
3. **Suite struct naming**: Follow the pattern `{HandlerName}HandlerSuite` (e.g., `SettingsHandlerSuite`, `ShareHandlerSuite`)

### Required Imports

Include these standard imports for API handler tests:
```go
import (
    "context"
    "encoding/json"
    "net/http"
    "sync"
    "testing"

    "github.com/danielgtaylor/huma/v2/autopatch"
    "github.com/danielgtaylor/huma/v2/humatest"
    "github.com/dianlight/srat/api"
    "github.com/dianlight/srat/config"
    "github.com/dianlight/srat/converter"
    "github.com/dianlight/srat/dto"
    "github.com/dianlight/srat/service"
    "github.com/ovechkin-dm/mockio/v2/matchers"
    "github.com/ovechkin-dm/mockio/v2/mock"
    "github.com/stretchr/testify/suite"
    "go.uber.org/fx"
    "go.uber.org/fx/fxtest"
)
```

### Test Suite Fields

Each test suite should include:
- The handler being tested (e.g., `api *api.SettingsHandler`)
- Mock services used by the handler
- Dependency injection app (`app *fxtest.App`)
- Context and cancellation function for async operations
- Any required configuration or shared state

### SetupTest Method

Follow this pattern for `SetupTest`:

```go
func (suite *HandlerNameSuite) SetupTest() {
    suite.app = fxtest.New(suite.T(),
        fx.Provide(
            // Mock controller
            func() *matchers.MockController { return mock.NewMockController(suite.T()) },
            
            // Context with cancellation
            func() (context.Context, context.CancelFunc) {
                return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
            },
            
            // Handler constructor
            api.NewHandlerName,
            
            // Real services (when not mocked)
            service.NewDirtyDataService,
            
            // Mock services
            mock.Mock[service.InterfaceName],
            
            // Context state provider
            func() *dto.ContextState {
                // Initialize with test data
                return &dto.ContextState{
                    ReadOnlyMode: false,
                    Heartbeat: 1,
                    DockerInterface: "hassio",
                    DockerNet: "172.30.32.0/23",
                }
            },
            
            // Configuration provider
            func() config.Config {
                // Load test configuration
            },
        ),
        // Populate all dependencies
        fx.Populate(&suite.mockService),
        fx.Populate(&suite.handler),
        // ... other dependencies
    )
    suite.app.RequireStart()
    
    // Set up mock expectations
    mock.When(suite.mockService.Method(mock.Any[Type]())).ThenReturn(expected, nil)
}
```

### TearDownTest Method

Always include proper cleanup:

```go
func (suite *HandlerNameSuite) TearDownTest() {
    if suite.cancel != nil {
        suite.cancel()
        suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
    }
    suite.app.RequireStop()
}
```

### Test Methods

#### HTTP Test Pattern
For HTTP endpoint tests using humatest:

```go
func (suite *HandlerNameSuite) TestMethodName() {
    _, api := humatest.New(suite.T())
    suite.handler.RegisterHandlerName(api)
    
    // For PATCH operations, add autopatch
    autopatch.AutoPatch(api)
    
    // Prepare test data
    input := dto.SomeType{
        Field: "value",
    }
    
    // Make HTTP request
    resp := api.Get("/endpoint")  // or Post, Patch, etc.
    suite.Require().Equal(http.StatusOK, resp.Code)
    
    // Parse response
    var result dto.SomeType
    err := json.Unmarshal(resp.Body.Bytes(), &result)
    suite.Require().NoError(err)
    
    // Assertions
    suite.Equal(expected, result)
    
    // Verify dirty state if applicable
    suite.True(suite.dirtyService.GetDirtyDataTracker().SomeField)
}
```

#### Direct Handler Test Pattern
For testing handlers directly (like in shares_test.go):

```go
func (suite *HandlerNameSuite) TestMethodName() {
    // Prepare input
    input := dto.SomeType{Name: "test"}
    
    // Configure mock expectations
    mock.When(suite.mockService.Method(mock.Any[dto.SomeType]())).ThenReturn(expected, nil)
    
    // Prepare request input
    requestInput := &struct {
        Body dto.SomeType `required:"true"`
    }{
        Body: input,
    }
    
    // Execute
    result, err := suite.handler.Method(context.Background(), requestInput)
    
    // Assert
    suite.NoError(err)
    suite.NotNil(result)
    suite.Equal(http.StatusCreated, result.Status)
    
    // Verify mock calls
    mock.Verify(suite.mockService, matchers.Times(1)).Method()
}
```

### Error Handling Tests

Always test error scenarios:

```go
func (suite *HandlerNameSuite) TestMethodNameError() {
    // Configure mock to return error
    expectedErr := dto.ErrorSomeCondition
    mock.When(suite.mockService.Method(mock.Any[dto.SomeType]())).ThenReturn(nil, expectedErr)
    
    // Execute
    result, err := suite.handler.Method(context.Background(), input)
    
    // Assert error
    suite.Error(err)
    suite.Nil(result)
    
    // Check specific error type if needed
    statusErr, ok := err.(huma.StatusError)
    suite.True(ok)
    suite.Equal(409, statusErr.GetStatus()) // or appropriate status
}
```

### Test Runner

Always include the test runner function:

```go
func TestHandlerNameSuite(t *testing.T) {
    suite.Run(t, new(HandlerNameSuite))
}
```

### General Guidelines

1. **Mock external dependencies**: Use mockio/v2 for mocking services and repositories
2. **Test both success and error paths**: Include tests for various error conditions
3. **Verify dirty state**: Check that `dirtyService.SetDirty*()` methods are called when expected
4. **Async operations**: Note that `NotifyClient()` calls are async and may not be verifiable in tests without synchronization
5. **Use meaningful test names**: Follow the pattern `Test{Method}{Scenario}` (e.g., `TestCreateShareSuccess`, `TestCreateShareAlreadyExists`)
6. **Clean up resources**: Always implement proper teardown in `TearDownTest`
7. **Use testify assertions**: Prefer `suite.Equal`, `suite.NoError`, etc. over raw Go comparisons

### Configuration Files

Test data should be placed in `backend/test/data/` directory. Reference configuration files using relative paths like `"../../test/data/config.json"`.

### Template Files

When tests need template files, reference them from the templates directory: `"../templates/smb.gtpl"`.
