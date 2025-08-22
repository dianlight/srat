## COPILOT: SRAT repository — quick orientation for coding agents

This file highlights the must-know, discoverable rules and workflows for productive changes in this repo. Keep it short and actionable.

- Languages: Go backend (Go 1.25), TypeScript React frontend (Bun runtime). See `backend/go.mod` and `frontend/package.json`/`bun.lockb`.
- Builds: backend via `backend/Makefile` (root `make` proxies many targets). Frontend uses Bun: `cd frontend && bun install && bun run build`.
- Pre-commit and hooks: repository uses `pre-commit`. Do not edit `.git/hooks` manually. See `.pre-commit-config.yaml` and run `pre-commit install` locally.

Key patterns and where to find them
- API handlers: `backend/src/api/*` — handlers use constructor `NewXHandler` and `RegisterXHandler(api huma.API)`.
- Services: `backend/src/service/*` — each service has an interface and an implementation wired via FX (`fx.In` param structs).
- Repositories/DB: `backend/src/repository/*` and `backend/src/dbom/*` — GORM models live in `dbom`, converters in `converter/` and `goverter`-generated files.
- DTOs: `backend/src/dto` define domain errors (see `dto/error_code.go`) and request/response shapes.
- Logging: `backend/src/tlog` — use `tlog` helpers for structured logs and sensitive-data masking.

Testing and mocks
- Tests use `testify/suite` and `mockio/v2` for mocks. Test packages are named `{pkg}_test` and suites follow `XHandlerSuite` pattern. See `backend/src/api/*.go` and `*_test.go` for examples.
- HTTP handler tests often use `humatest.New` to create a test API and `autopatch.AutoPatch(api)` for PATCH endpoints.

Dev and quality gates
- Run documentation checks: `make docs-validate` (root). Docs and examples are in `docs/`.
- Security: `make security` runs `gosec` (backend). CI expects gosec/pass for commits.
- Formatting: use `gofmt` for Go; frontend uses Biome/biome config (see `biome.json`).

Quick examples
- Start backend dev build: `cd backend && make run` (see `backend/Makefile`).
- Run backend unit tests: `cd backend && go test ./...`.
- Build all: `make ALL` (root Makefile).

Integration notes for agents
- Many constructors and handlers are wired via Uber FX — when changing signatures, update providers and `fx.Populate` calls in tests.
- Converters are generated (`*gen.go`) — if you change DTO/dbom shapes, regenerate converters or update `converter/*` accordingly.
- Sensitive secrets (Rollbar) are optional and guarded; don't hardcode secrets — use env or config flags the project expects.

Files to open first
- `backend/Makefile`, `backend/src/api/*`, `backend/src/service/*`, `backend/src/dto/error_code.go`, `backend/src/converter/*`.
- `frontend/src/pages/` and `frontend/src/components/` for UI changes.

If uncertain, run these checks before PR
- `pre-commit run --all-files`
- `make docs-validate`
- `make security`

If this file misses anything important, tell me which area (build, tests, DI, logging, frontend) and I will expand with concrete examples.

1. **Mock Dependencies**: Use `mockio/v2` for mocking services and repositories.
2. **Test Paths**: Test both success and error paths.
3. **Verify State**: Check that `dirtyService.SetDirty*()` methods are called when data is modified.
4. **Meaningful Names**: Use the pattern `Test{Method}{Scenario}` (e.g., `TestCreateShareSuccess`).
5. **Use Assertions**: Prefer `testify/assert` or `testify/require` (e.g., `suite.Equal(...)`).

### Test Suite Structure

- **Package**: `{package}_test`
- **Suite Struct**: Name it `{HandlerName}HandlerSuite`. It must embed `suite.Suite` and contain fields for the handler, mock services, `fxtest.App`, and a `context`.

### SetupTest / TearDownTest Methods

