//go:build !goverter

package converter

import (
	"database/sql"
	"database/sql/driver" // Added for driver.Valuer
	"fmt"
	"log/slog" // Added for logging
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
		newvalueReflected := reflect.ValueOf(source).FieldByName(key)

		// Ensure the field is valid and its value can be accessed.
		// This handles unexported fields or if funk.Keys returned a non-existent field name.
		if !newvalueReflected.IsValid() || !newvalueReflected.CanInterface() {
			slog.Warn("Skipping invalid or un-interfaceable field in SettingsToProperties", "key", key)
			continue
		}

		// Default value to set is the direct interface of the field's value
		valToSet := newvalueReflected.Interface()

		// Get the actual Go value from reflect.Value
		iface := newvalueReflected.Interface()

		// Check if the Go value implements driver.Valuer
		if valuer, ok := iface.(driver.Valuer); ok {
			var dv driver.Value
			var errValue error

			// Check if 'iface' (which is the valuer) is a nil pointer.
			// If it is, Value() would panic if called on a nil pointer receiver.
			// database/sql's internal callValuerValue handles this by returning (nil, nil).
			rvIface := reflect.ValueOf(iface)
			if rvIface.Kind() == reflect.Pointer && rvIface.IsNil() {
				dv = nil // Treat as SQL NULL
				errValue = nil
			} else {
				// If 'iface' is not a nil pointer (or not a pointer at all),
				// it's safe to call Value().
				dv, errValue = valuer.Value()
			}

			if errValue != nil {
				slog.Error("driver.Valuer Value() method returned error", "key", key, "error", errValue)
				return errors.Wrapf(errValue, "failed to get value from driver.Valuer for key %s", key)
			}
			valToSet = dv // Use the value from Value() method
		}

		prop := (*target)[key]
		if prop == (dbom.Property{}) { // Check if property exists or is zero-value
			prop = dbom.Property{Key: key, Value: valToSet}
		} else {
			prop.Value = valToSet
		}
		(*target)[key] = prop
	}
	return nil
}

func (c *DtoToDbomConverterImpl) PropertiesToSettings(source dbom.Properties, target *dto.Settings) error {
	var scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

	for _, prop := range source {
		newvalue := reflect.ValueOf(target).Elem().FieldByName(prop.Key)
		if newvalue.IsValid() {
			if prop.Value == nil {
				newvalue.Set(reflect.Zero(newvalue.Type()))
			} else if reflect.ValueOf(prop.Value).CanConvert(newvalue.Type()) {
				newvalue.Set(reflect.ValueOf(prop.Value).Convert(newvalue.Type()))
			} else if newvalue.CanAddr() && newvalue.Addr().Type().Implements(scannerType) {
				// If the field implements sql.Scanner, use its Scan method.
				scanner := newvalue.Addr().Interface().(sql.Scanner)
				err := scanner.Scan(prop.Value)
				if err != nil {
					return fmt.Errorf("error scanning field %s: %w", prop.Key, err)
				}
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
