package service_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/dianlight/srat/dto"
	resolutionapi "github.com/dianlight/srat/homeassistant/resolution"
	"github.com/dianlight/srat/service"
	"github.com/oapi-codegen/runtime/types"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
)

// ResolutionServiceSuite tests the ResolutionService create and delete issue methods.
type ResolutionServiceSuite struct {
	suite.Suite
	ctrl   *matchers.MockController
	state  *dto.ContextState
	client resolutionapi.ClientWithResponsesInterface
	svc    service.ResolutionServiceInterface
}

func TestResolutionServiceSuite(t *testing.T) {
	suite.Run(t, new(ResolutionServiceSuite))
}

func (suite *ResolutionServiceSuite) SetupTest() {
	suite.ctrl = mock.NewMockController(suite.T())
	suite.state = &dto.ContextState{
		SupervisorURL: "http://supervisor/",
		HACoreReady:   true,
	}
	suite.client = mock.Mock[resolutionapi.ClientWithResponsesInterface](suite.ctrl)
	params := service.ResolutionServiceParams{
		ApiContext:       context.Background(),
		ApiContextCancel: func() {},
		ResolutionClient: suite.client,
		State:            suite.state,
	}
	suite.svc = service.NewResolutionService(params)
}

func (suite *ResolutionServiceSuite) TearDownTest() {
	// no-op
}

func (suite *ResolutionServiceSuite) TestCreateIssue_NoOpDemoMode() {
	suite.state.SupervisorURL = "demo"
	err := suite.svc.CreateIssue(dto.ResolutionIssue{})
	suite.NoError(err)
	// Should not call DismissIssue on demo mode
	mock.Verify(suite.client, matchers.Times(0)).DismissIssueWithResponse(mock.AnyContext(), mock.Any[types.UUID]())
}

func (suite *ResolutionServiceSuite) TestCreateIssue_NoOpHACoreNotReady() {
	suite.state.SupervisorURL = "http://supervisor/"
	suite.state.HACoreReady = false
	err := suite.svc.CreateIssue(dto.ResolutionIssue{})
	suite.NoError(err)
	// Should not call DismissIssue when HA core not ready
	mock.Verify(suite.client, matchers.Times(0)).DismissIssueWithResponse(mock.AnyContext(), mock.Any[types.UUID]())
}

func (suite *ResolutionServiceSuite) TestDeleteIssue_NoOpDemoMode() {
	suite.state.SupervisorURL = "demo"
	err := suite.svc.DeleteIssue(types.UUID{})
	suite.NoError(err)
	mock.Verify(suite.client, matchers.Times(0)).DismissIssueWithResponse(mock.AnyContext(), mock.Any[types.UUID]())
}

func (suite *ResolutionServiceSuite) TestDeleteIssue_NoOpHACoreNotReady() {
	suite.state.SupervisorURL = "http://supervisor/"
	suite.state.HACoreReady = false
	err := suite.svc.DeleteIssue(types.UUID{})
	suite.NoError(err)
	mock.Verify(suite.client, matchers.Times(0)).DismissIssueWithResponse(mock.AnyContext(), mock.Any[types.UUID]())
}

func (suite *ResolutionServiceSuite) TestDeleteIssue_ClientError() {
	// Arrange
	uuid := types.UUID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	mock.When(suite.client.DismissIssueWithResponse(mock.AnyContext(), mock.Any[types.UUID]())).ThenReturn(nil, errors.New("client fail"))
	// Act
	err := suite.svc.DeleteIssue(uuid)
	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "client fail")
}

func (suite *ResolutionServiceSuite) TestDeleteIssue_Non200Status() {
	// Arrange
	resp := &resolutionapi.DismissIssueResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
	}
	mock.When(suite.client.DismissIssueWithResponse(mock.AnyContext(), mock.Any[types.UUID]())).ThenReturn(resp, nil)
	// Act
	err := suite.svc.DeleteIssue(types.UUID{})
	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "Error deleting issue: 500")
}

func (suite *ResolutionServiceSuite) TestDeleteIssue_Success() {
	// Arrange
	resp := &resolutionapi.DismissIssueResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
	}
	mock.When(suite.client.DismissIssueWithResponse(mock.AnyContext(), mock.Any[types.UUID]())).ThenReturn(resp, nil)
	// Act
	err := suite.svc.DeleteIssue(types.UUID{})
	// Assert
	suite.NoError(err)
}
