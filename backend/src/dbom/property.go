package dbom

import (
	"time"

	"github.com/thoas/go-funk"
	"gorm.io/gorm"
)

type Property struct {
	Key       string      `json:"key" gorm:"primaryKey" mapper:"key"`
	Value     interface{} `json:"value" mapper:"value" gorm:"serializer:json"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Properties []Property

func (self *Properties) Load() error {
	return db.Find(self).Error
}

func (self *Properties) Save() error {
	return db.Save(self).Error
}

func (self *Properties) DeleteAll() error {
	result := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Property{})
	if result.Error != nil {
		return result.Error
	}
	*self = nil
	return nil
}

func (self *Properties) Add(key string, value interface{}) error {
	prop := Property{
		Key:   key,
		Value: value,
	}

	db.Unscoped().Model(&Property{}).Where("key", key).Update("deleted_at", nil)

	result := db.Where(Property{Key: key}).Assign(prop).FirstOrCreate(&prop)
	if result.Error != nil {
		return result.Error
	}

	for i, existingProp := range *self {
		if existingProp.Key == key {
			(*self)[i] = prop
			return nil
		}
	}
	*self = append(*self, prop)
	return nil
}

func (self *Properties) Remove(key string) error {
	result := db.Where("key = ?", key).Delete(&Property{})
	if result.Error != nil {
		return result.Error
	}

	// Remove the property from the slice
	for i, prop := range *self {
		if prop.Key == key {
			*self = append((*self)[:i], (*self)[i+1:]...)
			break
		}
	}

	return nil
}

func (self *Properties) Get(key string) (*Property, error) {
	var prop Property
	result := db.Where("key = ?", key).First(&prop)
	if result.Error != nil {
		return nil, result.Error
	}
	if val, ok := funk.Find(*self, func(i Property) bool { return i.Key == key }).(Property); !ok {
		*self = append(*self, val)
	}
	return &prop, nil
}

// New GetValue method
func (self *Properties) GetValue(key string) (interface{}, error) {
	prop, err := self.Get(key)
	if err != nil {
		return nil, err
	}
	return prop.Value, nil
}
