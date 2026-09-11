package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00019, Down00019)
}

// Up00019 migrates the SmartMode string property to the SmartOn boolean property.
// Mapping: SmartMode="none" → SmartOn=false, otherwise → SmartOn=true.
// A missing SmartMode row means the legacy default ("legacy") → SmartOn=true.
// The stale SmartMode row no longer maps to any dto.Settings field and is removed.
func Up00019(ctx context.Context, db *sql.DB) error {
	var smartMode string
	row := db.QueryRowContext(ctx, "SELECT value FROM properties WHERE key = 'SmartMode'")
	if err := row.Scan(&smartMode); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	smartOn := "true"
	if smartMode == `"none"` || smartMode == "none" {
		smartOn = "false"
	}

	if _, err := db.ExecContext(ctx,
		"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('SmartOn', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		smartOn,
	); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, "DELETE FROM properties WHERE key = 'SmartMode'"); err != nil {
		return err
	}

	return nil
}

func Down00019(ctx context.Context, db *sql.DB) error {
	var smartOn string
	row := db.QueryRowContext(ctx, "SELECT value FROM properties WHERE key = 'SmartOn'")
	if err := row.Scan(&smartOn); err != nil && err != sql.ErrNoRows {
		return err
	}

	smartMode := `"legacy"`
	if smartOn == "false" || smartOn == `"false"` {
		smartMode = `"none"`
	}

	if _, err := db.ExecContext(ctx,
		"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('SmartMode', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		smartMode,
	); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, "DELETE FROM properties WHERE key = 'SmartOn'"); err != nil {
		return err
	}

	return nil
}
