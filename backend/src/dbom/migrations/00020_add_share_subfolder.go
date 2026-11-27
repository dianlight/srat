package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00020, Down00020)
}

// exportedSharesHasSubfolder reports whether exported_shares already has the
// subfolder column (GORM AutoMigrate runs before goose and may have added it).
func exportedSharesHasSubfolder(ctx context.Context, db *sql.DB) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM pragma_table_info('exported_shares') WHERE name = 'subfolder'",
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// Up00020 adds the exported_shares.subfolder column (issue #184): the folder
// inside the volume a share exposes. Empty, the default, keeps every existing
// share on its volume root. Idempotent.
func Up00020(ctx context.Context, db *sql.DB) error {
	exists, err := exportedSharesHasSubfolder(ctx, db)
	if err != nil || exists {
		return err
	}
	_, err = db.ExecContext(ctx, "ALTER TABLE exported_shares ADD COLUMN subfolder TEXT NOT NULL DEFAULT ''")
	return err
}

// Down00020 drops the subfolder column, returning every share to its volume
// root. Idempotent.
func Down00020(ctx context.Context, db *sql.DB) error {
	exists, err := exportedSharesHasSubfolder(ctx, db)
	if err != nil || !exists {
		return err
	}
	_, err = db.ExecContext(ctx, "ALTER TABLE exported_shares DROP COLUMN subfolder")
	return err
}
