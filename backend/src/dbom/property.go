package dbom

import (
	"time"

	"gorm.io/gorm"
)

type Property struct {
	Key       string      `json:"key" gorm:"primaryKey" mapper:"key"`
	Value     interface{} `json:"value" mapper:"value" gorm:"serializer:json"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Properties map[string]Property

func (self *Properties) Load() error {
	var props []Property
	err := db.Model(&Property{}).Find(&props).Error
	if err != nil {
		return err
	}
	*self = make(Properties, len(props))
	for _, prop := range props {
		(*self)[prop.Key] = prop
	}
	return nil
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

func (self *Properties) Add(key string, value any) error {
	prop := Property{
		Key:   key,
		Value: value,
	}

	db.Unscoped().Model(&Property{}).Where("key", key).Update("deleted_at", nil)

	result := db.Where(Property{Key: key}).Assign(prop).FirstOrCreate(&prop)
	if result.Error != nil {
		return result.Error
	}

	(*self)[key] = prop
	return nil
}

func (self *Properties) Remove(key string) error {
	result := db.Where("key = ?", key).Delete(&Property{})
	if result.Error != nil {
		return result.Error
	}

	delete(*self, key)

	return nil
}

func (self *Properties) Get(key string) (*Property, error) {
	var prop Property
	result := db.Where("key = ?", key).First(&prop)
	if result.Error != nil {
		return nil, result.Error
	}
	(*self)[key] = prop

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
