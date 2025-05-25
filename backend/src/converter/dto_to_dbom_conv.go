//go:build !goverter

package converter

import (
	"fmt"
	"reflect"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/thoas/go-funk"
	"gitlab.com/tozd/go/errors"
)

type DtoToDbomConverterInterface interface {
	DtoToDbomConverter
	SharedResourceToExportedShare(source dto.SharedResource, target *dbom.ExportedShare) error
	ExportedShareToSharedResource(source dbom.ExportedShare, target *dto.SharedResource) error
	SettingsToProperties(source dto.Settings, target *dbom.Properties) error
	PropertiesToSettings(source dbom.Properties, target *dto.Settings) error
}

func (c *DtoToDbomConverterImpl) SettingsToProperties(source dto.Settings, target *dbom.Properties) error {
	keys := funk.Keys(source)
	for _, key := range keys.([]string) {
		newvalue := reflect.ValueOf(source).FieldByName(key)
		if newvalue.IsZero() {
			continue
		}
		prop := (*target)[key]
		if prop == (dbom.Property{}) {
			prop = dbom.Property{Key: key, Value: newvalue.Interface()}
		} else {
			prop.Value = newvalue.Interface()
		}
		(*target)[key] = prop
	}
	return nil
}

func (c *DtoToDbomConverterImpl) PropertiesToSettings(source dbom.Properties, target *dto.Settings) error {
	for _, prop := range source {
		newvalue := reflect.ValueOf(target).Elem().FieldByName(prop.Key)
		if newvalue.IsValid() {
			if reflect.ValueOf(prop.Value).CanConvert(newvalue.Type()) {
				newvalue.Set(reflect.ValueOf(prop.Value).Convert(newvalue.Type()))
			} else {
				if newvalue.Kind() == reflect.Slice {
					newElem := reflect.New(newvalue.Type().Elem()).Elem()
					for _, value := range prop.Value.([]interface{}) {
						newElem.Set(reflect.ValueOf(value).Convert(newElem.Type()))
						newvalue.Set(reflect.Append(newvalue, newElem))
					}
				} else {
					return fmt.Errorf("Type mismatch for field: %s", prop.Key)
				}
			}
		} /*else {
			return fmt.Errorf("Field not found: %s", prop.Key)
		}*/
	}
	return nil
}

func (c *DtoToDbomConverterImpl) SharedResourceToExportedShare(source dto.SharedResource, target *dbom.ExportedShare) error {
	err := c.SharedResourceToExportedShareNoUsersNoMountPointPath(source, target)
	if err != nil {
		return errors.WithStack(err)
	}
	target.Users = make([]dbom.SambaUser, 0, len(source.Users))
	target.RoUsers = make([]dbom.SambaUser, 0, len(source.RoUsers))
	for _, _dtoUser := range source.Users {
		var user dbom.SambaUser
		err := c.UserToSambaUser(_dtoUser, &user)
		if err != nil {
			return errors.WithStack(err)
		}
		target.Users = append(target.Users, user)
	}
	for _, _dtoUser := range source.RoUsers {
		var user dbom.SambaUser
		err := c.UserToSambaUser(_dtoUser, &user)
		if err != nil {
			return errors.WithStack(err)
		}
		target.RoUsers = append(target.RoUsers, user)
	}
	if source.MountPointData != nil {
		target.MountPointData = dbom.MountPointPath{}
		err = c.MountPointDataToMountPointPath(*source.MountPointData, &target.MountPointData)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (c *DtoToDbomConverterImpl) ExportedShareToSharedResource(source dbom.ExportedShare, target *dto.SharedResource) error {
	err := c.ExportedShareToSharedResourceNoMountPointData(source, target)
	if err != nil {
		return errors.WithStack(err)
	}
	target.MountPointData = &dto.MountPointData{}
	err = c.MountPointPathToMountPointData(source.MountPointData, target.MountPointData)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
