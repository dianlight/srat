package service_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/google/go-github/v82/github"
	"github.com/jarcoal/httpmock"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type IssueReportServiceSuite struct {
	suite.Suite
	app                *fxtest.App
	issueReportService service.IssueReportServiceInterface
	mockAddonService   service.AddonsServiceInterface
	mockSettingService service.SettingServiceInterface
	ctx                context.Context
	cancel             context.CancelFunc
}

//const githubGistURL = "https://api.github.com/gists"

func TestIssueReportServiceSuite(t *testing.T) {
	suite.Run(t, new(IssueReportServiceSuite))
}

func (suite *IssueReportServiceSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Ensure ALL outbound HTTP calls are intercepted by httpmock.
	// Any request without an explicit responder will fail the test (prevents real GitHub calls).
	httpmock.Activate()
	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected http call (missing httpmock responder): %s %s", req.Method, req.URL.String())
	})

	githubHTTPClient := &http.Client{}
	httpmock.ActivateNonDefault(githubHTTPClient)

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			service.NewIssueReportService,
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			func() *github.Client {
				return github.NewClient(githubHTTPClient)
			},
		),
		fx.Populate(&suite.issueReportService),
		fx.Populate(&suite.mockAddonService),
		fx.Populate(&suite.mockSettingService),
	)
	suite.app.RequireStart()
}

func (suite *IssueReportServiceSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
	httpmock.DeactivateAndReset()
}

func (suite *IssueReportServiceSuite) TestGenerateIssueReport_FrontendUI() {
	// Arrange
	settings := &dto.Settings{}
	mock.When(suite.mockSettingService.Load()).ThenReturn(settings, nil)

	request := &dto.IssueReportRequest{
		ProblemType: dto.ProblemTypes.PROBLEMTYPEFRONTENDUI,
		Description: "Button not working",
	}

	// Act
	report, err := suite.issueReportService.GenerateIssueReport(suite.ctx, request)

	// Assert
	suite.Require().NoError(err)
	suite.Require().NotNil(report)
	suite.Contains(report.GitHubURL, "github.com/dianlight/srat")
	suite.Contains(report.IssueTitle, "[UI]")
	suite.Contains(report.GitHubURL, "Button+not+working")
	mock.Verify(suite.mockSettingService, matchers.Times(0)).Load()
}

func (suite *IssueReportServiceSuite) TestGenerateIssueReport_AddonProblem() {
	// Arrange
	settings := &dto.Settings{}
	mock.When(suite.mockSettingService.Load()).ThenReturn(settings, nil)

	request := &dto.IssueReportRequest{
		ProblemType:       dto.ProblemTypes.PROBLEMTYPEADDONCRASH,
		Description:       "Addon won't start",
		IncludeSRATConfig: false,
		IncludeAddonLogs:  false,
	}

	// Act
	report, err := suite.issueReportService.GenerateIssueReport(suite.ctx, request)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(report)
	suite.Contains(report.GitHubURL, "https://github.com/dianlight/hassio-addons")
	suite.Contains(report.IssueTitle, "[SambaNas2]")
	//suite.Contains(report.IssueBody, "Addon won't start")
}

func (suite *IssueReportServiceSuite) TestGenerateIssueReport_WithConfig() {
	// Arrange
	settings := &dto.Settings{}
	mock.When(suite.mockSettingService.Load()).ThenReturn(settings, nil)

	request := &dto.IssueReportRequest{
		ProblemType:       dto.ProblemTypes.PROBLEMTYPEHAINTEGRATION,
		Description:       "Integration not connecting",
		IncludeSRATConfig: true,
	}

	// Act
	report, err := suite.issueReportService.GenerateIssueReport(suite.ctx, request)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(report)
	//suite.NotNil(report.SanitizedConfig)
	mock.Verify(suite.mockSettingService, matchers.Times(1)).Load()
}

func (suite *IssueReportServiceSuite) TestGenerateIssueReport_SambaProblem() {
	// Arrange
	settings := &dto.Settings{}
	mock.When(suite.mockSettingService.Load()).ThenReturn(settings, nil)

	request := &dto.IssueReportRequest{
		ProblemType: dto.ProblemTypes.PROBLEMTYPESAMBA,
		Description: "Samba service crash",
	}

	// Act
	report, err := suite.issueReportService.GenerateIssueReport(suite.ctx, request)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(report)
	suite.Contains(report.GitHubURL, "https://github.com/dianlight/srat")
	suite.Contains(report.IssueTitle, "[Samba]")
}

func (suite *IssueReportServiceSuite) TestGenerateIssueReport_SambaProblem_WithLargeDiagnostics() {
	// Arrange
	settings := &dto.Settings{}
	mock.When(suite.mockSettingService.Load()).ThenReturn(settings, nil)

	addonConfig := strings.Repeat("a", 1200)
	addonLogs := strings.Repeat("l", 9000)
	consoleErrors := []string{strings.Repeat("e", 1200)}

	suite.Require().NoError(os.MkdirAll("/data", 0o755))
	suite.Require().NoError(os.WriteFile("/data/options.json", []byte(addonConfig), 0o600))
	suite.T().Cleanup(func() {
		_ = os.Remove("/data/options.json")
	})

	mock.When(suite.mockAddonService.GetLatestLogs(suite.ctx)).ThenReturn(addonLogs, nil)

	httpmock.RegisterResponder("POST", "https://api.github.com/gists",
		httpmock.NewJsonResponderOrPanic(http.StatusCreated, map[string]any{
			"files": map[string]any{
				"addon_logs.txt": map[string]any{
					"raw_url": "https://gist.github.com/raw/addon_logs.txt",
				},
			},
		}),
	)

	request := &dto.IssueReportRequest{
		ProblemType:          dto.ProblemTypes.PROBLEMTYPESAMBA,
		Title:                "Diagnostics",
		Description:          "Samba issue with large diagnostics",
		IncludeSRATConfig:    true,
		IncludeAddonConfig:   true,
		IncludeAddonLogs:     true,
		IncludeConsoleErrors: true,
		ConsoleErrors:        consoleErrors,
	}

	// Act
	report, err := suite.issueReportService.GenerateIssueReport(suite.ctx, request)

	// Assert
	suite.Require().NoError(err)
	suite.Require().NotNil(report)
	suite.Contains(report.GitHubURL, "https://github.com/dianlight/srat")
	suite.Contains(report.IssueTitle, "[Samba]")
	suite.Contains(report.GitHubURL, "addon_config=")
	suite.Contains(report.GitHubURL, "srat_config=")
	suite.Contains(report.GitHubURL, "logs="+url.QueryEscape("https://gist.github.com/raw/addon_logs.txt"))

	callInfo := httpmock.GetCallCountInfo()
	suite.Equal(1, callInfo["POST https://api.github.com/gists"])
	mock.Verify(suite.mockSettingService, matchers.Times(1)).Load()
}
