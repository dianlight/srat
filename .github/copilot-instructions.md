# SRAT Project Development Guidelines

This document provides a comprehensive set of rules, conventions, and guidelines for developing the SRAT project. It consolidates information on backend (Go) and frontend (React) development, testing, documentation, and project workflow.

## 1. Git & Project Workflow

### Git Hooks and pre-commit Policy

Always manage git hooks via `pre-commit`. Do not place scripts in `.git/hooks` or configure `core.hooksPath`.

1.  **Modify Configuration**: To add or change hooks, edit the repository's `.pre-commit-config.yaml`.
2.  **Hook Type**: Prefer `local` hooks with `language: system` that delegate to existing `Makefile` targets (e.g., `make -C backend gosec`).
3.  **Configuration**: Ensure `stages` are set appropriately (e.g., `commit`, `push`) and `pass_filenames` is `false` for repo-wide checks.
4.  **Installation**: Install hooks locally with:
    *   `pre-commit install` (for default hook types)
    *   `pre-commit install --hook-type pre-push` (for push-stage hooks)

*Note: The project enforces a `gosec` scan on commit for backend Go changes and a quick Go build+test on push.*

### CHANGELOG.md Rules

Update `CHANGELOG.md` for any significant changes.

-   **When to Update**: Update the changelog **after** validating that the changes work correctly (i.e., tests and builds pass).
-   **What to Document**:
    -   API changes (new endpoints, schema modifications)
    -   Breaking changes requiring user action
    -   New user-facing features
    -   Significant bug fixes
    -   Security updates
    -   Build, deployment, or database schema changes
-   **Format**: Follow the established emoji-based format.

    ```markdown
    ## 2025.08.* [ 🚧 Unreleased ]

    ### ✨ Features

    - New share management API endpoints [#123](https://github.com/dianlight/srat/issues/123)

    ### 🐛 Bug Fixes

    - Fix share enable/disable functionality not working as expected

    ### 🏗 Chore

    - Updated Huma v2 framework integration
    ```
-   **Required Information**: Include a clear description, issue/PR references (`[#123](...)`), and migration steps for any breaking changes.

## 2. Backend Development (Go)

### Package Organization

1.  **Package Naming**: Use descriptive, lowercase package names without underscores.
2.  **Project Structure**: Adhere to the established structure:
    -   `api/`: HTTP handlers and API endpoints
    -   `service/`: Business logic and service layer
    -   `repository/`: Data access layer
    -   `dto/`: Data Transfer Objects
    -   `dbom/`: Database Object Models
    -   `converter/`: Object conversion utilities
    -   `config/`: Configuration management
    -   `internal/`: Internal utilities and app setup
3.  **Import Organization**: Group imports in this order:
    1.  Standard library
    2.  External libraries
    3.  Internal project imports (`github.com/dianlight/srat/...`)

### Code Quality & Naming Conventions

-   **Receiver Name**: Use `self` as the receiver name for methods.
-   **Interface Naming**: End interface names with `Interface` (e.g., `SomeServiceInterface`).
-   **Constructor Naming**: Use the `NewTypeName` pattern.
-   **Error Handling**: Always handle errors explicitly; do not ignore them.
-   **Thread Safety**: Use mutexes for shared state in repositories.
-   **Formatting**: Use `gofmt` and follow standard Go conventions.
-   **Import Aliases**: Use consistent aliases (e.g., `errors` for `gitlab.com/tozd/go/errors`).

### API Layer (Handlers)

-   **Constructor**: Use a `NewSomethingHandler` function to instantiate handlers.
-   **Registration**: Register HTTP routes in a `RegisterSomethingHandler(api huma.API)` method.
-   **Method Signatures**: Follow standard patterns for handler methods, clearly defining inputs, outputs, and path parameters.
-   **Request Body**: Define request bodies inside an anonymous `input` struct.
-   **Response Body**: Use anonymous structs for responses (e.g., `*struct{ Body dto.Something }`).
-   **Operation Tags**: Use consistent tags for API grouping (e.g., `huma.OperationTags("system")`).

### Service Layer

-   **Interfaces**: Define an interface for every service.
-   **Implementation**: Use parameter structs with `fx.In` for dependency injection.

### Repository Layer

-   **Interfaces**: Define interfaces with GORM operations.
-   **Implementation**: Use GORM with a `sync.RWMutex` to ensure thread safety.

### Dependency Injection (FX)

-   **Service Registration**: Register all services, repositories, and handlers as FX providers.
-   **Parameter Structs**: Use `fx.In` structs for dependency injection in constructors.

### Error Handling

-   **Error Types**: Define domain-specific errors in `dto/error_code.go` (e.g., `ErrorSomethingNotFound`).
-   **Error Wrapping**: Use `gitlab.com/tozd/go/errors` for wrapping errors to preserve context.
-   **HTTP Mapping**: Map domain errors to specific HTTP status codes in the API layer (e.g., `huma.Error404NotFound`).

### Logging and Observability

-   **Structured Logging**: Use `slog` for structured, key-value based logging.
-   **Error Context**: Provide meaningful context in error messages and logs.

### Configuration and Context

-   **Context State**: Use `dto.ContextState` to manage application-wide configuration and state.
-   **Template Handling**: Load templates from the `/templates` directory.

### Async Operations and Broadcasting

-   **Dirty State**: After modifications, mark data as dirty using `dirtyService.SetDirtySomething()`.
-   **Notifications**: Use goroutines for asynchronous operations like client notifications.
-   **WaitGroup Context**: Use `context.WithValue(context.Background(), "wg", &sync.WaitGroup{})` to manage the lifecycle of async operations in tests.

### Data Transfer (DTOs) & Conversion

