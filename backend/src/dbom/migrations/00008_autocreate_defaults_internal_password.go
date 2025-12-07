package migrations

//package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/dianlight/srat/internal/osutil"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00008, Down00008)
}

func Up00008(ctx context.Context, db *sql.DB) error {

	_ha_mount_user_password_, errc := osutil.GenerateSecurePassword()
	if errc != nil {
		slog.ErrorContext(ctx, "Cant generate password", "errc", errc)
		_ha_mount_user_password_ = "changeme"
	}

	// Update the _ha_mount_user_password_ setting in the settings table
	queryUpdate := "INSERT OR IGNORE INTO properties (key,value,internal) VALUES (?, ?, ?)"
	if _, err := db.ExecContext(ctx, queryUpdate, "_ha_mount_user_password_", "\""+_ha_mount_user_password_+"\"", 1); err != nil {
		return err
	}
	return nil
}

func Down00008(ctx context.Context, db *sql.DB) error {
	return nil
}
