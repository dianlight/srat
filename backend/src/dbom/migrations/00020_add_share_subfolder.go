package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00020, Down00020)
}

// columnExists reports whether the given column exists on the given table.
func columnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt *string
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// Up00020 ensures the exported_shares table has a subfolder column.
// Idempotent: GORM AutoMigrate may already have created the column, in which
// case this is a no-op (avoids "duplicate column name" errors).
func Up00020(ctx context.Context, db *sql.DB) error {
	exists, err := columnExists(ctx, db, "exported_shares", "subfolder")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE exported_shares ADD COLUMN subfolder TEXT DEFAULT ''"); err != nil {
		return fmt.Errorf("failed to add subfolder column: %w", err)
	}
	return nil
}

// Down00020 drops the subfolder column when supported. SQLite < 3.35 does not
// support DROP COLUMN, so this is best-effort and non-fatal.
func Down00020(ctx context.Context, db *sql.DB) error {
	exists, err := columnExists(ctx, db, "exported_shares", "subfolder")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	_, err = db.ExecContext(ctx, "ALTER TABLE exported_shares DROP COLUMN subfolder")
	return err
}
