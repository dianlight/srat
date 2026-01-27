package repository

/*
type PropertyRepository struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

type PropertyRepositoryInterface interface {
	All() (dbom.Properties, errors.E)
	SaveAll(props *dbom.Properties) errors.E
	Value(key string) (interface{}, errors.E)
	SetValue(key string, value interface{}) errors.E
	DumpTable() (string, errors.E)
}

func NewPropertyRepositoryRepository(db *gorm.DB) PropertyRepositoryInterface {
	return &PropertyRepository{
		mutex: sync.RWMutex{},
		db:    db,
	}
}

func (self *PropertyRepository) All() (dbom.Properties, errors.E) {
	var props []dbom.Property
	var err error
	err = self.db.Model(&dbom.Property{}).Find(&props).Error
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

func (self *PropertyRepository) SaveAll(props *dbom.Properties) errors.E {
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
func (self *PropertyRepository) SetValue(key string, value interface{}) errors.E {
	prop := dbom.Property{
		Key:   key,
		Value: value,
	}
	return errors.WithStack(self.db.Save(&prop).Error)
}

func (self *PropertyRepository) Value(key string) (interface{}, errors.E) {
	var prop dbom.Property
	res := self.db.Model(&dbom.Property{}).First(&prop, "key = ?", key)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, errors.WithStack(dto.ErrorNotFound)
		}
		//slog.Error("Error retrieving property", "key", key, "include_internal", include_internal, "res", res)
		return nil, errors.WithStack(res.Error)
	}
	return prop.Value, nil
}

func (self *PropertyRepository) DumpTable() (string, errors.E) {
	ret := strings.Builder{}
	ret.WriteString("Properties Table Dump:\n")
	var props []dbom.Property
	err := self.db.Model(&dbom.Property{}).Find(&props).Error
	if err != nil {
		return "", errors.WithStack(err)
	}
	for _, prop := range props {
		ret.WriteString(fmt.Sprintf("Key: %s, Value: %v\n", prop.Key, prop.Value))
	}
	return ret.String(), nil
}
*/
