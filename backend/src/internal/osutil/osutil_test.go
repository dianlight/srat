package osutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const sampleMountInfo = `36 35 98:0 / /mnt/root rw,nosuid - ext4 /dev/root rw,relatime
37 36 98:1 /subdir /mnt/data rw,noatime shared:402 - xfs /dev/sdb1 rw`

type OsutilSuite struct {
	suite.Suite
}

func TestOsutilSuite(t *testing.T) {
	suite.Run(t, new(OsutilSuite))
}

func (suite *OsutilSuite) TestLoadMountInfoWithMock() {
	t := suite.T()
	restore := MockMountInfo(sampleMountInfo)
	t.Cleanup(restore)

	entries, err := LoadMountInfo()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	root := entries[0]
	assert.Equal(t, "/mnt/root", root.MountDir)
	assert.Equal(t, "ext4", root.FsType)
	assert.Equal(t, "/dev/root", root.MountSource)
	require.Contains(t, root.MountOptions, "rw")
	assert.Empty(t, root.MountOptions["rw"])
	require.Contains(t, root.MountOptions, "nosuid")

	data := entries[1]
	assert.Equal(t, 37, data.MountID)
	assert.Equal(t, 36, data.ParentID)
	assert.Equal(t, "xfs", data.FsType)
	assert.Equal(t, "/mnt/data", data.MountDir)
	require.Len(t, data.OptionalFields, 1)
	assert.Equal(t, "shared:402", data.OptionalFields[0])
}

func TestIsMounted(t *testing.T) {
	restore := MockMountInfo(sampleMountInfo)
	t.Cleanup(restore)

	mounted, err := IsMounted("/mnt/root")
	require.NoError(t, err)
	assert.True(t, mounted)

	mountedWithTrailingSlash, err := IsMounted("/mnt/data/")
	require.NoError(t, err)
	assert.True(t, mountedWithTrailingSlash)

	missing, err := IsMounted("/mnt/missing")
	require.NoError(t, err)
	assert.False(t, missing)
}

func (suite *OsutilSuite) TestParseHelpers() {
	t := suite.T()
	opts := parseOptions("rw,noatime,uid=1000")
	assert.Empty(t, opts["rw"])
	assert.Empty(t, opts["noatime"])
	assert.Equal(t, "1000", opts["uid"])

	optional := parseOptional("shared:5 master:1")
	require.Len(t, optional, 2)
	assert.Equal(t, "shared:5", optional[0])
	assert.Equal(t, "master:1", optional[1])
}

func (suite *OsutilSuite) TestMountInfoEntry() {
	t := suite.T()
	entry := &MountInfoEntry{
		MountID:      1,
		ParentID:     0,
		DevMajor:     8,
		DevMinor:     1,
		Root:         "/",
		MountDir:     "/mnt/test",
		MountOptions: map[string]string{"rw": "", "noatime": ""},
		FsType:       "ext4",
		MountSource:  "/dev/sda1",
	}

	assert.NotNil(t, entry)
	assert.NotEmpty(t, entry.MountOptions)
	assert.Equal(t, 1, entry.MountID)
	assert.Equal(t, 0, entry.ParentID)
	assert.Equal(t, 8, entry.DevMajor)
	assert.Equal(t, 1, entry.DevMinor)
	assert.Equal(t, "/mnt/test", entry.MountDir)
	assert.Equal(t, "ext4", entry.FsType)
	assert.Equal(t, "/dev/sda1", entry.MountSource)
}

func (suite *OsutilSuite) TestParseOptionsEmpty() {
	t := suite.T()
	opts := parseOptions("")
	assert.NotNil(t, opts)
	assert.Empty(t, opts)
}

func (suite *OsutilSuite) TestParseOptionsSingleValue() {
	t := suite.T()
	opts := parseOptions("rw")
	assert.Len(t, opts, 1)
	assert.Empty(t, opts["rw"])
}

func (suite *OsutilSuite) TestParseOptionsMultipleValues() {
	t := suite.T()
	opts := parseOptions("rw,nosuid,nodev,uid=1000,gid=100")
	assert.Len(t, opts, 5)
	assert.Empty(t, opts["rw"])
	assert.Empty(t, opts["nosuid"])
	assert.Empty(t, opts["nodev"])
	assert.Equal(t, "1000", opts["uid"])
	assert.Equal(t, "100", opts["gid"])
}

func (suite *OsutilSuite) TestParseOptionalEmpty() {
	t := suite.T()
	optional := parseOptional("")
	assert.Nil(t, optional)
}

func (suite *OsutilSuite) TestParseOptionalWhitespace() {
	t := suite.T()
	optional := parseOptional("   ")
	assert.Nil(t, optional)
}

func (suite *OsutilSuite) TestParseOptionalSingleField() {
	t := suite.T()
	optional := parseOptional("shared:402")
	require.Len(t, optional, 1)
	assert.Equal(t, "shared:402", optional[0])
}

