package repository

import (
	"sync"

	"github.com/dianlight/srat/dbom"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type PropertyRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type PropertyRepositoryInterface interface {
	All() (dbom.Properties, error)
	SaveAll(props *dbom.Properties) error
	DeleteAll() (dbom.Properties, error)
}

func NewPropertyRepositoryRepository(db *gorm.DB) PropertyRepositoryInterface {
	return &PropertyRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (self *PropertyRepository) All() (dbom.Properties, error) {
	var props []dbom.Property
	err := self.db.Model(&dbom.Property{}).Find(&props).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var propss dbom.Properties
	propss = make(dbom.Properties, len(props))
	for _, prop := range props {
		propss[prop.Key] = prop
	}
	return propss, nil
}

func (self *PropertyRepository) SaveAll(props *dbom.Properties) error {
	for _, prop := range *props {
		err := self.db.Save(&prop).Error
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (self *PropertyRepository) DeleteAll() (dbom.Properties, error) {
	result := self.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&dbom.Property{}).Delete(&dbom.Property{})
	if result.Error != nil {
		return nil, result.Error
	}
	return self.All()
}

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
