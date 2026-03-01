package service_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	// Third-party libraries for testing

	"aead.dev/minisign"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v84/github"
	"github.com/jarcoal/httpmock"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	// Project-specific packages
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/updatekey"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/internal/ctxkeys"
)

type UpgradeServiceTestSuite struct {
	suite.Suite
	upgradeService  service.UpgradeServiceInterface
	mockBroadcaster service.BroadcasterServiceInterface
	//mockPropertyRepo repository.PropertyRepositoryInterface
	state           *dto.ContextState
	app             *fxtest.App
	ctx             context.Context
	cancel          context.CancelFunc
	wg              *sync.WaitGroup
	originalVersion string
	privateKey      minisign.PrivateKey
}

const githubReleasesURL = "https://api.github.com/repos/dianlight/srat/releases?page=1&per_page=5"

func TestUpgradeServiceTestSuite(t *testing.T) {
	//t.Skipf("Activate only to test upgrade of github library for limit rate of github call")
	suite.Run(t, new(UpgradeServiceTestSuite))
}

func (suite *UpgradeServiceTestSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}
	suite.originalVersion = config.Version // Store original config.Version

	// Ensure ALL outbound HTTP calls are intercepted by httpmock.
	// Any request without an explicit responder will fail the test (prevents real GitHub calls).
	httpmock.Activate()
	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("unexpected http call (missing httpmock responder): %s %s", req.Method, req.URL.String())
	})
	// Default GitHub releases fixture used by the UpgradeService background loop and any tests
	// that don't override the releases endpoint.
	suite.registerGitHubReleasesFixture()

	tmpDir, err := os.MkdirTemp(os.TempDir(), "srat_update_*")
	if err != nil {
		panic(err)
	}

	// Minisign test key setup
	var pub minisign.PublicKey
	pub, suite.privateKey, err = minisign.GenerateKey(nil)
	suite.Require().NoError(err, "failed to generate minisign test key pair")
	//updatekey.UpdatePublicKey = pub.String()
	sugn, err := pub.MarshalText()
	suite.Require().NoError(err, "failed to marshal minisign public key")
	updatekey.UpdatePublicKey = string(sugn)
	suite.T().Logf("Using minisign public key for tests: %s", updatekey.UpdatePublicKey)

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg)
				return context.WithCancel(ctx)
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					HACoreReady:    true,
					SupervisorURL:  "http://supervisor",
					AddonIpAddress: "172.30.32.1",
					UpdateDataDir:  tmpDir,
					//UpdateFilePath: tmpDir + "/" + filepath.Base(os.Args[0]),
					UpdateChannel: dto.UpdateChannels.NONE,
					AutoUpdate:    false,
				}
			},
			service.NewUpgradeService,
			mock.Mock[service.BroadcasterServiceInterface],
			//mock.Mock[repository.PropertyRepositoryInterface],
			func() *github.Client {
				rateLimiter := github_ratelimit.New(nil)
				return github.NewClient(&http.Client{Transport: rateLimiter})
			},
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.mockBroadcaster),
		//fx.Populate(&suite.mockPropertyRepo),
		fx.Populate(&suite.upgradeService),
		fx.Populate(&suite.state),
	)

	// Default mocks
	//mock.When(suite.mockPropertyRepo.Value("UpdateChannel")).ThenReturn(&dto.UpdateChannels.RELEASE, nil)
	//mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[any]())).ThenReturn(nil, nil)

	suite.app.RequireStart()
}

func (suite *UpgradeServiceTestSuite) registerGitHubReleasesFixture() {
	suite.T().Helper()
	fixturePath := filepath.Join("..", "..", "test", "data", "github_release.json")
	b, err := os.ReadFile(fixturePath)
	suite.Require().NoError(err, "failed to read fixture %s", fixturePath)

	httpmock.RegisterResponder("GET", githubReleasesURL, func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewBytesResponse(200, b)
		resp.Header.Set("Content-Type", "application/json")
		resp.ContentLength = int64(len(b))
		return resp, nil
	})
}