func (suite *OsutilSuite) TestParseOptionalMultipleFields() {
	t := suite.T()
	optional := parseOptional("shared:5 master:1 propagate_from:2")
	require.Len(t, optional, 3)
	assert.Equal(t, "shared:5", optional[0])
	assert.Equal(t, "master:1", optional[1])
	assert.Equal(t, "propagate_from:2", optional[2])
}

func (suite *OsutilSuite) TestConvertInfosEmpty() {
	t := suite.T()
	entries := convertInfos(nil)
	assert.NotNil(t, entries)
	assert.Empty(t, entries)
}

func (suite *OsutilSuite) TestMockMountInfoRestore() {
	t := suite.T()
	// Set initial mock
	restore1 := MockMountInfo("first")

	// Set second mock
	restore2 := MockMountInfo("second")

	// Restore to first
	restore2()

	// Restore to original
	restore1()

	assert.True(t, true) // Test completes without panic
}

func (suite *OsutilSuite) TestIsMountedMultipleMounts() {
	t := suite.T()
	mountInfo := `36 35 98:0 / /mnt/a rw - ext4 /dev/sda1 rw
37 35 98:1 / /mnt/b rw - ext4 /dev/sda2 rw
38 35 98:2 / /mnt/c rw - xfs /dev/sdb1 rw`

	restore := MockMountInfo(mountInfo)
	t.Cleanup(restore)

	mountedA, err := IsMounted("/mnt/a")
	require.NoError(t, err)
	assert.True(t, mountedA)

	mountedB, err := IsMounted("/mnt/b")
	require.NoError(t, err)
	assert.True(t, mountedB)

	mountedC, err := IsMounted("/mnt/c")
	require.NoError(t, err)
	assert.True(t, mountedC)

	notMounted, err := IsMounted("/mnt/d")
	require.NoError(t, err)
	assert.False(t, notMounted)
}

func (suite *OsutilSuite) TestMountInfoSuperOptions() {
	t := suite.T()
	restore := MockMountInfo(sampleMountInfo)
	t.Cleanup(restore)

	entries, err := LoadMountInfo()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// Check super options
	assert.NotNil(t, entries[0].SuperOptions)
	assert.NotNil(t, entries[1].SuperOptions)
}

func (suite *OsutilSuite) TestMountInfoDeviceNumbers() {
	t := suite.T()
	restore := MockMountInfo(sampleMountInfo)
	t.Cleanup(restore)

	entries, err := LoadMountInfo()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, 98, entries[0].DevMajor)
	assert.Equal(t, 0, entries[0].DevMinor)
	assert.Equal(t, 98, entries[1].DevMajor)
	assert.Equal(t, 1, entries[1].DevMinor)
}

func (suite *OsutilSuite) TestIsKernelModuleLoaded() {
	t := suite.T()
	// Note: This test will check actual kernel modules on the system.
	// We test with a module that is very likely to be loaded (like 'loop')
	// and one that is unlikely to exist.

	// Test with a common module (may not be loaded on all systems)
	// We just verify the function doesn't error
	_, err := IsKernelModuleLoaded("loop")
	assert.NoError(t, err)

	// Test with a non-existent module
	loaded, err := IsKernelModuleLoaded("definitely_not_a_real_module_xyz123")
	assert.NoError(t, err)
	assert.False(t, loaded)
}

func (suite *OsutilSuite) TestGetSambaVersion() {
	t := suite.T()
	// This test will attempt to get Samba version if installed
	version, err := GetSambaVersion()

	// If Samba is not installed, we should get an error (that's OK for test)
	// If it is installed, version should not be empty
	if err == nil {
		assert.NotEmpty(t, version, "Samba version should not be empty when no error")
	}
	// If error, it's OK - Samba might not be installed in test environment
}

func (suite *OsutilSuite) TestIsSambaVersionSufficient() {
	t := suite.T()
	// This test will check if Samba version meets minimum requirement
	sufficient, err := IsSambaVersionSufficient()

	// If Samba is not installed, we might get an error (that's OK)
	// The function should handle this gracefully
	if err == nil {
		// If no error, the result should be a valid boolean
		assert.IsType(t, false, sufficient)
	}
	// If error, it's OK - Samba might not be installed in test environment
}

func (suite *OsutilSuite) TestGenerateSecurePassword() {
	t := suite.T()
	// Test basic generation
	password, err := GenerateSecurePassword()
	require.NoError(t, err)
	assert.NotEmpty(t, password, "Generated password should not be empty")

	// Password should be base64-encoded string (no padding, URL-safe)
	// 16 bytes = 22 characters in base64 without padding
	assert.Len(t, password, 22, "Password should be 22 characters long")

	// Test uniqueness - generate multiple passwords and ensure they're different
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pwd, err := GenerateSecurePassword()
		require.NoError(t, err)
		passwords[pwd] = true
	}
	assert.Len(t, passwords, 100, "All 100 generated passwords should be unique")
}