-   **DTOs**: Define DTOs with JSON tags and validation rules.
-   **Converters**: Use `goverter` for automatic conversions between `dto` and `dbom` objects.

## 3. Frontend Development (React)

### Component Organization

-   **Generic Components**: `src/components/` is **only** for generic, reusable components that can be used across multiple pages.
-   **Page-Specific Components**: All components specific to a single page **must** be located within that page's directory.
    -   **Example**: For a "Dashboard" page, the structure should be:
        -   `src/pages/dashboard/Dashboard.tsx` (The page itself)
        -   `src/pages/dashboard/DashboardWidget.tsx` (A component used only on the dashboard)
        -   `src/pages/dashboard/ChartPanel.tsx` (Another component used only on the dashboard)

## 4. Testing (Go)

Go tests for API handlers must follow these patterns, based on `testify/suite`.

### General Guidelines

1.  **Mock Dependencies**: Use `mockio/v2` for mocking services and repositories.
2.  **Test Paths**: Test both success and error paths.
3.  **Verify State**: Check that `dirtyService.SetDirty*()` methods are called when data is modified.
4.  **Meaningful Names**: Use the pattern `Test{Method}{Scenario}` (e.g., `TestCreateShareSuccess`).
5.  **Use Assertions**: Prefer `testify/assert` or `testify/require` (e.g., `suite.Equal(...)`).

### Test Suite Structure

-   **Package**: `package api_test`
-   **Suite Struct**: Name it `{HandlerName}HandlerSuite`. It must embed `suite.Suite` and contain fields for the handler, mock services, `fxtest.App`, and a `context`.

### SetupTest / TearDownTest Methods

-   **`SetupTest`**: Use `fxtest.New` to build the dependency graph. Provide all mocks, real services, configuration, and context. Use `fx.Populate` to inject dependencies into the suite fields. Set up mock expectations using `mock.When(...)`.
-   **`TearDownTest`**: Clean up resources. Cancel the context and wait for any `WaitGroup` to finish, then call `suite.app.RequireStop()`.

### HTTP Test Pattern (`humatest`)

1.  **Initialize**: `_, api := humatest.New(suite.T())`
2.  **Register Handler**: `suite.handler.RegisterSomething(api)`
3.  **Enable AutoPatch**: If testing a `PATCH` endpoint, call `autopatch.AutoPatch(api)`.
4.  **Make Request**: `resp := api.Get("/endpoint")`
5.  **Assert Status**: `suite.Require().Equal(http.StatusOK, resp.Code)`
6.  **Assert Body**: Unmarshal the response and assert its contents.

### Direct Handler Test Pattern

1.  **Prepare Input**: Create the required input DTO.
2.  **Configure Mock**: `mock.When(suite.mockService.Method(...)).ThenReturn(...)`
3.  **Execute**: Call the handler method directly: `result, err := suite.handler.Method(ctx, input)`
4.  **Assert**: Check the error and the result.
5.  **Verify Mock**: `mock.Verify(suite.mockService, matchers.Times(1)).Method()`

### Error Handling Tests

-   Configure the mock to return a specific error.
-   Execute the handler method.
-   Assert that an error is returned (`suite.Error(err)`).
-   If applicable, check for a specific `huma.StatusError` and status code.

### Test Runner

-   Always include the test runner function:
    ```go
    func TestHandlerNameSuite(t *testing.T) {
        suite.Run(t, new(HandlerNameSuite))
    }
    ```

### Test Data

-   Place test configuration data in `backend/test/data/`.
-   Reference files using relative paths (e.g., `"../../test/data/config.json"`).

## 5. Documentation

### General Markdown Rules

-   **Update `CHANGELOG.md`** for all significant changes.
-   Use proper heading hierarchy and consistent formatting.
-   Include fenced code blocks with language identifiers.
-   Keep all documentation, examples, and links current and valid.

### Package Documentation and Examples (Go)

When adding or changing features, update the corresponding documentation.

1.  **READMEs**: Update package `README.md` files with new features, API documentation, usage examples, and best practices.
2.  **API Reference (GoDoc)**: Ensure all public functions, types, and constants are fully documented. Include examples for complex APIs.
3.  **Examples**: Update or create runnable example programs that demonstrate new features, error handling, and best practices.

### API Documentation (OpenAPI)

-   Update OpenAPI specs when API endpoints change.
-   Include clear request/response examples.
-   Document all error codes and their meanings.

### Implementation Docs

-   Update `docs/implementation/*.md` files when architectural decisions change.
-   Document the reasoning behind technical choices.

## 6. Quality & Security

### Quality Gates

-   **Before Merge**: All code must be formatted, and all documentation must be spell-checked with valid links and tested examples.
-   **Review**: Documentation and breaking changes require maintainer review. New features must be documented.

### Security Scans (gosec)

-   Run `gosec` on the backend codebase to detect common security issues before submitting changes.
-   **How to run**: `make security` from the repo root.
-   Include the result in the "Quality Gates" section of your PR summary: `Security (gosec): PASS`.

## 7. Build & Deployment

### Build Conventions

1.  **Binary Structure**: Follow the established output structure.
    -   `backend/dist/${ARCH}/`: Production builds (stripped, optimized).
    -   `backend/tmp/`: Development/test builds (with debug symbols).
2.  **Binary Naming**: Use the convention `srat-server`, `srat-cli`, etc.
3.  **Makefile**: Use the `Makefile` targets for building. When adding a new binary, update the `Makefile` accordingly.
4.  **Multi-architecture**: Ensure builds work for `amd64` (x86_64), `armv7`, and `arm64` (aarch64).
5.  **Build Flags**: Use `-ldflags="-s -w"` for production and `-gcflags=all="-N -l"` for development. Always inject the version information.
6.  **Asset Embedding**: Frontend assets are embedded into the production binary using the `embedallowed` build tag.
