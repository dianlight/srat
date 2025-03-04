// endpoints_test.go
package api_test

import (
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/go-fuego/fuego"
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
	samba := api.NewSambaHanler(&apiContextState, suite.mockSambaService)
	ctx := fuego.NewMockContextNoBody()

	ok, err := samba.ApplySamba(ctx)
	suite.Require().NoError(err)
	suite.True(ok)
}

/*
func (suite *SambaHandlerSuite) checkStringInSMBConfig(testvalue string, expected string, t *testing.T) bool {
	stream, err := suite.CreateConfigStream(testContext)
	require.NoError(t, err)
	assert.NotNil(t, stream)

	rexpt := fmt.Sprintf(expected, testvalue)

	m, err := regexp.MatchString(rexpt, string(*stream))
	require.NoError(t, err)
	assert.True(t, m, "Wrong Match `%s` not found in stream \n%s", rexpt, string(*stream))

	return true
}
*/

// check migrate config don't duplicate share

/*
func TestGetSambaProcessStatus(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "GET", "/samba/status", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/samba/status", GetSambaProcessStatus).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, rr.Code, "Expected status code 200 or 404, got %d with Body %s", rr.Code, rr.Body.String())
}
*/

func (suite *SambaHandlerSuite) TestGetSambaConfig() {
	samba := api.NewSambaHanler(&apiContextState, suite.mockSambaService)
	ctx := fuego.NewMockContextNoBody()

	config, err := samba.GetSambaConfig(ctx)
	suite.Require().NoError(err)
	suite.NotNil(config)
	suite.Equal("Test", config.Data)
}
