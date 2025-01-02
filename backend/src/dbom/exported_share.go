package dbom

import "gorm.io/gorm"

type ExportedShare struct {
	gorm.Model
	Name        string      `json:"name,omitempty"`
	Path        string      `json:"path"`
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
