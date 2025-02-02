package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	api := api.NewSystemHanler()
	req, err := http.NewRequestWithContext(testContext, "GET", "/nics", nil)
	require.NoError(suite.T(), err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.GetNICsHandler)

	handler.ServeHTTP(rr, req)

	assert.Equal(suite.T(), rr.Code, http.StatusOK, "Expected status code 200, got %d", rr.Code)

	expectedContentType := "application/json"
	assert.Equal(suite.T(), rr.Header().Get("Content-Type"), expectedContentType, "Expected content type %s, got %s", expectedContentType, rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	suite.T().Logf("%v", response)

	if nics, ok := response["nics"].([]interface{}); !ok || len(nics) == 0 {
		suite.T().Errorf("Response does not contain any network interfaces")
	}
}

func (suite *SystemHandlerSuite) TestGetFSHandler() {
	api := api.NewSystemHanler()
	req, err := http.NewRequestWithContext(testContext, "GET", "/filesystems", nil)
	require.NoError(suite.T(), err)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.GetFSHandler)

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(suite.T(), rr.Code, http.StatusOK, "Expected status code 200, got %d", rr.Code)

	// Check the content type
	expectedContentType := "application/json"
	assert.Equal(suite.T(), rr.Header().Get("Content-Type"), expectedContentType, "Expected content type %s, got %s", expectedContentType, rr.Header().Get("Content-Type"))

	// Check the response body
	var fileSystems []string
	err = json.Unmarshal(rr.Body.Bytes(), &fileSystems)
	require.NoError(suite.T(), err)
	suite.T().Logf("%v", fileSystems)
	assert.NotEmpty(suite.T(), fileSystems, "Response does not contain any file systems")
}
