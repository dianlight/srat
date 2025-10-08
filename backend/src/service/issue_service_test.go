package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type IssueServiceSuite struct {
	suite.Suite
	issueService service.IssueServiceInterface
	issueRepo    repository.IssueRepositoryInterface
	app          *fxtest.App
}

func TestIssueServiceSuite(t *testing.T) {
	suite.Run(t, new(IssueServiceSuite))
}

func (suite *IssueServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			service.NewIssueService,
			mock.Mock[repository.IssueRepositoryInterface],
		),
		fx.Populate(&suite.issueService, &suite.issueRepo),
	)
	suite.app.RequireStart()
}

func (suite *IssueServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *IssueServiceSuite) TestCreateExistingIncrementsRepeatingSuccess() {
	// Arrange
	existing := &dbom.Issue{
		Title:     "test",
		Repeating: 1,
	}
	mock.When(suite.issueRepo.FindByTitle(mock.Any[string]())).ThenReturn(existing, nil)
	mock.When(suite.issueRepo.Update(existing)).ThenReturn(nil)
	// Ensure Create is not called
	mock.When(suite.issueRepo.Create(mock.Any[*dbom.Issue]())).ThenReturn(errors.New("should not be called"))

	// Act
	err := suite.issueService.Create(&dto.Issue{Title: "test"})

	// Assert
	suite.NoError(err)
	mock.Verify(suite.issueRepo, matchers.Times(1)).FindByTitle(mock.Any[string]())
	mock.Verify(suite.issueRepo, matchers.Times(1)).Update(existing)
	mock.Verify(suite.issueRepo, matchers.Times(0)).Create(mock.Any[*dbom.Issue]())
}

func (suite *IssueServiceSuite) TestCreateNewCreatesSuccess() {
	// Arrange
	mock.When(suite.issueRepo.FindByTitle(mock.Any[string]())).ThenReturn(nil, nil)
	mock.When(suite.issueRepo.Create(mock.Any[*dbom.Issue]())).ThenReturn(nil)

	// Act
	err := suite.issueService.Create(&dto.Issue{Title: "new"})

	// Assert
	suite.NoError(err)
	mock.Verify(suite.issueRepo, matchers.Times(1)).FindByTitle(mock.Any[string]())
	mock.Verify(suite.issueRepo, matchers.Times(1)).Create(mock.Any[*dbom.Issue]())
	mock.Verify(suite.issueRepo, matchers.Times(0)).Update(mock.Any[*dbom.Issue]())
}

func (suite *IssueServiceSuite) TestCreateFindByTitleError() {
	// Arrange
	mockErr := errors.New("db find error")
	mock.When(suite.issueRepo.FindByTitle(mock.Any[string]())).ThenReturn(nil, mockErr)

	// Act
	err := suite.issueService.Create(&dto.Issue{Title: "err"})

	// Assert
	suite.Error(err)
	mock.Verify(suite.issueRepo, matchers.Times(1)).FindByTitle(mock.Any[string]())
	mock.Verify(suite.issueRepo, matchers.Times(0)).Create(mock.Any[*dbom.Issue]())
	mock.Verify(suite.issueRepo, matchers.Times(0)).Update(mock.Any[*dbom.Issue]())
}

func (suite *IssueServiceSuite) TestCreateUpdateError() {
	// Arrange
	existing := &dbom.Issue{
		Title:     "uerr",
		Repeating: 2,
	}
	mock.When(suite.issueRepo.FindByTitle(mock.Any[string]())).ThenReturn(existing, nil)
	mockErr := errors.New("update fail")
	mock.When(suite.issueRepo.Update(existing)).ThenReturn(mockErr)

	// Act
	err := suite.issueService.Create(&dto.Issue{Title: "uerr"})

	// Assert
	suite.Error(err)
	mock.Verify(suite.issueRepo, matchers.Times(1)).FindByTitle(mock.Any[string]())
	mock.Verify(suite.issueRepo, matchers.Times(1)).Update(existing)
}

func (suite *IssueServiceSuite) TestCreateCreateError() {
	// Arrange
	mock.When(suite.issueRepo.FindByTitle(mock.Any[string]())).ThenReturn(nil, nil)
	mockErr := errors.New("create fail")
	mock.When(suite.issueRepo.Create(mock.Any[*dbom.Issue]())).ThenReturn(mockErr)

	// Act
	err := suite.issueService.Create(&dto.Issue{Title: "cerr"})

	// Assert
	suite.Error(err)
	mock.Verify(suite.issueRepo, matchers.Times(1)).FindByTitle(mock.Any[string]())
	mock.Verify(suite.issueRepo, matchers.Times(1)).Create(mock.Any[*dbom.Issue]())
}

func (suite *IssueServiceSuite) TestResolveSuccess() {
	// Arrange
	mock.When(suite.issueRepo.Delete(uint(1))).ThenReturn(nil)

	// Act
	err := suite.issueService.Resolve(1)

	// Assert
	suite.NoError(err)
	mock.Verify(suite.issueRepo, matchers.Times(1)).Delete(uint(1))
}

func (suite *IssueServiceSuite) TestResolveError() {
	// Arrange
	mockErr := errors.New("delete failed")
	mock.When(suite.issueRepo.Delete(uint(999))).ThenReturn(mockErr)

	// Act
	err := suite.issueService.Resolve(999)

	// Assert
	suite.Error(err)
	mock.Verify(suite.issueRepo, matchers.Times(1)).Delete(uint(999))
}

func (suite *IssueServiceSuite) TestFindOpenSuccess() {
	// Arrange
	openIssues := []*dbom.Issue{
		{ID: 1, Title: "Issue 1"},
		{ID: 2, Title: "Issue 2"},
	}
	mock.When(suite.issueRepo.FindOpen()).ThenReturn(openIssues, nil)

	// Act
	issues, err := suite.issueService.FindOpen()

	// Assert
	suite.NoError(err)
	suite.NotNil(issues)
	suite.Len(issues, 2)
	mock.Verify(suite.issueRepo, matchers.Times(1)).FindOpen()
}

func (suite *IssueServiceSuite) TestFindOpenError() {
	// Arrange
	mockErr := errors.New("database error")
	mock.When(suite.issueRepo.FindOpen()).ThenReturn(nil, mockErr)

	// Act
	issues, err := suite.issueService.FindOpen()

	// Assert
	suite.Error(err)
	suite.Nil(issues)
	mock.Verify(suite.issueRepo, matchers.Times(1)).FindOpen()
}

func (suite *IssueServiceSuite) TestUpdateSuccess() {
	// Arrange
	issue := &dto.Issue{Title: "Updated", Description: "Updated description"}
	mock.When(suite.issueRepo.Update(mock.Any[*dbom.Issue]())).ThenReturn(nil)

	// Act
	result, err := suite.issueService.Update(issue)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	mock.Verify(suite.issueRepo, matchers.Times(1)).Update(mock.Any[*dbom.Issue]())
}

func (suite *IssueServiceSuite) TestUpdateError() {
	// Arrange
	issue := &dto.Issue{Title: "Updated", Description: "Updated description"}
	mockErr := errors.New("update failed")
	mock.When(suite.issueRepo.Update(mock.Any[*dbom.Issue]())).ThenReturn(mockErr)

	// Act
	result, err := suite.issueService.Update(issue)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	mock.Verify(suite.issueRepo, matchers.Times(1)).Update(mock.Any[*dbom.Issue]())
}
