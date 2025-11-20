package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/shomali11/util/xhashes"
	"gitlab.com/tozd/go/errors"
)

// These tests extend VolumeHandlerSuite (declared in volumes_test.go) and reuse its fx setup.

func (suite *VolumeHandlerSuite) TestMountVolumeSuccess() {
	path := "/mnt/testvol"
	mount := dto.MountPointData{Path: path, Type: "HOST"}
	mount.PathHash = xhashes.SHA1(mount.Path)

	mock.When(suite.mockVolumeSvc.MountVolume(mock.Any[*dto.MountPointData]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Post(fmt.Sprintf("/volume/%s/mount", mount.PathHash), mount)
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.MountPointData
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal(path, out.Path)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).MountVolume(mock.Any[*dto.MountPointData]())

}

func (suite *VolumeHandlerSuite) TestMountVolumeInvalidHash() {
	path := "/mnt/testvol"
	mount := dto.MountPointData{Path: path, PathHash: "badhash", Type: "HOST"}

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Post("/volume/badhash/mount", mount)
	suite.Require().Equal(http.StatusConflict, resp.Code)
}

func (suite *VolumeHandlerSuite) TestUmountVolumeSuccess() {
	mountPath := "/mnt/testvol"
	hash := xhashes.SHA1(mountPath)

	mock.When(suite.mockVolumeSvc.PathHashToPath(mock.Any[string]())).ThenReturn(mountPath, nil)
	mock.When(suite.mockShareSvc.SetShareFromPathEnabled(mock.Any[string](), mock.Any[bool]())).ThenReturn(nil, nil)
	mock.When(suite.mockVolumeSvc.UnmountVolume(mock.Any[string](), mock.Any[bool](), mock.Any[bool]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Delete(fmt.Sprintf("/volume/%s/mount", hash))
	suite.Require().Equal(http.StatusNoContent, resp.Code)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).PathHashToPath(mock.Any[string]())
	mock.Verify(suite.mockShareSvc, matchers.Times(1)).SetShareFromPathEnabled(mock.Any[string](), mock.Any[bool]())
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).UnmountVolume(mock.Any[string](), mock.Any[bool](), mock.Any[bool]())
}

/*
func (suite *VolumeHandlerSuite) TestEjectDiskSuccess() {
	mock.When(suite.mockVolumeSvc.EjectDisk(mock.Any[string]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Post("/volume/disk/sda/eject", struct{}{})
	suite.Require().Equal(http.StatusNoContent, resp.Code)
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).EjectDisk(mock.Any[string]())
}
*/

func (suite *VolumeHandlerSuite) TestPatchVolumeSettingsSuccess() {
	mountPath := "/mnt/testvol"
	hash := xhashes.SHA1(mountPath)
	inputDto := dto.MountPointData{Path: mountPath, Type: "HOST"}

	updated := inputDto
	mock.When(suite.mockVolumeSvc.PathHashToPath(mock.Any[string]())).ThenReturn(mountPath, nil)
	mock.When(suite.mockVolumeSvc.PatchMountPointSettings(mock.Any[string](), mock.Any[dto.MountPointData]())).ThenReturn(&updated, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Patch(fmt.Sprintf("/volume/%s/settings", hash), inputDto)
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.MountPointData
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal(mountPath, out.Path)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).PathHashToPath(mock.Any[string]())
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).PatchMountPointSettings(mock.Any[string](), mock.Any[dto.MountPointData]())
}

func (suite *VolumeHandlerSuite) TestMountVolumeErrorBranches() {
	path := "/mnt/testvol"
	mount := dto.MountPointData{Path: path, Type: "HOST"}
	mount.PathHash = xhashes.SHA1(mount.Path)
	ctrl := mock.NewMockController(suite.T())

	// Mount fail with details -> 422
	vmock1 := mock.Mock[service.VolumeServiceInterface](ctrl)
	mockErr := errors.WithDetails(dto.ErrorMountFail, "Detail", "failure")
	mock.When(vmock1.MountVolume(mock.Any[*dto.MountPointData]())).ThenReturn(mockErr)
	h1 := api.NewVolumeHandler(vmock1, suite.mockShareSvc, &dto.ContextState{})
	_, apiInst1 := humatest.New(suite.T())
	h1.RegisterVolumeHandlers(apiInst1)
	resp := apiInst1.Post(fmt.Sprintf("/volume/%s/mount", mount.PathHash), mount)
	suite.Require().Equal(http.StatusUnprocessableEntity, resp.Code)

	// Device not found -> 404
	vmock2 := mock.Mock[service.VolumeServiceInterface](ctrl)
	mock.When(vmock2.MountVolume(mock.Any[*dto.MountPointData]())).ThenReturn(errors.WithStack(dto.ErrorDeviceNotFound))
	h2 := api.NewVolumeHandler(vmock2, suite.mockShareSvc, &dto.ContextState{})
	_, apiInst2 := humatest.New(suite.T())
	h2.RegisterVolumeHandlers(apiInst2)
	resp2 := apiInst2.Post(fmt.Sprintf("/volume/%s/mount", mount.PathHash), mount)
	suite.Require().Equal(http.StatusNotFound, resp2.Code)

	// Invalid parameter -> 406
	vmock3 := mock.Mock[service.VolumeServiceInterface](ctrl)
	mock.When(vmock3.MountVolume(mock.Any[*dto.MountPointData]())).ThenReturn(errors.WithStack(dto.ErrorInvalidParameter))
	h3 := api.NewVolumeHandler(vmock3, suite.mockShareSvc, &dto.ContextState{})
	_, apiInst3 := humatest.New(suite.T())
	h3.RegisterVolumeHandlers(apiInst3)
	resp3 := apiInst3.Post(fmt.Sprintf("/volume/%s/mount", mount.PathHash), mount)
	suite.Require().Equal(http.StatusNotAcceptable, resp3.Code)

	// Unknown error -> 500
	vmock4 := mock.Mock[service.VolumeServiceInterface](ctrl)
	mock.When(vmock4.MountVolume(mock.Any[*dto.MountPointData]())).ThenReturn(errors.New("boom"))
	h4 := api.NewVolumeHandler(vmock4, suite.mockShareSvc, &dto.ContextState{})
	_, apiInst4 := humatest.New(suite.T())
	h4.RegisterVolumeHandlers(apiInst4)
	resp4 := apiInst4.Post(fmt.Sprintf("/volume/%s/mount", mount.PathHash), mount)
	suite.Require().Equal(http.StatusInternalServerError, resp4.Code)
}