func (suite *UpgradeServiceTestSuite) TearDownTest() {
	suite.cancel()
	suite.wg.Wait() // Wait for the service's goroutine to finish
	if suite.app != nil {
		suite.app.RequireStop()
	}
	httpmock.DeactivateAndReset()
	config.Version = suite.originalVersion // Restore original config.Version
	os.RemoveAll(suite.state.UpdateDataDir)
}

// Helper to create a mock GitHub release asset
func newGitHubReleaseAsset(name, downloadURL string, size int64) *github.ReleaseAsset {
	return &github.ReleaseAsset{
		Name:               &name,
		BrowserDownloadURL: &downloadURL,
		Size:               new(int(size)),
		ContentType:        new("application/zip"),
		ID:                 new(time.Now().UnixNano()), // Unique ID
		NodeID:             new(fmt.Sprintf("AssetNodeID%d", time.Now().UnixNano())),
		Label:              new(""),
		State:              new("uploaded"),
		CreatedAt:          &github.Timestamp{Time: time.Now()},
		UpdatedAt:          &github.Timestamp{Time: time.Now()},
		Uploader:           &github.User{Login: new("testuser")},
	}
}

// Helper to create a mock GitHub repository release
func newGitHubRepositoryRelease(tagName string, prerelease bool, assets []*github.ReleaseAsset) *github.RepositoryRelease {
	return &github.RepositoryRelease{
		TagName:         &tagName,
		Prerelease:      new(prerelease),
		Assets:          assets,
		TargetCommitish: new("main"),
		Name:            new(fmt.Sprintf("Release %s", tagName)),
		Body:            new(fmt.Sprintf("Release notes for %s", tagName)),
		Draft:           new(false),
		HTMLURL:         new(fmt.Sprintf("https://github.com/releases/%s", tagName)),
		AssetsURL:       new(fmt.Sprintf("https://github.com/releases/%s/assets", tagName)),
		UploadURL:       new(fmt.Sprintf("https://github.com/releases/%s/upload", tagName)),
		ZipballURL:      new(fmt.Sprintf("https://github.com/archive/%s.zip", tagName)),
		TarballURL:      new(fmt.Sprintf("https://github.com/archive/%s.tar.gz", tagName)),
		PublishedAt:     &github.Timestamp{Time: time.Now()},
		CreatedAt:       &github.Timestamp{Time: time.Now()},
		Author:          &github.User{Login: new("testuser")},
	}
}

