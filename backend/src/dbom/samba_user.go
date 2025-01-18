package dbom

import (
	"time"

	"gorm.io/gorm"
)

type SambaUsers []SambaUser

type SambaUser struct {
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Username  string         `json:"username" gorm:"primaryKey"`
	Password  string         `json:"password"`
	IsAdmin   bool           `json:"is_admin"`
}

func (p *SambaUsers) Load() error {
	return db.Find(p).Error
}

func (p *SambaUsers) Save() error {
	return db.Save(p).Error
}

func (p *SambaUsers) DeleteAll() error {
	result := db.Delete(&SambaUser{})
	if result.Error != nil {
		return result.Error
	}
	*p = nil
	return nil
}

/*
func (self SambaUsers) Users() ([]SambaUser, error) {
	tmp := reflect.ValueOf(slices.Clone(self)).Interface().([]SambaUser)
	result := slices.DeleteFunc(tmp, func(u SambaUser) bool { return u.IsAdmin })
	return result, nil
}

func (self *SambaUsers) AdminUser() (*SambaUser, error) {
	var user SambaUser
	err := db.Where("is_admin = ?", true).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
*/

/*
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

func (p *SambaUser) FromSettings(setting dto.Settings) error {
	return db.Transaction(func(tx *gorm.DB) error {
		err := p.DeleteAll()
		if err != nil {
			return tracerr.Wrap(err)
		}
		mapSetting := setting.ToMap()
		for key, value := range mapSetting {
			err := p.Add(key, value)
			if err != nil {
				return tracerr.Wrap(err)
			}
		}
		return nil
	})
}
*/

//----------------------------------------------------------------

func (share *SambaUser) Save() error {
	db.Unscoped().Model(&SambaUser{}).Where("username", share.Username).Update("deleted_at", nil)
	return db.Save(share).Error
}

func (share *SambaUser) Delete() error {
	return db.Delete(share).Error
}

func (share *SambaUser) Get() error {
	return db.First(share).Error
}

/*
func (p *SambaUser) Get(username string) (*SambaUser, error) {
	var user SambaUser
	result := db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}
*/
