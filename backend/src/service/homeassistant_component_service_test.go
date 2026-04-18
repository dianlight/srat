package service_test

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type HomeAssistantComponentServiceSuite struct {
	suite.Suite
	app        *fxtest.App
	state      *dto.ContextState
	service    service.HomeAssistantComponentServiceInterface
	problemSvc service.ProblemServiceInterface
	tempRoot   string
}

func TestHomeAssistantComponentServiceSuite(t *testing.T) {
	suite.Run(t, new(HomeAssistantComponentServiceSuite))
}

func (suite *HomeAssistantComponentServiceSuite) SetupTest() {
	suite.tempRoot = suite.T().TempDir()
	suite.state = &dto.ContextState{
		CustomComponentsPath: suite.tempRoot,
	}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState { return suite.state },
			mock.Mock[service.ProblemServiceInterface],
			service.NewHomeAssistantComponentService,
		),
		fx.Populate(&suite.service),
		fx.Populate(&suite.problemSvc),
	)
	suite.app.RequireStart()
}

func (suite *HomeAssistantComponentServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *HomeAssistantComponentServiceSuite) TestGetStatus_MissingAndDisconnected() {
	status, err := suite.service.GetStatus()
	suite.Require().NoError(err)
	suite.Equal(dto.HomeAssistantComponentSRAT, status.Component)
	suite.False(status.Installed)
	suite.False(status.Connected)
	suite.Nil(status.InstalledVersion)
	suite.Nil(status.ConnectedVersion)
}

func (suite *HomeAssistantComponentServiceSuite) TestGetStatus_InstalledWithManifestVersion() {
	componentDir := filepath.Join(suite.tempRoot, dto.CustomComponentSRATName)
	err := os.MkdirAll(componentDir, 0o755)
	suite.Require().NoError(err)
	err = os.WriteFile(filepath.Join(componentDir, "manifest.json"), []byte(`{"version":"2026.04.1"}`), 0o644)
	suite.Require().NoError(err)

	status, err := suite.service.GetStatus()
	suite.Require().NoError(err)
	suite.True(status.Installed)
	suite.NotNil(status.InstalledVersion)
	suite.Equal("2026.04.1", *status.InstalledVersion)
	suite.False(status.Connected)
}

func (suite *HomeAssistantComponentServiceSuite) TestGetStatus_ConnectedFromWebSocketState() {
	connectedAt := time.Now().Add(-time.Minute)
	haVersion := "2026.4.0"
	entryID := "entry-123"
	suite.state.HAWsComponent = &dto.HomeAssistantComponentConnection{
		Component:   dto.HomeAssistantComponentSRAT,
		Version:     "2026.04.2",
		HAVersion:   &haVersion,
		EntryID:     &entryID,
		ConnectedAt: connectedAt,
	}

	status, err := suite.service.GetStatus()
	suite.Require().NoError(err)
	suite.True(status.Connected)
	suite.NotNil(status.ConnectedVersion)
	suite.Equal("2026.04.2", *status.ConnectedVersion)
	suite.NotNil(status.ConnectedAt)
	suite.Equal(entryID, *status.EntryID)
	suite.Equal(haVersion, *status.HAVersion)
}

func (suite *HomeAssistantComponentServiceSuite) TestUninstall_RemovesComponentDirectory() {
	componentDir := filepath.Join(suite.tempRoot, dto.CustomComponentSRATName)
	err := os.MkdirAll(componentDir, 0o755)
	suite.Require().NoError(err)
	err = os.WriteFile(filepath.Join(componentDir, "manifest.json"), []byte(`{"version":"2026.04.1"}`), 0o644)
	suite.Require().NoError(err)

	err = suite.service.Uninstall(suite.T().Context())
	suite.Require().NoError(err)

	_, statErr := os.Stat(componentDir)
	suite.True(os.IsNotExist(statErr), "component directory should be removed")
}

