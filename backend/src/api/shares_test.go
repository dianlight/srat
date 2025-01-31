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
	"github.com/xorcare/pointer"
	gomock "go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

type ShareHandlerSuite struct {
	suite.Suite
	mockBrocker *MockBrokerInterface
	// VariableThatShouldStartAtFive int
}

func TestShareHandlerSuite(t *testing.T) {
	csuite := new(ShareHandlerSuite)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sharedContex := api.StateFromContext(testContext)
	csuite.mockBrocker = NewMockBrokerInterface(ctrl)
	sharedContex.SSEBroker = csuite.mockBrocker
	csuite.mockBrocker.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
	csuite.mockBrocker.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	suite.Run(t, csuite)
}

func (suite *ShareHandlerSuite) TestListShares() {
	shareHandler := api.NewShareHandler(testContext)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/shares", nil)
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(shareHandler.ListShares)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusOK, rr.Code, "Body %#v", rr.Body.String())

	// Check the response body is what we expect.
	resultsDto := []dto.SharedResource{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &resultsDto)
	require.NoError(suite.T(), jsonError, "Body %#v", rr.Body.String())
	assert.NotEmpty(suite.T(), resultsDto)
	var config config.Config
	config.FromContext(testContext)
	assert.Len(suite.T(), resultsDto, 10)

	for _, sdto := range resultsDto {
		assert.NotEmpty(suite.T(), sdto.MountPointData.Path)
		if sdto.MountPointData.IsInvalid {
			assert.NoDirExists(suite.T(), sdto.MountPointData.Path, "DeviceId %s is Invalid=true but %s exist (%s)", sdto.MountPointData.Source, sdto.MountPointData.Path, *sdto.MountPointData.InvalidError)
		} else {
			assert.DirExists(suite.T(), sdto.MountPointData.Path, "DeviceId %s is Invalid=false but %s doesn't exist", sdto.MountPointData.Source, sdto.MountPointData.Path)
		}
	}

}

func (suite *ShareHandlerSuite) TestGetShareHandler() {
	shareHandler := api.NewShareHandler(testContext)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/share/LIBRARY", nil)
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", shareHandler.GetShare).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	require.Equal(suite.T(), http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	//context_state := (&dto.ContextState{}).FromContext(testContext)
	resultShare := dto.SharedResource{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &resultShare)
	require.NoError(suite.T(), jsonError, "Body %#v", rr.Body.String())

	var config config.Config
	config.FromContext(testContext)
	var conv converter.ConfigToDtoConverterImpl
	var expected dto.SharedResource
	conv.ShareToSharedResource(config.Shares["LIBRARY"], &expected, []dto.User{
		{Username: pointer.String("dianlight"), Password: pointer.String("hassio2010"), IsAdmin: pointer.Bool(true)},
		{Username: pointer.String("rouser"), Password: pointer.String("rouser"), IsAdmin: pointer.Bool(false)},
	})
	expected.ID = resultShare.ID // Fix for testing
	expected.MountPointData.ID = resultShare.MountPointData.ID
	expected.MountPointData.IsInvalid = resultShare.MountPointData.IsInvalid
	expected.MountPointData.InvalidError = resultShare.MountPointData.InvalidError
	//assert.Equal(suite.T(), config.Shares["LIBRARY"], resultShare)
	assert.EqualValues(suite.T(), expected, resultShare, "Body %#v", rr.Body.String())
}

func (suite *ShareHandlerSuite) TestCreateShareHandler() {
	shareHandler := api.NewShareHandler(testContext)

	share := dto.SharedResource{
		Name: "PIPPODD",
		MountPointData: &dto.MountPointData{
			Path:   "/pippo",
			FSType: "tmpfs"},
	}

	jsonBody, jsonError := json.Marshal(share)
	require.NoError(suite.T(), jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/share", strings.NewReader(string(jsonBody)))
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share", shareHandler.CreateShare).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	var result dto.SharedResource
	jsonError = json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(suite.T(), jsonError)
	share.ID = result.ID
	share.Users = []dto.User{
		{Username: pointer.String("dianlight"), Password: pointer.String("hassio2010"), IsAdmin: pointer.Bool(true)},
	} // Fix for testing
	//share.Usage = "none"
	share.MountPointData.ID = result.MountPointData.ID
	share.MountPointData.IsInvalid = result.MountPointData.IsInvalid
	share.MountPointData.InvalidError = result.MountPointData.InvalidError
	assert.EqualValues(suite.T(), share, result)
}

func (suite *ShareHandlerSuite) TestCreateShareDuplicateHandler() {
	shareHandler := api.NewShareHandler(testContext)

	share := dto.SharedResource{
		Name: "LIBRARY",
		MountPointData: &dto.MountPointData{
			Path:   "/mnt/LIBRARY",
			FSType: "ext4",
		},
		RoUsers: []dto.User{
			{Username: pointer.String("rouser")},
		},
		TimeMachine: true,
		Users: []dto.User{
			{Username: pointer.String("dianlight")},
		},
		Usage: "media",
	}

	jsonBody, jsonError := json.Marshal(share)
	require.NoError(suite.T(), jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/share/LIBRARY", strings.NewReader(string(jsonBody)))
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", shareHandler.CreateShare).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusConflict, rr.Code)

	// Check the response body is what we expect.
	assert.Contains(suite.T(), rr.Body.String(), "Share already exists")
}

func (suite *ShareHandlerSuite) TestUpdateShareHandler() {
	shareHandler := api.NewShareHandler(testContext)

	share := dto.SharedResource{
		MountPointData: &dto.MountPointData{
			Path: "/pippo_efi",
		},
	}

	jsonBody, jsonError := json.Marshal(share)
	require.NoError(suite.T(), jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/share/UPDATER", strings.NewReader(string(jsonBody)))
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", shareHandler.UpdateShare).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var rshare dto.SharedResource
	jsonError = json.Unmarshal(rr.Body.Bytes(), &rshare)
	require.NoError(suite.T(), jsonError)

	assert.EqualValues(suite.T(), share, rshare)

	/*
	   // Check the response body is what we expect.
	   share.FS = "ext4"
	   share.Name = "LIBRARY"

	   	share.RoUsers = []dto.User{
	   		{Username: pointer.String("rouser")},
	   	}

	   share.TimeMachine = true

	   	share.Users = []dto.User{
	   		{Username: pointer.String("dianlight")},
	   	}

	   share.Usage = dto.UsageAsMedia
	   expected, jsonError := json.Marshal(share)
	   require.NoError(suite.T(), jsonError)
	   assert.Equal(suite.T(), string(expected)[:len(expected)-3], rr.Body.String()[:len(expected)-3])
	*/
}

func (suite *ShareHandlerSuite) TestDeleteShareHandler() {
	shareHandler := api.NewShareHandler(testContext)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/share/EFI", nil)
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", shareHandler.DeleteShare).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusNoContent, rr.Code)

	// Refresh shares list anche check that LIBRARY don't exists
	share := dbom.ExportedShare{
		Name: "EFI",
	}
	err = share.FromName("EFI")
	if assert.Error(suite.T(), err, "Share %+v should not exist", share) {
		assert.Equal(suite.T(), gorm.ErrRecordNotFound, err)
	}
}
