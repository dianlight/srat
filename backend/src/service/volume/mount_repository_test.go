// Package volume_test holds black-box suites for the volume subpackage.
package volume_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service/volume"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

// volumeTestFixture wires a real temp-file SQLite repository, a real DiskMap
// and a real event bus with the MountRepository under test.
type volumeTestFixture struct {
	suite.Suite
	app         *fxtest.App
	repo        *volume.MountRepository
	mountRepoIf repository.MountPointPathRepositoryInterface
	eventBus    events.EventBusInterface
	disks       *dto.DiskMap
	ctx         context.Context
	cancel      context.CancelFunc
	db          *gorm.DB
}

func (f *volumeTestFixture) SetupTest() {
	f.app = fxtest.New(f.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{DatabasePath: filepath.Join(f.T().TempDir(), "test.db")}
			},
			func() *dto.DiskMap { return dto.NewDiskMap() },
			dbom.NewDB,
			events.NewEventBus,
			repository.NewMountPointPathRepository,
		),
		fx.Populate(&f.ctx),
		fx.Populate(&f.cancel),
		fx.Populate(&f.db),
		fx.Populate(&f.disks),
		fx.Populate(&f.eventBus),
	)
	f.app.RequireStart()
	mountRepo := repository.NewMountPointPathRepository(f.db)
	f.mountRepoIf = mountRepo
	f.repo = volume.NewMountRepository(f.ctx, mountRepo, f.disks, f.eventBus)
}

func (f *volumeTestFixture) TearDownTest() {
	if f.cancel != nil {
		f.cancel()
	}
	if f.ctx != nil {
		if wg := f.ctx.Value(ctxkeys.WaitGroup); wg != nil {
			wg.(*sync.WaitGroup).Wait()
		}
	}
	if f.app != nil {
		f.app.RequireStop()
	}
}

func seedPartition(diskID, partID string) *dto.Partition {
	return &dto.Partition{Id: &partID, DiskId: &diskID}
}

// MountRepositoryTestSuite covers LoadByDevice/LoadByPath/Persist/Reconcile/
// Evict against in-memory SQLite semantics via temp-file DBs.
type MountRepositoryTestSuite struct {
	volumeTestFixture
}

func TestMountRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(MountRepositoryTestSuite))
}

func (s *MountRepositoryTestSuite) TestPersistAndLoadByPath_RoundTrip() {
	md := dto.MountPointData{
		Path:        "/mnt/repo-roundtrip",
		Root:        "/",
		DeviceId:    "part-repo-1",
		Type:        "ADDON",
		Flags:       &dto.MountFlags{},
		CustomFlags: &dto.MountFlags{},
	}
	s.Require().NoError(s.repo.Persist(&md))

	got, found, err := s.repo.LoadByPath("/mnt/repo-roundtrip", "/")
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal("/mnt/repo-roundtrip", got.Path)
	s.Equal("part-repo-1", got.DeviceId)
}

func (s *MountRepositoryTestSuite) TestLoadByPath_Missing() {
	got, found, err := s.repo.LoadByPath("/mnt/does-not-exist", "/")
	s.Require().NoError(err)
	s.False(found)
	s.Nil(got)
}

func (s *MountRepositoryTestSuite) TestLoadByDevice_EmptyID() {
	got, err := s.repo.LoadByDevice(&dto.Partition{})
	s.Require().NoError(err)
	s.Nil(got)
}

func (s *MountRepositoryTestSuite) TestReconcile_PruneStaleOrphansKeepsLive() {
	part := seedPartition("disk-repo-1", "part-repo-1")
	// Two rows for one device: one live (startup-marked), one orphan.
	startup := true
	for _, md := range []dto.MountPointData{
		{Path: "/mnt/live_repo", Root: "/", DeviceId: "part-repo-1", Type: "ADDON", IsToMountAtStartup: &startup, Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{}},
		{Path: "/mnt/gone_repo", Root: "/", DeviceId: "part-repo-1", Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{}},
	} {
		s.Require().NoError(s.repo.Persist(&md))
	}

	loaded, err := s.repo.LoadByDevice(part)
	s.Require().NoError(err)
	_, liveOK := loaded["/mnt/live_repo"]
	s.True(liveOK, "live row must survive reconciliation")
	_, goneOK := loaded["/mnt/gone_repo"]
	s.False(goneOK, "stale orphan must be evicted")

	_, found, err := s.repo.LoadByPath("/mnt/gone_repo", "/")
	s.Require().NoError(err)
	s.False(found, "evicted row must be hard-deleted")
}

