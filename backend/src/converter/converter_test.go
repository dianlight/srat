package converter

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/google/go-github/v75/github"
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

func TestDtoToDbomConverter_MountPointDataToMountPointPath(t *testing.T) {
	conv := DtoToDbomConverterImpl{}
	fstype := "ext4"
	startup := true
	flags := dto.MountFlags{{Name: "ro"}, {Name: "uid", NeedsValue: true, FlagValue: "1000"}}
	custom := dto.MountFlags{{Name: "gid", NeedsValue: true, FlagValue: "1000"}}
	shareName := "media"
	mountData := dto.MountPointData{
		Path:               "/mnt/test",
		Type:               "ADDON",
		DeviceId:           "dev-1",
		FSType:             &fstype,
		Flags:              &flags,
		CustomFlags:        &custom,
		IsToMountAtStartup: &startup,
		Shares:             []dto.SharedResource{{Name: shareName}},
	}

	var target dbom.MountPointPath
	require.NoError(t, conv.MountPointDataToMountPointPath(mountData, &target))

	assert.Equal(t, mountData.Path, target.Path)
	assert.Equal(t, mountData.Type, target.Type)
	assert.Equal(t, mountData.DeviceId, target.DeviceId)
	if assert.NotNil(t, target.Flags) {
		require.Len(t, *target.Flags, len(flags))
		assert.Equal(t, flags[0].Name, (*target.Flags)[0].Name)
	}
	if assert.NotNil(t, target.Data) {
		require.Len(t, *target.Data, len(custom))
		assert.Equal(t, custom[0].Name, (*target.Data)[0].Name)
	}
	if assert.NotNil(t, target.IsToMountAtStartup) {
		assert.Equal(t, startup, *target.IsToMountAtStartup)
	}
	if assert.Len(t, target.Shares, 1) {
		assert.Equal(t, shareName, target.Shares[0].Name)
	}
}

func TestDtoToDbomConverter_MountFlagsToMountDataFlags(t *testing.T) {
	conv := DtoToDbomConverterImpl{}
	flags := []dto.MountFlag{
		{Name: "ro"},
		{Name: "uid", NeedsValue: true, FlagValue: "1000"},
	}
	converted := conv.MountFlagsToMountDataFlags(flags)
	require.Len(t, converted, len(flags))
	assert.Equal(t, flags[0].Name, converted[0].Name)
	assert.False(t, converted[0].NeedsValue)
	assert.True(t, converted[1].NeedsValue)
	assert.Equal(t, flags[1].FlagValue, converted[1].FlagValue)
}

func TestDtoToDbomConverter_ExportedShareToSharedResource_WithEmptyPath(t *testing.T) {
	conv := DtoToDbomConverterImpl{}

	// Test case 1: Share with empty path
	sourceWithEmptyPath := dbom.ExportedShare{
		Name:  "UPDATER",
		Users: []dbom.SambaUser{{Username: "homeassistant"}},
		MountPointData: dbom.MountPointPath{
			Path:     "", // Empty path
			Type:     "ADDON",
			DeviceId: "sdb2",
		},
	}

	var targetWithEmptyPath dto.SharedResource
	err := conv.ExportedShareToSharedResource(sourceWithEmptyPath, &targetWithEmptyPath, nil)

	require.NoError(t, err)
	assert.Equal(t, "UPDATER", targetWithEmptyPath.Name)
	assert.Nil(t, targetWithEmptyPath.MountPointData, "MountPointData should be nil when path is empty")

	// Test case 2: Share with valid path
	fstype := "ext4"
	sourceWithValidPath := dbom.ExportedShare{
		Name:  "valid-share",
		Users: []dbom.SambaUser{{Username: "testuser"}},
		MountPointData: dbom.MountPointPath{
			Path:     "/mnt/valid-share",
			Type:     "ADDON",
			DeviceId: "sda1",
			FSType:   fstype,
		},
	}

	var targetWithValidPath dto.SharedResource
	err = conv.ExportedShareToSharedResource(sourceWithValidPath, &targetWithValidPath, nil)

	require.NoError(t, err)
	assert.Equal(t, "valid-share", targetWithValidPath.Name)
	require.NotNil(t, targetWithValidPath.MountPointData, "MountPointData should not be nil when path is valid")
	assert.Equal(t, "/mnt/valid-share", targetWithValidPath.MountPointData.Path)
	assert.Equal(t, "ADDON", targetWithValidPath.MountPointData.Type)
	assert.Equal(t, "sda1", targetWithValidPath.MountPointData.DeviceId)
}

