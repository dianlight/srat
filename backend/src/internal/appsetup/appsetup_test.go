package appsetup

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
	"github.com/dianlight/srat/homeassistant/core_api"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/host"
	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/homeassistant/resolution"
	"github.com/dianlight/srat/homeassistant/root"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/service"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

func TestNewFXLoggerOption(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	app := fx.New(
		fx.Provide(func() *slog.Logger { return logger }),
		NewFXLoggerOption(),
	)

	require.NoError(t, app.Err())
}

func TestProvideHAClientDependencies(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	params := BaseAppParams{
		Ctx:      ctx,
		CancelFn: cancel,
		StaticConfig: &dto.ContextState{
			SupervisorURL:   "http://example.org",
			SupervisorToken: "token",
		},
	}

	var (
		addonsClient     addons.ClientWithResponsesInterface
		hardwareClient   hardware.ClientWithResponsesInterface
		mountClient      mount.ClientWithResponsesInterface
		hostClient       host.ClientWithResponsesInterface
		resolutionClient resolution.ClientWithResponsesInterface
		coreAPIClient    core_api.ClientWithResponsesInterface
		rootClient       root.ClientWithResponsesInterface
		ingressClient    ingress.ClientWithResponsesInterface
		websocketClient  websocket.ClientInterface
	)

	app := fx.New(
		ProvideHAClientDependencies(params),
		fx.Populate(
			&addonsClient,
			&hardwareClient,
			&mountClient,
			&hostClient,
			&resolutionClient,
			&coreAPIClient,
			&rootClient,
			&ingressClient,
			&websocketClient,
		),
	)
	require.NoError(t, app.Err())
	require.NoError(t, app.Start(context.Background()))
	t.Cleanup(func() { _ = app.Stop(context.Background()) })

	require.NotNil(t, addonsClient)
	require.NotNil(t, hardwareClient)
	require.NotNil(t, mountClient)
	require.NotNil(t, hostClient)
	require.NotNil(t, resolutionClient)
	require.NotNil(t, coreAPIClient)
	require.NotNil(t, rootClient)
	require.NotNil(t, ingressClient)
	if client, ok := addonsClient.(*addons.ClientWithResponses); ok {
		if core, ok := client.ClientInterface.(*addons.Client); ok {
			assert.Equal(t, "http://example.org/", core.Server)
		} else {
			t.Fatalf("unexpected addons client interface type %T", client.ClientInterface)
		}
	} else {
		t.Fatalf("unexpected addons client type %T", addonsClient)
	}

	require.NotNil(t, websocketClient)
}

func TestProvideCoreDependenciesReturnsOption(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	params := BaseAppParams{
		Ctx:      ctx,
		CancelFn: cancel,
		StaticConfig: &dto.ContextState{
			DatabasePath: "file::memory:?cache=shared",
		},
	}

	option := ProvideCoreDependencies(params)
	require.NotNil(t, option)
}

type shareServiceStub struct{}

func (shareServiceStub) SaveAll(*[]dto.SharedResource) errors.E          { return nil }
func (shareServiceStub) ListShares() ([]dto.SharedResource, errors.E)    { return nil, nil }
func (shareServiceStub) GetShare(string) (*dto.SharedResource, errors.E) { return nil, nil }
func (shareServiceStub) CreateShare(dto.SharedResource) (*dto.SharedResource, errors.E) {
	return nil, nil
}
func (shareServiceStub) UpdateShare(string, dto.SharedResource) (*dto.SharedResource, errors.E) {
	return nil, nil
}
func (shareServiceStub) DeleteShare(string) errors.E                             { return nil }
func (shareServiceStub) DisableShare(string) (*dto.SharedResource, errors.E)     { return nil, nil }
func (shareServiceStub) EnableShare(string) (*dto.SharedResource, errors.E)      { return nil, nil }
func (shareServiceStub) GetShareFromPath(string) (*dto.SharedResource, errors.E) { return nil, nil }
func (shareServiceStub) SetShareFromPathEnabled(string, bool) (*dto.SharedResource, errors.E) {
	return nil, nil
}
func (shareServiceStub) NotifyClient()                            {}
func (shareServiceStub) VerifyShare(*dto.SharedResource) errors.E { return nil }

type volumeServiceStub struct{}

func (volumeServiceStub) MountVolume(*dto.MountPointData) errors.E  { return nil }
func (volumeServiceStub) UnmountVolume(string, bool, bool) errors.E { return nil }
func (volumeServiceStub) GetVolumesData() *[]dto.Disk               { return nil }
func (volumeServiceStub) PathHashToPath(string) (string, errors.E)  { return "", nil }

// func (volumeServiceStub) EjectDisk(string) error                    { return nil }
func (volumeServiceStub) UpdateMountPointSettings(string, dto.MountPointData) (*dto.MountPointData, errors.E) {
	return nil, nil
}
func (volumeServiceStub) PatchMountPointSettings(string, dto.MountPointData) (*dto.MountPointData, errors.E) {
	return nil, nil
}

// func (volumeServiceStub) NotifyClient()                                               {}
func (volumeServiceStub) CreateAutomountFailureNotification(string, string, errors.E) {}
func (volumeServiceStub) CreateUnmountedPartitionNotification(string, string)         {}
func (volumeServiceStub) DismissAutomountNotification(string, string)                 {}
func (volumeServiceStub) CheckUnmountedAutomountPartitions() errors.E                 { return nil }
func (volumeServiceStub) MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error))  {}
func (volumeServiceStub) CreateBlockDevice(string) error                              { return nil }

func TestProvideCyclicDependencyWorkaroundOption(t *testing.T) {
	app := fx.New(
		fx.Provide(func() service.ShareServiceInterface { return shareServiceStub{} }),
		fx.Provide(func() service.VolumeServiceInterface { return volumeServiceStub{} }),
		ProvideCyclicDependencyWorkaroundOption(),
	)
	require.NoError(t, app.Err())
}

func TestProvideFrontendOption(t *testing.T) {
	original := internal.Frontend
	internal.Frontend = nil
	t.Cleanup(func() { internal.Frontend = original })

	var fs http.FileSystem
	app := fx.New(
		ProvideFrontendOption(),
		fx.Populate(&fs),
	)
	require.NoError(t, app.Err())
	require.NoError(t, app.Start(context.Background()))
	t.Cleanup(func() { _ = app.Stop(context.Background()) })

	require.NotNil(t, fs)
}
