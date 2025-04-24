// endpoints_test.go
package api_test

/*
import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/suite"
	gomock "go.uber.org/mock/gomock"
)

type SambaHandlerSuite struct {
	suite.Suite
	mockSambaService *MockSambaServiceInterface
	// VariableThatShouldStartAtFive int
}

func TestSambaHandlerSuite(t *testing.T) {
	csuite := new(SambaHandlerSuite)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	csuite.mockSambaService = NewMockSambaServiceInterface(ctrl)
	body := []byte("Test")
	csuite.mockSambaService.EXPECT().CreateConfigStream().AnyTimes().Return(&body, nil)
	csuite.mockSambaService.EXPECT().WriteSambaConfig().AnyTimes()
	csuite.mockSambaService.EXPECT().TestSambaConfig().AnyTimes()
	csuite.mockSambaService.EXPECT().RestartSambaService().AnyTimes()
	//csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()

	suite.Run(t, csuite)
}

func (suite *SambaHandlerSuite) TestApplySambaHandler() {
	sambaHanlder := api.NewSambaHanler(&apiContextState, suite.mockSambaService)
	_, api := humatest.New(suite.T())
	sambaHanlder.RegisterSambaHandler(api)

	rr := api.Put("/samba/apply")
	suite.Equal(http.StatusNoContent, rr.Code, "Expected status code 204, got %d with Body %#v", rr.Code, rr.Body.String())
}

func (suite *SambaHandlerSuite) TestGetSambaConfig() {
	sambaHanlder := api.NewSambaHanler(&apiContextState, suite.mockSambaService)
	_, api := humatest.New(suite.T())
	sambaHanlder.RegisterSambaHandler(api)

	rr := api.Get("/samba/config")
	suite.Equal(http.StatusOK, rr.Code, "Expected status code 200, got %d with Body %#v", rr.Code, rr.Body.String())

	// Check the response body is what we expect.
	var responseBody dto.SmbConf
	err := json.Unmarshal(rr.Body.Bytes(), &responseBody)
	suite.Require().NoError(err)

	// Compare the response body with the expected SmbConf
	suite.Equal("Test", responseBody.Data)
}
*/