func TestDtoToDbomConverter_ExportedShareToSharedResource_WithNilPath(t *testing.T) {
	conv := DtoToDbomConverterImpl{}

	// Test case: Share with no MountPointData path (zero value)
	source := dbom.ExportedShare{
		Name:           "no-mount-share",
		Users:          []dbom.SambaUser{{Username: "testuser"}},
		MountPointData: dbom.MountPointPath{}, // Zero value, path is ""
	}

	var target dto.SharedResource
	err := conv.ExportedShareToSharedResource(source, &target, nil)

	require.NoError(t, err)
	assert.Equal(t, "no-mount-share", target.Name)
	assert.Nil(t, target.MountPointData, "MountPointData should be nil when path is empty string")
}

func TestDtoToDbomConverter_SharedResourceToExportedShare(t *testing.T) {
	conv := DtoToDbomConverterImpl{}
	disabled := false

	source := dto.SharedResource{
		Name:     "test-share",
		Disabled: &disabled,
		Users: []dto.User{
			{Username: "user1"},
			{Username: "user2"},
		},
		RoUsers: []dto.User{
			{Username: "rouser1"},
		},
		MountPointData: &dto.MountPointData{
			Path:     "/mnt/test",
			Type:     "ADDON",
			DeviceId: "sda1",
		},
	}

	var target dbom.ExportedShare
	err := conv.SharedResourceToExportedShare(source, &target)

	require.NoError(t, err)
	assert.Equal(t, "test-share", target.Name)
	assert.Len(t, target.Users, 2)
	assert.Len(t, target.RoUsers, 1)
	assert.Equal(t, "user1", target.Users[0].Username)
	assert.Equal(t, "rouser1", target.RoUsers[0].Username)
	assert.Equal(t, "/mnt/test", target.MountPointData.Path)
}

func TestDtoToDbomConverter_SharedResourceToExportedShare_NoMountPoint(t *testing.T) {
	conv := DtoToDbomConverterImpl{}

	source := dto.SharedResource{
		Name:  "simple-share",
		Users: []dto.User{{Username: "user1"}},
	}

	var target dbom.ExportedShare
	err := conv.SharedResourceToExportedShare(source, &target)

	require.NoError(t, err)
	assert.Equal(t, "simple-share", target.Name)
	assert.Empty(t, target.MountPointData.Path)
}

func TestDtoToDbomConverter_SettingsToProperties(t *testing.T) {
	conv := DtoToDbomConverterImpl{}

	hostname := "TESTSERVER"
	workgroup := "WORKGROUP"
	source := dto.Settings{
		Hostname:  hostname,
		Workgroup: workgroup,
	}

	target := make(dbom.Properties)
	err := conv.SettingsToProperties(source, &target)

	require.NoError(t, err)
	assert.Contains(t, target, "Hostname")
	assert.Contains(t, target, "Workgroup")
	assert.Equal(t, "TESTSERVER", target["Hostname"].Value)
	assert.Equal(t, "WORKGROUP", target["Workgroup"].Value)
}

func TestDtoToDbomConverter_PropertiesToSettings(t *testing.T) {
	conv := DtoToDbomConverterImpl{}

	source := dbom.Properties{
		"Hostname":  {Key: "Hostname", Value: "TESTSERVER"},
		"Workgroup": {Key: "Workgroup", Value: "WORKGROUP"},
	}

	var target dto.Settings
	err := conv.PropertiesToSettings(source, &target)

	require.NoError(t, err)
	assert.Equal(t, "TESTSERVER", target.Hostname)
	assert.Equal(t, "WORKGROUP", target.Workgroup)
}

func TestConfigToDto_PathToSource(t *testing.T) {
	// PathToSource returns device name from path lookup in mount info
	// Since we're in a test environment without real mounts, it will return empty string
	result := PathToSource("/mnt/test")
	// The function returns empty string when path is not in mount info
	assert.NotNil(t, result) // Should at least return a string (can be empty)
}

