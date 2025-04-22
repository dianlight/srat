package dbom

import (
	"time"

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

/*
	func (self *Properties) Add(key string, value any) error {
		prop := Property{
			Key:   key,
			Value: value,
		}

		tx := db.WithContext(context.Background()).Begin()
		tx.Unscoped().Model(&Property{}).Where("key", key).Update("deleted_at", nil)

		result := tx.Model(&Property{}).Where(Property{Key: key}).Assign(prop).FirstOrCreate(&prop)
		if result.Error != nil {
			return errors.WithStack(result.Error)
		}

		(*self)[key] = prop
		return errors.WithStack(tx.Commit().Error)
	}

	func (self *Properties) Remove(key string) error {
		result := db.WithContext(context.Background()).Model(&Property{}).Where("key = ?", key).Delete(&Property{})
		if result.Error != nil {
			return result.Error
		}

		delete(*self, key)

		return nil
	}
*/
func (self *Properties) Get(key string) (*Property, error) {
	prop, ok := (*self)[key]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &prop, nil
}

func (self *Properties) GetValue(key string) (interface{}, error) {
	prop, err := self.Get(key)
	if err != nil {
		return nil, err
	}
	return prop.Value, nil
}

/*
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
*/
