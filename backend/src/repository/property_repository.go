package repository

import (
	"sync"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type PropertyRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type PropertyRepositoryInterface interface {
	All(include_internal bool) (dbom.Properties, error)
	SaveAll(props *dbom.Properties) error
	//DeleteAll() (dbom.Properties, error)
	Value(key string, include_internal bool) (interface{}, error)
	SetValue(key string, value interface{}) error
	SetInternalValue(key string, value interface{}) error
}

func NewPropertyRepositoryRepository(db *gorm.DB) PropertyRepositoryInterface {
	return &PropertyRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (self *PropertyRepository) All(include_internal bool) (dbom.Properties, error) {
	var props []dbom.Property
	err := self.db.Model(&dbom.Property{}).Find(&props, "internal = ? or internal = false", include_internal).Error
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

// SetValue saves a property with the given key and value.
// The property is marked as not internal.
func (self *PropertyRepository) SetValue(key string, value interface{}) error {
	prop := dbom.Property{
		Key:      key,
		Value:    value,
		Internal: false,
	}
	return errors.WithStack(self.db.Save(&prop).Error)
}

// SetInternalValue saves a property with the given key and value.
// The property is marked as internal.
func (self *PropertyRepository) SetInternalValue(key string, value interface{}) error {
	prop := dbom.Property{
		Key:      key,
		Value:    value,
		Internal: true,
	}
	return errors.WithStack(self.db.Save(&prop).Error)
}

func (self *PropertyRepository) Value(key string, include_internal bool) (interface{}, error) {
	var prop dbom.Property
	err := self.db.Model(&dbom.Property{}).First(&prop, "key = ? and (internal = ? or internal = false)", key, include_internal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithStack(dto.ErrorNotFound)
		}
		return nil, errors.WithStack(err)
	}
	return prop.Value, nil
}
