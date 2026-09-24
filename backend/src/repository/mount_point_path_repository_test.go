// Package repository_test covers MountPointPathRepository against temp-file
// SQLite databases.
package repository_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/repository"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type MountPointPathRepositoryTestSuite struct {
	suite.Suite
	app  *fxtest.App
	repo repository.MountPointPathRepositoryInterface
	ctx  context.Context
}

func TestMountPointPathRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(MountPointPathRepositoryTestSuite))
}

func (s *MountPointPathRepositoryTestSuite) SetupTest() {
	var db *gorm.DB
	var ctx context.Context
	var cancel context.CancelFunc
	s.app = fxtest.New(s.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{DatabasePath: filepath.Join(s.T().TempDir(), "test.db")}
			},
			dbom.NewDB,
		),
		fx.Populate(&db),
		fx.Populate(&ctx),
		fx.Populate(&cancel),
	)
	s.app.RequireStart()
	s.T().Cleanup(func() {
		cancel()
		s.app.RequireStop()
	})
	s.ctx = ctx
	s.repo = repository.NewMountPointPathRepository(db)
}

func (s *MountPointPathRepositoryTestSuite) seed() dto.MountPointData {
	md := dto.MountPointData{
		Path: "/mnt/repo-seed", Root: "/", DeviceId: "part-repo-seed",
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}
	s.Require().NoError(s.repo.Upsert(s.ctx, md))
	return md
}

func (s *MountPointPathRepositoryTestSuite) TestUpsertFindByPath_RoundTrip() {
	md := s.seed()

	rows, err := s.repo.FindByPath(s.ctx, "/mnt/repo-seed", "/")
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.Equal("part-repo-seed", rows[0].DeviceId)
	s.Equal(md.Type, rows[0].Type)
}

func (s *MountPointPathRepositoryTestSuite) TestFindByPath_Missing() {
	rows, err := s.repo.FindByPath(s.ctx, "/mnt/nope", "/")
	s.Require().NoError(err)
	s.Empty(rows)
}

func (s *MountPointPathRepositoryTestSuite) TestFindByDeviceID() {
	s.seed()
	rows, err := s.repo.FindByDeviceID(s.ctx, "part-repo-seed")
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.Equal("/mnt/repo-seed", rows[0].Path)

	rows, err = s.repo.FindByDeviceID(s.ctx, "part-unknown")
	s.Require().NoError(err)
	s.Empty(rows)
}

func (s *MountPointPathRepositoryTestSuite) TestUpsert_PreservesAbsentConfig() {
	s.seed()
	startup := true
	update := dto.MountPointData{
		Path: "/mnt/repo-seed", Root: "/", DeviceId: "part-repo-seed",
		Type: "ADDON", IsToMountAtStartup: &startup, IsMounted: true,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}
	s.Require().NoError(s.repo.Upsert(s.ctx, update))

	rows, err := s.repo.FindByPath(s.ctx, "/mnt/repo-seed", "/")
	s.Require().NoError(err)
	s.Require().Len(rows, 1)
	s.Require().NotNil(rows[0].IsToMountAtStartup)
	s.True(*rows[0].IsToMountAtStartup)
}

func (s *MountPointPathRepositoryTestSuite) TestPatch_UpdatesAndReloads() {
	s.seed()
	startup := true
	patch := dto.MountPointData{IsToMountAtStartup: &startup}
	row, err := s.repo.Patch(s.ctx, "/", "/mnt/repo-seed", patch)
	s.Require().NoError(err)
	s.Require().NotNil(row.IsToMountAtStartup)
	s.True(*row.IsToMountAtStartup)
	// Persisted flags survive the patch round-trip.
	s.NotNil(row.Flags)
}

func (s *MountPointPathRepositoryTestSuite) TestPatch_Missing() {
	_, err := s.repo.Patch(s.ctx, "/", "/mnt/nope", dto.MountPointData{})
	s.Require().Error(err)
	s.ErrorIs(err, dto.ErrorNotFound)
}

func (s *MountPointPathRepositoryTestSuite) TestPatch_NoOp() {
	s.seed()
	row, err := s.repo.Patch(s.ctx, "/", "/mnt/repo-seed", dto.MountPointData{})
	s.Require().NoError(err, "all-nil patch degrades to a successful no-op")
	s.Equal("/mnt/repo-seed", row.Path)
}

func (s *MountPointPathRepositoryTestSuite) TestPatch_InvalidPathRejected() {
	s.seed()
	_, err := s.repo.Patch(s.ctx, "/", "/mnt/repo-seed", dto.MountPointData{Path: "a:b"})
	s.Require().Error(err, "model validation must reject an invalid patch path")

	rows, err := s.repo.FindByPath(s.ctx, "/mnt/repo-seed", "/")
	s.Require().NoError(err)
	s.Require().Len(rows, 1, "failed patch must leave the original row untouched")
}

func (s *MountPointPathRepositoryTestSuite) TestSave_AndDeleteHard() {
	s.seed()
	rows, err := s.repo.FindByPath(s.ctx, "/mnt/repo-seed", "/")
	s.Require().NoError(err)
	s.Require().Len(rows, 1)

	startup := true
	rows[0].IsToMountAtStartup = &startup
	s.Require().NoError(s.repo.Save(s.ctx, &rows[0]))

	reloaded, err := s.repo.FindByPath(s.ctx, "/mnt/repo-seed", "/")
	s.Require().NoError(err)
	s.Require().Len(reloaded, 1)
	s.Require().NotNil(reloaded[0].IsToMountAtStartup)
	s.True(*reloaded[0].IsToMountAtStartup)

	s.Require().NoError(s.repo.DeleteHard(s.ctx, "/mnt/repo-seed", "/"))
	gone, err := s.repo.FindByPath(s.ctx, "/mnt/repo-seed", "/")
	s.Require().NoError(err)
	s.Empty(gone, "hard delete must remove the row, not soft-delete it")
}

func (s *MountPointPathRepositoryTestSuite) TestCancelledContext_SurfacesErrors() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.repo.FindByDeviceID(ctx, "part-x")
	s.Require().Error(err)

	_, err = s.repo.FindByPath(ctx, "/mnt/x", "/")
	s.Require().Error(err)

	s.Require().Error(s.repo.Upsert(ctx, dto.MountPointData{Path: "/mnt/x", Root: "/"}))

	_, err = s.repo.Patch(ctx, "/", "/mnt/x", dto.MountPointData{})
	s.Require().Error(err)

	s.Require().Error(s.repo.Save(ctx, &dbom.MountPointPath{}))
	s.Require().Error(s.repo.DeleteHard(ctx, "/mnt/x", "/"))
}
