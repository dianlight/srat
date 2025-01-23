package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExportedShare struct {
	ID               uint `gorm:"primarykey"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
	Name             string         `gorm:"uniqueIndex"`
	Disabled         bool
	Users            []SambaUser `gorm:"many2many:user_rw_share;constraint:OnDelete:CASCADE;"`
	RoUsers          []SambaUser `gorm:"many2many:user_ro_share;constraint:OnDelete:CASCADE;"`
	TimeMachine      bool
	Usage            dto.HAMountUsage
	MountPointDataID uint
	MountPointData   MountPointData `gorm:"foreignKey:MountPointDataID;references:ID;"`
	//Invalid        bool             `json:"invalid,omitempty"`
}

type ExportedShares []ExportedShare

func (p *ExportedShares) Load() error {
	return db.Preload(clause.Associations).Find(p).Error
}

func (p *ExportedShares) Save() error {
	err := db.Save(p).Error
	if err != nil {
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

/*
func (share *ExportedShare) FromNameOrPath(name string, path string) error {
	return db.Preload(clause.Associations).Limit(1).Find(share, db.Where("name =?", name).Or(db.Where("path = ?", path))).Error
}
*/

func (u *ExportedShare) BeforeSave(tx *gorm.DB) error {
	if u.Name == "" {
		return tracerr.Errorf("Invalid name for exported share")
	}
	return nil
}
