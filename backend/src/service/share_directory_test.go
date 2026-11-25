package service

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

// mockMountedVolume makes isMountPoint report mounted (or not) for the test.
func mockMountedVolume(t *testing.T, mounted bool, err error) {
	t.Helper()
	previous := isMountPoint
	isMountPoint = func(string) (bool, error) { return mounted, err }
	t.Cleanup(func() { isMountPoint = previous })
}

func requireShareValidationError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.True(t, errors.Is(err, dto.ErrorShareValidation), "expected ErrorShareValidation, got %v", err)
}

func TestResolveShareDirectoryVolumeRoot(t *testing.T) {
	// No subfolder never touches the filesystem, mounted or not.
	mockMountedVolume(t, false, nil)
	dir, err := resolveShareDirectory("/mnt/not-there", "", true)
	require.NoError(t, err)
	assert.Equal(t, "/mnt/not-there", dir)
}

func TestResolveShareDirectoryCreatesFoldersLikeVolumeRoot(t *testing.T) {
	mockMountedVolume(t, true, nil)
	volume := t.TempDir()
	require.NoError(t, os.Chmod(volume, 0o775|fs.ModeSetgid))

	dir, err := resolveShareDirectory(volume, "photos/2024", true)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(volume, "photos", "2024"), dir)

	rootInfo, errS := os.Stat(volume)
	require.NoError(t, errS)
	for _, created := range []string{filepath.Join(volume, "photos"), dir} {
		info, errS := os.Stat(created)
		require.NoError(t, errS)
		assert.True(t, info.IsDir())
		assert.Equal(t, rootInfo.Mode()&(fs.ModePerm|fs.ModeSetgid), info.Mode()&(fs.ModePerm|fs.ModeSetgid), created)
	}

	// Resolving again finds the existing folders.
	again, err := resolveShareDirectory(volume, "photos/2024", false)
	require.NoError(t, err)
	assert.Equal(t, dir, again)
}

func TestResolveShareDirectoryNeverCreatesInUnmountedVolume(t *testing.T) {
	volume := t.TempDir()

	mockMountedVolume(t, false, nil)
	_, err := resolveShareDirectory(volume, "emperor", true)
	requireShareValidationError(t, err)
	assert.NoDirExists(t, filepath.Join(volume, "emperor"))

	mockMountedVolume(t, false, errors.New("mountinfo unreadable"))
	_, err = resolveShareDirectory(volume, "emperor", true)
	requireShareValidationError(t, err)
	assert.NoDirExists(t, filepath.Join(volume, "emperor"))
}

func TestResolveShareDirectoryMissingFolder(t *testing.T) {
	volume := t.TempDir()
	_, err := resolveShareDirectory(volume, "gone", false)
	requireShareValidationError(t, err)
}

func TestResolveShareDirectoryInaccessibleVolume(t *testing.T) {
	_, err := resolveShareDirectory(filepath.Join(t.TempDir(), "missing-volume"), "emperor", false)
	requireShareValidationError(t, err)
}

func TestResolveShareDirectoryRejectsFile(t *testing.T) {
	volume := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(volume, "notes"), []byte("x"), 0o600))
	_, err := resolveShareDirectory(volume, "notes", false)
	requireShareValidationError(t, err)
}

func TestResolveShareDirectoryFollowsSymlinkInsideVolume(t *testing.T) {
	mockMountedVolume(t, true, nil)
	volume := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(volume, "real"), 0o755))
	require.NoError(t, os.Symlink("real", filepath.Join(volume, "link")))

	dir, err := resolveShareDirectory(volume, "link/new", true)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(volume, "link", "new"), dir)
	assert.DirExists(t, filepath.Join(volume, "real", "new"))
}

func TestResolveShareDirectoryRejectsSymlinkOutsideVolume(t *testing.T) {
	mockMountedVolume(t, true, nil)
	volume := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(volume, "escape")))

	_, err := resolveShareDirectory(volume, "escape", false)
	requireShareValidationError(t, err)

	// The walk stops at the escaping segment: nothing is created outside.
	_, err = resolveShareDirectory(volume, "escape/new", true)
	requireShareValidationError(t, err)
	assert.NoDirExists(t, filepath.Join(outside, "new"))
}

func TestResolveShareDirectoryReportsMkdirFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	mockMountedVolume(t, true, nil)
	volume := t.TempDir()
	require.NoError(t, os.Chmod(volume, 0o555))
	t.Cleanup(func() { _ = os.Chmod(volume, 0o755) })

	_, err := resolveShareDirectory(volume, "emperor", true)
	requireShareValidationError(t, err)
}

func TestIsWithin(t *testing.T) {
	assert.True(t, isWithin("/mnt/DATA1", "/mnt/DATA1"))
	assert.True(t, isWithin("/mnt/DATA1", "/mnt/DATA1/emperor"))
	assert.True(t, isWithin("/mnt/DATA1", "/mnt/DATA1/..hidden"))
	assert.False(t, isWithin("/mnt/DATA1", "/mnt"))
	assert.False(t, isWithin("/mnt/DATA1", "/mnt/DATA10"))
	assert.False(t, isWithin("/mnt/DATA1", "/etc"))
}
