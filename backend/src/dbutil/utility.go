package dbutil

import (
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/repository"
	"gitlab.com/tozd/go/errors"
)

func FirstTimeJSONImporter(config config.Config,
	mount_repository repository.MountPointPathRepositoryInterface,
	props_repository repository.PropertyRepositoryInterface,
	export_share_repository repository.ExportedShareRepositoryInterface,
	users_repository repository.SambaUserRepositoryInterface,
) (err error) {

	var conv converter.ConfigToDbomConverterImpl
	shares := &[]dbom.ExportedShare{}
	properties := &dbom.Properties{}
	users := &dbom.SambaUsers{}

	err = conv.ConfigToDbomObjects(config, properties, users, shares)
	if err != nil {
		return errors.WithStack(err)
	}
	err = props_repository.SaveAll(properties)
	if err != nil {
		return errors.WithStack(err)
	}
	err = users_repository.SaveAll(users)
	if err != nil {
		return errors.WithStack(err)
	}
	for i, share := range *shares {
		err = mount_repository.Save(&share.MountPointData)
		if err != nil {
			return errors.WithStack(err)
		}
		//		slog.Debug("Share ", "id", share.MountPointData.ID)
		(*shares)[i] = share
	}
	err = export_share_repository.SaveAll(shares)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func JSONFromDatabase(
	export_share_repository repository.ExportedShareRepositoryInterface,
	props_repository repository.PropertyRepositoryInterface,
	users_repository repository.SambaUserRepositoryInterface,
) (tconfig config.Config, err error) {
	var conv converter.ConfigToDbomConverterImpl
	shares := []dbom.ExportedShare{}

	properties, err := props_repository.All()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}
	users, err := users_repository.All()
	if err != nil {
		return tconfig, errors.WithStack(err)
	}
	err = export_share_repository.All(&shares)
	if err != nil {
		return tconfig, errors.WithStack(err)
	}

	tconfig = config.Config{}
	err = conv.DbomObjectsToConfig(properties, users, shares, &tconfig)
	if err != nil {
		return tconfig, errors.WithStack(err)
	}
	for _, cshare := range tconfig.Shares {
		if cshare.Usage == "media" {
			tconfig.Medialibrary.Enable = true
			break
		}
	}

	return tconfig, nil
}
