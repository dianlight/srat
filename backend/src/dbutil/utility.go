package dbutil

import (
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/repository"
	"github.com/ztrue/tracerr"
)

func FirstTimeJSONImporter(config config.Config, mount_repository repository.MountPointPathRepositoryInterface) (err error) {

	var conv converter.ConfigToDbomConverterImpl
	shares := &dbom.ExportedShares{}
	properties := &dbom.Properties{}
	users := &dbom.SambaUsers{}

	err = conv.ConfigToDbomObjects(config, properties, users, shares)
	if err != nil {
		return tracerr.Wrap(err)
	}
	err = properties.Save()
	if err != nil {
		return tracerr.Wrap(err)
	}
	err = users.Save()
	if err != nil {
		return tracerr.Wrap(err)
	}
	for i, share := range *shares {
		err = mount_repository.Save(&share.MountPointData)
		if err != nil {
			return tracerr.Wrap(err)
		}
		(*shares)[i] = share
	}
	err = shares.Save()
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func JSONFromDatabase() (tconfig config.Config, err error) {
	var conv converter.ConfigToDbomConverterImpl
	shares := dbom.ExportedShares{}
	properties := dbom.Properties{}
	users := dbom.SambaUsers{}

	err = properties.Load()
	if err != nil {
		return tconfig, tracerr.Wrap(err)
	}
	err = users.Load()
	if err != nil {
		return tconfig, tracerr.Wrap(err)
	}
	err = shares.Load()
	if err != nil {
		return tconfig, tracerr.Wrap(err)
	}

	tconfig = config.Config{}
	err = conv.DbomObjectsToConfig(properties, users, shares, &tconfig)
	if err != nil {
		return tconfig, tracerr.Wrap(err)
	}
	for _, cshare := range tconfig.Shares {
		if cshare.Usage == "media" {
			tconfig.Medialibrary.Enable = true
			break
		}
	}

	return tconfig, nil
}