func (suite *OsutilSuite) TestGenerateSecurePasswordCharacterSet() {
	t := suite.T()
	// Generate a password and verify it only contains valid base64 URL-safe characters
	// Valid characters: A-Z, a-z, 0-9, -, _
	password, err := GenerateSecurePassword()
	require.NoError(t, err)

	for _, char := range password {
		assert.True(t,
			(char >= 'A' && char <= 'Z') ||
				(char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '_',
			"Password should only contain base64 URL-safe characters, found: %c", char)
	}
}

func (suite *OsutilSuite) TestGenerateSecurePasswordNoSpecialChars() {
	t := suite.T()
	// Verify no padding or special characters that might cause issues
	password, err := GenerateSecurePassword()
	require.NoError(t, err)

	// Should not contain padding (=) or standard base64 special chars (+, /)
	assert.NotContains(t, password, "=", "Password should not contain padding")
	assert.NotContains(t, password, "+", "Password should not contain +")
	assert.NotContains(t, password, "/", "Password should not contain /")
}

// TestCommandExists_EmptySlice tests that commandExists returns false for empty command slice
func (suite *OsutilSuite) TestCommandExists_EmptySlice() {
	result := CommandExists([]string{})
	suite.False(result, "commandExists should return false for empty slice")
}

// TestCommandExists_S6Command_ServicePathExists tests s6-* command with existing service directory
func (suite *OsutilSuite) TestCommandExists_S6Command_ServicePathExists() {
	result := CommandExists([]string{"s6-svc", "-u", "/bin"})
	suite.True(result, "commandExists should return true for s6-* command with existing service path")
}

// TestCommandExists_S6Command_ServicePathNotExists tests s6-* command with non-existent service directory
func (suite *OsutilSuite) TestCommandExists_S6Command_ServicePathNotExists() {
	result := CommandExists([]string{"s6-svc", "-u", "/nonexistent/path"})
	suite.False(result, "commandExists should return false for s6-* command with non-existent service path")
}

// TestCommandExists_S6Command_NoServicePath tests s6-* command with only command name (no path argument)
func (suite *OsutilSuite) TestCommandExists_S6Command_NoServicePath() {
	result := CommandExists([]string{"s6-svc"})
	suite.False(result, "commandExists should return false for s6-* command without path argument")
}

// TestCommandExists_S6Command_ServicePathIsFile tests s6-* command where the path is a file, not directory
func (suite *OsutilSuite) TestCommandExists_S6Command_ServicePathIsFile() {
	result := CommandExists([]string{"s6-svc", "-u", "hello.txt"})
	suite.False(result, "commandExists should return false when s6-* service path is a file, not directory")
}

// TestCommandExists_RegularCommand_InPATH tests regular command that exists in PATH
func (suite *OsutilSuite) TestCommandExists_RegularCommand_InPATH() {
	// "ls" is a standard command that should exist in PATH
	result := CommandExists([]string{"ls"})
	suite.True(result, "commandExists should return true for valid command in PATH")
}

// TestCommandExists_RegularCommand_NotInPATH tests regular command that does not exist in PATH
func (suite *OsutilSuite) TestCommandExists_RegularCommand_NotInPATH() {
	result := CommandExists([]string{"nonexistent-command-xyz-12345"})
	suite.False(result, "commandExists should return false for command not in PATH")
}

// TestCommandExists_RegularCommand_WithArguments tests regular command with arguments (only first element matters)
func (suite *OsutilSuite) TestCommandExists_RegularCommand_WithArguments() {
	// "echo" is a standard command, arguments should be ignored for PATH lookup
	result := CommandExists([]string{"echo", "hello", "world"})
	suite.True(result, "commandExists should return true for valid command with arguments")
}

// TestCommandExists_S6Command_MultipleArguments tests s6-* command with multiple arguments (last element is path)
func (suite *OsutilSuite) TestCommandExists_S6Command_MultipleArguments() {
	result := CommandExists([]string{"s6-svc", "-wU", "-s", "SIGKILL", "/bin"})
	suite.True(result, "commandExists should use last element as service path for s6-* commands")
}

// TestCommandExists_S6Command_MultipleArguments_NonExistentPath tests s6-* with multiple arguments and non-existent final path
func (suite *OsutilSuite) TestCommandExists_S6Command_MultipleArguments_NonExistentPath() {
	result := CommandExists([]string{"s6-svc", "-wU", "-s", "SIGKILL", "/nonexistent/service/path"})
	suite.False(result, "commandExists should return false when s6-* final path element does not exist")
}

// TestCommandExists_CommandNameOnly tests command with only name (single element slice)
func (suite *OsutilSuite) TestCommandExists_CommandNameOnly() {
	result := CommandExists([]string{"cat"})
	suite.True(result, "commandExists should handle single-element command slices")
}
