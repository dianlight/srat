package api_test

import (
	"context"
	"encoding/json"
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
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type IssueHandlerSuite struct {
	suite.Suite
	app          *fxtest.App
	handler      *api.IssueAPI
	mockIssueSvc service.IssueServiceInterface
	ctx          context.Context
	cancel       context.CancelFunc
}

func TestIssueHandlerSuite(t *testing.T) {
	suite.Run(t, new(IssueHandlerSuite))
}

func (suite *IssueHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			api.NewIssueAPI,
			mock.Mock[service.IssueServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockIssueSvc),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *IssueHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value("wg").(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func (suite *IssueHandlerSuite) TestGetIssuesSuccess() {
	expected := []*dto.Issue{{ID: 1, Title: "A"}, {ID: 2, Title: "B"}}
	mock.When(suite.mockIssueSvc.FindOpen()).ThenReturn(expected, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)

	resp := apiInst.Get("/issues")
	suite.Require().Equal(http.StatusOK, resp.Code)
	mock.Verify(suite.mockIssueSvc, matchers.Times(1)).FindOpen()
}

func (suite *IssueHandlerSuite) TestGetIssuesError() {
	mock.When(suite.mockIssueSvc.FindOpen()).ThenReturn(nil, errors.New("db"))
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)
	resp := apiInst.Get("/issues")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
	mock.Verify(suite.mockIssueSvc, matchers.Times(1)).FindOpen()
}

func (suite *IssueHandlerSuite) TestCreateIssueSuccess() {
	mock.When(suite.mockIssueSvc.Create(mock.Any[*dto.Issue]())).ThenReturn(nil)
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)
	// Provide all required fields to pass validation
	resp := apiInst.Post("/issues", map[string]any{
		"id":          0,
		"date":        "2025-01-01T00:00:00Z",
		"title":       "T",
		"description": "D",
		"repeating":   0,
		"ignored":     false,
	})
	suite.Require().Equal(http.StatusOK, resp.Code)
	mock.Verify(suite.mockIssueSvc, matchers.Times(1)).Create(mock.Any[*dto.Issue]())
}

func (suite *IssueHandlerSuite) TestCreateIssueError() {
	mock.When(suite.mockIssueSvc.Create(mock.Any[*dto.Issue]())).ThenReturn(errors.New("create"))
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)
	resp := apiInst.Post("/issues", map[string]any{
		"id":          0,
		"date":        "2025-01-01T00:00:00Z",
		"title":       "T",
		"description": "D",
		"repeating":   0,
		"ignored":     false,
	})
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
	mock.Verify(suite.mockIssueSvc, matchers.Times(1)).Create(mock.Any[*dto.Issue]())
}

func (suite *IssueHandlerSuite) TestResolveIssueSuccess() {
	mock.When(suite.mockIssueSvc.Resolve(uint(1))).ThenReturn(nil)
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)
	resp := apiInst.Delete("/issues/1")
	suite.Require().Equal(http.StatusNoContent, resp.Code)
	mock.Verify(suite.mockIssueSvc, matchers.Times(1)).Resolve(uint(1))
}

func (suite *IssueHandlerSuite) TestResolveIssueError() {
	mock.When(suite.mockIssueSvc.Resolve(uint(1))).ThenReturn(errors.New("resolve"))
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)
	resp := apiInst.Delete("/issues/1")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
	mock.Verify(suite.mockIssueSvc, matchers.Times(1)).Resolve(uint(1))
}

func (suite *IssueHandlerSuite) TestUpdateIssueSuccess() {
	updated := &dto.Issue{ID: 1, Title: "U"}
	mock.When(suite.mockIssueSvc.Update(mock.Any[*dto.Issue]())).ThenReturn(updated, nil)
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)
	resp := apiInst.Put("/issues/1", map[string]any{
		"id":          1,
		"date":        "2025-01-01T00:00:00Z",
		"title":       "U",
		"description": "DD",
		"repeating":   0,
		"ignored":     false,
	})
	suite.Require().Equal(http.StatusOK, resp.Code)
	var out dto.Issue
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal(uint(1), out.ID)
	mock.Verify(suite.mockIssueSvc, matchers.Times(1)).Update(mock.Any[*dto.Issue]())
}

func (suite *IssueHandlerSuite) TestUpdateIssueError() {
	mock.When(suite.mockIssueSvc.Update(mock.Any[*dto.Issue]())).ThenReturn(nil, errors.New("update"))
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)
	resp := apiInst.Put("/issues/1", map[string]any{
		"id":          1,
		"date":        "2025-01-01T00:00:00Z",
		"title":       "U",
		"description": "DD",
		"repeating":   0,
		"ignored":     false,
	})
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
	mock.Verify(suite.mockIssueSvc, matchers.Times(1)).Update(mock.Any[*dto.Issue]())
}

// TestCreateIssueValidationError ensures missing required fields trigger 422 before hitting service layer
func (suite *IssueHandlerSuite) TestCreateIssueValidationError() {
	// No mock expectation: validation should intercept
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterIssueHandler(apiInst)
	// Omitting required fields like id, date, repeating, ignored
	resp := apiInst.Post("/issues", map[string]any{"title": "T"})
	suite.Require().Equal(http.StatusUnprocessableEntity, resp.Code)
	// Ensure Create was never invoked
	mock.Verify(suite.mockIssueSvc, matchers.Times(0)).Create(mock.Any[*dto.Issue]())
}
