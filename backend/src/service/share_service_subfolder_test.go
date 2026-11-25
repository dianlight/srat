package service_test

import (
	"context"
	"path/filepath"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"gitlab.com/tozd/go/errors"
)

// subfolderVolume returns a mounted test volume backed by a temp directory
// (issue #184) and stubs the admin user every share needs.
func (suite *ShareServiceSuite) subfolderVolume(deviceID string) *dto.MountPointData {
	service.MockIsMountPoint(func(string) (bool, error) { return true, nil })
	suite.T().Cleanup(func() { service.MockIsMountPoint(osutil.IsMounted) })
	mock.When(suite.userService.GetAdmin()).ThenReturn(&dto.User{Username: "homeassistant"}, nil)
	return &dto.MountPointData{
		IsMounted: true,
		Path:      suite.T().TempDir(),
		DeviceId:  deviceID,
		Type:      "ADDON",
	}
}

func subfolderShare(name, subfolder string, volume *dto.MountPointData) dto.SharedResource {
	return dto.SharedResource{
		Name:           name,
		Disabled:       new(false),
		Usage:          dto.UsageAsShare,
		Subfolder:      &subfolder,
		MountPointData: volume,
		Users:          []dto.User{{Username: "homeassistant"}},
	}
}

func (suite *ShareServiceSuite) TestCreateShareWithSubfolderCreatesFolder() {
	volume := suite.subfolderVolume("sub-create")

	created, err := suite.shareService.CreateShare(subfolderShare("sub-create", " photos/2024/ ", volume))

	suite.Require().NoError(err)
	suite.Require().NotNil(created.Subfolder)
	suite.Equal("photos/2024", *created.Subfolder, "subfolder is stored cleaned")
	suite.DirExists(filepath.Join(volume.Path, "photos", "2024"))

	stored, err := suite.shareService.GetShare("sub-create")
	suite.Require().NoError(err)
	suite.Equal("photos/2024", stored.SubfolderOrRoot())
}

// TestCreateSharesOnSubfoldersOfOneVolume is the per-person layout of #184:
// one volume, one share per folder, each with its own users.
func (suite *ShareServiceSuite) TestCreateSharesOnSubfoldersOfOneVolume() {
	volume := suite.subfolderVolume("sub-family")

	for _, share := range []dto.SharedResource{
		subfolderShare("family-root", "", volume),
		subfolderShare("family-emperor", "emperor", volume),
		subfolderShare("family-pavel", "pavel", volume),
	} {
		_, err := suite.shareService.CreateShare(share)
		suite.Require().NoError(err, share.Name)
	}

	_, err := suite.shareService.CreateShare(subfolderShare("family-emperor-2", "emperor", volume))
	suite.True(errors.Is(err, dto.ErrorShareFolderInUse), "same folder twice, got %v", err)

	_, err = suite.shareService.CreateShare(subfolderShare("family-root-2", "", volume))
	suite.True(errors.Is(err, dto.ErrorShareFolderInUse), "volume root twice, got %v", err)

	shares, err := suite.shareService.ListShares()
	suite.Require().NoError(err)
	names := make([]string, 0, len(shares))
	for _, share := range shares {
		names = append(names, share.Name)
	}
	suite.Subset(names, []string{"family-root", "family-emperor", "family-pavel"})
	suite.NotContains(names, "family-emperor-2")
	suite.NotContains(names, "family-root-2")
}

func (suite *ShareServiceSuite) TestCreateShareRejectsInvalidSubfolder() {
	volume := suite.subfolderVolume("sub-invalid")

	for _, subfolder := range []string{"../etc", "/etc", "photos//2024", `photos\2024`} {
		created, err := suite.shareService.CreateShare(subfolderShare("sub-invalid", subfolder, volume))
		suite.Nil(created, subfolder)
		suite.True(errors.Is(err, dto.ErrorShareValidation), "%q: expected ErrorShareValidation, got %v", subfolder, err)
	}
	_, err := suite.shareService.GetShare("sub-invalid")
	suite.True(errors.Is(err, dto.ErrorShareNotFound), "a refused share is never stored")
}

func (suite *ShareServiceSuite) TestCreateShareSubfolderOnUnmountedVolume() {
	volume := suite.subfolderVolume("sub-unmounted")
	service.MockIsMountPoint(func(string) (bool, error) { return false, nil })

	_, err := suite.shareService.CreateShare(subfolderShare("sub-unmounted", "emperor", volume))

	suite.True(errors.Is(err, dto.ErrorShareValidation), "got %v", err)
	suite.NoDirExists(filepath.Join(volume.Path, "emperor"))
}

func (suite *ShareServiceSuite) TestCreateInternalShareWithSubfolderRejected() {
	volume := suite.subfolderVolume("sub-internal")
	share := subfolderShare("sub-internal", "emperor", volume)
	share.Usage = dto.UsageAsInternal

	_, err := suite.shareService.CreateShare(share)

	suite.True(errors.Is(err, dto.ErrorShareValidation), "got %v", err)
}

