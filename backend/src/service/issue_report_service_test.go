package service_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
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
	mockSettingService service.SettingServiceInterface
	ctx                context.Context
	cancel             context.CancelFunc
}

func TestIssueReportServiceSuite(t *testing.T) {
	suite.Run(t, new(IssueReportServiceSuite))
}

func (suite *IssueReportServiceSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			service.NewIssueReportService,
			mock.Mock[service.SettingServiceInterface],
		),
		fx.Populate(&suite.issueReportService),
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
}

func (suite *IssueReportServiceSuite) TestGenerateIssueReport_FrontendUI() {
	// Arrange
	settings := &dto.Settings{}
	mock.When(suite.mockSettingService.Load()).ThenReturn(settings, nil)

	request := &dto.IssueReportRequest{
		ProblemType:        dto.ProblemTypes.PROBLEMTYPEFRONTENDUI,
		Description:        "Button not working",
		IncludeContextData: true,
		CurrentURL:         "http://localhost/shares",
		BrowserInfo:        "Chrome 120",
	}

	// Act
	report, err := suite.issueReportService.GenerateIssueReport(suite.ctx, request)

	// Assert
	suite.NoError(err)
	suite.NotNil(report)
	suite.Contains(report.GitHubURL, "github.com/dianlight/srat")
	suite.Contains(report.IssueTitle, "[UI]")
	//suite.Contains(report.IssueBody, "Button not working")
	//suite.Contains(report.IssueBody, "http://localhost/shares")
	mock.Verify(suite.mockSettingService, matchers.Times(1)).Load()
}

func (suite *IssueReportServiceSuite) TestGenerateIssueReport_AddonProblem() {
	// Arrange
	settings := &dto.Settings{}
	mock.When(suite.mockSettingService.Load()).ThenReturn(settings, nil)

	request := &dto.IssueReportRequest{
		ProblemType:        dto.ProblemTypes.PROBLEMTYPEADDONCRASH,
		Description:        "Addon won't start",
		IncludeContextData: false,
		IncludeSRATConfig:  false,
		IncludeAddonLogs:   false,
	}

	// Act
	report, err := suite.issueReportService.GenerateIssueReport(suite.ctx, request)

	// Assert
	suite.NoError(err)
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
	suite.NoError(err)
	suite.NotNil(report)
	//suite.NotNil(report.SanitizedConfig)
	mock.Verify(suite.mockSettingService, matchers.Times(2)).Load()
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
	suite.NoError(err)
	suite.NotNil(report)
	suite.Contains(report.GitHubURL, "https://github.com/dianlight/hassio-addons")
	suite.Contains(report.IssueTitle, "[SambaNas2]")
}