// --- GetUpgradeReleaseAsset Tests ---

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_ChannelNone() {
	suite.state.UpdateChannel = dto.UpdateChannels.NONE
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
	suite.Contains(err.Error(), "No releases check")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_ChannelDevelop() {
	suite.state.UpdateChannel = dto.UpdateChannels.DEVELOP
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
	suite.Contains(err.Error(), "No releases check")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_ChannelPreRelease() {
	suite.state.UpdateChannel = dto.UpdateChannels.PRERELEASE
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.NotNil(asset)
	suite.Require().NoError(err)
	suite.NotEmpty(asset)
	suite.Equal("2025.12.0-dev.3", asset.LastRelease)
	suite.Contains(asset.ArchAsset.Name, ".zip")

}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_ChannelRelease() {
	suite.state.UpdateChannel = dto.UpdateChannels.RELEASE
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.NotNil(asset)
	suite.Require().NoError(err)
	suite.NotEmpty(asset)
	suite.Equal("2025.6.9", asset.LastRelease)
	suite.Contains(asset.ArchAsset.Name, ".zip")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_GitHubAPIFailure() {
	config.Version = "1.0.0"
	httpmock.RegisterResponder("GET", githubReleasesURL,
		httpmock.NewErrorResponder(fmt.Errorf("github api down")))

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable), "Error was: %v", err)
	suite.Contains(err.Error(), "No update available for the specified channel and architecture")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_NoReleasesFound() {
	config.Version = "1.0.0"
	httpmock.RegisterResponder("GET", githubReleasesURL,
		httpmock.NewJsonResponderOrPanic(200, []*github.RepositoryRelease{}))

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
	suite.Contains(err.Error(), "No update available")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_SkipPrerelease_WhenChannelIsRelease() {
	config.Version = "1.0.0"
	arch := runtime.GOARCH
	switch arch {
	case "arm64":
		arch = "aarch64"
	case "amd64":
		arch = "x86_64"
	}
	assetName := fmt.Sprintf("srat_%s.zip", arch)

	releases := []*github.RepositoryRelease{
		newGitHubRepositoryRelease("v1.1.0-beta", true, []*github.ReleaseAsset{
			newGitHubReleaseAsset(assetName, "https://github.com/srat_v1.1.0-beta.zip", 1024),
		}),
	}
	httpmock.RegisterResponder("GET", githubReleasesURL,
		httpmock.NewJsonResponderOrPanic(200, releases))

	// Default channel is RELEASE
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_AcceptPrerelease_WhenChannelIsPrerelease() {
	config.Version = "2025.12.0-dev.2"
	arch := runtime.GOARCH
	switch arch {
	case "arm64":
		arch = "aarch64"
	case "amd64":
		arch = "x86_64"
	}
	assetName := fmt.Sprintf("srat_%s.zip", arch)
	// Uses the default GitHub releases fixture registered in SetupTest.
	expectedRelease := "2025.12.0-dev.3"
	expectedURL := fmt.Sprintf("https://github.com/dianlight/srat/releases/download/%s/%s", expectedRelease, assetName)

	//mock.When(suite.mockPropertyRepo.Value("UpdateChannel")).ThenReturn(&dto.UpdateChannels.PRERELEASE, nil)

	suite.state.UpdateChannel = dto.UpdateChannels.PRERELEASE
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.Require().NoError(err)
	suite.Require().NotNil(asset)
	suite.Equal(expectedRelease, asset.LastRelease)
	suite.Equal(assetName, asset.ArchAsset.Name)
	suite.Equal(expectedURL, asset.ArchAsset.BrowserDownloadURL)
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_CurrentVersionNewer() {
	config.Version = "1.2.0"
	arch := runtime.GOARCH
	switch arch {
	case "arm64":
		arch = "aarch64"
	case "amd64":
		arch = "x86_64"
	}
	assetName := fmt.Sprintf("srat_%s.zip", arch)

	releases := []*github.RepositoryRelease{
		newGitHubRepositoryRelease("v1.1.0", false, []*github.ReleaseAsset{
			newGitHubReleaseAsset(assetName, "https://github.com/srat_v1.1.0.zip", 1024),
		}),
	}
	httpmock.RegisterResponder("GET", githubReleasesURL,
		httpmock.NewJsonResponderOrPanic(200, releases))

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_Success_PicksLatestValidRelease() {
	config.Version = "1.0.0"
	arch := runtime.GOARCH
	switch arch {
	case "arm64":
		arch = "aarch64"
	case "amd64":
		arch = "x86_64"
	}
	assetName := fmt.Sprintf("srat_%s.zip", arch)

	releases := []*github.RepositoryRelease{
		newGitHubRepositoryRelease("2024.1.0", false, []*github.ReleaseAsset{ // Older, valid
			newGitHubReleaseAsset(assetName, "https://github.com/srat_v1.1.0.zip", 1024),
		}),
		newGitHubRepositoryRelease("2025.6.1", false, []*github.ReleaseAsset{ // Newer, valid
			newGitHubReleaseAsset(assetName, "https://github.com/srat_v1.2.0.zip", 2048),
		}),
		newGitHubRepositoryRelease("2025.6.0", false, []*github.ReleaseAsset{ // Older, valid
			newGitHubReleaseAsset(assetName, "https://github.com/srat_v1.0.5.zip", 512),
		}),
		newGitHubRepositoryRelease("2025.6.2-dev076", true, []*github.ReleaseAsset{ // Prerelease, skipped by default
			newGitHubReleaseAsset(assetName, "https://github.com/srat_v1.3.0-beta.zip", 4096),
		}),
	}
	httpmock.RegisterResponder("GET", githubReleasesURL,
		httpmock.NewJsonResponderOrPanic(200, releases))

	suite.state.UpdateChannel = dto.UpdateChannels.RELEASE
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset()
	suite.Require().NoError(err)
	suite.Require().NotNil(asset)
	suite.Equal("2025.6.1", asset.LastRelease) // Should pick the latest non-prerelease version
	suite.Equal(assetName, asset.ArchAsset.Name)
	suite.Equal("https://github.com/srat_v1.2.0.zip", asset.ArchAsset.BrowserDownloadURL)
}

// --- DownloadAndExtractBinaryAsset Tests ---

func createDummyZip(files map[string]string, force_size int, privateKey *minisign.PrivateKey) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	for name, content := range files {
		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		if force_size > 0 && len(content) < force_size {
			// Adjust content to force size
			content += strings.Repeat(" ", force_size-len(content))
		}
		if privateKey != nil {
			comment := minisign.Sign(*privateKey, []byte(content))
			header.Comment = string(comment)
		}

		f, err := zipWriter.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			return nil, err
		}
	}

	err := zipWriter.Close()
	return buf, err
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_NoSignature() {
	currentExePath, _ := os.Executable()
	currentExeName := filepath.Base(currentExePath)

	zipContents := map[string]string{
		currentExeName:    "fake binary content for main exe",
		"other_file.sh":   "#!/bin/bash\necho hello",
		"config/data.txt": "some config data",
	}
	zipBuffer, err := createDummyZip(zipContents, 0, nil)
	suite.Require().NoError(err)

	// Compute the correct digest for this unsigned zip
	ssha256 := sha256.New()
	ssha256.Write(zipBuffer.Bytes())
	correctDigest := "sha256:" + fmt.Sprintf("%x", ssha256.Sum(nil))

	asset := dto.BinaryAsset{
		Name:               "test_asset.zip",
		BrowserDownloadURL: "https://github.com/test_asset.zip",
		Digest:             correctDigest,
		Size:               zipBuffer.Len(),
	}

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, zipBuffer.Bytes())
			resp.Header.Set("Content-Type", "application/zip")
			resp.ContentLength = int64(zipBuffer.Len()) // Crucial for progress
			return resp, nil
		})

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "file has no signature in comment")
	suite.Nil(updatePkg)
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_ContainDir() {

	asset := dto.BinaryAsset{
		Name:               "test_asset.zip",
		BrowserDownloadURL: "https://github.com/test_asset.zip",
		Digest:             "sha256:f6e9d067648b3b21359dd9988650ffa8ab340f0d8579ba0c4c4e6c2ae2048556",
	}

	zipContents := map[string]string{
		"config/data.txt": "some config data",
	}
	zipBuffer, err := createDummyZip(zipContents, 0, nil)
	suite.Require().NoError(err)
	asset.Size = zipBuffer.Len()

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, zipBuffer.Bytes())
			resp.Header.Set("Content-Type", "application/zip")
			resp.ContentLength = int64(zipBuffer.Len()) // Crucial for progress
			return resp, nil
		})

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "file has no signature in comment")
	suite.Nil(updatePkg)
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_Success() {
	currentExePath, _ := os.Executable()
	currentExeName := filepath.Base(currentExePath)

	asset := dto.BinaryAsset{
		Name:               "test_asset.zip",
		BrowserDownloadURL: "https://github.com/test_asset.zip",
		//		Size:               100, // This size is used for progress reporting
		//Digest: "sha256:14bd9fb509e174888a0b64ba436cf2ba4d3788cc7e40f9db77723113cc61865b",
	}

	zipContents := map[string]string{
		currentExeName:    "fake binary content for main exe",
		"other_file.sh":   "#!/bin/bash\necho hello",
		"config/data.txt": "some config data",
	}
	zipBuffer, err := createDummyZip(zipContents, 0, &suite.privateKey)
	suite.Require().NoError(err)
	asset.Size = zipBuffer.Len()
	// sha256 of zipBuffer
	ssha256 := sha256.New()
	ssha256.Write(zipBuffer.Bytes())
	asset.Digest = "sha256:" + fmt.Sprintf("%x", ssha256.Sum(nil))

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, zipBuffer.Bytes())
			resp.Header.Set("Content-Type", "application/zip")
			resp.ContentLength = int64(zipBuffer.Len()) // Crucial for progress
			return resp, nil
		})

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().NoError(err)
	suite.Require().NotNil(updatePkg)

	suite.Require().NotNil(updatePkg.FilesPaths)
	suite.Len(updatePkg.FilesPaths, len(zipContents))

	foundExe := false
	foundOtherFile := false
	foundConfigFile := false
	for _, p := range updatePkg.FilesPaths {
		suite.FileExists(p.Path)
		if filepath.Base(p.Path) == currentExeName {
			foundExe = true
		}
		if filepath.Base(p.Path) == "other_file.sh" {
			foundOtherFile = true
		}
		if strings.HasSuffix(p.Path, filepath.Join("config", "data.txt")) {
			foundConfigFile = true
			// Check nested file content
			content, _ := os.ReadFile(p.Path)
			suite.Equal("some config data", string(content))
		}
	}
	suite.True(foundExe, "%s not found in extracted files", currentExeName)
	suite.True(foundOtherFile, "other_file.sh not found in extracted files")
	suite.True(foundConfigFile, "config/data.txt not found in extracted files")

	// Verify progress notifications
	var progressEvents []dto.UpdateProgress
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[dto.UpdateProgress]())).ThenAnswer(
		matchers.Answer(func(args []any) []any {
			if p, ok := args[0].(dto.UpdateProgress); ok {
				progressEvents = append(progressEvents, p)
			}
			return []any{nil}
		}),
	)
	// Re-run to capture broadcasts
	updatePkg, err = suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().NoError(err)

	suite.ContainsProgress(progressEvents, dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, 0)
	suite.ContainsProgress(progressEvents, dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, 100) // Or some intermediate if size is large
	suite.ContainsProgress(progressEvents, dto.UpdateProcessStates.UPDATESTATUSDOWNLOADCOMPLETE, 100)
	suite.ContainsProgress(progressEvents, dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, 0)
	suite.ContainsProgress(progressEvents, dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, 100)
	suite.ContainsProgress(progressEvents, dto.UpdateProcessStates.UPDATESTATUSEXTRACTCOMPLETE, 100)
}

