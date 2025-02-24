package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExportedShare struct {
	Name             string `gorm:"primarykey"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
	Disabled         bool
	Users            []SambaUser `gorm:"many2many:user_rw_share;constraint:OnDelete:CASCADE;"`
	RoUsers          []SambaUser `gorm:"many2many:user_ro_share;constraint:OnDelete:CASCADE;"`
	TimeMachine      bool
	Usage            dto.HAMountUsage
	MountPointDataID uint
	MountPointData   MountPointPath `gorm:"foreignKey:MountPointDataID;references:ID;"`
}

type ExportedShares []ExportedShare

func (p *ExportedShares) Load() error {
	return db.Preload(clause.Associations).Find(p).Error
}

func (p *ExportedShares) Save() error {
	err := db.Save(p).Error
	if err != nil {
		//		tracerr.PrintSourceColor(tracerr.Wrap(err))
		return tracerr.Wrap(err)
	}
	return nil
}

//------------------------------------------------------------------------------

func (share *ExportedShare) Save() error {
	return db.Save(share).Error
}

func (share *ExportedShare) Delete() error {
	return db.Delete(share).Error
}

func (share *ExportedShare) FromName(name string) error {
	return db.Preload(clause.Associations).Where("name =?", name).First(share).Error
}

func (share *ExportedShare) Get() error {
	return db.Preload(clause.Associations).First(share).Error
}

func (u *ExportedShare) BeforeSave(tx *gorm.DB) error {
	if u.Name == "" {
		return tracerr.Errorf("Invalid name for exported share")
	}
	return nil
}
