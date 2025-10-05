package repository

import (
	"log/slog"
	"sync"

	"github.com/dianlight/srat/dbom"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MountPointPathRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type MountPointPathRepositoryInterface interface {
	All() ([]dbom.MountPointPath, errors.E)
	AllByDeviceId() (map[string]dbom.MountPointPath, errors.E)
	Save(mp *dbom.MountPointPath) errors.E
	FindByPath(path string) (*dbom.MountPointPath, errors.E)
	FindByDevice(device string) (*dbom.MountPointPath, errors.E)
	Exists(id string) (bool, errors.E)
	Delete(path string) errors.E
}

func NewMountPointPathRepository(db *gorm.DB) MountPointPathRepositoryInterface {
	return &MountPointPathRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (r *MountPointPathRepository) Save(mp *dbom.MountPointPath) errors.E {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return errors.WithStack(r.db.Transaction(func(tx *gorm.DB) error {
		// Check if record exists to decide between Create and Update
		var count int64
		// mp.Path is the primary key. BeforeSave hook in MountPointPath ensures it's not empty.
		if err := tx.Unscoped().Model(&dbom.MountPointPath{}).Where("path = ?", mp.Path).Count(&count).Error; err != nil {
			return errors.WithStack(err)
		}

		var opErr error
		if count > 0 { // Record exists, so update
			// Updates(struct) only updates non-zero fields. For pointers, nil is the zero-value.
			// This means if a pointer field in 'mp' (e.g., mp.Flags) is nil,
			// that field will NOT be included in the UPDATE statement, effectively ignoring it.
			// clause.Returning{} will repopulate 'mp' with the current DB state after the update.
			if err := tx.Model(&dbom.MountPointPath{}).Unscoped().Where("path = ?", mp.Path).UpdateColumn("deleted_at", gorm.Expr("NULL")).Error; err != nil {
				slog.Error("Failed to explicitly undelete exported_share before update", "path", mp.Path, "error", err)
				return errors.WithDetails(err, "path", mp.Path, "details", mp)
			}
			opErr = tx. /*Debug().*/ /*.Model(&dbom.MountPointPath{Path: mp.Path})*/ Clauses(clause.Returning{}).Updates(mp).Error
		} else {
			// Record does not exist, so create
			opErr = tx.Clauses(clause.Returning{}).Create(mp).Error
		}
		return errors.WithStack(opErr) // opErr will be nil on success, errors.WithStack(nil) is nil
	}))
}

func (r *MountPointPathRepository) FindByPath(path string) (*dbom.MountPointPath, errors.E) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var mp dbom.MountPointPath
	err := r.db.Where("path = ?", path).First(&mp).Error
	return &mp, errors.WithStack(err)
}

func (r *MountPointPathRepository) FindByDevice(device string) (*dbom.MountPointPath, errors.E) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var mp dbom.MountPointPath
	// Ensure we search for the device id (can include /dev/disk/by-id/ or similar)
	err := r.db.Where("device_id = ?", device).First(&mp).Error
	return &mp, errors.WithStack(err)
}
func (r *MountPointPathRepository) All() (Data []dbom.MountPointPath, Error errors.E) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var mps []dbom.MountPointPath
	err := r.db.Find(&mps).Error
	return mps, errors.WithStack(err)
}

func (r *MountPointPathRepository) AllByDeviceId() (map[string]dbom.MountPointPath, errors.E) {
	all, err := r.All()
	if err != nil {
		return nil, err
	}
	result := make(map[string]dbom.MountPointPath, len(all))
	for _, mp := range all {
		if mp.DeviceId != "" {
			result[mp.DeviceId] = mp
		}
	}
	return result, nil
}

func (r *MountPointPathRepository) Exists(path string) (bool, errors.E) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var mp dbom.MountPointPath
	err := r.db.Where("path = ?", path).First(&mp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return true, errors.WithStack(err)
}

func (r *MountPointPathRepository) Delete(path string) errors.E {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return errors.WithStack(r.db.Delete(&dbom.MountPointPath{Path: path}).Error)
}
