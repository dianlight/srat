package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ProblemServiceSuite struct {
	suite.Suite
	app      *fxtest.App
	svc      service.ProblemServiceInterface
	eventBus events.EventBusInterface
}

func TestProblemServiceSuite(t *testing.T) {
	suite.Run(t, new(ProblemServiceSuite))
}

func (suite *ProblemServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(func() *matchers.MockController { return mock.NewMockController(suite.T()) }, func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
		},
			func() *dto.ContextState {
				return &dto.ContextState{
					DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)",
				}
			},
			dbom.NewDB,
			mock.Mock[events.EventBusInterface],
			service.NewProblemService,
		),
		fx.Populate(&suite.svc),
		fx.Populate(&suite.eventBus),
	)
	suite.app.RequireStart()
}

func (suite *ProblemServiceSuite) TearDownTest() {
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func (suite *ProblemServiceSuite) TestUpsertCreate() {
	p, err := suite.svc.Upsert(&dto.Problem{
		ProblemKey:  "test_key",
		Title:       "Test Problem",
		Description: "Some description",
		Severity:    dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(p)
	suite.Equal("test_key", p.ProblemKey)
	suite.Equal("Test Problem", p.Title)
	suite.Equal(uint(1), p.Repeating)
}

func (suite *ProblemServiceSuite) TestUpsertUpdateIncrementsRepeating() {
	_, err := suite.svc.Upsert(&dto.Problem{
		ProblemKey: "dup_key",
		Title:      "First",
		Severity:   dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
	})
	suite.Require().NoError(err)

	updated, err := suite.svc.Upsert(&dto.Problem{
		ProblemKey:  "dup_key",
		Title:       "Updated",
		Severity:    dto.ProblemSeverities.PROBLEMSEVERITYERROR,
		Description: "updated desc",
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(updated)
	suite.Equal(uint(2), updated.Repeating)
	suite.Equal("Updated", updated.Title)
}

func (suite *ProblemServiceSuite) TestUpsertRequiresProblemKeyOrTitle() {
	_, err := suite.svc.Upsert(&dto.Problem{})
	suite.Require().Error(err)
}

func (suite *ProblemServiceSuite) TestGet() {
	_, err := suite.svc.Upsert(&dto.Problem{
		ProblemKey: "get_key",
		Title:      "Get Test",
		Severity:   dto.ProblemSeverities.PROBLEMSEVERITYINFO,
	})
	suite.Require().NoError(err)

	p, err := suite.svc.Get("get_key")
	suite.Require().NoError(err)
	suite.Require().NotNil(p)
	suite.Equal("get_key", p.ProblemKey)
}

func (suite *ProblemServiceSuite) TestGetMissing() {
	_, err := suite.svc.Get("no_such_key")
	suite.Require().Error(err)
}

func (suite *ProblemServiceSuite) TestList() {
	_, err := suite.svc.Upsert(&dto.Problem{ProblemKey: "list_a", Title: "A", Severity: dto.ProblemSeverities.PROBLEMSEVERITYWARNING})
	suite.Require().NoError(err)
	_, err = suite.svc.Upsert(&dto.Problem{ProblemKey: "list_b", Title: "B", Severity: dto.ProblemSeverities.PROBLEMSEVERITYINFO})
	suite.Require().NoError(err)

	list, err := suite.svc.List()
	suite.Require().NoError(err)
	suite.GreaterOrEqual(len(list), 2)
}

func (suite *ProblemServiceSuite) TestDismiss() {
	_, err := suite.svc.Upsert(&dto.Problem{ProblemKey: "dismiss_key", Title: "To Dismiss", Severity: dto.ProblemSeverities.PROBLEMSEVERITYWARNING})
	suite.Require().NoError(err)

	err = suite.svc.Dismiss("dismiss_key")
	suite.Require().NoError(err)

	_, err = suite.svc.Get("dismiss_key")
	suite.Require().Error(err)
}

func (suite *ProblemServiceSuite) TestDismissMissing() {
	err := suite.svc.Dismiss("no_such_key")
	suite.Require().Error(err)
}

func (suite *ProblemServiceSuite) TestApplyLifecycle() {
	_, err := suite.svc.Upsert(&dto.Problem{
		ProblemKey: "lifecycle_key",
		Title:      "Lifecycle",
		Severity:   dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
	})
	suite.Require().NoError(err)

	errMsg := "something went wrong"
	p, err := suite.svc.ApplyLifecycle("lifecycle_key", dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR, &errMsg)
	suite.Require().NoError(err)
	suite.Require().NotNil(p)
	suite.Equal(dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR, p.Status)
	suite.Require().NotNil(p.LastError)
	suite.Equal(errMsg, *p.LastError)
}

func (suite *ProblemServiceSuite) TestApplyLifecycleMissing() {
	_, err := suite.svc.ApplyLifecycle("missing_key", dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR, nil)
	suite.Require().Error(err)
}
