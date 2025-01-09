package dbom

import "gorm.io/gorm"

type ExportedShare struct {
	gorm.Model
	Name        string      `json:"name,omitempty" gorm:"unique"`
	Path        string      `json:"path" gorm:"unique"`
	FS          string      `json:"fs"`
	Disabled    bool        `json:"disabled,omitempty"`
	Users       []SambaUser `json:"users,omitempty" gorm:"many2many:user_rw_share;"`
	RoUsers     []SambaUser `json:"ro_users,omitempty" gorm:"many2many:user_ro_share;"`
	TimeMachine bool        `json:"timemachine,omitempty"`
	Usage       string      `json:"usage,omitempty"`
	DeviceId    *uint64     `json:"device_id,omitempty"`
	Invalid     bool        `json:"invalid,omitempty"`
}

type ExportedShares []ExportedShare

func (p *ExportedShares) Load() error {
	return db.Find(p).Error
}

func (p *ExportedShares) Save() error {
	return db.Save(p).Error
}

//------------------------------------------------------------------------------

func (share *ExportedShare) Save() error {
	return db.Save(share).Error
}

func (share *ExportedShare) Delete() error {
	return db.Delete(share).Error
}

func (share *ExportedShare) FromName(name string) error {
	return db.Where("name =?", name).First(share).Error
}

func (share *ExportedShare) Get() error {
	return db.First(share).Error
}

func (share *ExportedShare) FromNameOrPath(name string, path string) error {
	return db.Limit(1).Find(share, db.Where("name =?", name).Or(db.Where("path = ?", path))).Error
}
