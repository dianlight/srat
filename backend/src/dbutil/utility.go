package dbutil

import (
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
)

func FirstTimeJSONImporter(config config.Config) (err error) {

	var conv converter.ConfigToDbomConverterImpl
	shares := &dbom.ExportedShares{}
	properties := &dbom.Properties{}
	users := &dbom.SambaUsers{}

	err = conv.ConfigToDbomObjects(config, properties, users, shares)
	if err != nil {
		return err
	}
	err = properties.Save()
	if err != nil {
		return err
	}
	err = users.Save()
	if err != nil {
		return err
	}
	err = shares.Save()
	if err != nil {
		return err
	}
	return nil
}
