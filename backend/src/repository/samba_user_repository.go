package repository

import (
	"sync"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/unixsamba"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SambaUserRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type SambaUserRepositoryInterface interface {
	GetAdmin() (dbom.SambaUser, errors.E)
	All() (dbom.SambaUsers, errors.E)
	SaveAll(users *dbom.SambaUsers) errors.E

	Save(user *dbom.SambaUser) errors.E
	Create(user *dbom.SambaUser) errors.E

	Delete(name string) errors.E
	GetUserByName(name string) (*dbom.SambaUser, errors.E)

	Rename(oldname string, newname string) errors.E
}

func NewSambaUserRepository(db *gorm.DB) SambaUserRepositoryInterface {
	return &SambaUserRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (p *SambaUserRepository) GetAdmin() (dbom.SambaUser, errors.E) {
	var user dbom.SambaUser
	ret := p.db.Model(&dbom.SambaUser{}).Preload(clause.Associations).Where("is_admin = ?", true).First(&user)
	if ret.Error != nil {
		return dbom.SambaUser{}, errors.WithStack(ret.Error)
	}
	if ret.RowsAffected == 0 {
		return dbom.SambaUser{}, errors.WithStack(gorm.ErrRecordNotFound)
	}
	return user, nil
}

func (self *SambaUserRepository) All() (dbom.SambaUsers, errors.E) {
	var users []dbom.SambaUser
	err := self.db.Model(&dbom.SambaUser{}).Preload(clause.Associations).Find(&users).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return users, nil
}

func (self *SambaUserRepository) Save(user *dbom.SambaUser) errors.E {
	self.db.Unscoped().Model(&dbom.SambaUser{}).Where("username = ?", user.Username).Update("deleted_at", nil)
	return errors.WithStack(self.db.Debug().Save(user).Error)
}

func (self *SambaUserRepository) Create(user *dbom.SambaUser) errors.E {
	ret := self.db.Unscoped().Model(&dbom.SambaUser{}).Where("username = ?", user.Username).Update("deleted_at", nil)
	if ret.Error != nil {
		return errors.WithStack(ret.Error)
	}
	if ret.RowsAffected == 0 {
		return errors.WithStack(self.db.Create(user).Error)
	}
	return errors.WithStack(self.db.Save(user).Error)
}

func (self *SambaUserRepository) GetUserByName(name string) (*dbom.SambaUser, errors.E) {
	var user dbom.SambaUser
	err := self.db.Model(&dbom.SambaUser{}).Preload(clause.Associations).Where("username = ? and is_admin = false", name).First(&user).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &user, nil
}

func (self *SambaUserRepository) Delete(name string) errors.E {
	return errors.WithStack(self.db.Model(&dbom.SambaUser{}).Where("username = ? and is_admin = false", name).Delete(&dbom.SambaUser{Username: name}).Error)
}

func (self *SambaUserRepository) SaveAll(users *dbom.SambaUsers) errors.E {
	for _, user := range *users {
		err := self.db.Save(&user).Error
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (self *SambaUserRepository) Rename(oldname string, newname string) errors.E {
	return errors.WithStack(self.db.Transaction(func(tx *gorm.DB) error {
		var smbuser dbom.SambaUser
		// First, retrieve the user to get the password *before* updating the name.
		// We need the original password for the unixsamba.RenameUsername call.
		if err := tx.Where("username = ?", oldname).First(&smbuser).Error; err != nil {
			return errors.Wrapf(err, "failed to find user %s before renaming", oldname)
		}

		// Attempt to rename the user in the underlying system (Samba/Unix) first
		if err := unixsamba.RenameUsername(oldname, newname, false, smbuser.Password); err != nil {
			return errors.Wrapf(err, "failed to rename user in unix/samba from %s to %s", oldname, newname)
		}

		// Update the username in the database
		if err := tx.Model(&dbom.SambaUser{}).Where("username = ?", oldname).Update("username", newname).Error; err != nil {
			return errors.Wrapf(err, "failed to update username in database from %s to %s", oldname, newname)
		}
		return nil
	}))
}

//func (self *SambaUserRepository)
