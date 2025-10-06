# Backend Test Coverage Improvement Roadmap

This document provides a detailed roadmap for achieving 60% minimum test coverage across all backend packages.

## Current Status (2025-10-06)

**Overall Backend Coverage**: 34.3%  
**Target**: 60% minimum per package  
**Packages Meeting Target**: 10 out of 29 packages (34.5%)

## Executive Summary

### Packages Already Meeting 60% Threshold ✅

| Package                   | Coverage | Status    |
| ------------------------- | -------- | --------- |
| `internal`                | 88.6%    | Excellent |
| `tempio`                  | 85.7%    | Excellent |
| `tlog`                    | 83.9%    | Excellent |
| `internal/appsetup`       | 80.0%    | Excellent |
| `internal/osutil`         | 77.4%    | Excellent |
| `unixsamba`               | 75.1%    | Good      |
| `config`                  | 71.1%    | Good      |
| `repository`              | 71.0%    | Good      |
| `dbom/migrations`         | 63.8%    | Good      |
| `homeassistant/websocket` | 60.9%    | Good      |

**These packages require no immediate action.**

### Priority Packages Requiring Improvement

#### Tier 1: High Priority (Close to target, high impact)

| Package   | Current | Target | Gap   | Effort | Impact |
| --------- | ------- | ------ | ----- | ------ | ------ |
| `api`     | 48.1%   | 60%    | 11.9% | Medium | High   |
| `server`  | 33.8%   | 60%    | 26.2% | Medium | High   |
| `service` | 33.6%   | 60%    | 26.4% | High   | High   |

#### Tier 2: Medium Priority (Core business logic)

| Package                  | Current | Target | Gap   | Effort | Impact |
| ------------------------ | ------- | ------ | ----- | ------ | ------ |
| `homeassistant/mount`    | 36.8%   | 60%    | 23.2% | Medium | Medium |
| `homeassistant/core_api` | 45.0%   | 60%    | 15.0% | Low    | Medium |
| `converter`              | 27.2%   | 60%    | 32.8% | Low    | Medium |
| `homeassistant/host`     | 27.3%   | 60%    | 32.7% | Medium | Medium |
| `homeassistant/core`     | 25.8%   | 60%    | 34.2% | Medium | Medium |

#### Tier 3: Low Priority (Supporting packages)

| Package                    | Current | Target | Gap   | Effort | Impact |
| -------------------------- | ------- | ------ | ----- | ------ | ------ |
| `homeassistant/ingress`    | 20.8%   | 60%    | 39.2% | Medium | Low    |
| `homeassistant/root`       | 19.6%   | 60%    | 40.4% | Medium | Low    |
| `dto`                      | 19.1%   | 60%    | 40.9% | Medium | Medium |
| `dbom`                     | 19.0%   | 60%    | 41.0% | Medium | Medium |
| `cmd/srat-openapi`         | 17.9%   | 60%    | 42.1% | Low    | Low    |
| `homeassistant/hardware`   | 16.3%   | 60%    | 43.7% | Medium | Low    |
| `homeassistant/addons`     | 13.2%   | 60%    | 46.8% | Medium | Low    |
| `homeassistant/resolution` | 13.7%   | 60%    | 46.3% | Medium | Low    |
| `cmd/srat-cli`             | 5.7%    | 60%    | 54.3% | High   | Low    |
| `cmd/srat-server`          | 5.4%    | 60%    | 54.6% | High   | Low    |

## Implementation Phases

### Phase 1: Quick Wins (1-2 weeks)

**Target: Improve 3 high-priority packages to 60%**

#### 1. API Package (48.1% → 60%)

**Files needing tests:**

- [ ] `issue.go` - Issue CRUD operations
- [ ] `telemetry.go` - Telemetry configuration endpoints
- [ ] `upgrade.go` - System upgrade handlers
- [ ] `volumes.go` - Volume mount/unmount operations
- [ ] `sse.go` - Server-Sent Events handlers
- [ ] `ws.go` - WebSocket connection handlers

**Test Pattern to Follow:**

