package repository

import (
	"log/slog"
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
	FindByPath(path string) (*dbom.MountPointPath, error)
	Exists(id string) (bool, error)
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

	existingRecord := dbom.MountPointPath{}
	res := tx.Take(&existingRecord, "path = ?", mp.Path)

	slog.Warn("Return", "res", res, "ext", existingRecord)
	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		// Errore Generico
		return errors.WithStack(res.Error)
	} else if res.Error == nil {
		// Record Found
		if mp.DeviceId != 0 && existingRecord.DeviceId != 0 && existingRecord.DeviceId != mp.DeviceId {
			return errors.Errorf("DeviceId mismatch for %s | mp:%d db:%d", mp.Path, mp.DeviceId, existingRecord.DeviceId)
		}
		slog.Warn("Save checkpoint", "mp", mp, "exists", existingRecord)
		err := copier.CopyWithOption(&existingRecord, mp, copier.Option{IgnoreEmpty: true})
		if err != nil {
			return errors.WithStack(err)
		}
		*mp = existingRecord
		slog.Warn("Save checkpoint", "mp", mp, "exists", existingRecord)
	} else {
		// Record Not Found
		slog.Debug("New MountPoint", "mp", mp)
	}

	if strings.HasPrefix(mp.Device, "/dev") {
		mp.Device = strings.TrimPrefix(mp.Device, "/dev/")
		//panic(errors.Errorf("Invalid Source with /dev prefix %v", mp))
	}
	// slog.Debug("Save checkpoint", "mp", mp)
	err := tx.Save(mp).Error
	if err != nil {
		return errors.WithStack(err)
	}
	tx.Commit()
	return nil

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

func (r *MountPointPathRepository) Exists(path string) (bool, error) {
	var mp dbom.MountPointPath
	err := r.db.Where("path = ?", path).First(&mp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return true, err
}
