package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	gomock "go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

type ShareHandlerSuite struct {
	suite.Suite
	mockBoradcaster     *MockBroadcasterServiceInterface
	dirtyService        service.DirtyDataServiceInterface
	exported_share_repo repository.ExportedShareRepositoryInterface
}

func TestShareHandlerSuite(t *testing.T) {
	csuite := new(ShareHandlerSuite)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
	csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()

	csuite.exported_share_repo = exported_share_repo

	csuite.dirtyService = service.NewDirtyDataService(testContext)

	suite.Run(t, csuite)
}

func (suite *ShareHandlerSuite) TestListShares() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	_, api := humatest.New(suite.T())
	shareHandler.RegisterShareHandler(api)
	rr := api.Get("/shares")
	suite.Equal(http.StatusOK, rr.Code, "Body %#v", rr.Body.String())

	resultsDto := []dto.SharedResource{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &resultsDto)
	suite.Require().NoError(jsonError, "Body %#v", rr.Body.String())
	suite.NotEmpty(resultsDto)
	var config config.Config
	config.FromContext(testContext)
	suite.Len(resultsDto, 10)

	for _, sdto := range resultsDto {
		suite.NotEmpty(sdto.MountPointData.Path)
		if sdto.MountPointData.IsInvalid {
			suite.NoDirExists(sdto.MountPointData.Path, "DeviceId %s is Invalid=true but %s exist (%s)", sdto.MountPointData.Source, sdto.MountPointData.Path, *sdto.MountPointData.InvalidError)
		} else {
			suite.DirExists(sdto.MountPointData.Path, "DeviceId %s is Invalid=false but %s doesn't exist", sdto.MountPointData.Source, sdto.MountPointData.Path)
		}
	}

}

func (suite *ShareHandlerSuite) TestGetShareHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	_, api := humatest.New(suite.T())
	shareHandler.RegisterShareHandler(api)
	rr := api.Get("/share/LIBRARY")

	suite.Require().Equal(http.StatusOK, rr.Code)

	resultShare := dto.SharedResource{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &resultShare)
	suite.Require().NoError(jsonError, "Body %#v", rr.Body.String())

	var config config.Config
	config.FromContext(testContext)
	var conv converter.ConfigToDtoConverterImpl
	var expected dto.SharedResource
	conv.ShareToSharedResource(config.Shares["LIBRARY"], &expected, []dto.User{
		{Username: pointer.String("dianlight"), Password: pointer.String("hassio2010"), IsAdmin: pointer.Bool(true)},
		{Username: pointer.String("rouser"), Password: pointer.String("rouser"), IsAdmin: pointer.Bool(false)},
	})
	expected.Name = resultShare.Name // Fix for testing
	expected.MountPointData.ID = resultShare.MountPointData.ID
	expected.MountPointData.IsInvalid = resultShare.MountPointData.IsInvalid
	expected.MountPointData.InvalidError = resultShare.MountPointData.InvalidError
	//assert.Equal(suite.T(), config.Shares["LIBRARY"], resultShare)
	suite.Equal(expected, resultShare, "Body %#v", rr.Body.String())
}

func (suite *ShareHandlerSuite) TestCreateShareHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	_, api := humatest.New(suite.T())
	shareHandler.RegisterShareHandler(api)

	share := dto.SharedResource{
		Name: "PIPPODD",
		MountPointData: &dto.MountPointData{
			Path:   "/pippo",
			FSType: "tmpfs"},
	}
	rr := api.Post("/share", share)
	suite.Equal(http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	var result dto.SharedResource
	jsonError := json.Unmarshal(rr.Body.Bytes(), &result)
	suite.Require().NoError(jsonError)
	share.Name = result.Name
	share.Users = []dto.User{
		{Username: pointer.String("dianlight"), Password: pointer.String("hassio2010"), IsAdmin: pointer.Bool(true)},
	} // Fix for testing
	//share.Usage = "none"
	share.MountPointData.ID = result.MountPointData.ID
	share.MountPointData.IsInvalid = result.MountPointData.IsInvalid
	share.MountPointData.InvalidError = result.MountPointData.InvalidError
	suite.Equal(share, result)
}