- **`SetupTest`**: Use `fxtest.New` to build the dependency graph. Provide all mocks, real services, configuration, and context. Use `fx.Populate` to inject dependencies into the suite fields. Set up mock expectations using `mock.When(...)`.
- **`TearDownTest`**: Clean up resources. Cancel the context and wait for any `WaitGroup` to finish, then call `suite.app.RequireStop()`.

### HTTP Test Pattern (`humatest`)

1. **Initialize**: `_, api := humatest.New(suite.T())`
2. **Register Handler**: `suite.handler.RegisterSomething(api)`
3. **Enable AutoPatch**: If testing a `PATCH` endpoint, call `autopatch.AutoPatch(api)`.
4. **Make Request**: `resp := api.Get("/endpoint")`
5. **Assert Status**: `suite.Require().Equal(http.StatusOK, resp.Code)`
6. **Assert Body**: Unmarshal the response and assert its contents.

### Direct Handler Test Pattern

1. **Prepare Input**: Create the required input DTO.
2. **Configure Mock**: `mock.When(suite.mockService.Method(...)).ThenReturn(...)`
3. **Execute**: Call the handler method directly: `result, err := suite.handler.Method(ctx, input)`
4. **Assert**: Check the error and the result.
5. **Verify Mock**: `mock.Verify(suite.mockService, matchers.Times(1)).Method()`

### Error Handling Tests

- Configure the mock to return a specific error.
- Execute the handler method.
- Assert that an error is returned (`suite.Error(err)`).
- If applicable, check for a specific `huma.StatusError` and status code.

### Test Runner

- Always include the test runner function:

  ```go
  func TestHandlerNameSuite(t *testing.T) {
      suite.Run(t, new(HandlerNameSuite))
  }
  ```

### Test Data

- Place test configuration data in `backend/test/data/`.
- Reference files using relative paths (e.g., `"../../test/data/config.json"`).

## 5. Documentation

### General Markdown Rules

- **Update `CHANGELOG.md`** for all significant changes.
- Use proper heading hierarchy and consistent formatting.
- Include fenced code blocks with language identifiers.
- Keep all documentation, examples, and links current and valid.

### Package Documentation and Examples (Go)

When adding or changing features, update the corresponding documentation.

1. **READMEs**: Update package `README.md` files with new features, API documentation, usage examples, and best practices.
2. **API Reference (GoDoc)**: Ensure all public functions, types, and constants are fully documented. Include examples for complex APIs.
3. **Examples**: Update or create runnable example programs that demonstrate new features, error handling, and best practices.

### API Documentation (OpenAPI)

- Update OpenAPI specs when API endpoints change.
- Include clear request/response examples.
- Document all error codes and their meanings.

### Implementation Docs

- Update `docs/implementation/*.md` files when architectural decisions change.
- Document the reasoning behind technical choices.

## 6. Quality & Security

### Quality Gates

- **Before Merge**: All code must be formatted, and all documentation must be spell-checked with valid links and tested examples.
- **Review**: Documentation and breaking changes require maintainer review. New features must be documented.

### Security Scans (gosec)

- Run `gosec` on the backend codebase to detect common security issues before submitting changes.
- **How to run**: `make security` from the repo root.
- Include the result in the "Quality Gates" section of your PR summary: `Security (gosec): PASS`.

## 7. Build & Deployment

### Build Conventions

1. **Binary Structure**: Follow the established output structure.
   - `backend/dist/${ARCH}/`: Production builds (stripped, optimized).
   - `backend/tmp/`: Development/test builds (with debug symbols).
2. **Binary Naming**: Use the convention `srat-server`, `srat-cli`, etc.
3. **Makefile**: Use the `Makefile` targets for building. When adding a new binary, update the `Makefile` accordingly.
4. **Multi-architecture**: Ensure builds work for `amd64` (x86_64), `armv7`, and `arm64` (aarch64).
5. **Build Flags**: Use `-ldflags="-s -w"` for production and `-gcflags=all="-N -l"` for development. Always inject the version information.
6. **Asset Embedding**: Frontend assets are embedded into the production binary using the `embedallowed` build tag.
