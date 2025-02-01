package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
	gomock "go.uber.org/mock/gomock"
)

type VolumeHandlerSuite struct {
	suite.Suite
	mockBoradcaster *MockBroadcasterServiceInterface
	previus_device  uint
}

func TestVolumeHandlerSuite(t *testing.T) {
	csuite := new(VolumeHandlerSuite)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
	csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
	csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	suite.Run(t, csuite)
}

func (suite *VolumeHandlerSuite) TestListVolumessHandler() {
	volume := api.NewVolumeHandler(testContext, suite.mockBoradcaster)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/volumes", nil)
	if err != nil {
		suite.T().Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(volume.ListVolumes)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		suite.T().Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	//suite.T().Log(pretty.Sprint(rr.Body))
	if len(rr.Body.String()) == 0 {
		suite.T().Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), "[]")
	}

	var volumes dto.BlockInfo
	err2 := json.NewDecoder(rr.Body).Decode(&volumes)
	if err2 != nil {
		suite.T().Errorf("handler error in decode body %v", err2)
	}
	//suite.T(),Log(pretty.Sprint(volumes))

	assert.NotNil(suite.T(), funk.Find(volumes.Partitions, func(d *dto.BlockPartition) bool {
		return d.Label == "testvolume"
	}))
	assert.NotNil(suite.T(), funk.Find(volumes.Partitions, func(d *dto.BlockPartition) bool {
		return d.Label == "_EXT4"
	}), "Expected _EXT4 volume not found %+v", funk.Map(volumes.Partitions, func(d *dto.BlockPartition) string {
		return d.Label + "[" + d.Name + "]"
	}))

}

//var

func (suite *VolumeHandlerSuite) TestMountVolumeHandler() {
	volume := api.NewVolumeHandler(testContext, suite.mockBoradcaster)
	// Check if loop device is available for mounting
	volumes, err := volume.GetVolumesData()
	require.NoError(suite.T(), err)

	var mockMountData dbom.MountPointPath

	for _, d := range volumes.Partitions {
		if strings.HasPrefix(d.Name, "loop") && d.Label == "_EXT4" {
			mockMountData.Source = "/dev/" + d.Name
			mockMountData.Path = filepath.Join("/mnt", d.Label)
			mockMountData.FSType = d.Type
			mockMountData.Flags = []dto.MounDataFlag{dto.MS_NOATIME}
			suite.previus_device = d.MountPointData.ID
			suite.T().Logf("Selected loop device: %v", mockMountData)
			break
		}
	}
	if mockMountData.Source == "" {
		suite.T().Skip("Test failed: loop device not found for mounting")
		return
	}

	body, _ := json.Marshal(mockMountData)
	requestPath := fmt.Sprintf("/volume/%d/mount", suite.previus_device)
	suite.T().Logf("Request path: %s", requestPath)
	req, err := http.NewRequestWithContext(testContext, "POST", requestPath, bytes.NewBuffer(body))
	require.NoError(suite.T(), err)

	// Set up gorilla/mux router
	router := mux.NewRouter()
	router.HandleFunc("/volume/{id}/mount", volume.MountVolume).Methods("POST")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusOK, rr.Code, "Body %#v", rr.Body.String())

	// Check the response body is what we expecsuite.T().
	var responseData dbom.MountPointPath
	err = json.Unmarshal(rr.Body.Bytes(), &responseData)
	require.NoError(suite.T(), err)

	// Verify the response data
	if !strings.HasPrefix(responseData.Path, mockMountData.Path) {
		suite.T().Errorf("Unexpected path in response: got %v want %v", responseData.Path, mockMountData.Path)
	}
	if responseData.FSType != mockMountData.FSType {
		suite.T().Errorf("Unexpected FSType in response: got %v want %v", responseData.FSType, mockMountData.FSType)
	}
	if !reflect.DeepEqual(responseData.Flags, mockMountData.Flags) {
		suite.T().Errorf("Unexpected Flags in response: got %v want %v", responseData.Flags, mockMountData.Flags)
	}
}

func (suite *VolumeHandlerSuite) TestUmountVolumeNonExistent() {
	volume := api.NewVolumeHandler(testContext, suite.mockBoradcaster)
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/volume/999999/mount", nil)
	if err != nil {
		suite.T().Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/volume/{id}/mount", volume.UmountVolume).Methods("DELETE")

	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		suite.T().Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	expected := `{"code":404,"error":"MountPoint not found","body":null}`
	if rr.Body.String() != expected {
		suite.T().Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
func (suite *VolumeHandlerSuite) TestUmountVolumeSuccess() {

	volume := api.NewVolumeHandler(testContext, suite.mockBoradcaster)
	if suite.previus_device == 0 {
		suite.T().Skip("Test skip: not prevision mounted volume found")
	}

	require.NotEmpty(suite.T(), suite.previus_device, "Test skip: not prevision mounted volume found")

	// Create a request
	req, err := http.NewRequestWithContext(testContext, "DELETE", fmt.Sprintf("/volume/%d/mount", suite.previus_device), nil)
	require.NoError(suite.T(), err)

	// Set up gorilla/mux router
	router := mux.NewRouter()
	router.HandleFunc("/volume/{id}/mount", volume.UmountVolume).Methods("DELETE")

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(suite.T(), http.StatusNoContent, rr.Code, "Body %#v", rr.Body.String())

	// Check that the response body is empty
	assert.Empty(suite.T(), rr.Body.String(), "Body should be empty")
}
