package dbom

import (
	"time"

	"gorm.io/gorm"
)

type SambaUsers []SambaUser

type SambaUser struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Username  string         `gorm:"primaryKey"`
	Password  string
	IsAdmin   bool
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

func (p *SambaUsers) GetAdmin() error {
	return db.Where("is_admin = ?", true).First(p).Error
}

//----------------------------------------------------------------

func (share *SambaUser) Save() error {
	db.Unscoped().Model(&SambaUser{}).Where("username", share.Username).Update("deleted_at", nil)
	return db.Save(share).Error
}

func (share *SambaUser) Create() error {
	db.Unscoped().Model(&SambaUser{}).Where("username", share.Username).Update("deleted_at", nil)
	return db.Create(share).Error
}

func (share *SambaUser) Delete() error {
	return db.Delete(share).Error
}

func (share *SambaUser) Get() error {
	return db.First(share).Error
}

func (share *SambaUser) GetAdmin() error {
	return db.Where("is_admin", true).First(share).Error
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
