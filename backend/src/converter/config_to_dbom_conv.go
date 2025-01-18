//go:build !goverter

package converter

import (
	"fmt"
	"reflect"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/thoas/go-funk"
)

func (c *ConfigToDbomConverterImpl) ConfigToDbomObjects(source config.Config, properties *dbom.Properties, users *dbom.SambaUsers, shares *dbom.ExportedShares) (err error) {
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
		err = c.ShareToExportedShare(share, &sharedResource, *users)
		if err != nil {
			return
		}
		*shares = append(*shares, sharedResource)
	}
	return
}

func (c *ConfigToDbomConverterImpl) ConfigToProperties(source config.Config, target *dbom.Properties) error {
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

func (c *ConfigToDbomConverterImpl) PropertiesToConfig(source dbom.Properties, target *config.Config) error {
	for _, prop := range source {
		newvalue := reflect.ValueOf(target).FieldByName(prop.Key)
		if newvalue.IsValid() {
			newvalue.Set(reflect.ValueOf(prop.Value))
		} else {
			return fmt.Errorf("Field not found: %s", prop.Key)
		}
	}
	return nil
}

func (c *ConfigToDbomConverterImpl) DbomObjectsToConfig(properties dbom.Properties, users dbom.SambaUsers, shares dbom.ExportedShares, tconfig *config.Config) (err error) {
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
