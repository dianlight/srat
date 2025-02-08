package repository

import (
	"github.com/dianlight/srat/dbom"
	"gorm.io/gorm"
)

type MountPointPathRepository struct {
	db *gorm.DB
}

type MountPointPathRepositoryInterface interface {
	All() ([]dbom.MountPointPath, error)
	Save(mp *dbom.MountPointPath) error
	FindByID(id uint) (*dbom.MountPointPath, error)
	FindByPath(path string) (*dbom.MountPointPath, error)
	Exists(id uint) bool
	// Delete(mp *dbom.MountPointPath) error
	// Update(mp *dbom.MountPointPath) error
}

func NewMountPointPathRepository(db *gorm.DB) MountPointPathRepositoryInterface {
	return &MountPointPathRepository{db: db}
}

func (r *MountPointPathRepository) Save(mp *dbom.MountPointPath) error {
	return r.db.Save(mp).Error
	/*
		tx := db.Begin()
		var existingRecord MountPointPath
		res := tx.Limit(1).Find(&existingRecord, "path = ?", mp.Path)
		if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return tracerr.Wrap(res.Error)
		} else if res.RowsAffected > 0 {
			if mp.DeviceId != 0 && existingRecord.DeviceId != mp.DeviceId {
				return tracerr.Errorf("DeviceId mismatch for %s", mp.Path)
			}
			err = copier.CopyWithOption(mp, &existingRecord, copier.Option{IgnoreEmpty: true})
			if err != nil {
				return tracerr.Wrap(err)
			}
		}

		err = tx.Save(mp).Error
		if err != nil {
			return tracerr.Wrap(err)
		}
		tx.Commit()
		return nil
	*/

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

func (r *MountPointPathRepository) Exists(id uint) bool {
	var mp dbom.MountPointPath
	err := r.db.First(&mp, id).Error
	return err == nil
}
