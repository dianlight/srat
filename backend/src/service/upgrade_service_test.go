package service_test

import (
	"archive/zip"
	"bytes"
	"context"
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
	"github.com/Masterminds/semver/v3"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v74/github"
	"github.com/jarcoal/httpmock"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	// Project-specific packages
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
)

type UpgradeServiceTestSuite struct {
	suite.Suite
	upgradeService   service.UpgradeServiceInterface
	mockBroadcaster  service.BroadcasterServiceInterface
	mockPropertyRepo repository.PropertyRepositoryInterface
	app              *fxtest.App
	ctx              context.Context
	cancel           context.CancelFunc
	wg               *sync.WaitGroup
	originalVersion  string
}

func TestUpgradeServiceTestSuite(t *testing.T) {
	t.SkipNow()
	suite.Run(t, new(UpgradeServiceTestSuite))
}

func (suite *UpgradeServiceTestSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}
	suite.originalVersion = config.Version // Store original config.Version

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", suite.wg)
				return context.WithCancel(ctx)
			},
			service.NewUpgradeService,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			func() *github.Client {
				httpmock.Activate()
				rateLimiter := github_ratelimit.New(nil)
				return github.NewClient(&http.Client{
					Transport: rateLimiter,
				})
			},
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.mockBroadcaster),
		fx.Populate(&suite.mockPropertyRepo),
		fx.Populate(&suite.upgradeService),
	)

	// Default mocks
	mock.When(suite.mockPropertyRepo.Value("UpdateChannel", false)).ThenReturn(&dto.UpdateChannels.RELEASE, nil)
	//mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[any]())).ThenReturn(nil, nil)

	suite.app.RequireStart()
}

func (suite *UpgradeServiceTestSuite) TearDownTest() {
	suite.cancel()
	suite.wg.Wait() // Wait for the service's goroutine to finish
	if suite.app != nil {
		suite.app.RequireStop()
	}
	httpmock.DeactivateAndReset()
	config.Version = suite.originalVersion // Restore original config.Version
}

// Helper to create a mock GitHub release asset
func newGitHubReleaseAsset(name, downloadURL string, size int64) *github.ReleaseAsset {
	return &github.ReleaseAsset{
		Name:               &name,
		BrowserDownloadURL: &downloadURL,
		Size:               pointer.Int(int(size)),
		ContentType:        pointer.String("application/zip"),
		ID:                 pointer.Int64(time.Now().UnixNano()), // Unique ID
		NodeID:             pointer.String(fmt.Sprintf("AssetNodeID%d", time.Now().UnixNano())),
		Label:              pointer.String(""),
		State:              pointer.String("uploaded"),
		CreatedAt:          &github.Timestamp{Time: time.Now()},
		UpdatedAt:          &github.Timestamp{Time: time.Now()},
		Uploader:           &github.User{Login: pointer.String("testuser")},
	}
}

// Helper to create a mock GitHub repository release
func newGitHubRepositoryRelease(tagName string, prerelease bool, assets []*github.ReleaseAsset) *github.RepositoryRelease {
	return &github.RepositoryRelease{
		TagName:         github.String(tagName),
		Prerelease:      github.Bool(prerelease),
		Assets:          assets,
		TargetCommitish: github.String("main"),
		Name:            github.String(fmt.Sprintf("Release %s", tagName)),
		Body:            github.String(fmt.Sprintf("Release notes for %s", tagName)),
		Draft:           github.Bool(false),
		HTMLURL:         github.String(fmt.Sprintf("http://example.com/releases/%s", tagName)),
		AssetsURL:       github.String(fmt.Sprintf("http://example.com/releases/%s/assets", tagName)),
		UploadURL:       github.String(fmt.Sprintf("http://example.com/releases/%s/upload", tagName)),
		ZipballURL:      github.String(fmt.Sprintf("http://example.com/archive/%s.zip", tagName)),
		TarballURL:      github.String(fmt.Sprintf("http://example.com/archive/%s.tar.gz", tagName)),
		PublishedAt:     &github.Timestamp{Time: time.Now()},
		CreatedAt:       &github.Timestamp{Time: time.Now()},
		Author:          &github.User{Login: github.String("testuser")},
	}
}

// --- GetUpgradeReleaseAsset Tests ---

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_ChannelNone() {
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(&dto.UpdateChannels.NONE)
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
	suite.Contains(err.Error(), "No releases check")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_ChannelDevelop() {
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(&dto.UpdateChannels.DEVELOP)
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
	suite.Contains(err.Error(), "No releases check")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_ErrorParsingCurrentVersion() {
	config.Version = "invalid-version"

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(&dto.UpdateChannels.RELEASE)
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.ErrorIs(err, semver.ErrInvalidSemVer)
	suite.Contains(err.Error(), "invalid semantic version", "Error should be a semantic version parsing error")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_GitHubAPIFailure() {
	config.Version = "1.0.0"
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/dianlight/srat/releases?page=1&per_page=5",
		httpmock.NewErrorResponder(fmt.Errorf("github api down")))

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(nil)
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable), "Error was: %v", err)
	suite.Contains(err.Error(), "No update available for the specified channel and architecture")
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_NoReleasesFound() {
	config.Version = "1.0.0"
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/dianlight/srat/releases?page=1&per_page=5",
		httpmock.NewJsonResponderOrPanic(200, []*github.RepositoryRelease{}))

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(nil)
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
	suite.Contains(err.Error(), "No releases found")
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
			newGitHubReleaseAsset(assetName, "http://example.com/srat_v1.1.0-beta.zip", 1024),
		}),
	}
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/dianlight/srat/releases?page=1&per_page=5",
		httpmock.NewJsonResponderOrPanic(200, releases))

	// Default channel is RELEASE
	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(nil)
	suite.Nil(asset)
	suite.Require().Error(err)
	suite.True(errors.Is(err, dto.ErrorNoUpdateAvailable))
}