func (suite *HomeAssistantComponentServiceSuite) TestUninstall_MissingDirectoryIsNoop() {
	err := suite.service.Uninstall(suite.T().Context())
	suite.NoError(err)
}

func (suite *HomeAssistantComponentServiceSuite) TestInstallOrUpgradeFromZip_InstallsComponentFiles() {
	zipContent := createCustomComponentArchive(suite.T(), map[string]string{
		"srat/manifest.json": `{"version":"2026.05.1"}`,
		"srat/__init__.py":   "# init",
		"srat/sensor.py":     "# sensor",
	})

	err := suite.service.InstallOrUpgradeFromZip(suite.T().Context(), zipContent)
	suite.Require().NoError(err)

	status, err := suite.service.GetStatus()
	suite.Require().NoError(err)
	suite.True(status.Installed)
	suite.Require().NotNil(status.InstalledVersion)
	suite.Equal("2026.05.1", *status.InstalledVersion)

	componentDir := filepath.Join(suite.tempRoot, dto.CustomComponentSRATName)
	suite.FileExists(filepath.Join(componentDir, "manifest.json"))
	suite.FileExists(filepath.Join(componentDir, "sensor.py"))
}

func (suite *HomeAssistantComponentServiceSuite) TestInstallOrUpgradeFromZip_RejectsZipSlipEntries() {
	zipContent := createCustomComponentArchive(suite.T(), map[string]string{
		"../escape.txt": "owned",
	})

	err := suite.service.InstallOrUpgradeFromZip(suite.T().Context(), zipContent)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "illegal file path")
}

func (suite *HomeAssistantComponentServiceSuite) TestInstallOrUpgradeFromZip_UpgradeWhenAlreadyInstalled() {
	componentDir := filepath.Join(suite.tempRoot, dto.CustomComponentSRATName)
	suite.Require().NoError(os.MkdirAll(componentDir, 0o755))
	suite.Require().NoError(os.WriteFile(filepath.Join(componentDir, "manifest.json"), []byte(`{"version":"2026.04.1"}`), 0o644))
	suite.Require().NoError(os.WriteFile(filepath.Join(componentDir, "legacy.py"), []byte("# stale"), 0o644))

	zipContent := createCustomComponentArchive(suite.T(), map[string]string{
		"srat/manifest.json": `{"version":"2026.04.9"}`,
		"srat/__init__.py":   "# upgraded",
		"srat/new_sensor.py": "# new",
	})

	err := suite.service.InstallOrUpgradeFromZip(suite.T().Context(), zipContent)
	suite.Require().NoError(err)

	status, err := suite.service.GetStatus()
	suite.Require().NoError(err)
	suite.True(status.Installed)
	suite.Require().NotNil(status.InstalledVersion)
	suite.Equal("2026.04.9", *status.InstalledVersion)

	_, err = os.Stat(filepath.Join(componentDir, "legacy.py"))
	suite.True(os.IsNotExist(err), "stale files from previous install should be removed")
	suite.FileExists(filepath.Join(componentDir, "new_sensor.py"))
}

func (suite *HomeAssistantComponentServiceSuite) TestSyncIssueStatus_CreatesIssueWhenMissingDisconnected() {
	status := &dto.HomeAssistantCustomComponentStatus{
		Installed: false,
		Connected: false,
	}

	mock.When(suite.problemSvc.Upsert(mock.Any[*dto.Problem]())).ThenReturn(new(dto.Problem{}), nil)

	err := suite.service.SyncIssueStatus(status)
	suite.Require().NoError(err)

	_, _ = mock.Verify(suite.problemSvc, matchers.Times(1)).Upsert(mock.Any[*dto.Problem]())
}

func (suite *HomeAssistantComponentServiceSuite) TestSyncIssueStatus_ResolvesIssueWhenInstalled() {
	status := &dto.HomeAssistantCustomComponentStatus{
		Installed: true,
		Connected: false,
	}

	mock.When(suite.problemSvc.Dismiss(mock.Exact("custom_component_missing"))).ThenReturn(nil)

	err := suite.service.SyncIssueStatus(status)
	suite.Require().NoError(err)

	_ = mock.Verify(suite.problemSvc, matchers.Times(1)).Dismiss(mock.Exact("custom_component_missing"))
}

