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

	HASmbPassword, errc := osutil.GenerateSecurePassword()
	if errc != nil {
		slog.ErrorContext(ctx, "Cant generate password", "errc", errc)
		HASmbPassword = "changeme"
	}

	// Update the HASmbPassword setting in the settings table
	queryUpdate := "INSERT OR IGNORE INTO properties (key,value) VALUES (?, ?)"
	if _, err := db.ExecContext(ctx, queryUpdate, "HASmbPassword", "\""+HASmbPassword+"\""); err != nil {
		return err
	}
	return nil
}

func Down00008(ctx context.Context, db *sql.DB) error {
	return nil
}
