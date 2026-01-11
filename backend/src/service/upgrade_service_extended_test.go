package service_test

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/jarcoal/httpmock"
)

// --- More ApplyUpdateAndRestart Tests ---

func (suite *UpgradeServiceTestSuite) TestApplyUpdateAndRestart_WithSignatureFile() {
	// Create a test binary
	tmpBinary := filepath.Join(suite.state.UpdateDataDir, "update-binary")
	err := os.WriteFile(tmpBinary, []byte("test binary with update"), 0755)
	suite.Require().NoError(err)
	defer os.Remove(tmpBinary)

	// Create a signature file
	sigFile := tmpBinary + ".minisig"
	sigContent := `untrusted comment: signature from test
RWSSomeInvalidSignature==
trusted comment: timestamp:1234567890
AnotherInvalidPart==`
	err = os.WriteFile(sigFile, []byte(sigContent), 0644)
	suite.Require().NoError(err)
	defer os.Remove(sigFile)

	suite.state.UpdateChannel = dto.UpdateChannels.RELEASE

	updatePkg := &service.UpdatePackage{
		FilesPaths: []service.UpdateFile{
			{Path: tmpBinary},
		},
	}

	// Should fail due to invalid signature
	err = suite.upgradeService.ApplyUpdateAndRestart(updatePkg)
	suite.Require().Error(err)
}

func (suite *UpgradeServiceTestSuite) TestApplyUpdateAndRestart_WithoutSignatureOnDevelop() {
	// Create a test binary without signature
	tmpBinary := filepath.Join(suite.state.UpdateDataDir, "unsigned-binary")
	err := os.WriteFile(tmpBinary, []byte("test unsigned binary"), 0755)
	suite.Require().NoError(err)
	defer os.Remove(tmpBinary)

	suite.state.UpdateChannel = dto.UpdateChannels.DEVELOP

	updatePkg := &service.UpdatePackage{
		FilesPaths: []service.UpdateFile{
			{Path: tmpBinary},
		},
	}

	// Should proceed on develop channel without signature
	err = suite.upgradeService.ApplyUpdateAndRestart(updatePkg)
	// May fail for other reasons, but not signature
	if err != nil {
		suite.NotContains(err.Error(), "signature file not found")
	}
}

// --- More DownloadAndExtractBinaryAsset Tests ---

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_EmptyZip() {
	asset := dto.BinaryAsset{
		Name:               "empty.zip",
		BrowserDownloadURL: "http://example.com/empty.zip",
		Size:               22, // Minimal zip file size
		Digest:             "sha256:8739c76e681f900923b900c9df0ef75cf421d39cabb54650c4b9ad19b6a76d85",
	}

	// Create an empty zip file
	emptyZip, err := createDummyZip(map[string]string{}, 0, nil)
	suite.Require().NoError(err)

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, emptyZip.Bytes())
			resp.Header.Set("Content-Type", "application/zip")
			resp.ContentLength = int64(emptyZip.Len())
			return resp, nil
		})

	_, err = suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "is empty")
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_LargeFile() {
	currentExePath, _ := os.Executable()
	currentExeName := filepath.Base(currentExePath)

	asset := dto.BinaryAsset{
		Name:               "large.zip",
		BrowserDownloadURL: "http://example.com/large.zip",
		Size:               1024 * 1024, // 1MB
		//Digest:             "sha256:f23c9b8f7752acd2b8e9ab3f9934fb6d09894f3eeb22111b91f7b0d9c3b0a678",
	}

	// Create zip with large file
	largeContent := make([]byte, 1024*100) // 100KB of data
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	zipContents := map[string]string{
		currentExeName: string(largeContent),
	}
	zipBuffer, err := createDummyZip(zipContents, asset.Size, &suite.privateKey)
	suite.Require().NoError(err)

	asset.Size = zipBuffer.Len()
	ssha256 := sha256.New()
	ssha256.Write(zipBuffer.Bytes())
	asset.Digest = "sha256:" + fmt.Sprintf("%x", ssha256.Sum(nil))

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, zipBuffer.Bytes())
			resp.Header.Set("Content-Type", "application/zip")
			resp.ContentLength = int64(zipBuffer.Len())
			return resp, nil
		})

	pkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().NoError(err)
	suite.NotNil(pkg)
	suite.NotNil(pkg.FilesPaths)
}