```go
// Use fx-based dependency injection like shares_test.go
suite.app = fxtest.New(suite.T(),
    fx.Provide(
        func() *matchers.MockController { return mock.NewMockController(suite.T()) },
        api.NewYourHandler,
        mock.Mock[service.YourServiceInterface],
    ),
    fx.Populate(&suite.handler, &suite.mockService),
)
```

**Estimated Effort:** 3-4 days  
**Impact:** High - API is the entry point for all operations

#### 2. Server Package (33.8% → 60%)

**Areas needing tests:**

- [ ] Server initialization with different configurations
- [ ] Middleware chain execution
- [ ] WebSocket connection lifecycle
- [ ] SSE connection management
- [ ] Error handling and recovery
- [ ] Graceful shutdown

**Estimated Effort:** 2-3 days  
**Impact:** High - Core server functionality

#### 3. Service Package (33.6% → 60%)

**Priority services to test:**

- [ ] Filesystem service - mount/unmount operations
- [ ] Share service - share creation, modification, deletion
- [ ] User service - user management operations
- [ ] System service - system information and control

**Strategy:**

- Focus on untested service methods
- Test error conditions and edge cases
- Test service interactions
- Mock repository dependencies

**Estimated Effort:** 4-5 days  
**Impact:** High - Core business logic

### Phase 2: Core Business Logic (2-3 weeks)

**Target: Improve medium-priority packages**

#### 4. Converter Package (27.2% → 60%)

- Test all goverter-generated converters
- Test null/empty value handling
- Test type conversion edge cases
- Test nested object conversions

**Estimated Effort:** 1-2 days  
**Impact:** Medium - Data transformation layer

#### 5. DBOM Package (19.0% → 60%)

- Test model methods and validations
- Test GORM relationships
- Test custom callbacks (BeforeCreate, AfterFind)
- Test JSON marshaling for complex fields

**Estimated Effort:** 2-3 days  
**Impact:** Medium - Data persistence layer

#### 6. DTO Package (19.1% → 60%)

- Test DTO validation rules
- Test error code definitions
- Test serialization/deserialization
- Test DTO helper methods

**Estimated Effort:** 2-3 days  
**Impact:** Medium - Data transfer layer

### Phase 3: Home Assistant Integration (2-3 weeks)

**Target: Improve homeassistant/\* packages**

All homeassistant packages follow similar patterns:

- Mock HTTP responses from supervisor
- Test error handling for network failures
- Test response parsing
- Test authentication flows

**Packages:**

- `homeassistant/mount` (36.8%)
- `homeassistant/core_api` (45.0%)
- `homeassistant/host` (27.3%)
- `homeassistant/core` (25.8%)
- `homeassistant/ingress` (20.8%)
- `homeassistant/root` (19.6%)
- `homeassistant/hardware` (16.3%)
- `homeassistant/addons` (13.2%)
- `homeassistant/resolution` (13.7%)

**Estimated Effort:** 4-5 days  
**Impact:** Medium - External integration layer

### Phase 4: CLI and Entry Points (Optional)

**Note:** These packages are difficult to test and have lower priority.

- `cmd/srat-cli` (5.7%)
- `cmd/srat-server` (5.4%)
- `cmd/srat-openapi` (17.9%)

**Strategy:**

- Focus on flag validation and command parsing
- Add integration tests for critical commands
- Accept lower coverage for main entry points

**Estimated Effort:** 2-3 days  
**Impact:** Low - Entry points, business logic tested elsewhere

## Testing Best Practices

### 1. FX-Based Dependency Injection

Always use `fxtest.New()` for proper dependency management:

```go
suite.app = fxtest.New(suite.T(),
    fx.Provide(
        func() *matchers.MockController { return mock.NewMockController(suite.T()) },
        service.NewYourService,
        mock.Mock[repository.RepositoryInterface],
    ),
    fx.Populate(&suite.service, &suite.mockRepo),
)
suite.app.RequireStart()
```

### 2. Mock Pattern

Use mockio v2 for mocking:

