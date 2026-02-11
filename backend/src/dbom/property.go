package dbom

import (
	"log/slog"
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

func (self *Property) BeforeSave(tx *gorm.DB) (err error) {
	if self.Key == "HASmbPassword" && self.Value == "\"\"" {
		slog.Error("Try to save HASmbPassword with empty value, skipping to prevent potential issues with SMB authentication. This may indicate a problem with the SMB password retrieval process.")
		return gorm.ErrInvalidData
	}
	return nil
}