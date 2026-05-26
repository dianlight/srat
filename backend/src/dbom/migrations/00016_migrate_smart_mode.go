package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00016, Down00016)
}

// Up00016 migrates the boolean DisableSmart property to the SmartMode string property.
// Mapping: DisableSmart=true → SmartMode="none", otherwise → SmartMode="legacy".
func Up00016(ctx context.Context, db *sql.DB) error {
	// Determine the current value of DisableSmart (default to false if not present)
	var disableSmart string
	row := db.QueryRowContext(ctx, "SELECT value FROM properties WHERE key = 'DisableSmart'")
	if err := row.Scan(&disableSmart); err != nil && err != sql.ErrNoRows {
		return err
	}

	smartMode := `"legacy"`
	if disableSmart == "true" || disableSmart == `"true"` {
		smartMode = `"none"`
	}

	// Insert or replace SmartMode property
	if _, err := db.ExecContext(ctx,
		"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('SmartMode', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		smartMode,
	); err != nil {
		return err
	}

	return nil
}

func Down00016(ctx context.Context, db *sql.DB) error {
	// Reverse: read SmartMode and write DisableSmart boolean
	var smartMode string
	row := db.QueryRowContext(ctx, "SELECT value FROM properties WHERE key = 'SmartMode'")
	if err := row.Scan(&smartMode); err != nil && err != sql.ErrNoRows {
		return err
	}

	disableSmart := "false"
	if smartMode == `"none"` {
		disableSmart = "true"
	}

	if _, err := db.ExecContext(ctx,
		"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('DisableSmart', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		disableSmart,
	); err != nil {
		return err
	}

	// Remove SmartMode property
	if _, err := db.ExecContext(ctx, "DELETE FROM properties WHERE key = 'SmartMode'"); err != nil {
		return err
	}

	return nil
}
