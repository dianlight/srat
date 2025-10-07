package dbom_test

import (
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/stretchr/testify/assert"
)

func TestMountPointPathBeforeSave_ValidPath(t *testing.T) {
	mpp := dbom.MountPointPath{
		Path: "/valid/path",
	}

	err := mpp.BeforeSave(nil)
	assert.NoError(t, err)
}

func TestMountPointPathBeforeSave_EmptyPath(t *testing.T) {
	mpp := dbom.MountPointPath{
		Path: "",
	}

	err := mpp.BeforeSave(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path cannot be empty")
}

func TestMountPointPathBeforeSave_PathNotStartingWithSlash(t *testing.T) {
	mpp := dbom.MountPointPath{
		Path: "relative/path",
	}

	err := mpp.BeforeSave(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must start with '/'")
}

func TestMountPointPathBeforeSave_PathWithNullCharacter(t *testing.T) {
	mpp := dbom.MountPointPath{
		Path: "/path/with\x00null",
	}

	err := mpp.BeforeSave(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot contain null characters")
}

func TestMountPointPathBeforeSave_PathWithInvalidCharacters(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"space", "/path with space"},
		{"special char", "/path@with$special"},
		{"bracket", "/path[with]bracket"},
		{"asterisk", "/path*with/asterisk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpp := dbom.MountPointPath{
				Path: tt.path,
			}

			err := mpp.BeforeSave(nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid characters")
		})
	}
}

func TestMountPointPathFields(t *testing.T) {
	mpp := dbom.MountPointPath{
		Path:     "/mnt/test",
		Type:     "ext4",
		DeviceId: "/dev/sda1",
		FSType:   "ext4",
	}

	assert.Equal(t, "/mnt/test", mpp.Path)
	assert.Equal(t, "ext4", mpp.Type)
	assert.Equal(t, "/dev/sda1", mpp.DeviceId)
	assert.Equal(t, "ext4", mpp.FSType)
}

func TestMountPointPathWithValidCharacters(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"simple", "/mnt/data"},
		{"with underscore", "/mnt/my_data"},
		{"with hyphen", "/mnt/my-data"},
		{"with dot", "/mnt/data.backup"},
		{"nested", "/mnt/storage/data"},
		{"complex", "/mnt/my_data-2024.backup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpp := dbom.MountPointPath{
				Path: tt.path,
			}

			err := mpp.BeforeSave(nil)
			assert.NoError(t, err, "Path %s should be valid", tt.path)
		})
	}
}