func (suite *HomeAssistantComponentServiceSuite) TestUpsertRestartRequiredRepair_UsesProblemService() {
	mock.When(suite.problemSvc.Upsert(mock.Any[*dto.Problem]())).ThenReturn(new(dto.Problem{}), nil)

	err := suite.service.UpsertRestartRequiredRepair(context.Background())
	suite.Require().NoError(err)

	_, _ = mock.Verify(suite.problemSvc, matchers.Times(1)).Upsert(mock.Any[*dto.Problem]())
}

func (suite *HomeAssistantComponentServiceSuite) TestDismissRestartRequiredRepair_UsesProblemService() {
	mock.When(suite.problemSvc.Dismiss(mock.Exact("custom_component_restart_required"))).ThenReturn(nil)

	err := suite.service.DismissRestartRequiredRepair(context.Background())
	suite.Require().NoError(err)

	_ = mock.Verify(suite.problemSvc, matchers.Times(1)).Dismiss(mock.Exact("custom_component_restart_required"))
}

// TestGetStatus_CanUpgrade_NotInstalledIsFalse verifies CanUpgrade is false when not installed.
func (suite *HomeAssistantComponentServiceSuite) TestGetStatus_CanUpgrade_NotInstalledIsFalse() {
	status, err := suite.service.GetStatus()
	suite.Require().NoError(err)
	suite.False(status.Installed)
	suite.False(status.CanUpgrade, "CanUpgrade must be false when component is not installed")
}

// TestGetStatus_CanUpgrade_InstalledVersionNewerThanLatestIsFalse verifies that CanUpgrade
// is false when the installed version is newer than the embedded latest version.
func (suite *HomeAssistantComponentServiceSuite) TestGetStatus_CanUpgrade_InstalledVersionNewerThanLatestIsFalse() {
	componentDir := filepath.Join(suite.tempRoot, dto.CustomComponentSRATName)
	suite.Require().NoError(os.MkdirAll(componentDir, 0o755))
	suite.Require().NoError(os.WriteFile(
		filepath.Join(componentDir, "manifest.json"),
		[]byte(`{"version":"9999.99.99"}`), 0o644,
	))

	status, err := suite.service.GetStatus()
	suite.Require().NoError(err)
	suite.True(status.Installed)
	suite.False(status.CanUpgrade, "CanUpgrade must be false when installed version is newer than latest")
}

// TestGetStatus_CanUpgrade_InstalledVersionSameAsLatestIsFalse verifies that CanUpgrade
// is false when the installed version equals the embedded latest version.
func (suite *HomeAssistantComponentServiceSuite) TestGetStatus_CanUpgrade_InstalledVersionSameAsLatestIsFalse() {
	componentDir := filepath.Join(suite.tempRoot, dto.CustomComponentSRATName)
	suite.Require().NoError(os.MkdirAll(componentDir, 0o755))

	status, err := suite.service.GetStatus()
	// Use the actual LatestVersion to create a matching InstalledVersion
	suite.Require().NoError(err)
	suite.Require().NotNil(status.LatestVersion)
	latestVersion := *status.LatestVersion

	suite.Require().NoError(os.WriteFile(
		filepath.Join(componentDir, "manifest.json"),
		[]byte(`{"version":"`+latestVersion+`"}`), 0o644,
	))

	status, err = suite.service.GetStatus()
	suite.Require().NoError(err)
	suite.True(status.Installed)
	suite.False(status.CanUpgrade, "CanUpgrade must be false when installed version equals latest version")
}

func createCustomComponentArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	buffer := new(bytes.Buffer)
	writer := zip.NewWriter(buffer)

	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create archive entry %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write archive entry %s: %v", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close archive writer: %v", err)
	}

	return buffer.Bytes()
}
