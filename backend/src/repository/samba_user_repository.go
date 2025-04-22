package repository

import (
	"sync"

	"github.com/dianlight/srat/dbom"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
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
}

func NewSambaUserRepository(db *gorm.DB) SambaUserRepositoryInterface {
	return &SambaUserRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (p *SambaUserRepository) GetAdmin() (dbom.SambaUser, error) {
	ret := p.db.Where("is_admin = ?", true).First(p)
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
	err := self.db.Model(&dbom.SambaUser{}).Find(&users).Error
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
	err := self.db.Model(&dbom.SambaUser{}).Where("username = ? and is_admin = false", name).First(&user).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &user, nil
}

func (self *SambaUserRepository) Delete(name string) error {
	return self.db.Unscoped().Model(&dbom.SambaUser{}).Where("username = ? and is_admin = false", name).Delete(&dbom.SambaUser{}).Error
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

/*
func (self *PropertyRepository) DeleteAll() (dbom.Properties, error) {
	result := self.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&dbom.Property{}).Delete(&dbom.Property{})
	if result.Error != nil {
		return nil, result.Error
	}
	return self.All()
}
*/

/*
func (p *PropertyRepository) UpdateName(old_name string, new_name string) error {
	// Get / Save Users end RoUsers association

	err := p.db.
		Model(&dbom.ExportedShare{Name: old_name}).Update("name", new_name).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}



func (r *ExportedShareRepository) All(shares *[]dbom.ExportedShare) error {
	err := r.db.Preload(clause.Associations).Find(shares).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return err
}

func (p *ExportedShareRepository) FindByName(name string) (*dbom.ExportedShare, error) {
	var share dbom.ExportedShare
	err := p.db.Preload(clause.Associations).First(&share, "name = ?", name).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &share, nil
}

func (p *ExportedShareRepository) SaveAll(shares *[]dbom.ExportedShare) error {
	err := p.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(shares).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (p *ExportedShareRepository) Save(share *dbom.ExportedShare) error {
	err := p.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(share).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (p *ExportedShareRepository) Delete(name string) error {
	return p.db.Select(clause.Associations).Delete(&dbom.ExportedShare{Name: name}).Error
}
*/
