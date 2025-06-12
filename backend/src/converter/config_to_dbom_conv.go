//go:build !goverter

package converter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/thoas/go-funk"
	"gitlab.com/tozd/go/errors"
)

func (c *ConfigToDbomConverterImpl) ConfigToDbomObjects(source config.Config, properties *dbom.Properties, users *dbom.SambaUsers, shares *[]dbom.ExportedShare) (err error) {
	err = c.ConfigToProperties(source, properties)
	if err != nil {
		return
	}
	for _, user := range source.OtherUsers {
		var tuser dbom.SambaUser
		err = c.UserToUSambaUser(user, &tuser)
		if err != nil {
			return
		}
		tuser.IsAdmin = false
		*users = append(*users, tuser)
	}
	var auser dbom.SambaUser
	err = c.ConfigToSambaUser(source, &auser)
	if err != nil {
		return
	}
	auser.IsAdmin = true
	*users = append(*users, auser)
	for _, share := range source.Shares {
		var sharedResource dbom.ExportedShare
		err = c.ShareToExportedShare(share, &sharedResource, users)
		if err != nil {
			return
		}
		*shares = append(*shares, sharedResource)
	}
	return
}

func (c *ConfigToDbomConverterImpl) ConfigToProperties(source config.Config, target *dbom.Properties) error {
	vsource := reflect.Indirect(reflect.ValueOf(source))
	for i := 0; i < vsource.NumField(); i++ {
		key := vsource.Type().Field(i).Name
		if funk.Contains([]string{"Shares", "OtherUsers", "ACL", "Medialibrary"}, key) {
			continue
		}
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

func (c *ConfigToDbomConverterImpl) PropertiesToConfig(source dbom.Properties, target *config.Config) error {
	for _, prop := range source {
		newvalue := reflect.ValueOf(target).Elem().FieldByName(prop.Key)
		if newvalue.IsValid() && !newvalue.IsZero() {
			if reflect.ValueOf(prop.Value).CanConvert(newvalue.Type()) {
				newvalue.Set(reflect.ValueOf(prop.Value).Convert(newvalue.Type()))
			} else {
				if newvalue.Kind() == reflect.Slice {
					newElem := reflect.New(newvalue.Type().Elem()).Elem()
					for _, value := range prop.Value.([]interface{}) {
						newElem.Set(reflect.ValueOf(value).Convert(newElem.Type()))
						newvalue.Set(reflect.Append(newvalue, newElem))
					}
				} else if newvalue.Kind() == reflect.Bool && reflect.TypeOf(prop.Value).Kind() == reflect.String {
					if strings.ToLower(prop.Value.(string)) == "true" || strings.ToLower(prop.Value.(string)) == "t" ||
						strings.ToLower(prop.Value.(string)) == "yes" || strings.ToLower(prop.Value.(string)) == "y" ||
						strings.ToLower(prop.Value.(string)) == "on" ||
						strings.ToLower(prop.Value.(string)) == "enabled" ||
						strings.ToLower(prop.Value.(string)) == "1" ||
						strings.ToLower(prop.Value.(string)) == strings.ToLower(prop.Key) {
						newvalue.SetBool(true)
					} else {
						newvalue.SetBool(false)
					}
				} else {
					return fmt.Errorf("Type mismatch for field: %s %T->%T", prop.Key, prop.Value, newvalue.Interface())
				}
			}
		}
	}
	return nil
}

func (c *ConfigToDbomConverterImpl) DbomObjectsToConfig(properties dbom.Properties, users dbom.SambaUsers, shares []dbom.ExportedShare, tconfig *config.Config) (err error) {
	err = c.PropertiesToConfig(properties, tconfig)
	if err != nil {
		return
	}
	for _, user := range users {
		if user.IsAdmin {
			tconfig.Username = user.Username
			tconfig.Password = user.Password
		} else {
			var tuser config.User
			err = c.SambaUserToUser(user, &tuser)
			if err != nil {
				return
			}
			tconfig.OtherUsers = append(tconfig.OtherUsers, tuser)
		}
	}
	if tconfig.Shares == nil {
		tconfig.Shares = config.Shares{}
	}
	for _, share := range shares {
		var tshare config.Share
		err = c.ExportedShareToShare(share, &tshare)
		if err != nil {
			return
		}
		tconfig.Shares[share.Name] = tshare
	}
	return nil
}

func (c *ConfigToDbomConverterImpl) ShareToExportedShare(source config.Share, target *dbom.ExportedShare, context *dbom.SambaUsers) error {
	err := c.ShareToExportedShareNoMountPointPath(source, target, context)
	if err != nil {
		return errors.WithStack(err)
	}
	target.MountPointData = dbom.MountPointPath{}
	err = c.ShareToMountPointPath(source, &target.MountPointData)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
