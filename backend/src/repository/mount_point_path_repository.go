// Package repository holds persistence adapters for SRAT domain objects.
// MountPointPathRepository is the sole place that issues GORM queries for
// mount point rows; services must use it instead of touching *gorm.DB for
// mount data directly.
package repository

import (
	"context"
	"log/slog"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MountPointPathRepositoryInterface isolates mount point persistence so the
// volume service layers never issue direct db.* queries.
type MountPointPathRepositoryInterface interface {
	FindByDeviceID(ctx context.Context, deviceID string) ([]dbom.MountPointPath, errors.E)
	FindByPath(ctx context.Context, path, root string) ([]dbom.MountPointPath, errors.E)
	Upsert(ctx context.Context, md dto.MountPointData) errors.E
	Patch(ctx context.Context, root, path string, patch dto.MountPointData) (dbom.MountPointPath, errors.E)
	Save(ctx context.Context, row *dbom.MountPointPath) errors.E
	DeleteHard(ctx context.Context, path, root string) errors.E
}

type mountPointPathRepository struct {
	db   *gorm.DB
	conv converter.DtoToDbomConverterImpl
}

// NewMountPointPathRepository creates the GORM-backed mount point repository.
func NewMountPointPathRepository(db *gorm.DB) MountPointPathRepositoryInterface {
	return &mountPointPathRepository{db: db}
}

// FindByDeviceID loads all persisted mount rows for one partition device id.
func (r *mountPointPathRepository) FindByDeviceID(ctx context.Context, deviceID string) ([]dbom.MountPointPath, errors.E) {
	rows, err := gorm.G[dbom.MountPointPath](r.db).
		Preload("ExportedShare", nil).
		Where(g.MountPointPath.DeviceId.Eq(deviceID)).
		Find(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rows, nil
}

// FindByPath loads persisted mount rows by primary key (path, root).
func (r *mountPointPathRepository) FindByPath(ctx context.Context, path, root string) ([]dbom.MountPointPath, errors.E) {
	rows, err := gorm.G[dbom.MountPointPath](r.db).
		Where(g.MountPointPath.Path.Eq(path), g.MountPointPath.Root.Eq(root)).
		Find(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rows, nil
}

// Upsert persists one mount point without wiping configuration absent from
// the payload: the existing record (if any) is loaded first and converted
// into, so automount flags and custom flags survive partial event payloads.
func (r *mountPointPathRepository) Upsert(ctx context.Context, md dto.MountPointData) errors.E {
	root := "/"
	if md.Root != "" {
		root = md.Root
	}

	existing, err := gorm.G[dbom.MountPointPath](r.db).
		Where(g.MountPointPath.Path.Eq(md.Path), g.MountPointPath.Root.Eq(root)).
		First(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(err)
	}
	dbMountData := existing

	if err := r.conv.MountPointDataToMountPointPath(md, &dbMountData); err != nil {
		return errors.WithStack(err)
	}
	// close mountpath loop before save
	if dbMountData.ExportedShare != nil {
		dbMountData.ExportedShare.MountPointData = dbMountData
		dbMountData.ExportedShare.MountPointDataPath = &dbMountData.Path
		dbMountData.ExportedShare.MountPointDataRoot = dbMountData.Root
	}

	if err := r.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).
		Create(&dbMountData).Error; err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Patch applies a partial mount point update and re-reads the row so the
// caller observes the true persisted state: struct-based Updates skips
// zero-value fields, so the in-memory copy cannot be trusted for the reply.
func (r *mountPointPathRepository) Patch(ctx context.Context, root, path string, patch dto.MountPointData) (dbom.MountPointPath, errors.E) {
	dbMountData, err := gorm.G[dbom.MountPointPath](r.db).
		Where(g.MountPointPath.Root.Eq(root), g.MountPointPath.Path.Eq(path)).First(ctx)
	if err != nil {
		return dbom.MountPointPath{}, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
	}

	if err := r.conv.MountPointDataToMountPointPath(patch, &dbMountData); err != nil {
		return dbom.MountPointPath{}, errors.WithStack(err)
	}

	affected, err := gorm.G[*dbom.MountPointPath](r.db).
		Where(g.MountPointPath.Root.Eq(root), g.MountPointPath.Path.Eq(path)).
		Updates(ctx, &dbMountData)
	if err != nil {
		// First above already proved the row exists; a concurrent delete
		// surfaces below as NotFound on the re-read instead.
		return dbom.MountPointPath{}, errors.WithStack(err)
	}
	if affected == 0 {
		slog.DebugContext(ctx, "PatchMountPointSettings: no fields changed (no-op)", "root", root, "path", path)
	}

	if dbMountData, err = gorm.G[dbom.MountPointPath](r.db).
		Where(g.MountPointPath.Root.Eq(root), g.MountPointPath.Path.Eq(path)).First(ctx); err != nil {
		return dbom.MountPointPath{}, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
	}
	return dbMountData, nil
}

// Save writes back one row, used when reconciliation merges legacy flags.
func (r *mountPointPathRepository) Save(ctx context.Context, row *dbom.MountPointPath) errors.E {
	if err := r.db.WithContext(ctx).Save(row).Error; err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// DeleteHard removes one row with Unscoped delete: the model carries
// DeletedAt (soft delete), but a soft-deleted row keeps its (path, root)
// primary key and would be resurrected by the Upsert OnConflict UpdateAll.
func (r *mountPointPathRepository) DeleteHard(ctx context.Context, path, root string) errors.E {
	if err := r.db.WithContext(ctx).Unscoped().Where("path = ? AND root = ?", path, root).Delete(&dbom.MountPointPath{}).Error; err != nil {
		return errors.WithStack(err)
	}
	return nil
}
