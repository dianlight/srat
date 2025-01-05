package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

type SambaUsers []SambaUser

type SambaUser struct {
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Username  string         `json:"username" gorm:"primaryKey"`
	Password  string         `json:"password"`
}

func (p *SambaUsers) Load() error {
	return db.Find(p).Error
}

func (p *SambaUsers) DeleteAll() error {
	result := db.Delete(&SambaUser{})
	if result.Error != nil {
		return result.Error
	}
	*p = nil
	return nil
}

func (p *SambaUser) Add(value interface{}) error {
	switch value.(type) {
	case SambaUser:
		db.Unscoped().Model(&SambaUser{}).Where("username", value.(SambaUser).Username).Update("deleted_at", nil)
		result := db.Where(SambaUser{Username: value.(SambaUser).Username}).Assign(value).FirstOrCreate(&value)
		if result.Error != nil {
			return result.Error
		}
		return nil
	default:
		var sambaUser SambaUser
		copier.Copy(&sambaUser, value)
		return p.Add(sambaUser)
	}
}

func (p *SambaUser) Remove(username string) error {
	result := db.Where("username = ?", username).Delete(&SambaUser{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (p *SambaUser) Get(username string) (*SambaUser, error) {
	var user SambaUser
	result := db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (p *SambaUser) FromSettings(setting dto.Settings) error {
	return db.Transaction(func(tx *gorm.DB) error {
		err := p.DeleteAll()
		if err != nil {
			return err
		}
		mapSetting := setting.ToMap()
		for key, value := range mapSetting {
			err := p.Add(key, value)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *SambaUser) ToSettings(setting *dto.Settings) {
	mapSetting := setting.ToMap()
	for _, prop := range *p {
		mapSetting[prop.Key] = prop.Value
	}
}