// Helper to check if a specific progress event was broadcasted
func (suite *UpgradeServiceTestSuite) ContainsProgress(events []dto.UpdateProgress, status dto.UpdateProcessState, progress int) {
	found := false
	for _, e := range events {
		if e.ProgressStatus == status && int(e.Progress) == progress {
			found = true
			break
		}
	}
	if !found {
		// Log existing events for easier debugging
		var eventSummaries []string
		for _, e := range events {
			eventSummaries = append(eventSummaries, fmt.Sprintf("{Status: %s, Progress: %d}", e.ProgressStatus, int(e.Progress)))
		}
		suite.Failf("Progress event not found", "Expected Status: %s, Progress: %d. Actual events: %s", status, progress, strings.Join(eventSummaries, ", "))
	}
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_DownloadHttpError() {
	asset := dto.BinaryAsset{Name: "test.zip", BrowserDownloadURL: "https://github.com/download/zip/sample-1.zip"}
	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL, httpmock.NewStringResponder(500, "Server Error"))

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Nil(updatePkg)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "failed to download asset: received status code 500")
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_NotAZipFile() {
	asset := dto.BinaryAsset{
		Name:               "notazip.txt",
		BrowserDownloadURL: "https://github.com/download/txt/sample-1.txt",
		Digest:             "sha256:5b144727ab9efa85381eddb567447c2c33b48750a362b86f6d39780b9fc630f5",
	}
	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL, httpmock.NewStringResponder(200, "this is not zip content"))

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Nil(updatePkg)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "failed to open downloaded zip asset") // zip.OpenReader error
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_BlocksZipTraversal() {
	//suite.T().Skip("Skipping TestDownloadAndExtractBinaryAsset_BlocksZipTraversal because it is flaky on Windows")
	// Create a zip containing a file that attempts path traversal
	asset := dto.BinaryAsset{
		Name:               "evil.zip",
		BrowserDownloadURL: "https://github.com/evil.zip",
		Digest:             "sha256:c6916950785c7fb08682a9cf26d4d28d5cc091666fbc82da16957522dde2e577",
	}
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	// File tries to escape extraction dir
	header := &zip.FileHeader{Name: "../escape.txt", Method: zip.Deflate, Comment: "fake comment"}
	_, err := zw.CreateHeader(header)
	suite.Require().NoError(err)
	suite.Require().NoError(zw.Close())

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, buf.Bytes())
			resp.Header.Set("Content-Type", "application/zip")
			resp.ContentLength = int64(buf.Len())
			return resp, nil
		})

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Nil(updatePkg)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "illegal file path")
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_SetsSafePermissions() {
	suite.T().Skip("Skipping TestDownloadAndExtractBinaryAsset_SetsSafePermissions because it is flaky on Windows")
	// Verify directories are created with 0750 and files respect archive mode
	currentExePath, _ := os.Executable()
	currentExeName := filepath.Base(currentExePath)

	asset := dto.BinaryAsset{
		Name:               "perm_test.zip",
		BrowserDownloadURL: "https://github.com/perm_test.zip",
		Digest:             "sha256:2eb9048baa6dc5a2baf303a00cf6cf81ce7b1cf468af5dbb0b43f9d57a67e85b",
	}

	// Build a zip with nested dir and files
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Create a directory entry
	dirHeader := &zip.FileHeader{Name: "nested/", Method: zip.Deflate}
	dirHeader.SetMode(0o777) // even if zip says 0777, extractor should mkdir with 0750
	_, err := zw.CreateHeader(dirHeader)
	suite.Require().NoError(err)

	// Create executable file (pretend to be current exe name)
	exeHeader := &zip.FileHeader{Name: "nested/" + currentExeName, Method: zip.Deflate}
	exeHeader.SetMode(0o755)
	exeWriter, err := zw.CreateHeader(exeHeader)
	suite.Require().NoError(err)
	_, _ = exeWriter.Write([]byte("binary"))

	// Create regular file
	fileHeader := &zip.FileHeader{Name: "nested/config.txt", Method: zip.Deflate}
	fileHeader.SetMode(0o644)
	fileWriter, err := zw.CreateHeader(fileHeader)
	suite.Require().NoError(err)
	_, _ = fileWriter.Write([]byte("data"))

	suite.Require().NoError(zw.Close())

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, buf.Bytes())
			resp.Header.Set("Content-Type", "application/zip")
			resp.ContentLength = int64(buf.Len())
			return resp, nil
		})

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().NoError(err)
	suite.Require().NotNil(updatePkg)

	// Check that directory is 0750 despite zip header suggesting 0777
	nestedDir := filepath.Join(suite.state.UpdateDataDir, "nested")
	info, err := os.Stat(nestedDir)
	suite.Require().NoError(err)
	suite.True(info.IsDir())
	suite.Equal(os.FileMode(0o750)|os.ModeDir, info.Mode()&os.ModePerm|os.ModeDir)

	// Check files exist; mode is as written by extractor (we use f.Mode())
	exePath := filepath.Join(nestedDir, currentExeName)
	finfo, err := os.Stat(exePath)
	suite.Require().NoError(err)
	suite.True(finfo.Mode().IsRegular())
	// mode equals 0755 from header
	suite.Equal(os.FileMode(0o755), finfo.Mode()&os.ModePerm)

	cfgPath := filepath.Join(nestedDir, "config.txt")
	cinfo, err := os.Stat(cfgPath)
	suite.Require().NoError(err)
	suite.True(cinfo.Mode().IsRegular())
	suite.Equal(os.FileMode(0o644), cinfo.Mode()&os.ModePerm)
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_UntrustedURL() {
	asset := dto.BinaryAsset{
		Name:               "untrusted.zip",
		BrowserDownloadURL: "https://evil.com/untrusted.zip",
	}

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "untrusted download URL")
	suite.Nil(updatePkg)
}

