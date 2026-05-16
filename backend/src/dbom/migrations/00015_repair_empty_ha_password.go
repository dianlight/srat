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
	goose.AddMigrationNoTxContext(Up00015, Down00015)
}

// Up00015 repairs databases where migration 14 ran incorrectly due to the wrong function
// registration bug (Up00008 was registered instead of Up00014). It re-runs the same UPDATE
// logic as Up00014 idempotently: only rows with an empty JSON-encoded HASmbPassword are touched.
func Up00015(ctx context.Context, db *sql.DB) error {
	HASmbPassword, errc := osutil.GenerateSecurePassword()
	if errc != nil {
		slog.ErrorContext(ctx, "Cant generate password", "errc", errc)
		HASmbPassword = "changeme"
	}

	// Idempotent: only updates rows where the value is a JSON-encoded empty string ("").
	queryUpdate := "UPDATE properties SET value = ? WHERE key = ? AND value = ?"
	if _, err := db.ExecContext(ctx, queryUpdate, "\""+HASmbPassword+"\"", "HASmbPassword", "\"\""); err != nil {
		return err
	}
	return nil
}

func Down00015(ctx context.Context, db *sql.DB) error {
	return nil
}
