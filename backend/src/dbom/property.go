package dbom

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type Property struct {
	Key       string      `json:"key" gorm:"primaryKey" from_map:"key"`
	Value     interface{} `json:"value" from_map:"value" gorm:"serializer:json"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Properties []Property

func (p *Properties) Load() error {
	return db.Find(p).Error
}

func (self *Properties) Save() error {
	return db.Save(self).Error
}

func (p *Properties) DeleteAll() error {
	result := db.Delete(&Property{})
	if result.Error != nil {
		return result.Error
	}
	*p = nil
	return nil
}

func (p *Properties) Add(key string, value interface{}) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	prop := Property{
		Key:   key,
		Value: string(jsonValue),
	}

	db.Unscoped().Model(&Property{}).Where("key", key).Update("deleted_at", nil)

	result := db.Where(Property{Key: key}).Assign(prop).FirstOrCreate(&prop)
	if result.Error != nil {
		return result.Error
	}

	for i, existingProp := range *p {
		if existingProp.Key == key {
			(*p)[i] = prop
			return nil
		}
	}
	*p = append(*p, prop)
	return nil
}

func (p *Properties) Remove(key string) error {
	result := db.Where("key = ?", key).Delete(&Property{})
	if result.Error != nil {
		return result.Error
	}

	// Remove the property from the slice
	for i, prop := range *p {
		if prop.Key == key {
			*p = append((*p)[:i], (*p)[i+1:]...)
			break
		}
	}

	return nil
}

func (p *Properties) Get(key string) (*Property, error) {
	var prop Property
	result := db.Where("key = ?", key).First(&prop)
	if result.Error != nil {
		return nil, result.Error
	}
	return &prop, nil
}

/*
func (p *Property) UnmarshalValue(v interface{}) error {
	return json.Unmarshal([]byte(p.Value), v)
}
*/
// New GetValue method
func (p *Properties) GetValue(key string) (interface{}, error) {
	//var v interface{}
	prop, err := p.Get(key)
	if err != nil {
		return nil, err
	}
	//	err = prop.UnmarshalValue(v)
	//	return &v, err
	return prop.Value, nil
}

/*


func (p *Properties) FromSettings(setting dto.Settings) error {
	return db.Transaction(func(tx *gorm.DB) error {
		err := p.DeleteAll()
		if err != nil {
			return err
		}
		mapSetting := setting.ToMap()
		for key, value := range mapSetting {
			err := p.Add(key, value)
			if err != nil {
				return err
			}
		}
		return nil
	})
}


func (p *Properties) ToSettings(setting *dto.Settings) {
	mapSetting := setting.ToMap()
	for _, prop := range *p {
		mapSetting[prop.Key] = prop.Value
	}
}

*/
