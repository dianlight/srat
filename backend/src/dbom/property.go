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

func (self *Properties) Populate(props []Property) {
	for _, prop := range props {
		(*self)[prop.Key] = prop
	}
}

/*
func (self *Properties) AddInternalValue(key string, value any) error {
	prop, err := self.Get(key)
	if err != nil {
		prop = &Property{Key: key}
	}

	prop.Value = value
	prop.Internal = true
	(*self)[key] = *prop
	return nil
}
*/
