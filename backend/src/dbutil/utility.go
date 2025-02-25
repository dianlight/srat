package dbutil

import (
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/repository"
	"github.com/ztrue/tracerr"
)

func FirstTimeJSONImporter(config config.Config,
	mount_repository repository.MountPointPathRepositoryInterface,
	export_share_repository repository.ExportedShareRepositoryInterface,
) (err error) {

	var conv converter.ConfigToDbomConverterImpl
	shares := &[]dbom.ExportedShare{}
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
		//		slog.Debug("Share ", "id", share.MountPointData.ID)
		(*shares)[i] = share
	}
	err = export_share_repository.SaveAll(shares)
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func JSONFromDatabase(export_share_repository repository.ExportedShareRepositoryInterface) (tconfig config.Config, err error) {
	var conv converter.ConfigToDbomConverterImpl
	shares := []dbom.ExportedShare{}
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
	err = export_share_repository.All(&shares)
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