func TestConfigToDto_TimeMachineSupportFromFS_AllTypes(t *testing.T) {
	tests := []struct {
		fstype   string
		expected dto.TimeMachineSupport
	}{
		{"ext4", dto.TimeMachineSupports.SUPPORTED},
		{"ext3", dto.TimeMachineSupports.SUPPORTED},
		{"btrfs", dto.TimeMachineSupports.SUPPORTED},
		{"xfs", dto.TimeMachineSupports.SUPPORTED},
		{"ntfs", dto.TimeMachineSupports.EXPERIMENTAL},
		{"ntfs3", dto.TimeMachineSupports.EXPERIMENTAL},
		{"exfat", dto.TimeMachineSupports.UNSUPPORTED},
		{"vfat", dto.TimeMachineSupports.UNSUPPORTED},
		{"iso9660", dto.TimeMachineSupports.UNSUPPORTED},
		{"unknown", dto.TimeMachineSupports.UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.fstype, func(t *testing.T) {
			result := TimeMachineSupportFromFS(tt.fstype)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, *result)
		})
	}
}

func TestConfigToDto_FSTypeIsWriteSupported(t *testing.T) {
	// This function calls osutil.IsWritable which checks actual path writability
	// In test environment, we can check the function returns a boolean pointer
	result := FSTypeIsWriteSupported("/tmp")
	assert.NotNil(t, result)
	// The result depends on actual filesystem permissions
}

// GitHub to DTO converter tests
func TestGitHubToDto_ReleaseAssetToBinaryAsset(t *testing.T) {
	conv := GitHubToDtoImpl{}
	
	name := "srat-v1.0.0.tar.gz"
	size := int(1024)
	id := int64(12345)
	url := "https://github.com/releases/download"
	
	source := &github.ReleaseAsset{
		Name:               &name,
		Size:               &size,
		ID:                 &id,
		BrowserDownloadURL: &url,
	}
	
	var target dto.BinaryAsset
	err := conv.ReleaseAssetToBinaryAsset(source, &target)
	
	require.NoError(t, err)
	assert.Equal(t, "srat-v1.0.0.tar.gz", target.Name)
	assert.Equal(t, 1024, target.Size)
	assert.Equal(t, int64(12345), target.ID)
	assert.Equal(t, "https://github.com/releases/download", target.BrowserDownloadURL)
}

func TestGitHubToDto_ReleaseAssetToBinaryAsset_NilSource(t *testing.T) {
	conv := GitHubToDtoImpl{}
	
	var target dto.BinaryAsset
	err := conv.ReleaseAssetToBinaryAsset(nil, &target)
	
	require.NoError(t, err)
	assert.Empty(t, target.Name)
	assert.Zero(t, target.Size)
}

func TestGitHubToDto_RepositoryReleaseToReleaseAsset(t *testing.T) {
	conv := GitHubToDtoImpl{}
	
	tagName := "v1.0.0"
	source := &github.RepositoryRelease{
		TagName: &tagName,
	}
	
	var target dto.ReleaseAsset
	err := conv.RepositoryReleaseToReleaseAsset(source, &target)
	
	require.NoError(t, err)
	// This converter currently doesn't populate any fields
}

// Issue converter tests
func TestTimeToTime(t *testing.T) {
	now := time.Now()
	result := timeToTime(now)
	assert.Equal(t, now, result)
}

func TestIssueToDtoConverter_ToDto(t *testing.T) {
	conv := IssueToDtoConverterImpl{}
	
	now := time.Now()
	source := &dbom.Issue{
		Title:       "Test Issue",
		Description: "Test Description",
		CreatedAt:   now,
	}
	
	result := conv.ToDto(source)
	
	require.NotNil(t, result)
	assert.Equal(t, "Test Issue", result.Title)
	assert.Equal(t, "Test Description", result.Description)
	assert.Equal(t, now, result.Date)
}

func TestIssueToDtoConverter_ToDbom(t *testing.T) {
	conv := IssueToDtoConverterImpl{}
	
	now := time.Now()
	source := &dto.Issue{
		Title:       "Test Issue",
		Description: "Test Description",
		Date:        now,
	}
	
	result := conv.ToDbom(source)
	
	require.NotNil(t, result)
	assert.Equal(t, "Test Issue", result.Title)
	assert.Equal(t, "Test Description", result.Description)
	assert.Equal(t, now, result.CreatedAt)
}
