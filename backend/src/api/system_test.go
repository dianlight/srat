package api_test

import (
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/go-fuego/fuego"
	"github.com/stretchr/testify/suite"
)

type SystemHandlerSuite struct {
	suite.Suite
	mockBoradcaster *MockBroadcasterServiceInterface
	// VariableThatShouldStartAtFive int
}

func TestSystemHandlerSuite(t *testing.T) {
	csuite := new(SystemHandlerSuite)
	/*
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
		csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
		csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	*/
	suite.Run(t, csuite)
}

func (suite *SystemHandlerSuite) TestGetNICsHandler() {
	ctx := fuego.NewMockContextNoBody()
	response, err := api.NewSystemHanler().GetNICsHandler(ctx)
	suite.Require().NoError(err)
	suite.T().Logf("%v", response)

	if len(response.NICs) == 0 {
		suite.T().Errorf("Response does not contain any network interfaces")
	}
}

func (suite *SystemHandlerSuite) TestGetFSHandler() {
	ctx := fuego.NewMockContextNoBody()
	response, err := api.NewSystemHanler().GetFSHandler(ctx)
	suite.Require().NoError(err)
	suite.T().Logf("%v", response)

	if len(response) == 0 {
		suite.T().Errorf("Response does not contain any file systems")
	}
}