func (suite *ShareServiceSuite) TestUpdateShareSubfolder() {
	volume := suite.subfolderVolume("sub-update")
	_, err := suite.shareService.CreateShare(subfolderShare("sub-update", "emperor", volume))
	suite.Require().NoError(err)

	// Omitting the subfolder keeps it: a partial update never widens a share.
	keep := subfolderShare("sub-update", "", volume)
	keep.Subfolder = nil
	keep.GuestOk = new(true)
	updated, err := suite.shareService.UpdateShare("sub-update", keep)
	suite.Require().NoError(err)
	suite.Equal("emperor", updated.SubfolderOrRoot())
	suite.True(*updated.GuestOk)

	// A new subfolder is created and stored.
	updated, err = suite.shareService.UpdateShare("sub-update", subfolderShare("sub-update", "pavel", volume))
	suite.Require().NoError(err)
	suite.Equal("pavel", updated.SubfolderOrRoot())
	suite.DirExists(filepath.Join(volume.Path, "pavel"))

	// An explicit empty subfolder moves the share to the volume root.
	updated, err = suite.shareService.UpdateShare("sub-update", subfolderShare("sub-update", "", volume))
	suite.Require().NoError(err)
	suite.Empty(updated.SubfolderOrRoot())

	_, err = suite.shareService.UpdateShare("sub-update", subfolderShare("sub-update", "../escape", volume))
	suite.True(errors.Is(err, dto.ErrorShareValidation), "got %v", err)
}

func (suite *ShareServiceSuite) TestUpdateShareSubfolderInUse() {
	volume := suite.subfolderVolume("sub-inuse")
	_, err := suite.shareService.CreateShare(subfolderShare("sub-inuse-a", "a", volume))
	suite.Require().NoError(err)
	_, err = suite.shareService.CreateShare(subfolderShare("sub-inuse-b", "b", volume))
	suite.Require().NoError(err)

	_, err = suite.shareService.UpdateShare("sub-inuse-b", subfolderShare("sub-inuse-b", "a", volume))

	suite.True(errors.Is(err, dto.ErrorShareFolderInUse), "got %v", err)
	stored, err := suite.shareService.GetShare("sub-inuse-b")
	suite.Require().NoError(err)
	suite.Equal("b", stored.SubfolderOrRoot(), "a refused update leaves the share unchanged")
}

func (suite *ShareServiceSuite) TestVerifyShareSubfolder() {
	volume := suite.subfolderVolume("sub-verify")
	_, err := suite.shareService.CreateShare(subfolderShare("sub-verify", "emperor", volume))
	suite.Require().NoError(err)

	present := subfolderShare("sub-verify", "emperor", volume)
	suite.Require().NoError(suite.shareService.VerifyShare(&present))
	suite.True(present.Status.IsValid)

	// A missing folder invalidates the share instead of exposing the volume root.
	missing := subfolderShare("sub-verify", "gone", volume)
	suite.Require().NoError(suite.shareService.VerifyShare(&missing))
	suite.False(missing.Status.IsValid)
}

func (suite *ShareServiceSuite) TestGetShareFromPathPrefersVolumeRootShare() {
	volume := suite.subfolderVolume("sub-frompath")
	_, err := suite.shareService.CreateShare(subfolderShare("a-subfolder", "emperor", volume))
	suite.Require().NoError(err)
	_, err = suite.shareService.CreateShare(subfolderShare("z-volume-root", "", volume))
	suite.Require().NoError(err)

	share, err := suite.shareService.GetShareFromPath(volume.Path)

	suite.Require().NoError(err)
	suite.Equal("z-volume-root", share.Name)
}

// TestMountEventUpdatesEverySubfolderShare checks that all shares of a volume
// follow its mount state, not only the first one found.
func (suite *ShareServiceSuite) TestMountEventUpdatesEverySubfolderShare() {
	volume := suite.subfolderVolume("sub-mountevent")
	for _, name := range []string{"emperor", "pavel"} {
		_, err := suite.shareService.CreateShare(subfolderShare("mountevent-"+name, name, volume))
		suite.Require().NoError(err)
	}

	handlerCaptor := mock.Captor[func(context.Context, events.MountPointEvent) errors.E]()
	mock.Verify(suite.eventBus, mock.Once()).OnMountPoint(handlerCaptor.Capture())
	eventCaptor := mock.Captor[events.ShareEvent]()
	mock.When(suite.eventBus.EmitShare(eventCaptor.Capture())).ThenReturn(nil)

	err := handlerCaptor.Last()(suite.T().Context(), events.MountPointEvent{MountPoint: volume})

	suite.Require().NoError(err)
	updated := map[string]bool{}
	for _, event := range eventCaptor.Values() {
		if event.Type == events.EventTypes.UPDATE {
			updated[event.Share.Name] = true
		}
	}
	suite.Equal(map[string]bool{"mountevent-emperor": true, "mountevent-pavel": true}, updated)
}