func (s *MountRepositoryTestSuite) TestReconcile_SingletonNeverPruned() {
	part := seedPartition("disk-repo-2", "part-repo-2")
	md := dto.MountPointData{
		Path: "/mnt/singleton_repo", Root: "/", DeviceId: "part-repo-2",
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}
	s.Require().NoError(s.repo.Persist(&md))

	loaded, err := s.repo.LoadByDevice(part)
	s.Require().NoError(err)
	s.Len(loaded, 1, "singleton pending config must never be pruned")
}

func (s *MountRepositoryTestSuite) TestEvict_EmitsDiskUpdateAfterCacheRemoval() {
	diskID := "disk-repo-3"
	partID := "part-repo-3"
	md := dto.MountPointData{
		Path: "/mnt/evict_repo", Root: "/", DeviceId: partID,
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}
	s.Require().NoError(s.repo.Persist(&md))

	diskIDCopy, partIDCopy := diskID, partID
	part := dto.Partition{Id: &partIDCopy, DiskId: &diskIDCopy}
	parts := map[string]dto.Partition{partID: part}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))
	s.Require().NoError(s.disks.AddOrUpdateMountPoint(diskID, partID, md))

	var emitted []events.DiskEvent
	var stillCached []bool
	unsub := s.eventBus.OnDisk(func(_ context.Context, e events.DiskEvent) errors.E {
		emitted = append(emitted, e)
		// #971: the bus dispatches synchronously, so the cache must already
		// reflect the eviction when the listener runs.
		_, found := s.disks.GetMountPoint(diskID, partID, "/mnt/evict_repo")
		stillCached = append(stillCached, found)
		return nil
	})
	defer unsub()

	s.repo.Evict(seedPartition(diskID, partID), "/mnt/evict_repo")

	s.Require().NotEmpty(emitted, "eviction must emit a Disk UPDATE after cache removal")
	s.Equal(events.EventTypes.UPDATE, emitted[len(emitted)-1].Type)
	s.Require().NotEmpty(stillCached, "emit listener must have run")
	s.False(stillCached[len(stillCached)-1], "cache must be mutated before the emit")
	_, found, err := s.repo.LoadByPath("/mnt/evict_repo", "/")
	s.Require().NoError(err)
	s.False(found)
}

func (s *MountRepositoryTestSuite) TestReconcile_MergesDashDuplicateOntoLive() {
	part := seedPartition("disk-repo-4", "part-repo-4")
	startup := true
	// Live underscores row without the automount flag.
	s.Require().NoError(s.repo.Persist(&dto.MountPointData{
		Path: "/mnt/ata_WD_part4", Root: "/", DeviceId: "part-repo-4",
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))
	// Legacy dashes row carrying the automount intent.
	s.Require().NoError(s.repo.Persist(&dto.MountPointData{
		Path: "/mnt/ata-WD-part4", Root: "/", DeviceId: "part-repo-4",
		Type: "ADDON", IsToMountAtStartup: &startup,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))

	loaded, err := s.repo.LoadByDevice(part)
	s.Require().NoError(err)
	s.Len(loaded, 1, "collision must merge onto a single survivor")
	live, ok := loaded["/mnt/ata_WD_part4"]
	s.Require().True(ok, "underscores row must survive")
	s.Require().NotNil(live.IsToMountAtStartup)
	s.True(*live.IsToMountAtStartup, "automount intent must migrate to the survivor")

	_, found, err := s.repo.LoadByPath("/mnt/ata-WD-part4", "/")
	s.Require().NoError(err)
	s.False(found, "legacy dashes row must be hard-deleted")
}

func (s *MountRepositoryTestSuite) TestReconcile_KeepsMountedDashRow() {
	// IsMounted is derived live from the mount table during conversion, so
	// mock the table with the dashes path mounted.
	restore := osutil.MockMountInfo("1 1 0:1 / /mnt/ata-WD-part5 rw,relatime - ext4 /dev/sdz5 rw\n")
	s.T().Cleanup(restore)

	part := seedPartition("disk-repo-5", "part-repo-5")
	startup := true
	s.Require().NoError(s.repo.Persist(&dto.MountPointData{
		Path: "/mnt/ata_WD_part5", Root: "/", DeviceId: "part-repo-5",
		Type: "ADDON", IsToMountAtStartup: &startup,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))
	s.Require().NoError(s.repo.Persist(&dto.MountPointData{
		Path: "/mnt/ata-WD-part5", Root: "/", DeviceId: "part-repo-5",
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))

	loaded, err := s.repo.LoadByDevice(part)
	s.Require().NoError(err)
	s.Len(loaded, 2, "mounted dash row must survive the collision")
	_, ok := loaded["/mnt/ata-WD-part5"]
	s.True(ok)
}