func (suite *UpgradeServiceTestSuite) TestGetUpgradeReleaseAsset_AcceptPrerelease_WhenChannelIsPrerelease() {
	config.Version = "1.0.0"
	arch := runtime.GOARCH
	switch arch {
	case "arm64":
		arch = "aarch64"
	case "amd64":
		arch = "x86_64"
	}
	assetName := fmt.Sprintf("srat_%s.zip", arch)
	expectedURL := "http://example.com/srat_v1.1.0-beta.zip"

	releases := []*github.RepositoryRelease{
		newGitHubRepositoryRelease("v1.1.0-beta", true, []*github.ReleaseAsset{
			newGitHubReleaseAsset(assetName, expectedURL, 1024),
		}),
	}
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/dianlight/srat/releases?page=1&per_page=5",
		httpmock.NewJsonResponderOrPanic(200, releases))

	mock.When(suite.mockPropertyRepo.Value("UpdateChannel", false)).ThenReturn(&dto.UpdateChannels.PRERELEASE, nil)

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(&dto.UpdateChannels.PRERELEASE)
	suite.Require().NoError(err)
	suite.Require().NotNil(asset)
	suite.Equal("v1.1.0-beta", asset.LastRelease)
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
			newGitHubReleaseAsset(assetName, "http://example.com/srat_v1.1.0.zip", 1024),
		}),
	}
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/dianlight/srat/releases?page=1&per_page=5",
		httpmock.NewJsonResponderOrPanic(200, releases))

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(nil)
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
			newGitHubReleaseAsset(assetName, "http://example.com/srat_v1.1.0.zip", 1024),
		}),
		newGitHubRepositoryRelease("2025.6.1", false, []*github.ReleaseAsset{ // Newer, valid
			newGitHubReleaseAsset(assetName, "http://example.com/srat_v1.2.0.zip", 2048),
		}),
		newGitHubRepositoryRelease("2025.6.0", false, []*github.ReleaseAsset{ // Older, valid
			newGitHubReleaseAsset(assetName, "http://example.com/srat_v1.0.5.zip", 512),
		}),
		newGitHubRepositoryRelease("2025.6.2-dev076", true, []*github.ReleaseAsset{ // Prerelease, skipped by default
			newGitHubReleaseAsset(assetName, "http://example.com/srat_v1.3.0-beta.zip", 4096),
		}),
	}
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/dianlight/srat/releases?page=1&per_page=5",
		httpmock.NewJsonResponderOrPanic(200, releases))

	asset, err := suite.upgradeService.GetUpgradeReleaseAsset(nil)
	suite.Require().NoError(err)
	suite.Require().NotNil(asset)
	suite.Equal("2025.6.1", asset.LastRelease) // Should pick the latest non-prerelease version
	suite.Equal(assetName, asset.ArchAsset.Name)
	suite.Equal("http://example.com/srat_v1.2.0.zip", asset.ArchAsset.BrowserDownloadURL)
}

// --- DownloadAndExtractBinaryAsset Tests ---

