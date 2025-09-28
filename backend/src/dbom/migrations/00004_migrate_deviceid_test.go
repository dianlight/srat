package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDirEntry struct {
	name string
	mode os.FileMode
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.mode.IsDir() }
func (f fakeDirEntry) Type() os.FileMode          { return f.mode }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return fakeFileInfo{mode: f.mode}, nil }

type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string       { return "mock" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any           { return nil }

func TestGetMountPointPathaReturnsRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"path", "device"}).
		AddRow("/mnt/a", "sda1").
		AddRow("/mnt/b", "")

	mock.ExpectQuery("SELECT path, device FROM mount_point_paths").
		WillReturnRows(rows)

	paths, devices, err := getMountPointPatha(db)

	require.NoError(t, err)
	assert.Equal(t, []string{"/mnt/a", "/mnt/b"}, paths)
	assert.Equal(t, []string{"sda1", ""}, devices)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMountPointPathaHandlesNoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT path, device FROM mount_point_paths").
		WillReturnError(sql.ErrNoRows)

	paths, devices, err := getMountPointPatha(db)

	require.NoError(t, err)
	assert.Nil(t, paths)
	assert.Nil(t, devices)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUp00004UpdatesDeviceIDWhenSymlinkMatches(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"path", "device"}).
		AddRow("/mnt/a", "sda1")

	mock.ExpectQuery("SELECT path, device FROM mount_point_paths").
		WillReturnRows(rows)

	mock.ExpectExec("UPDATE mount_point_paths SET device_id = \\$1 WHERE path = \\$2").
		WithArgs("by-id-disk0", "/mnt/a").
		WillReturnResult(sqlmock.NewResult(0, 1))

	originalReadDir := readDirFunc
	originalEval := evalSymlinksFunc
	t.Cleanup(func() {
		readDirFunc = originalReadDir
		evalSymlinksFunc = originalEval
	})

	readDirFunc = func(path string) ([]os.DirEntry, error) {
		require.Equal(t, "/dev/disk/by-id/", path)
		return []os.DirEntry{fakeDirEntry{name: "disk0", mode: os.ModeSymlink}}, nil
	}

	evalSymlinksFunc = func(path string) (string, error) {
		if path == "/dev/disk/by-id/disk0" {
			return "/dev/sda1", nil
		}
		return "", fmt.Errorf("unexpected path: %s", path)
	}

	err = Up00004(context.Background(), db)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUp00004MarksEntryDeletedWhenNoMatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"path", "device"}).
		AddRow("/mnt/a", "sdb1")

	mock.ExpectQuery("SELECT path, device FROM mount_point_paths").
		WillReturnRows(rows)

	mock.ExpectExec("UPDATE mount_point_paths SET deleted_at = CURRENT_TIMESTAMP WHERE path = \\$1").
		WithArgs("/mnt/a").
		WillReturnResult(sqlmock.NewResult(0, 1))

	originalReadDir := readDirFunc
	originalEval := evalSymlinksFunc
	t.Cleanup(func() {
		readDirFunc = originalReadDir
		evalSymlinksFunc = originalEval
	})

	readDirFunc = func(path string) ([]os.DirEntry, error) {
		return nil, errors.New("boom")
	}

	evalSymlinksFunc = func(path string) (string, error) {
		return "", errors.New("unexpected call")
	}

	err = Up00004(context.Background(), db)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
