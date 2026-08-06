package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00018, Down00018)
}

// Up00018 migrates the legacy AddonMDNSRegistration property (addon-side direct
// mDNS) to the new master/proxy scheme:
//
//	AddonMDNSRegistration=true  → MDNSRegistration=true, UseComponentMDNSProxy=false
//	AddonMDNSRegistration=false → no change (defaults apply)
//
// The old MDNSRegistration property keeps its name but now means the master
// switch; a previously stored true (old HA proxy mode) is preserved as
// master=true with the proxy defaulting to true. The stale AddonMDNSRegistration
// row no longer maps to any dto.Settings field and is removed.
func Up00018(ctx context.Context, db *sql.DB) error {
	var addonMDNS string
	row := db.QueryRowContext(ctx, "SELECT value FROM properties WHERE key = 'AddonMDNSRegistration'")
	if err := row.Scan(&addonMDNS); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	if addonMDNS == "true" || addonMDNS == `"true"` {
		// Direct mode was active → master on, proxy off.
		if _, err := db.ExecContext(ctx,
			"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('MDNSRegistration', 'true', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx,
			"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('UseComponentMDNSProxy', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		); err != nil {
			return err
		}
	}

	// The old key no longer exists in dto.Settings — drop it.
	if _, err := db.ExecContext(ctx, "DELETE FROM properties WHERE key = 'AddonMDNSRegistration'"); err != nil {
		return err
	}

	return nil
}

func Down00018(ctx context.Context, db *sql.DB) error {
	var proxy string
	row := db.QueryRowContext(ctx, "SELECT value FROM properties WHERE key = 'UseComponentMDNSProxy'")
	if err := row.Scan(&proxy); err != nil && err != sql.ErrNoRows {
		return err
	}

	if proxy == "false" || proxy == `"false"` {
		// Direct mode was active → restore the legacy direct property.
		if _, err := db.ExecContext(ctx,
			"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('AddonMDNSRegistration', 'true', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx,
			"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('MDNSRegistration', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		); err != nil {
			return err
		}
	} else if proxy == "true" || proxy == `"true"` {
		// Component mode was active → restore the legacy proxy property.
		if _, err := db.ExecContext(ctx,
			"INSERT OR REPLACE INTO properties (key, value, created_at, updated_at) VALUES ('AddonMDNSRegistration', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		); err != nil {
			return err
		}
	}

	// The new proxy key no longer exists in the old scheme — drop it.
	if _, err := db.ExecContext(ctx, "DELETE FROM properties WHERE key = 'UseComponentMDNSProxy'"); err != nil {
		return err
	}

	return nil
}
