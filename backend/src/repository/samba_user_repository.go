package repository

import (
	"sync"

	"github.com/dianlight/srat/dbom"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SambaUserRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type SambaUserRepositoryInterface interface {
	GetAdmin() (dbom.SambaUser, error)
	All() (dbom.SambaUsers, error)
	SaveAll(users *dbom.SambaUsers) error

	Save(user *dbom.SambaUser) error
	Create(user *dbom.SambaUser) error

	Delete(name string) error
	GetUserByName(name string) (*dbom.SambaUser, error)

	Rename(oldname string, newname string) error
}

func NewSambaUserRepository(db *gorm.DB) SambaUserRepositoryInterface {
	return &SambaUserRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (p *SambaUserRepository) GetAdmin() (dbom.SambaUser, error) {
	ret := p.db.Model(&dbom.SambaUser{}).Preload(clause.Associations).Where("is_admin = ?", true).First(p)
	if ret.Error != nil {
		return dbom.SambaUser{}, ret.Error
	}
	if ret.RowsAffected == 0 {
		return dbom.SambaUser{}, gorm.ErrRecordNotFound
	}
	var user dbom.SambaUser
	err := ret.Scan(&user).Error
	if err != nil {
		return dbom.SambaUser{}, err
	}
	return user, nil
}

func (self *SambaUserRepository) All() (dbom.SambaUsers, error) {
	var users []dbom.SambaUser
	err := self.db.Model(&dbom.SambaUser{}).Preload(clause.Associations).Find(&users).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return users, nil
}

func (self *SambaUserRepository) Save(user *dbom.SambaUser) error {
	self.db.Unscoped().Model(&dbom.SambaUser{}).Where("username = ?", user.Username).Update("deleted_at", nil)
	return self.db.Save(user).Error
}

func (self *SambaUserRepository) Create(user *dbom.SambaUser) error {
	ret := self.db.Unscoped().Model(&dbom.SambaUser{}).Where("username = ?", user.Username).Update("deleted_at", nil)
	if ret.Error != nil {
		return errors.WithStack(ret.Error)
	}
	if ret.RowsAffected == 0 {
		return self.db.Create(user).Error
	}
	return self.db.Save(user).Error
}

func (self *SambaUserRepository) GetUserByName(name string) (*dbom.SambaUser, error) {
	var user dbom.SambaUser
	err := self.db.Model(&dbom.SambaUser{}).Preload(clause.Associations).Where("username = ? and is_admin = false", name).First(&user).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &user, nil
}

func (self *SambaUserRepository) Delete(name string) error {
	return self.db.Unscoped().Model(&dbom.SambaUser{}).Where("username = ? and is_admin = false", name).Delete(&dbom.SambaUser{Username: name}).Error
}

func (self *SambaUserRepository) SaveAll(users *dbom.SambaUsers) error {
	for _, user := range *users {
		err := self.db.Save(&user).Error
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (self *SambaUserRepository) Rename(oldname string, newname string) error {
	return self.db.Unscoped().Model(&dbom.SambaUser{}).Where("username = ?", oldname).Update("username", newname).Error
}

//func (self *SambaUserRepository)
