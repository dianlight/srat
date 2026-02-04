## Test Failure Analysis

The test `TestIssueReportServiceSuite/TestGenerateIssueReport_AddonProblem` is failing because the `AddonsServiceInterface` dependency is not provided in the test setup.

### Error
```
missing dependencies for function "github.com/dianlight/srat/service".NewIssueReportService:
missing type: service.AddonsServiceInterface (did you mean to Provide it?)
```

### Root Cause
In `backend/src/service/issue_report_service_test.go`, the `SetupTest` method only provides `SettingServiceInterface` but `NewIssueReportService` requires both `SettingServiceInterface` and `AddonsServiceInterface`.

### Solution
Add the missing mock provider in the test setup:

```go
suite.app = fxtest.New(suite.T(),
    fx.Provide(
        func() *matchers.MockController { return mock.NewMockController(suite.T()) },
        func() (context.Context, context.CancelFunc) {
            return suite.ctx, suite.cancel
        },
        service.NewIssueReportService,
        mock.Mock[service.SettingServiceInterface],
        mock.Mock[service.AddonsServiceInterface],  // ADD THIS LINE
    ),
    fx.Populate(&suite.issueReportService),
    fx.Populate(&suite.mockSettingService),
)
```