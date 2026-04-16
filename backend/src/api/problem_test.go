package api_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ProblemHandlerSuite struct {
	suite.Suite
	app         *fxtest.App
	handler     *api.ProblemAPI
	mockProblem service.ProblemServiceInterface
	ctx         context.Context
	cancel      context.CancelFunc
}

func TestProblemHandlerSuite(t *testing.T) {
	suite.Run(t, new(ProblemHandlerSuite))
}

func (suite *ProblemHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			api.NewProblemAPI,
			mock.Mock[service.ProblemServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockProblem),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *ProblemHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func sampleProblem() *dto.Problem {
	return &dto.Problem{
		ProblemKey:  "sample.problem",
		Title:       "Sample problem",
		Description: "desc",
		Severity:    dto.ProblemSeverities.PROBLEMSEVERITYERROR,
		Status:      dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (suite *ProblemHandlerSuite) TestGetProblemsSuccess() {
	expected := []*dto.Problem{sampleProblem()}
	mock.When(suite.mockProblem.List()).ThenReturn(expected, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterProblemHandler(apiInst)

	resp := apiInst.Get("/problems")
	suite.Require().Equal(http.StatusOK, resp.Code)
	_, _ = mock.Verify(suite.mockProblem, matchers.Times(1)).List()
}

func (suite *ProblemHandlerSuite) TestGetProblemSuccess() {
	expected := sampleProblem()
	mock.When(suite.mockProblem.Get(mock.Exact("sample.problem"))).ThenReturn(expected, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterProblemHandler(apiInst)

	resp := apiInst.Get("/problems/sample.problem")
	suite.Require().Equal(http.StatusOK, resp.Code)
	_, _ = mock.Verify(suite.mockProblem, matchers.Times(1)).Get(mock.Exact("sample.problem"))
}

func (suite *ProblemHandlerSuite) TestUpsertProblemSuccess() {
	item := sampleProblem()
	mock.When(suite.mockProblem.Upsert(mock.Any[*dto.Problem]())).ThenReturn(item, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterProblemHandler(apiInst)

	resp := apiInst.Post("/problems", map[string]any{
		"id":          0,
		"problem_key": "sample.problem",
		"title":       "Sample problem",
		"description": "desc",
		"severity":    "error",
		"status":      "created",
		"repeating":   0,
		"ignored":     false,
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
	})
	suite.Require().Equal(http.StatusOK, resp.Code)
	_, _ = mock.Verify(suite.mockProblem, matchers.Times(1)).Upsert(mock.Any[*dto.Problem]())
}

func (suite *ProblemHandlerSuite) TestDismissProblemSuccess() {
	mock.When(suite.mockProblem.Dismiss(mock.Exact("sample.problem"))).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterProblemHandler(apiInst)

	resp := apiInst.Delete("/problems/sample.problem")
	suite.Require().Equal(http.StatusNoContent, resp.Code)
	_ = mock.Verify(suite.mockProblem, matchers.Times(1)).Dismiss(mock.Exact("sample.problem"))
}

func (suite *ProblemHandlerSuite) TestGetProblemsError() {
	mock.When(suite.mockProblem.List()).ThenReturn(nil, errors.New("db"))

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterProblemHandler(apiInst)

	resp := apiInst.Get("/problems")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
	_, _ = mock.Verify(suite.mockProblem, matchers.Times(1)).List()
}

func (suite *ProblemHandlerSuite) TestPutProblemByKeySuccess() {
	item := sampleProblem()
	item.ProblemKey = "put.problem"
	mock.When(suite.mockProblem.Upsert(mock.Any[*dto.Problem]())).ThenReturn(item, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterProblemHandler(apiInst)

	resp := apiInst.Put("/problems/put.problem", map[string]any{
		"id":          0,
		"problem_key": "put.problem",
		"title":       "Put problem",
		"description": "desc",
		"severity":    "warning",
		"status":      "created",
		"repeating":   0,
		"ignored":     false,
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
	})
	suite.Require().Equal(http.StatusOK, resp.Code)
	_, _ = mock.Verify(suite.mockProblem, matchers.Times(1)).Upsert(mock.Any[*dto.Problem]())
}

func (suite *ProblemHandlerSuite) TestExecuteProblemActionSuccess() {
	item := sampleProblem()
	item.ProblemKey = "sample.problem"
	item.Actions = []dto.ProblemAction{{Key: "restart", Label: "Restart"}}
	fixed := sampleProblem()
	fixed.ProblemKey = "sample.problem"
	fixed.Status = dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED

	mock.When(suite.mockProblem.Get(mock.Exact("sample.problem"))).ThenReturn(item, nil)
	mock.When(
		suite.mockProblem.ApplyLifecycle(
			mock.Exact("sample.problem"),
			mock.Exact(dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED),
			mock.Any[*string](),
		),
	).ThenReturn(fixed, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterProblemHandler(apiInst)

	resp := apiInst.Post("/problems/sample.problem/actions/restart", map[string]any{})
	suite.Require().Equal(http.StatusOK, resp.Code)
	_, _ = mock.Verify(suite.mockProblem, matchers.Times(1)).Get(mock.Exact("sample.problem"))
	_, _ = mock.Verify(suite.mockProblem, matchers.Times(1)).ApplyLifecycle(
		mock.Exact("sample.problem"),
		mock.Exact(dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED),
		mock.Any[*string](),
	)
}
