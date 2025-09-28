package converter

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFileInfo struct {
	isDir bool
}

func (f fakeFileInfo) Name() string       { return "mock" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.isDir }
func (f fakeFileInfo) Sys() any           { return nil }

func TestStringToSambaUserExisting(t *testing.T) {
	users := dbom.SambaUsers{{Username: "alice"}}

	user, err := StringToSambaUser("alice", &users)

	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Len(t, users, 1)
}

func TestStringToSambaUserAddsNewUser(t *testing.T) {
	users := dbom.SambaUsers{}

	user, err := StringToSambaUser("bob", &users)

	require.NoError(t, err)
	assert.Equal(t, "bob", user.Username)
	assert.Len(t, users, 1)
	assert.Equal(t, "bob", users[0].Username)
}

func TestSambaUserToString(t *testing.T) {
	value := SambaUserToString(dbom.SambaUser{Username: "charlie"})
	assert.Equal(t, "charlie", value)
}

func TestIsPathDirNotExistsWhenDirectoryExists(t *testing.T) {
	original := osStat
	t.Cleanup(func() { MockFuncOsStat(original) })

	MockFuncOsStat(func(name string) (os.FileInfo, error) {
		return fakeFileInfo{isDir: true}, nil
	})

	exists, err := isPathDirNotExists("/tmp")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestIsPathDirNotExistsWhenFileExists(t *testing.T) {
	original := osStat
	t.Cleanup(func() { MockFuncOsStat(original) })

	MockFuncOsStat(func(name string) (os.FileInfo, error) {
		return fakeFileInfo{isDir: false}, nil
	})

	exists, err := isPathDirNotExists("/tmp/file")

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestIsPathDirNotExistsWhenMissing(t *testing.T) {
	original := osStat
	t.Cleanup(func() { MockFuncOsStat(original) })

	MockFuncOsStat(func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	})

	exists, err := isPathDirNotExists("/missing")

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestIsPathDirNotExistsReturnsWrappedError(t *testing.T) {
	original := osStat
	t.Cleanup(func() { MockFuncOsStat(original) })

	sentinel := errors.New("boom")
	MockFuncOsStat(func(name string) (os.FileInfo, error) {
		return nil, sentinel
	})

	exists, err := isPathDirNotExists("/boom")

	assert.True(t, exists)
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
}

func TestExportedShareToStringRoundTrip(t *testing.T) {
	share := dbom.ExportedShare{Name: "media"}
	assert.Equal(t, "media", exportedShareToString(share))

	converted := stringToExportedShare("media")
	assert.Equal(t, share.Name, converted.Name)
}

func TestPartitionFromDeviceId(t *testing.T) {
	id := "disk-1"
	partitions := []dto.Partition{{Id: &id}}
	disks := []dto.Disk{{Partitions: &partitions}}

	result := partitionFromDeviceId(id, disks)
	if assert.NotNil(t, result) {
		assert.Equal(t, id, *result.Id)
	}
}

func TestPartitionFromDeviceIdNotFound(t *testing.T) {
	disks := []dto.Disk{}
	result := partitionFromDeviceId("missing", disks)
	assert.Nil(t, result)
}

func TestTimeMachineSupportFromFS(t *testing.T) {
	if support := TimeMachineSupportFromFS("ext4"); assert.NotNil(t, support) {
		assert.Equal(t, dto.TimeMachineSupports.SUPPORTED, *support)
	}
	if experimental := TimeMachineSupportFromFS("ntfs"); assert.NotNil(t, experimental) {
		assert.Equal(t, dto.TimeMachineSupports.EXPERIMENTAL, *experimental)
	}
	if unknown := TimeMachineSupportFromFS("customfs"); assert.NotNil(t, unknown) {
		assert.Equal(t, dto.TimeMachineSupports.UNKNOWN, *unknown)
	}
}