func (suite *ShareHandlerSuite) TestCreateShareDuplicateHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	_, api := humatest.New(suite.T())
	shareHandler.RegisterShareHandler(api)

	share := dto.SharedResource{
		Name: "LIBRARY",
		MountPointData: &dto.MountPointData{
			Path:   "/mnt/LIBRARY",
			FSType: "ext4",
		},
		RoUsers: []dto.User{
			{Username: pointer.String("rouser")},
		},
		TimeMachine: pointer.Bool(true),
		Users: []dto.User{
			{Username: pointer.String("dianlight")},
		},
		Usage: "media",
	}

	rr := api.Post("/share", share)

	suite.Equal(http.StatusConflict, rr.Code)

	suite.Contains(rr.Body.String(), "Share already exists")
}

func (suite *ShareHandlerSuite) TestUpdateShareHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	_, api := humatest.New(suite.T())
	shareHandler.RegisterShareHandler(api)

	share := dto.SharedResource{
		MountPointData: &dto.MountPointData{
			Path: "/pippo_efi",
		},
	}

	rr := api.Put("/share/UPDATER", share)

	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var rshare dto.SharedResource
	jsonError := json.Unmarshal(rr.Body.Bytes(), &rshare)
	suite.Require().NoError(jsonError)

	suite.Equal("UPDATER", rshare.Name)
	suite.Equal(share.MountPointData.Path, rshare.MountPointData.Path)
}

func (suite *ShareHandlerSuite) TestUpdateShareHandlerEnableDisableShare() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	_, api := humatest.New(suite.T())
	shareHandler.RegisterShareHandler(api)

	share := dto.SharedResource{
		Disabled: pointer.Bool(true),
	}

	rr := api.Put("/share/UPDATER", share)
	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var rshare dto.SharedResource
	jsonError := json.Unmarshal(rr.Body.Bytes(), &rshare)
	suite.Require().NoError(jsonError)

	suite.Equal("UPDATER", rshare.Name)
	suite.True(*rshare.Disabled)

	share.Disabled = pointer.Bool(false)
	rr = api.Put("/share/UPDATER", share)
	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())
	var rshare2 dto.SharedResource
	jsonError = json.Unmarshal(rr.Body.Bytes(), &rshare2)
	suite.Require().NoError(jsonError)
	suite.Equal("UPDATER", rshare2.Name)
	suite.Nil(rshare2.Disabled)
}

func (suite *ShareHandlerSuite) TestDeleteShareHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	_, api := humatest.New(suite.T())
	shareHandler.RegisterShareHandler(api)
	rr := api.Delete("/share/EFI")
	suite.Equal(http.StatusNoContent, rr.Code)

	// Refresh shares list anche check that LIBRARY don't exists
	share, err := exported_share_repo.FindByName("EFI")
	if suite.Error(err, "Share %+v should not exist", share) {
		suite.Equal(gorm.ErrRecordNotFound, errors.Unwrap(err))
	}
}

func (suite *ShareHandlerSuite) TestUpdateShareNameHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	_, api := humatest.New(suite.T())
	shareHandler.RegisterShareHandler(api)

	// Prepare create a share named OLD_NAME with users
	old_share := dbom.ExportedShare{
		Name: "OLD_NAME",
		Users: []dbom.SambaUser{
			{
				Username: "dianlight_t",
				Password: "hassio2010_t",
				IsAdmin:  true,
			},
		},
		MountPointData: dbom.MountPointPath{
			Path: "/mnt/OLD_NAME",
		},
	}
	err := exported_share_repo.Save(&old_share)
	suite.Require().NoError(err)

	share := dto.SharedResource{
		Name: "NEW_NAME",
	}

	rr := api.Put("/share/OLD_NAME", share)

	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var rshare dto.SharedResource
	jsonError := json.Unmarshal(rr.Body.Bytes(), &rshare)
	suite.Require().NoError(jsonError)

	suite.Equal("NEW_NAME", rshare.Name)

	// Check that old name is not found
	_, err = exported_share_repo.FindByName("OLD_NAME")
	suite.Require().Error(err)
	suite.Equal(gorm.ErrRecordNotFound, errors.Unwrap(err))

	// Check that new name is found
	_, err = exported_share_repo.FindByName("NEW_NAME")
	suite.Require().NoError(err)

	// Return to ooriginal name
	err = exported_share_repo.Delete("NEW_NAME")
	suite.NoError(err)
}
