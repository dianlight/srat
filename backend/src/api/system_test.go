package api_test

/*
import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
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
	* /
	suite.Run(t, csuite)
}

func (suite *SystemHandlerSuite) TestGetNICsHandler() {
	systemHanlder := api.NewSystemHanler()
	_, api := humatest.New(suite.T())
	systemHanlder.RegisterSystemHanler(api)

	rr := api.Get("/nics")

	suite.Equal(http.StatusOK, rr.Code, "Expected status code 200, got %d", rr.Code)

	expectedContentType := "application/json"
	suite.Equal(expectedContentType, rr.Header().Get("Content-Type"), "Expected content type %s, got %s", expectedContentType, rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.T().Logf("%v", response)

	if nics, ok := response["nics"].([]interface{}); !ok || len(nics) == 0 {
		suite.T().Errorf("Response does not contain any network interfaces")
	}
}

func (suite *SystemHandlerSuite) TestGetFSHandler() {
	systemHanlder := api.NewSystemHanler()
	_, api := humatest.New(suite.T())
	systemHanlder.RegisterSystemHanler(api)

	rr := api.Get("/filesystems")

	// Check the status code
	suite.Equal(http.StatusOK, rr.Code, "Expected status code 200, got %d", rr.Code)

	// Check the content type
	expectedContentType := "application/json"
	suite.Equal(expectedContentType, rr.Header().Get("Content-Type"), "Expected content type %s, got %s", expectedContentType, rr.Header().Get("Content-Type"))

	// Check the response body
	var fileSystems []string
	err := json.Unmarshal(rr.Body.Bytes(), &fileSystems)
	suite.Require().NoError(err)
	suite.T().Logf("%v", fileSystems)
	suite.NotEmpty(fileSystems, "Response does not contain any file systems")
}
*/
