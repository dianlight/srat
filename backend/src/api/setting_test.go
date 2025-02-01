package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SettingsHandlerSuite struct {
	suite.Suite
	//mockBoradcaster *MockBroadcasterServiceInterface
	// VariableThatShouldStartAtFive int
}

func TestSettingsHandlerSuite(t *testing.T) {
	csuite := new(ShareHandlerSuite)
	suite.Run(t, csuite)
}
func (suite *SettingsHandlerSuite) TestGetSettingsHandler() {
	api := api.NewSettingsHanler()
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/global", nil)
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.GetSettings)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	require.Equal(suite.T(), http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var config config.Config
	err = config.FromContext(testContext)
	require.NoError(suite.T(), err)
	var expected dto.Settings
	var conv converter.ConfigToDtoConverterImpl
	err = conv.ConfigToSettings(config, &expected)
	//err = mapper.Map(context.Background(), &expected, config)
	require.NoError(suite.T(), err)

	// Check the response body is what we expect.
	var returned dto.Settings
	jsonError := json.Unmarshal(rr.Body.Bytes(), &returned)
	require.NoError(suite.T(), jsonError)

	assert.Equal(suite.T(), expected, returned)

	assert.False(suite.T(), apiContextState.DataDirtyTracker.Settings)
}

func (suite *SettingsHandlerSuite) TestUpdateSettingsHandler() {
	api := api.NewSettingsHanler()
	glc := dto.Settings{
		Workgroup: "pluto&admin",
	}

	jsonBody, jsonError := json.Marshal(glc)
	require.NoError(suite.T(), jsonError)
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/global", strings.NewReader(string(jsonBody)))
	require.NoError(suite.T(), err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/global", api.UpdateSettings).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.Settings
	err = json.Unmarshal(rr.Body.Bytes(), &res)
	require.NoError(suite.T(), err, "Body %#v", rr.Body.String())

	assert.Equal(suite.T(), glc.Workgroup, res.Workgroup)
	assert.EqualValues(suite.T(), []string{"10.0.0.0/8", "100.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "fe80::/10", "fc00::/7"}, res.AllowHost)
	assert.True(suite.T(), apiContextState.DataDirtyTracker.Settings)

	// Restore original state
	var properties dbom.Properties
	if err := properties.Load(); err != nil {
		suite.T().Fatalf("Failed to load properties: %v", err)
	}
	if err := properties.SetValue("Workgroup", "WORKGROUP"); err != nil {
		suite.T().Fatalf("Failed to add workgroup property: %v", err)
	}
}
