package repository

import (
	"strings"
	"sync"

	"github.com/dianlight/srat/dbom"
	"github.com/jinzhu/copier"
	"gitlab.com/tozd/go/errors"
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
	//slog.Debug("Save checkpoint", "mp", mp)
	if mp.ID == 0 {
		//data, _ := r.All()
		//slog.Debug("Chekp", "all", data)
		existingRecord := dbom.MountPointPath{}
		res := tx.Limit(1).Find(&existingRecord, "path = ? and source = ?", mp.Path, mp.Source)
		//slog.Debug("Return", "res", res, "ext", existingRecord)
		if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return errors.WithStack(res.Error)
		} else if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			if mp.DeviceId != 0 && existingRecord.DeviceId != 0 && existingRecord.DeviceId != mp.DeviceId {
				return errors.Errorf("DeviceId mismatch for %s | mp:%d db:%d", mp.Path, mp.DeviceId, existingRecord.DeviceId)
			}
			//slog.Debug("Save checkpoint", "mp", mp, "exists", existingRecord)
			err := copier.CopyWithOption(&existingRecord, mp, copier.Option{IgnoreEmpty: true})
			if err != nil {
				return errors.WithStack(err)
			}
			*mp = existingRecord
			//slog.Debug("Save checkpoint", "mp", mp, "exists", existingRecord)
		}
	}

	if strings.HasPrefix(mp.Source, "/dev") {
		panic(errors.Errorf("Invalid Source with /dev prefix %v", mp))
	}
	// slog.Debug("Save checkpoint", "mp", mp)
	err := tx.Save(mp).Error
	if err != nil {
		return errors.WithStack(err)
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

func (r *MountPointPathRepository) All() (Data []dbom.MountPointPath, Error error) {
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
