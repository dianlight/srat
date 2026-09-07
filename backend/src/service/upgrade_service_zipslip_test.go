package service

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type zipTestEntry struct {
	name      string
	body      string
	isSymlink bool
	comment   string
}

func buildZipFiles(t *testing.T, entries []zipTestEntry) map[string]*zip.File {
	t.Helper()
	zipPath := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(zipPath)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Store}
		if e.isSymlink {
			h.SetMode(os.ModeSymlink | 0o777)
		} else {
			h.SetMode(0o644)
			h.Comment = e.comment
		}
		fw, err := w.CreateHeader(h)
		require.NoError(t, err)
		_, err = fw.Write([]byte(e.body))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
	r, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close() })
	out := make(map[string]*zip.File)
	for _, zf := range r.File {
		out[zf.Name] = zf
	}
	return out
}

func newTestUpgradeService() *UpgradeService {
	return &UpgradeService{ctx: context.Background()}
}

func TestExtractFile_RejectsDotDotAndAbsoluteNames(t *testing.T) {
	svc := newTestUpgradeService()
	cases := []struct {
		name      string
		entryName string
		isSymlink bool
		body      string
	}{
		{name: "dotdot symlink name", entryName: "../evil", isSymlink: true, body: "target"},
		{name: "dotdot nested symlink name", entryName: "a/../../evil", isSymlink: true, body: "target"},
		{name: "absolute symlink name", entryName: "/abs/path", isSymlink: true, body: "target"},
		{name: "dotdot regular name", entryName: "../evil", isSymlink: false, body: "data"},
		{name: "absolute regular name", entryName: "/abs/path", isSymlink: false, body: "data"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dest := t.TempDir()
			comment := ""
			if !tc.isSymlink {
				comment = "dummy-signature"
			}
			files := buildZipFiles(t, []zipTestEntry{{name: tc.entryName, body: tc.body, isSymlink: tc.isSymlink, comment: comment}})
			zf, ok := files[tc.entryName]
			require.True(t, ok, "zip entry missing")
			_, err := svc.extractFile(zf, dest)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "illegal file path")
		})
	}
}

func TestExtractFile_RejectsUnsafeSymlinkTargets(t *testing.T) {
	svc := newTestUpgradeService()
	cases := []struct {
		name   string
		target string
	}{
		{name: "absolute target", target: "/etc/passwd"},
		{name: "dotdot target", target: "../outside"},
		{name: "nested dotdot target", target: "a/../../etc/passwd"},
		{name: "empty target", target: "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dest := t.TempDir()
			files := buildZipFiles(t, []zipTestEntry{{name: "good-link", body: tc.target, isSymlink: true}})
			_, err := svc.extractFile(files["good-link"], dest)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "symlink target")
		})
	}
}

func TestExtractFile_RejectsPlantedSymlinkEscape(t *testing.T) {
	svc := newTestUpgradeService()
	t.Run("regular file through planted link", func(t *testing.T) {
		dest := t.TempDir()
		require.NoError(t, os.Symlink("..", filepath.Join(dest, "link")))
		files := buildZipFiles(t, []zipTestEntry{{name: "link/evil.txt", body: "data", comment: "dummy-signature"}})
		_, err := svc.extractFile(files["link/evil.txt"], dest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "illegal file path")
	})
	t.Run("symlink entry through planted link", func(t *testing.T) {
		dest := t.TempDir()
		require.NoError(t, os.Symlink("..", filepath.Join(dest, "link")))
		files := buildZipFiles(t, []zipTestEntry{{name: "link/evil-link", body: "some-target", isSymlink: true}})
		_, err := svc.extractFile(files["link/evil-link"], dest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "illegal file path")
	})
	t.Run("target traverses planted link", func(t *testing.T) {
		dest := t.TempDir()
		outside := t.TempDir()
		require.NoError(t, os.Symlink(outside, filepath.Join(dest, "a")))
		files := buildZipFiles(t, []zipTestEntry{{name: "evil", body: "a/passwd", isSymlink: true}})
		_, err := svc.extractFile(files["evil"], dest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "illegal symlink target")
	})
	t.Run("regular file over existing symlink", func(t *testing.T) {
		dest := t.TempDir()
		outsideFile := filepath.Join(t.TempDir(), "outside.txt")
		require.NoError(t, os.WriteFile(outsideFile, []byte("secret"), 0o644))
		require.NoError(t, os.Symlink(outsideFile, filepath.Join(dest, "victim")))
		files := buildZipFiles(t, []zipTestEntry{{name: "victim", body: "data", comment: "dummy-signature"}})
		_, err := svc.extractFile(files["victim"], dest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "illegal file path")
	})
}

func TestExtractFile_SymlinkHappyPath(t *testing.T) {
	svc := newTestUpgradeService()
	dest := t.TempDir()
	files := buildZipFiles(t, []zipTestEntry{{name: "my-link", body: "my-target", isSymlink: true}})
	got, extractErr := svc.extractFile(files["my-link"], dest)
	require.NoError(t, extractErr)
	require.NotNil(t, got)
	assert.Equal(t, filepath.Join(dest, "my-link"), got.Path)
	info, lstatErr := os.Lstat(filepath.Join(dest, "my-link"))
	require.NoError(t, lstatErr)
	require.NotEqual(t, info.Mode()&os.ModeSymlink, 0)
	target, readlinkErr := os.Readlink(filepath.Join(dest, "my-link"))
	require.NoError(t, readlinkErr)
	assert.Equal(t, "my-target", target)
}
