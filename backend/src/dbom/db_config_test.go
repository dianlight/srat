package dbom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCheckFileSystemPermissions covers the filesystem-level pre-flight checks
// performed before opening the SQLite database, including the Go 1.27
// strings.CutLast based directory extraction for new database files.
func TestCheckFileSystemPermissions(t *testing.T) {
	t.Run("InMemoryDatabaseSkipsChecks", func(t *testing.T) {
		err := checkFileSystemPermissions("file::memory:?cache=shared")
		assert.NoError(t, err)
	})

	t.Run("NewDatabaseInWritableDirectory", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "new.sqlite")

		err := checkFileSystemPermissions(dbPath)
		assert.NoError(t, err)

		// The write-probe file must be cleaned up.
		assert.NoFileExists(t, filepath.Join(dir, ".db_write_test"))
	})

	t.Run("QueryParametersAreStripped", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "withparams.sqlite") + "?cache=shared&_pragma=foreign_keys(1)"

		err := checkFileSystemPermissions(dbPath)
		assert.NoError(t, err)
	})

	t.Run("BareRelativeFilenameFallsBackToCurrentDirectory", func(t *testing.T) {
		// A bare filename has no directory separator; the check must fall back
		// to "." (the cwd is writable in tests) instead of failing or panicking.
		dbPath := "cutlast_probe.sqlite"

		err := checkFileSystemPermissions(dbPath)
		assert.NoError(t, err)
		assert.NoFileExists(t, dbPath) // probe must not create the DB itself
	})

	t.Run("ExistingFileWithoutWritePermission", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses file permission checks")
		}
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "readonly.sqlite")
		assert.NoError(t, os.WriteFile(dbPath, []byte("x"), 0400))

		err := checkFileSystemPermissions(dbPath)
		assert.ErrorContains(t, err, "not writable")
	})

	t.Run("NewDatabaseInReadOnlyDirectory", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses file permission checks")
		}
		dir := t.TempDir()
		assert.NoError(t, os.Chmod(dir, 0555))
		t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

		err := checkFileSystemPermissions(filepath.Join(dir, "blocked.sqlite"))
		assert.ErrorContains(t, err, "not writable")
	})
}
