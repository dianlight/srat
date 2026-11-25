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

func TestSameDirectoryAsAny(t *testing.T) {
	volume := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(volume, "real"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(volume, "other"), 0o755))
	require.NoError(t, os.Symlink("real", filepath.Join(volume, "alias")))
	require.NoError(t, os.Symlink(".", filepath.Join(volume, "here")))

	assert.True(t, sameDirectoryAsAny(volume, "alias", []string{"other", "real"}), "symlink to a used folder")
	assert.True(t, sameDirectoryAsAny(volume, "here", []string{""}), "symlink to the volume root")
	assert.False(t, sameDirectoryAsAny(volume, "other", []string{"real", "missing"}))
	assert.False(t, sameDirectoryAsAny(volume, "missing", []string{"real"}))
	assert.False(t, sameDirectoryAsAny(volume, "real", nil))
	assert.False(t, sameDirectoryAsAny(filepath.Join(volume, "no-volume"), "real", []string{"real"}))
}

// folderTestService returns a ShareService that knows `volume` as a mount
// point, mounted or not.
func folderTestService(t *testing.T, volume string, mounted bool) *ShareService {
	t.Helper()
	diskID, partID := "disk-folders", "part-folders"
	disks := dto.NewDiskMap()
	require.NoError(t, disks.AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &map[string]dto.Partition{
		partID: {Id: &partID, DiskId: &diskID},
	}}))
	require.NoError(t, disks.AddOrUpdateMountPoint(diskID, partID, dto.MountPointData{Path: volume, IsMounted: mounted}))
	return &ShareService{disks: disks}
}

func TestListVolumeFolders(t *testing.T) {
	volume := t.TempDir()
	for _, dir := range []string{"Rodina/photos", "emperor", ".recycle", "lost+found", "Rodina/lost+found"} {
		require.NoError(t, os.MkdirAll(filepath.Join(volume, dir), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(volume, "notes.txt"), []byte("x"), 0o600))
	require.NoError(t, os.Symlink("emperor", filepath.Join(volume, "link")))
	s := folderTestService(t, volume, true)

	root, err := s.ListVolumeFolders(volume, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"emperor", "Rodina"}, root.Folders, "sorted, no files, hidden folders, symlinks or root lost+found")
	assert.Equal(t, volume, root.MountPath)
	assert.Empty(t, root.Path)
	assert.False(t, root.Truncated)

	nested, err := s.ListVolumeFolders(volume, "Rodina/")
	require.NoError(t, err)
	assert.Equal(t, "Rodina", nested.Path)
	assert.Equal(t, []string{"lost+found", "photos"}, nested.Folders, "lost+found is only hidden at the volume root")

	empty, err := s.ListVolumeFolders(volume, "Rodina/photos")
	require.NoError(t, err)
	assert.Empty(t, empty.Folders)
	assert.NotNil(t, empty.Folders)
}

func TestListVolumeFoldersRefusals(t *testing.T) {
	volume := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(volume, "escape")))
	s := folderTestService(t, volume, true)

	_, err := s.ListVolumeFolders(volume, "../etc")
	requireShareValidationError(t, err)
	_, err = s.ListVolumeFolders(volume, "escape")
	requireShareValidationError(t, err)
	_, err = s.ListVolumeFolders(volume, "missing")
	requireShareValidationError(t, err)

	_, err = s.ListVolumeFolders("/mnt/unknown", "")
	assert.True(t, errors.Is(err, dto.ErrorNotFound), "unknown volume, got %v", err)

	unmounted := folderTestService(t, volume, false)
	_, err = unmounted.ListVolumeFolders(volume, "")
	assert.True(t, errors.Is(err, dto.ErrorNotFound), "unmounted volume, got %v", err)
}

func TestListVolumeFoldersTruncates(t *testing.T) {
	previous := maxListedFolders
	maxListedFolders = 2
	t.Cleanup(func() { maxListedFolders = previous })
	volume := t.TempDir()
	for _, dir := range []string{"a", "b", "c"} {
		require.NoError(t, os.Mkdir(filepath.Join(volume, dir), 0o755))
	}

	listing, err := folderTestService(t, volume, true).ListVolumeFolders(volume, "")

	require.NoError(t, err)
	assert.Len(t, listing.Folders, 2)
	assert.True(t, listing.Truncated)
}

func TestListVolumeFoldersUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads any directory")
	}
	volume := t.TempDir()
	locked := filepath.Join(volume, "locked")
	require.NoError(t, os.Mkdir(locked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	_, err := folderTestService(t, volume, true).ListVolumeFolders(volume, "locked")

	requireShareValidationError(t, err)
}