```go
// Setup
mock.When(mockService.Method(matchers.Any[Type]())).ThenReturn(result, nil)

// Verify
mock.Verify(mockService, matchers.Times(1)).Method(matchers.Any[Type]())
```

### 3. Test Both Paths

Always test success and error scenarios:

```go
func (suite *Suite) TestMethodSuccess() { /* ... */ }
func (suite *Suite) TestMethodError() { /* ... */ }
func (suite *Suite) TestMethodNotFound() { /* ... */ }
```

### 4. Verify State Changes

For operations that modify data, verify dirty state:

```go
mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyShares()
```

### 5. Cleanup

Always implement proper cleanup:

```go
func (suite *Suite) TearDownTest() {
    if suite.cancel != nil {
        suite.cancel()
    }
    if suite.app != nil {
        suite.app.RequireStop()
    }
}
```

## Measuring Progress

### Running Coverage Reports

```bash
# Full coverage report
cd backend && make test

# Package-specific coverage
cd backend/src && go test -cover ./api

# Detailed coverage analysis
cd backend/src && go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage Tracking

Update `docs/TEST_COVERAGE.md` after each significant improvement:

```bash
# Run update script (when available)
./scripts/update-coverage-badges.sh

# Manual update
cd backend && make test | grep "Total coverage"
```

### Success Criteria

**Phase 1 Complete:**

- API package ≥ 60%
- Server package ≥ 60%
- Service package ≥ 60%
- Overall backend coverage ≥ 42%

**Phase 2 Complete:**

- Converter package ≥ 60%
- DBOM package ≥ 60%
- DTO package ≥ 60%
- Overall backend coverage ≥ 48%

**Phase 3 Complete:**

- All homeassistant/\* packages ≥ 60% (except already passing)
- Overall backend coverage ≥ 55%

**Final Goal:**

- All core packages ≥ 60%
- Overall backend coverage ≥ 60%

## Resources

- [TEST_COVERAGE.md](./TEST_COVERAGE.md) - Current coverage status and guidelines
- [Backend Testing Patterns](../backend/README.md#testing)
- [Copilot Instructions](../.github/copilot-instructions.md) - Coding agent guidelines
- Existing test files:
  - `backend/src/api/shares_test.go` - API handler pattern
  - `backend/src/service/telemetry_service_test.go` - Service pattern
  - `backend/src/repository/*_test.go` - Repository pattern

## Timeline Estimate

| Phase                   | Duration  | Cumulative | Target Coverage |
| ----------------------- | --------- | ---------- | --------------- |
| Phase 1: Quick Wins     | 1-2 weeks | 1-2 weeks  | 42%             |
| Phase 2: Core Logic     | 2-3 weeks | 3-5 weeks  | 48%             |
| Phase 3: HA Integration | 2-3 weeks | 5-8 weeks  | 55%             |
| Phase 4: CLI (Optional) | 2-3 weeks | 7-11 weeks | 60%+            |

**Recommended Approach:** Focus on Phases 1-3 first to achieve meaningful coverage improvement where it matters most.

## Next Steps

1. **Immediate (Week 1):**
   - Create test file for `api/issue.go`
   - Create test file for `api/telemetry.go`
   - Create test file for `api/volumes.go`

2. **Short-term (Week 2-3):**
   - Complete all API handler tests
   - Begin server package tests
   - Start service package improvement

3. **Medium-term (Month 2):**
   - Complete converter, dbom, dto packages
   - Begin homeassistant package improvements

4. **Long-term (Month 3+):**
   - Complete remaining homeassistant packages
   - Optionally improve CLI packages
   - Achieve 60%+ overall coverage

## Notes

- **cmd/\* packages**: Low priority due to difficulty in testing entry points. The business logic they call is tested elsewhere.
- **Auto-generated code**: Converter package uses goverter; tests focus on edge cases and integration rather than generated code itself.
- **Mocking complexity**: Some packages require extensive mocking setup. Refer to existing tests for patterns.
- **CI Integration**: Pre-commit hooks and GitHub Actions should enforce minimum coverage thresholds once targets are met.

---

*Last Updated: 2025-10-06*
