package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/dto"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/shomali11/util/xhashes"
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
	mock.Verify(suite.mockDirtySvc, matchers.Times(1)).SetDirtyVolumes()
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
	mock.When(suite.mockShareSvc.DisableShareFromPath(mock.Any[string]())).ThenReturn(nil, nil)
	mock.When(suite.mockVolumeSvc.UnmountVolume(mock.Any[string](), mock.Any[bool](), mock.Any[bool]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Delete(fmt.Sprintf("/volume/%s/mount", hash))
	suite.Require().Equal(http.StatusNoContent, resp.Code)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).PathHashToPath(mock.Any[string]())
	mock.Verify(suite.mockShareSvc, matchers.Times(1)).DisableShareFromPath(mock.Any[string]())
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).UnmountVolume(mock.Any[string](), mock.Any[bool](), mock.Any[bool]())
	mock.Verify(suite.mockDirtySvc, matchers.Times(1)).SetDirtyVolumes()
}

func (suite *VolumeHandlerSuite) TestEjectDiskSuccess() {
	mock.When(suite.mockVolumeSvc.EjectDisk(mock.Any[string]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Post("/volume/disk/sda/eject", struct{}{})
	suite.Require().Equal(http.StatusNoContent, resp.Code)
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).EjectDisk(mock.Any[string]())
}

func (suite *VolumeHandlerSuite) TestUpdateVolumeSettingsSuccess() {
	mountPath := "/mnt/testvol"
	hash := xhashes.SHA1(mountPath)
	inputDto := dto.MountPointData{Path: mountPath, Type: "HOST"}

	updated := inputDto
	mock.When(suite.mockVolumeSvc.PathHashToPath(mock.Any[string]())).ThenReturn(mountPath, nil)
	mock.When(suite.mockVolumeSvc.UpdateMountPointSettings(mock.Any[string](), mock.Any[dto.MountPointData]())).ThenReturn(&updated, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Put(fmt.Sprintf("/volume/%s/settings", hash), inputDto)
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out dto.MountPointData
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Equal(mountPath, out.Path)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).PathHashToPath(mock.Any[string]())
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).UpdateMountPointSettings(mock.Any[string](), mock.Any[dto.MountPointData]())
	mock.Verify(suite.mockDirtySvc, matchers.Times(1)).SetDirtyVolumes()
}

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
	mock.Verify(suite.mockDirtySvc, matchers.Times(1)).SetDirtyVolumes()
}
