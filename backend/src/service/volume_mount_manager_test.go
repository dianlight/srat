// Package service_test contains tests for the service layer.
package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// VolumeMountManagerTestSuite exercises the OS-level unmount flag mapping in
// volumeMountManager.Unmount (H6): a normal unmount must pass no flags so a
// busy filesystem surfaces the real error, and a force unmount must detach
// lazily (MNT_DETACH) so it always succeeds.
type VolumeMountManagerTestSuite struct {
	suite.Suite
	app       *fxtest.App
	mounter   service.VolumeMountManagerInterface
	mockFsSvc service.FilesystemServiceInterface
	ctrl      *matchers.MockController
	ctx       context.Context
	cancel    context.CancelFunc
	disks     *dto.DiskMap
	eventBus  events.EventBusInterface
}

func TestVolumeMountManagerTestSuite(t *testing.T) {
	suite.Run(t, new(VolumeMountManagerTestSuite))
}

func (suite *VolumeMountManagerTestSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)"}
			},
			func() *dto.DiskMap { return dto.NewDiskMap() },
			events.NewEventBus,
			service.NewVolumeMountManager,
			mock.Mock[service.FilesystemServiceInterface],
		),
		fx.Populate(&suite.mounter),
		fx.Populate(&suite.mockFsSvc),
		fx.Populate(&suite.ctrl),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.disks),
		fx.Populate(&suite.eventBus),
	)
	suite.app.RequireStart()
}

func (suite *VolumeMountManagerTestSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.ctx != nil {
		if wg := suite.ctx.Value(ctxkeys.WaitGroup); wg != nil {
			wg.(*sync.WaitGroup).Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// seedMountPoint registers a disk/partition/mount point in the cache so the
// manager can update it after a successful unmount.
func (suite *VolumeMountManagerTestSuite) seedMountPoint() *dto.MountPointData {
	diskID := "disk-h6"
	partID := "part-h6"
	devicePath := "/dev/h6"
	fsType := "ext4"

	disk := &dto.Disk{
		Id: &diskID,
		Partitions: &map[string]dto.Partition{
			partID: {Id: &partID, DiskId: &diskID, DevicePath: &devicePath},
		},
	}
	suite.Require().NoError(suite.disks.AddOrUpdate(disk))

	md := &dto.MountPointData{
		Path:      "/mnt/h6",
		DeviceId:  partID,
		IsMounted: true,
		FSType:    &fsType,
		Partition: &dto.Partition{
			Id:     &partID,
			DiskId: &diskID,
		},
	}
	suite.Require().NoError(suite.disks.AddOrUpdateMountPoint(diskID, partID, *md))
	return md
}

// TestUnmount_NormalPassesNoFlags verifies a normal (non-force) unmount calls
// UnmountPartition with (force=false, lazy=false): no MNT_DETACH, so a busy
// filesystem returns the real error instead of silently detaching.
func (suite *VolumeMountManagerTestSuite) TestUnmount_NormalPassesNoFlags() {
	md := suite.seedMountPoint()

	var capturedForce, capturedLazy bool
	mock.When(suite.mockFsSvc.UnmountPartition(
		mock.AnyContext(), mock.Exact(md.Path), mock.Exact("ext4"), mock.Any[bool](), mock.Any[bool](),
	)).
		ThenAnswer(matchers.Answer(func(args []any) []any {
			capturedForce = args[3].(bool)
			capturedLazy = args[4].(bool)
			return []any{nil}
		}))

	errE := suite.mounter.Unmount(md, false)
	suite.Require().NoError(errE)
	suite.False(capturedForce, "normal unmount must not set MNT_FORCE")
	suite.False(capturedLazy, "normal unmount must not set MNT_DETACH")
	suite.False(md.IsMounted, "successful unmount must mark the mount point as unmounted")
}

// TestUnmount_ForceDetachesLazily verifies a force unmount calls
// UnmountPartition with (force=false, lazy=true): MNT_DETACH guarantees the
// detach succeeds even when files are still open.
func (suite *VolumeMountManagerTestSuite) TestUnmount_ForceDetachesLazily() {
	md := suite.seedMountPoint()

	var capturedForce, capturedLazy bool
	mock.When(suite.mockFsSvc.UnmountPartition(
		mock.AnyContext(), mock.Exact(md.Path), mock.Exact("ext4"), mock.Any[bool](), mock.Any[bool](),
	)).
		ThenAnswer(matchers.Answer(func(args []any) []any {
			capturedForce = args[3].(bool)
			capturedLazy = args[4].(bool)
			return []any{nil}
		}))

	errE := suite.mounter.Unmount(md, true)
	suite.Require().NoError(errE)
	suite.False(capturedForce, "force unmount must not set MNT_FORCE")
	suite.True(capturedLazy, "force unmount must set MNT_DETACH (lazy)")
	suite.False(md.IsMounted, "successful unmount must mark the mount point as unmounted")
}

// TestUnmount_NormalBusyErrorPropagates verifies the busy error from the
// filesystem layer is surfaced to the caller for a normal unmount.
func (suite *VolumeMountManagerTestSuite) TestUnmount_NormalBusyErrorPropagates() {
	md := suite.seedMountPoint()

	mock.When(suite.mockFsSvc.UnmountPartition(
		mock.AnyContext(), mock.Exact(md.Path), mock.Exact("ext4"), mock.Any[bool](), mock.Any[bool](),
	)).
		ThenReturn(errors.New("target is busy"))

	errE := suite.mounter.Unmount(md, false)
	suite.Require().Error(errE)
	suite.ErrorIs(errE, dto.ErrorUnmountFail)
	suite.True(md.IsMounted, "failed unmount must leave the mount point marked as mounted")
}

// TestUnmount_NilMountPoint verifies a nil mount point is rejected.
func (suite *VolumeMountManagerTestSuite) TestUnmount_NilMountPoint() {
	errE := suite.mounter.Unmount(nil, false)
	suite.Require().Error(errE)
	suite.ErrorIs(errE, dto.ErrorInvalidParameter)
}