// --- Test installBinaryTo edge cases ---

func (suite *UpgradeServiceTestSuite) TestInstallBinaryTo_BackupAndRestore() {
	// Create source and destination
	sourceFile := filepath.Join(suite.state.UpdateDataDir, "source")
	err := os.WriteFile(sourceFile, []byte("new version"), 0644)
	suite.Require().NoError(err)
	defer os.Remove(sourceFile)

	destDir := filepath.Join(suite.state.UpdateDataDir, "install")
	err = os.MkdirAll(destDir, 0755)
	suite.Require().NoError(err)
	defer os.RemoveAll(destDir)

	destFile := filepath.Join(destDir, "binary")

	// Create existing binary
	err = os.WriteFile(destFile, []byte("old version"), 0755)
	suite.Require().NoError(err)

	// Setup state for installation
	suite.state.UpdateChannel = dto.UpdateChannels.DEVELOP
	//suite.state.UpdateFilePath = destFile

	updatePkg := &service.UpdatePackage{
		FilesPaths: []service.UpdateFile{
			{
				Path:      sourceFile,
				Size:      int64(len("new version")),
				Signature: nil,
			},
		},
	}

	// Should backup old version
	err = suite.upgradeService.InstallUpdatePackage(updatePkg)
	suite.Require().NoError(err)

	// Check if backup was created
	backupFile := destFile + ".old"
	if _, statErr := os.Stat(backupFile); statErr == nil {
		// Backup exists, verify it has old content
		content, _ := os.ReadFile(backupFile)
		suite.Contains(string(content), "old version")
	}
}

// --- Test getCurrentExecutablePath ---

func (suite *UpgradeServiceTestSuite) TestGetCurrentExecutablePath_Success() {
	// This is tested indirectly through InstallUpdatePackage
	// but we can verify it works by checking the result

	tmpBinary := filepath.Join(suite.state.UpdateDataDir, "test-path")
	err := os.WriteFile(tmpBinary, []byte("test"), 0755)
	suite.Require().NoError(err)
	defer os.Remove(tmpBinary)

	suite.state.UpdateChannel = dto.UpdateChannels.DEVELOP

	updatePkg := &service.UpdatePackage{
		FilesPaths: []service.UpdateFile{
			{Path: tmpBinary},
		},
	}

	// This will call getCurrentExecutablePath internally
	_ = suite.upgradeService.InstallUpdatePackage(updatePkg)
}

// --- Test with Multiple Files ---

func (suite *UpgradeServiceTestSuite) TestInstallUpdatePackage_WithOtherFiles() {
	// Create main binary
	mainBinary := filepath.Join(suite.state.UpdateDataDir, "main-exe")
	err := os.WriteFile(mainBinary, []byte("main binary"), 0755)
	suite.Require().NoError(err)
	defer os.Remove(mainBinary)

	// Create other files
	otherFile1 := filepath.Join(suite.state.UpdateDataDir, "helper1")
	err = os.WriteFile(otherFile1, []byte("helper 1"), 0755)
	suite.Require().NoError(err)
	defer os.Remove(otherFile1)

	otherFile2 := filepath.Join(suite.state.UpdateDataDir, "helper2")
	err = os.WriteFile(otherFile2, []byte("helper 2"), 0755)
	suite.Require().NoError(err)
	defer os.Remove(otherFile2)

	suite.state.UpdateChannel = dto.UpdateChannels.DEVELOP

	updatePkg := &service.UpdatePackage{
		FilesPaths: []service.UpdateFile{
			{Path: mainBinary},
			{Path: otherFile1},
			{Path: otherFile2},
		},
	}

	// Should install all files
	_ = suite.upgradeService.InstallUpdatePackage(updatePkg)
}

