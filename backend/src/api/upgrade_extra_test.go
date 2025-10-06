package api_test

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
)

// Tests that exercise UpdateHandler background flow and GetUpdateChannelsHandler logic.

func (suite *UpgradeHandlerSuite) TestUpdateHandlerStartsBackgroundFlow() {
	asset := &dto.ReleaseAsset{
		LastRelease: "v9.9.9",
		ArchAsset:   dto.BinaryAsset{Name: "srat-amd64.zip"},
	}

	// Expect GetUpgradeReleaseAsset to return the asset
	mock.When(suite.mockUpgradeService.GetUpgradeReleaseAsset(mock.Any[*dto.UpdateChannel]())).ThenReturn(asset, nil)
	// Expect DownloadAndExtractBinaryAsset to be called and return a fake path
	mock.When(suite.mockUpgradeService.DownloadAndExtractBinaryAsset(mock.Any[dto.BinaryAsset]())).ThenReturn(&service.UpdatePackage{TempDirPath: "/tmp/pkg"}, nil)
	// Expect InstallUpdatePackage to be called with the UpdatePackage
	mock.When(suite.mockUpgradeService.InstallUpdatePackage(mock.Any[*service.UpdatePackage]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterUpgradeHanler(apiInst)

	resp := apiInst.Put("/update", struct{}{})
	suite.Require().Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Body.String(), "v9.9.9")

	mock.Verify(suite.mockUpgradeService, matchers.Times(1)).GetUpgradeReleaseAsset(mock.Any[*dto.UpdateChannel]())
	mock.Verify(suite.mockUpgradeService, matchers.Times(1)).DownloadAndExtractBinaryAsset(mock.Any[dto.BinaryAsset]())
	mock.Verify(suite.mockUpgradeService, matchers.Times(1)).InstallUpdatePackage(mock.Any[*service.UpdatePackage]())
}

func (suite *UpgradeHandlerSuite) TestGetUpdateChannelsFiltersDevelopCorrectly() {
	// Set to stable version -> DEVELOP should be filtered
	orig := config.Version
	defer func() { config.Version = orig }()

	config.Version = "v1.2.3" // stable
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterUpgradeHanler(apiInst)
	resp := apiInst.Get("/update_channels")
	suite.Require().Equal(http.StatusOK, resp.Code)
	// Body should NOT contain DEVELOP
	suite.NotContains(resp.Body.String(), dto.UpdateChannels.DEVELOP.String())

	// Set to prerelease -> DEVELOP should be present
	config.Version = "v1.2.3-alpha"
	_, apiInst2 := humatest.New(suite.T())
	suite.handler.RegisterUpgradeHanler(apiInst2)
	resp2 := apiInst2.Get("/update_channels")
	suite.Require().Equal(http.StatusOK, resp2.Code)
	suite.Contains(resp2.Body.String(), dto.UpdateChannels.DEVELOP.String())
}
