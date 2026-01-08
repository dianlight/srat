package api_test

import (
	"net/http"
	"time"

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
	mock.When(suite.mockUpgradeService.GetUpgradeReleaseAsset()).ThenReturn(asset, nil)
	// Expect DownloadAndExtractBinaryAsset to be called and return a fake path
	mock.When(suite.mockUpgradeService.DownloadAndExtractBinaryAsset(mock.Any[dto.BinaryAsset]())).ThenReturn(&service.UpdatePackage{}, nil)
	// Expect ApplyUpdateAndRestart to be called with the UpdatePackage (replaces InstallUpdatePackage in new selfupdate flow)
	mock.When(suite.mockUpgradeService.ApplyUpdateAndRestart(mock.Any[*service.UpdatePackage]())).ThenReturn(nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterUpgradeHanler(apiInst)

	// Set testDone hook for deterministic synchronization instead of sleeping.
	done := make(chan struct{})
	suite.handler.SetTestDoneHook(done)

	resp := apiInst.Put("/update", struct{}{})
	suite.Require().Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Body.String(), "v9.9.9")

	// Wait for background goroutine to signal completion (with timeout)
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for update background goroutine to finish")
	}

	mock.Verify(suite.mockUpgradeService, matchers.Times(1)).GetUpgradeReleaseAsset()
	mock.Verify(suite.mockUpgradeService, matchers.Times(1)).DownloadAndExtractBinaryAsset(mock.Any[dto.BinaryAsset]())
	mock.Verify(suite.mockUpgradeService, matchers.Times(1)).ApplyUpdateAndRestart(mock.Any[*service.UpdatePackage]())
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