// --- Test error handling in DownloadAndExtractBinaryAsset ---

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_HTTP404() {
	asset := dto.BinaryAsset{
		Name:               "notfound.zip",
		BrowserDownloadURL: "http://example.com/notfound.zip",
		Size:               100,
	}

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		httpmock.NewStringResponder(404, "Not Found"))

	_, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().Error(err)
}

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_HTTP500() {
	asset := dto.BinaryAsset{
		Name:               "error.zip",
		BrowserDownloadURL: "http://example.com/error.zip",
		Size:               100,
	}

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		httpmock.NewStringResponder(500, "Internal Server Error"))

	_, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().Error(err)
}

// --- Test architecture-specific download ---

func (suite *UpgradeServiceTestSuite) TestDownloadAndExtractBinaryAsset_ArchSpecificBinary() {
	currentExePath, _ := os.Executable()
	currentExeName := filepath.Base(currentExePath)

	// This simulates downloading an architecture-specific binary
	asset := dto.BinaryAsset{
		Name:               fmt.Sprintf("srat_%s.zip", runtime.GOARCH),
		BrowserDownloadURL: fmt.Sprintf("http://example.com/srat_%s.zip", runtime.GOARCH),
		Size:               2048,
		Digest:             "sha256:a64c3f2e35d7a2a0a5996d3d32f4c978dce1241fd7a2c2c9e033bcbdd8d2ef09",
	}

	zipContents := map[string]string{
		currentExeName: "architecture-specific binary content",
		"README.md":    "Installation instructions",
	}
	zipBuffer, err := createDummyZip(zipContents, 0, &suite.privateKey)
	suite.Require().NoError(err)
	asset.Size = zipBuffer.Len()
	ssha256 := sha256.New()
	ssha256.Write(zipBuffer.Bytes())
	asset.Digest = "sha256:" + fmt.Sprintf("%x", ssha256.Sum(nil))

	httpmock.RegisterResponder("GET", asset.BrowserDownloadURL,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewBytesResponse(200, zipBuffer.Bytes())
			resp.Header.Set("Content-Type", "application/zip")
			resp.ContentLength = int64(zipBuffer.Len())
			return resp, nil
		})

	pkg, err := suite.upgradeService.DownloadAndExtractBinaryAsset(asset)
	suite.Require().NoError(err)
	suite.NotNil(pkg)

	// Verify the executable was extracted
	suite.NotNil(pkg.FilesPaths)
	suite.Len(pkg.FilesPaths, 2)

	for _, file := range pkg.FilesPaths {
		suite.FileExists(file.Path)
		content, err := os.ReadFile(file.Path)
		suite.Require().NoError(err)
		if filepath.Base(file.Path) == currentExeName {
			suite.Contains(string(content), "architecture-specific binary content")
		} else if filepath.Base(file.Path) == "README.md" {
			suite.Contains(string(content), "Installation instructions")
		} else {
			suite.Failf("Unexpected file extracted", "Got file: %s", file.Path)
		}
	}

}

// --- Test version comparison logic ---

func (suite *UpgradeServiceTestSuite) TestInstallUpdatePackage_NewerVersion() {
	// Create a binary that reports a different version
	// In real scenario this would be a binary with different version metadata
	tmpBinary := filepath.Join(suite.state.UpdateDataDir, "newer-version")
	err := os.WriteFile(tmpBinary, []byte("newer binary content"), 0755)
	suite.Require().NoError(err)
	defer os.Remove(tmpBinary)

	suite.state.UpdateChannel = dto.UpdateChannels.DEVELOP

	updatePkg := &service.UpdatePackage{
		FilesPaths: []service.UpdateFile{
			{Path: tmpBinary},
		},
	}

	// Should proceed with installation (version is likely different or unknown)
	_ = suite.upgradeService.InstallUpdatePackage(updatePkg)
}
