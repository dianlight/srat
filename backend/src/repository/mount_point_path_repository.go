package repository

import (
	"errors"
	"sync"

	"github.com/dianlight/srat/dbom"
	"github.com/jinzhu/copier"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type MountPointPathRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type MountPointPathRepositoryInterface interface {
	All() ([]dbom.MountPointPath, error)
	Save(mp *dbom.MountPointPath) error
	FindByID(id uint) (*dbom.MountPointPath, error)
	FindByPath(path string) (*dbom.MountPointPath, error)
	Exists(id uint) (bool, error)
	// Delete(mp *dbom.MountPointPath) error
	// Update(mp *dbom.MountPointPath) error
}

func NewMountPointPathRepository(db *gorm.DB) MountPointPathRepositoryInterface {
	return &MountPointPathRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (r *MountPointPathRepository) Save(mp *dbom.MountPointPath) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	tx := r.db.Begin()
	defer tx.Rollback()
	//	slog.Debug("Save checkpoint", "mp", mp)
	if mp.ID == 0 {
		var existingRecord dbom.MountPointPath
		res := tx.Limit(1).Find(&existingRecord, "path = ? and source = ?", mp.Path, mp.Source)
		if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return tracerr.Wrap(res.Error)
		} else if res.RowsAffected > 0 {
			if mp.DeviceId != 0 && existingRecord.DeviceId != mp.DeviceId {
				return tracerr.Errorf("DeviceId mismatch for %s", mp.Path)
			}
			err := copier.CopyWithOption(&existingRecord, mp, copier.Option{IgnoreEmpty: true})
			if err != nil {
				return tracerr.Wrap(err)
			}
			*mp = existingRecord
		}
	}

	// slog.Debug("Save checkpoint", "mp", mp)
	err := tx.Save(mp).Error
	if err != nil {
		tracerr.PrintSourceColor(tracerr.Wrap(err))
		return tracerr.Wrap(err)
	}
	tx.Commit()
	return nil

}

func (r *MountPointPathRepository) FindByID(id uint) (*dbom.MountPointPath, error) {
	var mp dbom.MountPointPath
	err := r.db.First(&mp, id).Error
	return &mp, err
}

func (r *MountPointPathRepository) FindByPath(path string) (*dbom.MountPointPath, error) {
	var mp dbom.MountPointPath
	err := r.db.Where("path = ?", path).First(&mp).Error
	return &mp, err
}

func (r *MountPointPathRepository) All() ([]dbom.MountPointPath, error) {
	var mps []dbom.MountPointPath
	err := r.db.Find(&mps).Error
	return mps, err
}

/*
func (r *MountPointPathRepository) Delete(mp *dbom.MountPointPath) error {
	return r.db.Delete(mp).Error
}

func (r *MountPointPathRepository) Update(mp *dbom.MountPointPath) error {
	return r.db.Save(mp).Error
}
*/

// Exists checks if a MountPointPath exists in the database by its ID.

func (r *MountPointPathRepository) Exists(id uint) (bool, error) {
	var mp dbom.MountPointPath
	err := r.db.First(&mp, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return true, err
}
