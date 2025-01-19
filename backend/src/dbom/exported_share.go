package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExportedShare struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt   `gorm:"index"`
	Name        string           `json:"name,omitempty" gorm:"unique,index"`
	Path        string           `json:"path" gorm:"unique"`
	FS          string           `json:"fs"`
	Disabled    bool             `json:"disabled,omitempty"`
	Users       []SambaUser      `json:"users,omitempty" gorm:"many2many:user_rw_share;"`
	RoUsers     []SambaUser      `json:"ro_users,omitempty" gorm:"many2many:user_ro_share;"`
	TimeMachine bool             `json:"timemachine,omitempty"`
	Usage       dto.HAMountUsage `json:"usage,omitempty"`
	DeviceId    *uint64          `json:"device_id,omitempty"`
	Invalid     bool             `json:"invalid,omitempty"`
}

type ExportedShares []ExportedShare

func (p *ExportedShares) Load() error {
	return db.Preload(clause.Associations).Find(p).Error
}

func (p *ExportedShares) Save() error {
	return db.Debug().Save(p).Error
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

func (share *ExportedShare) FromNameOrPath(name string, path string) error {
	return db.Preload(clause.Associations).Limit(1).Find(share, db.Where("name =?", name).Or(db.Where("path = ?", path))).Error
}