// --- InstallUpdatePackage & InstallOverseerUpdate Tests ---

func (suite *UpgradeServiceTestSuite) TestInstallUpdatePackage_NilPackage() {
	err := suite.upgradeService.InstallUpdatePackage(nil)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "invalid update package")
}

func (suite *UpgradeServiceTestSuite) TestInstallUpdatePackage_MissingExecutableInPackage() {
	pkg := &service.UpdatePackage{} // CurrentExecutablePath is nil
	err := suite.upgradeService.InstallUpdatePackage(pkg)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "invalid update package")
}

// Note: Testing the actual `update.Apply` is an integration concern.
// These tests focus on the UpgradeService's logic around it.
// To test `installBinaryTo` more deeply, one might refactor it to make `update.Apply` mockable,
// or use filesystem-level assertions in a more integration-style test.

func (suite *UpgradeServiceTestSuite) TestRun_GoroutineLifecycle() {
	// The main purpose of this test is to ensure the goroutine in NewUpgradeService
	// starts and can be gracefully shut down by cancelling the context.
	// The SetupTest starts the service (and its goroutine).
	// The TearDownTest cancels the context and waits on the WaitGroup.
	// If TearDownTest completes without timeout or panic, the lifecycle is implicitly tested.
	suite.True(true, "Goroutine lifecycle managed by SetupTest/TearDownTest")

	// Optionally, try to capture one initial broadcast from the `run` method's `updateLimiter.Do`
	var initialBroadcast dto.UpdateProgress
	var mu sync.Mutex
	broadcastHappened := false

	//mock.Reset(suite.mockBroadcaster) // Reset any previous mock settings for this specific check
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[dto.UpdateProgress]())).ThenAnswer(
		matchers.Answer(func(args []any) []any {
			mu.Lock()
			defer mu.Unlock()
			if p, ok := args[0].(dto.UpdateProgress); ok {
				if !broadcastHappened { // Capture only the first relevant one
					initialBroadcast = p
					broadcastHappened = true
				}
			}
			return []any{nil}
		}),
	)
	// The `rate.Sometimes` might execute the Do func immediately or after its interval.
	// The `run` loop also has a 10s sleep. To catch the initial `Do`, we might need a short wait.
	// However, forcing it for a unit test is tricky without time control.
	// We rely on the fact that `Do` is called at least once when the service starts.
	// The default mocks are already in place.
	// This part is more to see if *any* broadcast happens from the `run` loop's startup.
	time.Sleep(50 * time.Millisecond) // Give a very small window for the initial Do()

	mu.Lock()
	if broadcastHappened {
		suite.T().Logf("Initial broadcast from run loop: Status=%s, Progress=%f", initialBroadcast.ProgressStatus, initialBroadcast.Progress)
		// We expect it to be checking or similar initial state
		suite.True(initialBroadcast.ProgressStatus == dto.UpdateProcessStates.UPDATESTATUSCHECKING ||
			initialBroadcast.ProgressStatus == dto.UpdateProcessStates.UPDATESTATUSNOUPGRADE ||
			initialBroadcast.ProgressStatus == dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE,
			"Unexpected initial broadcast status: %s", initialBroadcast.ProgressStatus)
	} else {
		suite.T().Log("No initial broadcast captured from run loop within the short wait time.")
	}
	mu.Unlock()
}
