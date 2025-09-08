package migrations

//package main

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00006, Down00006)
}

func Up00006(ctx context.Context, db *sql.DB) error {
	// Drop the device_id column from the mount_point_paths table
	query := "ALTER TABLE mount_point_paths DROP COLUMN device"
	if _, err := db.ExecContext(ctx, query); err != nil {
		if !strings.Contains(err.Error(), "no such column") {
			return err
		}
		return nil
	}
	return nil
}

func Down00006(ctx context.Context, db *sql.DB) error {
	// Re-add the device_id column to the mount_point_paths table
	query := "ALTER TABLE mount_point_paths ADD COLUMN device TEXT"
	if _, err := db.ExecContext(ctx, query); err != nil {
		return err
	}
	return nil
}
