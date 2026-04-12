package service_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type HomeAssistantComponentServiceSuite struct {
	suite.Suite
	app      *fxtest.App
	state    *dto.ContextState
	service  service.HomeAssistantComponentServiceInterface
	tempRoot string
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
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState { return suite.state },
			service.NewHomeAssistantComponentService,
		),
		fx.Populate(&suite.service),
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

	err = suite.service.Uninstall()
	suite.Require().NoError(err)

	_, statErr := os.Stat(componentDir)
	suite.True(os.IsNotExist(statErr), "component directory should be removed")
}

func (suite *HomeAssistantComponentServiceSuite) TestUninstall_MissingDirectoryIsNoop() {
	err := suite.service.Uninstall()
	suite.NoError(err)
}