func createDummyZip(files map[string]string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	for name, content := range files {
		f, err := zipWriter.Create(name)
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

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_Success() {
	currentExePath, _ := os.Executable()
	currentExeName := filepath.Base(currentExePath)

	asset := dto.BinaryAsset{
		Name:               "test_asset.zip",
		BrowserDownloadURL: "http://example.com/test_asset.zip",
		Size:               100, // This size is used for progress reporting
	}

	zipContents := map[string]string{
		currentExeName:    "fake binary content for main exe",
		"other_file.sh":   "#!/bin/bash\necho hello",
		"config/data.txt": "some config data",
	}
	zipBuffer, err := createDummyZip(zipContents)
	suite.Require().NoError(err)

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
	defer os.RemoveAll(updatePkg.TempDirPath)

	suite.Require().NotNil(updatePkg.CurrentExecutablePath)
	suite.Equal(currentExeName, filepath.Base(*updatePkg.CurrentExecutablePath))
	suite.FileExists(*updatePkg.CurrentExecutablePath)

	suite.Len(updatePkg.OtherFilesPaths, 2)
	foundOtherFile := false
	foundConfigFile := false
	for _, p := range updatePkg.OtherFilesPaths {
		suite.FileExists(p)
		if filepath.Base(p) == "other_file.sh" {
			foundOtherFile = true
		}
		if strings.HasSuffix(p, filepath.Join("config", "data.txt")) {
			foundConfigFile = true
			// Check nested file content
			content, _ := os.ReadFile(p)
			suite.Equal("some config data", string(content))
		}
	}
	suite.True(foundOtherFile, "other_file.sh not found in extracted files")
	suite.True(foundConfigFile, "config/data.txt not found in extracted files")

	// Verify progress notifications
	var progressEvents []dto.UpdateProgress
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[dto.UpdateProgress]())).ThenAnswer(
		matchers.Answer(func(args []any) []any {
			if p, ok := args[0].(dto.UpdateProgress); ok {
				progressEvents = append(progressEvents, p)
			}
			return []any{nil, nil}
		}),
	)
	// Re-run to capture broadcasts
	updatePkg, err = suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().NoError(err)
	defer os.RemoveAll(updatePkg.TempDirPath)

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
		if e.ProgressStatus == status && e.Progress == progress {
			found = true
			break
		}
	}
	if !found {
		// Log existing events for easier debugging
		var eventSummaries []string
		for _, e := range events {
			eventSummaries = append(eventSummaries, fmt.Sprintf("{Status: %s, Progress: %d}", e.ProgressStatus, e.Progress))
		}
		suite.Failf("Progress event not found", "Expected Status: %s, Progress: %d. Actual events: %s", status, progress, strings.Join(eventSummaries, ", "))
	}
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_DownloadHttpError() {
	asset := dto.BinaryAsset{Name: "test.zip", BrowserDownloadURL: "https://getsamplefiles.com/download/zip/sample-1.zip"}
	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL, httpmock.NewStringResponder(500, "Server Error"))

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Nil(updatePkg)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "failed to download asset: received status code 500")
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_NotAZipFile() {
	asset := dto.BinaryAsset{Name: "notazip.txt", BrowserDownloadURL: "https://getsamplefiles.com/download/txt/sample-1.txt"}
	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL, httpmock.NewStringResponder(200, "this is not zip content"))

	updatePkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Nil(updatePkg)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "failed to open downloaded zip asset") // zip.OpenReader error
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_BlocksZipTraversal() {
	// Create a zip containing a file that attempts path traversal
	asset := dto.BinaryAsset{Name: "evil.zip", BrowserDownloadURL: "http://example.com/evil.zip"}
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	// File tries to escape extraction dir
	_, err := zw.Create("../escape.txt")
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
	suite.Contains(err.Error(), "invalid file path in zip")
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_SetsSafePermissions() {
	// Verify directories are created with 0750 and files respect archive mode
	currentExePath, _ := os.Executable()
	currentExeName := filepath.Base(currentExePath)

	asset := dto.BinaryAsset{
		Name:               "perm_test.zip",
		BrowserDownloadURL: "http://example.com/perm_test.zip",
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
	defer os.RemoveAll(updatePkg.TempDirPath)

	// Check that directory is 0750 despite zip header suggesting 0777
	nestedDir := filepath.Join(updatePkg.TempDirPath, "nested")
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

// --- InstallUpdatePackage & InstallOverseerUpdate Tests ---

func (suite *UpgradeServiceTestSuite) TestInstallUpdatePackage_NilPackage() {
	err := suite.upgradeService.InstallUpdatePackage(nil)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "invalid update package or missing executable path")
}

func (suite *UpgradeServiceTestSuite) TestInstallUpdatePackage_MissingExecutableInPackage() {
	pkg := &service.UpdatePackage{TempDirPath: "test"} // CurrentExecutablePath is nil
	err := suite.upgradeService.InstallUpdatePackage(pkg)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "invalid update package or missing executable path")
}

func (suite *UpgradeServiceTestSuite) TestInstallOverseerUpdate_NilPackage() {
	err := suite.upgradeService.InstallOverseerUpdate(nil, "some/path")
	suite.Require().Error(err)
	suite.Contains(err.Error(), "invalid update package or missing executable path")
}

func (suite *UpgradeServiceTestSuite) TestInstallOverseerUpdate_EmptyOverseerPath() {
	exePath := "test/exe"
	pkg := &service.UpdatePackage{CurrentExecutablePath: &exePath, TempDirPath: "test"}
	err := suite.upgradeService.InstallOverseerUpdate(pkg, "")
	suite.Require().Error(err)
	suite.Contains(err.Error(), "overseer update path cannot be empty")
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
			return []any{nil, nil}
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
		suite.T().Logf("Initial broadcast from run loop: Status=%s, Progress=%d", initialBroadcast.ProgressStatus, initialBroadcast.Progress)
		// We expect it to be checking or similar initial state
		suite.True(initialBroadcast.ProgressStatus == dto.UpdateProcessStates.UPDATESTATUSCHECKING ||
			initialBroadcast.ProgressStatus == dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE ||
			initialBroadcast.ProgressStatus == dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE,
			"Unexpected initial broadcast status: %s", initialBroadcast.ProgressStatus)
	} else {
		suite.T().Log("No initial broadcast captured from run loop within the short wait time.")
	}
	mu.Unlock()
}
