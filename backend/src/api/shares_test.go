package api_test

import (
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"github.com/ztrue/tracerr"
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
	csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
	csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()

	csuite.exported_share_repo = exported_share_repo

	csuite.dirtyService = service.NewDirtyDataService(testContext)

	suite.Run(t, csuite)
}

func (suite *ShareHandlerSuite) TestListShares() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	ctx := fuego.NewMockContextNoBody()
	resultsDto, err := shareHandler.ListShares(ctx)
	suite.Require().NoError(err)
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
	ctx := fuego.NewMockContextNoBody()
	ctx.PathParams = map[string]string{
		"share_name": "LIBRARY",
	}
	resultShare, err := shareHandler.GetShare(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(resultShare)

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
	suite.EqualValues(expected, resultShare)
}

func (suite *ShareHandlerSuite) TestCreateShareHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)

	share := dto.SharedResource{
		Name: "PIPPODD",
		MountPointData: &dto.MountPointData{
			Path:   "/pippo",
			FSType: "tmpfs"},
	}

	ctx := fuego.NewMockContext(share)

	result, err := shareHandler.CreateShare(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(result)

	share.Name = result.Name
	share.Users = []dto.User{
		{Username: pointer.String("dianlight"), Password: pointer.String("hassio2010"), IsAdmin: pointer.Bool(true)},
	} // Fix for testing
	//share.Usage = "none"
	share.MountPointData.ID = result.MountPointData.ID
	share.MountPointData.IsInvalid = result.MountPointData.IsInvalid
	share.MountPointData.InvalidError = result.MountPointData.InvalidError
	suite.EqualValues(share, *result)
}

func (suite *ShareHandlerSuite) TestCreateShareDuplicateHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)

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
	ctx := fuego.NewMockContext(share)

	result, err := shareHandler.CreateShare(ctx)
	suite.Require().Error(err)
	suite.Nil(result)

	// Check the response body is what we expect.
	suite.ErrorAs(err, &fuego.ConflictError{})
}

func (suite *ShareHandlerSuite) TestUpdateShareHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)

	share := dto.SharedResource{
		MountPointData: &dto.MountPointData{
			Path: "/pippo_efi",
		},
	}

	ctx := fuego.NewMockContext(share)

	ctx.PathParams = map[string]string{
		"share_name": "UPDATER",
	}
	rshare, err := shareHandler.UpdateShare(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(rshare)

	suite.EqualValues("UPDATER", rshare.Name)
	suite.EqualValues(share.MountPointData.Path, rshare.MountPointData.Path)
}

func (suite *ShareHandlerSuite) TestUpdateShareHandlerEnableDisableShare() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)

	share := dto.SharedResource{
		Disabled: pointer.Bool(true),
	}
	ctx := fuego.NewMockContext(share)
	ctx.PathParams = map[string]string{
		"share_name": "UPDATER",
	}
	rshare, err := shareHandler.UpdateShare(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(rshare)

	suite.EqualValues("UPDATER", rshare.Name)
	suite.Assert().True(*rshare.Disabled)

	share.Disabled = pointer.Bool(false)
	ctx = fuego.NewMockContext(share)
	ctx.PathParams = map[string]string{
		"share_name": "UPDATER",
	}

	rshare, err = shareHandler.UpdateShare(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(rshare)
	suite.EqualValues("UPDATER", rshare.Name)
	suite.Assert().False(*rshare.Disabled)
}

func (suite *ShareHandlerSuite) TestDeleteShareHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)
	ctx := fuego.NewMockContextNoBody()
	ctx.PathParams = map[string]string{
		"share_name": "EFI",
	}
	result, err := shareHandler.DeleteShare(ctx)
	suite.Require().NoError(err)
	suite.True(result)

	// Refresh shares list anche check that LIBRARY don't exists
	share, err := exported_share_repo.FindByName("EFI")
	if suite.Error(err, "Share %+v should not exist", share) {
		suite.Equal(gorm.ErrRecordNotFound, tracerr.Unwrap(err))
	}
}

func (suite *ShareHandlerSuite) TestUpdateShareNameHandler() {
	shareHandler := api.NewShareHandler(suite.mockBoradcaster, &apiContextState, suite.dirtyService, suite.exported_share_repo)

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

	ctx := fuego.NewMockContext(share)
	ctx.PathParams = map[string]string{
		"share_name": "OLD_NAME",
	}

	rshare, err := shareHandler.UpdateShare(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(rshare)

	suite.EqualValues("NEW_NAME", rshare.Name)

	// Check that old name is not found
	_, err = exported_share_repo.FindByName("OLD_NAME")
	suite.Require().Error(err)
	suite.Equal(gorm.ErrRecordNotFound, tracerr.Unwrap(err))

	// Check that new name is found
	_, err = exported_share_repo.FindByName("NEW_NAME")
	suite.Require().NoError(err)

	// Return to ooriginal name
	err = exported_share_repo.Delete("NEW_NAME")
	suite.NoError(err)
}
