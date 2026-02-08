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

func Up00014(ctx context.Context, db *sql.DB) error {

	HASmbPassword, errc := osutil.GenerateSecurePassword()
	if errc != nil {
		slog.ErrorContext(ctx, "Cant generate password", "errc", errc)
		HASmbPassword = "changeme"
	}

	// Update the HASmbPassword setting in the settings table
	queryUpdate := "UPDATE properties SET value = ? WHERE key = ? and value = ?"
	if _, err := db.ExecContext(ctx, queryUpdate, "\""+HASmbPassword+"\"", "HASmbPassword", "\"\""); err != nil {
		return err
	}
	return nil
}

func Down00014(ctx context.Context, db *sql.DB) error {
	return nil
}
