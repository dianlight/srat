package dbom

import (
	"context"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/mapper"
)

func FirstTimeJSONImporter(config config.Config) error {
	// Migrate from JSON to DB
	var settings dto.Settings
	err := mapper.Map(context.Background(), &settings, config)
	if err != nil {
		return err
	}
	var properties Properties
	err = mapper.Map(context.Background(), &properties, settings)
	if err != nil {
		return err
	}
	err = properties.Save()
	if err != nil {
		return err
	}

	// Users
	var users []dto.User
	err = mapper.Map(context.Background(), &users, config)
	if err != nil {
		return err
	}
	var sambaUsers SambaUsers
	err = mapper.Map(context.Background(), &sambaUsers, users)
	if err != nil {
		return err
	}
	err = sambaUsers.Save()
	if err != nil {
		return err
	}
	// Shares
	var shares []dto.SharedResource
	err = mapper.Map(context.Background(), &shares, config)
	if err != nil {
		return err
	}
	var sambaShares ExportedShares
	err = mapper.Map(context.Background(), &sambaShares, shares)
	if err != nil {
		return err
	}
	err = sambaShares.Save()
	if err != nil {
		return err
	}

	return nil
}
