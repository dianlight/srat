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
}

func (_ ExportedShare) All() ([]ExportedShare, error) {
	var shares []ExportedShare
	err := db.Find(&shares).Error
	return shares, err
}

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

func (share *ExportedShare) FromNameOfMountPoint(name string, path string) error {
	return db.Limit(1).Find(share, db.Where("name =?", name).Or(db.Where("path = ?", path))).Error
}
