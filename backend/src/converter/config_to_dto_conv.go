//go:build !goverter

package converter

import (
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/xorcare/pointer"
	"github.com/ztrue/tracerr"
)

func (c *ConfigToDtoConverterImpl) ConfigToDtoObjects(source config.Config, settings *dto.Settings, users *[]dto.User, shares *[]dto.SharedResource) error {
	err := c.ConfigToSettings(source, settings)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, user := range source.OtherUsers {
		var tuser dto.User
		err := c.OtherUserToUser(user, &tuser)
		if err != nil {
			return tracerr.Wrap(err)
		}
		tuser.IsAdmin = pointer.Bool(false)
		*users = append(*users, tuser)
	}
	var auser dto.User
	err = c.ConfigToUser(source, &auser)
	if err != nil {
		return tracerr.Wrap(err)
	}
	auser.IsAdmin = pointer.Bool(true)
	*users = append(*users, auser)
	for _, share := range source.Shares {
		var sharedResource dto.SharedResource
		err := c.ShareToSharedResource(share, &sharedResource, *users)
		if err != nil {
			return tracerr.Wrap(err)
		}
		*shares = append(*shares, sharedResource)
	}
	return nil
}

func (c *ConfigToDtoConverterImpl) DtoObjectsToConfig(settings dto.Settings, users []dto.User, shares []dto.SharedResource, target *config.Config) error {
	err := c.SettingsToConfig(settings, target, c)
	if err != nil {
		return tracerr.Wrap(err)
	}
	for _, user := range users {
		var tuser config.User
		if *user.IsAdmin {
			target.Username = *user.Username
			target.Password = *user.Password
		} else {
			err := c.UserToOtherUser(user, &tuser)
			if err != nil {
				return tracerr.Wrap(err)
			}
			target.OtherUsers = append(target.OtherUsers, tuser)
		}
	}
	for _, share := range shares {
		var tshare config.Share
		err := c.SharedResourceToShare(share, &tshare)
		if err != nil {
			return tracerr.Wrap(err)
		}
		target.Shares[share.Name] = tshare
	}
	return nil
}

func (c *ConfigToDtoConverterImpl) ShareToSharedResource(source config.Share, target *dto.SharedResource, context []dto.User) error {
	err := c.ShareToSharedResourceNoMountPointData(source, target, context)
	if err != nil {
		return tracerr.Wrap(err)
	}
	var mountPointData dto.MountPointData
	err = c.ShareToMountPointData(source, &mountPointData)
	if err != nil {
		return tracerr.Wrap(err)
	}
	target.MountPointData = &mountPointData
	return nil
}
