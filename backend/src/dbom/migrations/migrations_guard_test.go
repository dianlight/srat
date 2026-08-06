package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"runtime"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goMigrationEntry holds the up/down function pair for a single Go migration.
type goMigrationEntry struct {
	version int
	upFn    interface{}
	downFn  interface{}
}

// allGoMigrations lists every Go-based migration in this package.
// When a new Go migration is added, add it here.
var allGoMigrations = []goMigrationEntry{
	{4, Up00004, Down00004},
	{6, Up00006, Down00006},
	{8, Up00008, Down00008},
	{9, Up00009, Down00009},
	{14, Up00014, Down00014},
	{15, Up00015, Down00015},
	{16, Up00016, Down00016},
	{18, Up00018, Down00018},
}

// funcName returns the short function name for any function value via reflection.
func funcName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

// TestMigrationInitFunctionsMatchNumber verifies that every Go migration's Up and Down
// functions contain the migration's zero-padded number in their names. This catches the
// class of bug where init() registers the wrong function pair (e.g. Up00008/Down00008
// instead of Up00014/Down00014 for migration 14).
func TestMigrationInitFunctionsMatchNumber(t *testing.T) {
	for _, m := range allGoMigrations {
		numStr := fmt.Sprintf("%05d", m.version)

		upName := funcName(m.upFn)
		assert.Contains(t, upName, numStr,
			"Up function for migration %d must contain %q in its name (got %q)",
			m.version, numStr, upName)

		downName := funcName(m.downFn)
		assert.Contains(t, downName, numStr,
			"Down function for migration %d must contain %q in its name (got %q)",
			m.version, numStr, downName)
	}
}

// TestUp00014UpdatesEmptyHASmbPassword verifies that Up00014 issues an UPDATE for
// properties rows where HASmbPassword holds a JSON-encoded empty string.
func TestUp00014UpdatesEmptyHASmbPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`UPDATE properties SET value = \? WHERE key = \? and value = \?`).
		WithArgs(sqlmock.AnyArg(), "HASmbPassword", `""`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = Up00014(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUp00015UpdatesEmptyHASmbPassword verifies that the compensating migration Up00015
// issues the same idempotent UPDATE as Up00014.
func TestUp00015UpdatesEmptyHASmbPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`UPDATE properties SET value = \? WHERE key = \? AND value = \?`).
		WithArgs(sqlmock.AnyArg(), "HASmbPassword", `""`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = Up00015(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUp00015IsIdempotentWhenNoEmptyPassword verifies that Up00015 succeeds without
// error when no rows match the empty-password condition (0 rows affected is fine).
func TestUp00015IsIdempotentWhenNoEmptyPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`UPDATE properties SET value = \? WHERE key = \? AND value = \?`).
		WithArgs(sqlmock.AnyArg(), "HASmbPassword", `""`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = Up00015(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUp00018MigratesAddonMDNSTrue verifies that an AddonMDNSRegistration=true
// property is migrated to MDNSRegistration=true + UseComponentMDNSProxy=false
// and the stale row is deleted.
func TestUp00018MigratesAddonMDNSTrue(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"value"}).AddRow("true")
	mock.ExpectQuery(`SELECT value FROM properties WHERE key = 'AddonMDNSRegistration'`).
		WillReturnRows(rows)
	mock.ExpectExec(`INSERT OR REPLACE INTO properties \(key, value, created_at, updated_at\) VALUES \('MDNSRegistration', 'true', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP\)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT OR REPLACE INTO properties \(key, value, created_at, updated_at\) VALUES \('UseComponentMDNSProxy', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP\)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM properties WHERE key = 'AddonMDNSRegistration'`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = Up00018(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUp00018IsIdempotentWhenNoAddonMDNS verifies that Up00018 succeeds without
// error and without writes when no AddonMDNSRegistration row exists.
func TestUp00018IsIdempotentWhenNoAddonMDNS(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT value FROM properties WHERE key = 'AddonMDNSRegistration'`).
		WillReturnError(sql.ErrNoRows)

	err = Up00018(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDown00018RestoresLegacyProperties verifies that Down00018 maps
// UseComponentMDNSProxy=false back to the legacy AddonMDNSRegistration=true
// scheme and removes the proxy key.
func TestDown00018RestoresLegacyProperties(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"value"}).AddRow("false")
	mock.ExpectQuery(`SELECT value FROM properties WHERE key = 'UseComponentMDNSProxy'`).
		WillReturnRows(rows)
	mock.ExpectExec(`INSERT OR REPLACE INTO properties \(key, value, created_at, updated_at\) VALUES \('AddonMDNSRegistration', 'true', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP\)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT OR REPLACE INTO properties \(key, value, created_at, updated_at\) VALUES \('MDNSRegistration', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP\)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM properties WHERE key = 'UseComponentMDNSProxy'`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = Down00018(context.Background(), db)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
