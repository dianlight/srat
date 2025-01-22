package dbom

import (
	"context"
	"time"

	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type Property struct {
	Key       string      `gorm:"primaryKey"`
	Value     interface{} `gorm:"serializer:json"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Properties map[string]Property

func (self *Properties) Load() error {
	var props []Property
	err := db.Model(&Property{}).Find(&props).Error
	if err != nil {
		return tracerr.Wrap(err)
	}
	*self = make(Properties, len(props))
	for _, prop := range props {
		(*self)[prop.Key] = prop
	}
	return nil
}

func (self *Properties) Save() error {
	for _, prop := range *self {
		err := db.Save(&prop).Error
		if err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (self *Properties) DeleteAll() error {
	result := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&Property{}).Delete(&Property{})
	if result.Error != nil {
		return result.Error
	}
	*self = nil
	return nil
}

func (self *Properties) Add(key string, value any) error {
	prop := Property{
		Key:   key,
		Value: value,
	}

	tx := db.WithContext(context.Background()).Begin()
	tx.Unscoped().Model(&Property{}).Where("key", key).Update("deleted_at", nil)

	result := tx.Model(&Property{}).Where(Property{Key: key}).Assign(prop).FirstOrCreate(&prop)
	if result.Error != nil {
		return tracerr.Wrap(result.Error)
	}

	(*self)[key] = prop
	return tracerr.Wrap(tx.Commit().Error)
}

func (self *Properties) Remove(key string) error {
	result := db.WithContext(context.Background()).Model(&Property{}).Where("key = ?", key).Delete(&Property{})
	if result.Error != nil {
		return result.Error
	}

	delete(*self, key)

	return nil
}

func (self *Properties) Get(key string) (*Property, error) {
	var prop Property
	result := db.WithContext(context.Background()).Table("properties").Where("key = ?", key).First(&prop)
	if result.Error != nil {
		return nil, result.Error
	}
	(*self)[key] = prop

	return &prop, nil
}

func (self *Properties) GetValue(key string) (interface{}, error) {
	prop, err := self.Get(key)
	if err != nil {
		return nil, err
	}
	return prop.Value, nil
}

func (self *Properties) SetValue(key string, value any) error {
	prop, err := self.Get(key)
	if err != nil {
		return err
	}

	prop.Value = value
	err = db.Save(&prop).Error
	if err != nil {
		return err
	}

	(*self)[key] = *prop
	return nil
}
