package dbom

import (
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/mapper"
)

func FirstTimeJSONImporter(config config.Config) error {
	// Migrate from JSON to DB
	var settings dto.Settings
	err := mapper.Map(&settings, config)
	if err != nil {
		return err
	}
	var properties Properties
	err = mapper.Map(&properties, settings)
	if err != nil {
		return err
	}
	err = properties.Save()
	if err != nil {
		return err
	}

	// Users
	var users []dto.User
	err = mapper.Map(&users, config)
	if err != nil {
		return err
	}
	var sambaUsers SambaUsers
	err = mapper.Map(&sambaUsers, users)
	if err != nil {
		return err
	}
	err = sambaUsers.Save()
	if err != nil {
		return err
	}
	// Shares
	var shares []dto.SharedResource
	err = mapper.Map(&shares, config)
	if err != nil {
		return err
	}
	var sambaShares ExportedShares
	err = mapper.Map(&sambaShares, shares)
	if err != nil {
		return err
	}
	err = sambaShares.Save()
	if err != nil {
		return err
	}

	return nil
}
