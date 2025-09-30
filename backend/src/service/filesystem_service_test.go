package service_test

import (
	"context"
	"strings"
	"syscall"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type FilesystemServiceTestSuite struct {
	suite.Suite
	fsService service.FilesystemServiceInterface
	ctx       context.Context
}

func TestFilesystemServiceTestSuite(t *testing.T) {
	suite.Run(t, new(FilesystemServiceTestSuite))
}

func (suite *FilesystemServiceTestSuite) TestSyscallDataToMountFlag() {
	testCases := []struct {
		name          string
		data          string
		expectedFlags []dto.MountFlag
	}{
		{
			name:          "Empty data string",
			data:          "",
			expectedFlags: []dto.MountFlag{},
		},
		{
			name: "Single option with value",
			data: "uid=1000",
			expectedFlags: []dto.MountFlag{
				{Name: "uid", FlagValue: "1000", NeedsValue: true, Description: "Set owner of all files to user ID", ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			},
		},
		{
			name: "Multiple options with values",
			data: "uid=1000,gid=1000",
			expectedFlags: []dto.MountFlag{
				{Name: "uid", FlagValue: "1000", NeedsValue: true, Description: "Set owner of all files to user ID", ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
				{Name: "gid", FlagValue: "1000", NeedsValue: true, Description: "Set group of all files to group ID", ValueDescription: "Group ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			},
		},
		{
			name: "Option without value",
			data: "acl",
			expectedFlags: []dto.MountFlag{
				{Name: "acl", NeedsValue: false, Description: "Enable POSIX Access Control Lists support"},
			},
		},
		{
			name: "Mixed options with and without values",
			data: "uid=1000,acl",
			expectedFlags: []dto.MountFlag{
				{Name: "uid", FlagValue: "1000", NeedsValue: true, Description: "Set owner of all files to user ID", ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
				{Name: "acl", NeedsValue: false, Description: "Enable POSIX Access Control Lists support"},
			},
		},
		{
			name: "Empty options are skipped",
			data: "uid=1000,,gid=1000",
			expectedFlags: []dto.MountFlag{
				{Name: "uid", FlagValue: "1000", NeedsValue: true, Description: "Set owner of all files to user ID", ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
				{Name: "gid", FlagValue: "1000", NeedsValue: true, Description: "Set group of all files to group ID", ValueDescription: "Group ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
			},
		},
		{
			name: "Unknown option",
			data: "unknown=value",
			expectedFlags: []dto.MountFlag{
				{Name: "unknown", FlagValue: "value", NeedsValue: true},
			},
		},
		{
			name: "Known and unknown options mixed",
			data: "uid=1000,unknown=value",
			expectedFlags: []dto.MountFlag{
				{Name: "uid", FlagValue: "1000", NeedsValue: true, Description: "Set owner of all files to user ID", ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
				{Name: "unknown", FlagValue: "value", NeedsValue: true},
			},
		},
		{
			name: "Option with spaces around",
			data: "  uid = 1000  ,  acl  ",
			expectedFlags: []dto.MountFlag{
				{Name: "uid", FlagValue: "1000", NeedsValue: true, Description: "Set owner of all files to user ID", ValueDescription: "User ID (numeric)", ValueValidationRegex: `^[0-9]+$`},
				{Name: "acl", NeedsValue: false, Description: "Enable POSIX Access Control Lists support"},
			},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			flags, err := suite.fsService.SyscallDataToMountFlag(tc.data)
			require.NoError(t, err)
			assert.Len(t, flags, len(tc.expectedFlags), "Number of flags should match")

			for i, expectedFlag := range tc.expectedFlags {
				t.Logf("Checking case %s flag %s", tc.name, expectedFlag.Name)
				assert.Equal(t, expectedFlag.Name, flags[i].Name, "Flag name mismatch")
				assert.Equal(t, expectedFlag.FlagValue, flags[i].FlagValue, "Flag value mismatch")
				assert.Equal(t, expectedFlag.NeedsValue, flags[i].NeedsValue, "NeedsValue mismatch")
				if expectedFlag.Description != "" {
					assert.Equal(t, expectedFlag.Description, flags[i].Description, "Description mismatch")
					assert.Equal(t, expectedFlag.ValueDescription, flags[i].ValueDescription, "ValueDescription mismatch")
					assert.Equal(t, expectedFlag.ValueValidationRegex, flags[i].ValueValidationRegex, "ValueValidationRegex mismatch")
				}
			}
		})
	}
}

func (suite *FilesystemServiceTestSuite) TestSyscallFlagToMountFlag() {
	testCases := []struct {
		name            string
		syscallFlag     uintptr
		expectedFlags   []string // List of expected flag names
		unexpectedFlags []string // List of flags that should NOT be present
	}{
		{
			name:            "No flags set",
			syscallFlag:     0,
			expectedFlags:   []string{},
			unexpectedFlags: []string{"ro", "nosuid", "nodev"},
		},
		{
			name:            "MS_RDONLY set",
			syscallFlag:     syscall.MS_RDONLY,
			expectedFlags:   []string{"ro"},
			unexpectedFlags: []string{"nosuid", "nodev"},
		},
		{
			name:            "MS_NOSUID and MS_NOEXEC set",
			syscallFlag:     syscall.MS_NOSUID | syscall.MS_NOEXEC,
			expectedFlags:   []string{"nosuid", "noexec"},
			unexpectedFlags: []string{"ro", "nodev"},
		},
		{
			name:          "All supported flags set",
			syscallFlag:   syscall.MS_RDONLY | syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV | syscall.MS_SYNCHRONOUS | syscall.MS_REMOUNT | syscall.MS_MANDLOCK | syscall.MS_DIRSYNC | syscall.MS_NOATIME | syscall.MS_NODIRATIME | syscall.MS_BIND | syscall.MS_REC | syscall.MS_SILENT | syscall.MS_POSIXACL | syscall.MS_UNBINDABLE | syscall.MS_PRIVATE | syscall.MS_SLAVE | syscall.MS_SHARED | syscall.MS_RELATIME | syscall.MS_STRICTATIME,
			expectedFlags: []string{"ro", "nosuid", "noexec", "nodev", "sync", "remount", "mand", "dirsync", "noatime", "nodiratime", "bind", "rec", "silent", "acl", "unbindable", "private", "slave", "shared", "relatime", "strictatime"},
		},
		/*
			{
				name:            "MS_POSIXACL alias acl",
				syscallFlag:     syscall.MS_POSIXACL,
				expectedFlags:   []string{"acl"}, // Expecting the alias
				unexpectedFlags: []string{"posixacl"},
			},
		*/
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			flags, err := suite.fsService.SyscallFlagToMountFlag(tc.syscallFlag)
			require.NoError(t, err, "Unexpected error")

			// Check for expected flags
			for _, expectedFlagName := range tc.expectedFlags {
				found := false
				for _, flag := range flags {
					if flag.Name == expectedFlagName {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected flag '%s' not found", expectedFlagName)
			}

			// Check for unexpected flags
			for _, unexpectedFlagName := range tc.unexpectedFlags {
				for _, flag := range flags {
					assert.NotEqual(t, unexpectedFlagName, flag.Name, "Unexpected flag '%s' found", unexpectedFlagName)
				}
			}
		})
	}
}

func (suite *FilesystemServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.fsService = service.NewFilesystemService(suite.ctx)
	suite.Require().NotNil(suite.fsService, "FilesystemService should be initialized")
}

func (suite *FilesystemServiceTestSuite) TestGetStandardMountFlags() {
	stdFlags, err := suite.fsService.GetStandardMountFlags()
	suite.Require().NoError(err)
	suite.Require().NotNil(stdFlags)

	// Check for a few expected flags
	foundRo := false
	foundNoExec := false
	for _, flag := range stdFlags {
		if flag.Name == "ro" {
			foundRo = true
		}
		if flag.Name == "noexec" {
			foundNoExec = true
		}
	}
	suite.True(foundRo, "Standard flag 'ro' not found")
	suite.True(foundNoExec, "Standard flag 'noexec' not found")

	// Verify it's not empty
	suite.NotEmpty(stdFlags)
}

func (suite *FilesystemServiceTestSuite) TestGetFilesystemSpecificMountFlags() {
	// Test with a known filesystem type (ntfs)
	ntfsFlags, err := suite.fsService.GetFilesystemSpecificMountFlags("ntfs")
	suite.Require().NoError(err)
	suite.Require().NotNil(ntfsFlags)
	suite.NotEmpty(ntfsFlags, "Expected specific flags for ntfs")

	foundUID := false
	for _, flag := range ntfsFlags {
		if flag.Name == "uid" {
			foundUID = true
			suite.True(flag.NeedsValue, "ntfs uid flag should need a value")
		}
	}
	suite.True(foundUID, "ntfs specific flag 'uid' not found")

	// Test with another known filesystem type (ntfs3)
	ntfs3Flags, err := suite.fsService.GetFilesystemSpecificMountFlags("ntfs3")
	suite.Require().NoError(err)
	suite.Require().NotNil(ntfs3Flags)
	suite.NotEmpty(ntfs3Flags, "Expected specific flags for ntfs3")
	foundForce := false
	for _, flag := range ntfs3Flags {
		if flag.Name == "force" {
			foundForce = true
			suite.False(flag.NeedsValue, "ntfs3 force flag should not need a value")
		}
	}
	suite.True(foundForce, "ntfs3 specific flag 'force' not found")

	// Test with an unknown filesystem type
	unknownFlags, err := suite.fsService.GetFilesystemSpecificMountFlags("someunknownfs")
	suite.Require().NoError(err)
	suite.Require().NotNil(unknownFlags)
	suite.Empty(unknownFlags, "Expected no specific flags for an unknown filesystem type")

	// Test with zfs
	zfsFlags, err := suite.fsService.GetFilesystemSpecificMountFlags("zfs")
	suite.Require().NoError(err)
	suite.Require().NotNil(zfsFlags)
	suite.NotEmpty(zfsFlags)
	foundContext := false
	for _, flag := range zfsFlags {
		if flag.Name == "context" {
			foundContext = true
			suite.True(flag.NeedsValue, "zfs context flag should need a value")
		}
	}
	suite.True(foundContext, "zfs specific flag 'context' not found")

	// Test with xfs
	xfsFlags, err := suite.fsService.GetFilesystemSpecificMountFlags("xfs")
	suite.Require().NoError(err)
	suite.Require().NotNil(xfsFlags)
	suite.NotEmpty(xfsFlags, "Expected specific flags for xfs")

	var foundInode64, foundAllocsize bool
	for _, flag := range xfsFlags {
		if flag.Name == "inode64" {
			foundInode64 = true
			suite.False(flag.NeedsValue, "xfs inode64 flag should not need a value")
		}
		if flag.Name == "allocsize" {
			foundAllocsize = true
			suite.True(flag.NeedsValue, "xfs allocsize flag should need a value")
			suite.Equal("Size in bytes optionally with K, M, or G suffix (e.g., 1G)", flag.ValueDescription)
			suite.Equal(`^[0-9]+([kKmMgG])?$`, flag.ValueValidationRegex)
		}
	}
	suite.True(foundInode64, "xfs specific flag 'inode64' not found")
	suite.True(foundAllocsize, "xfs specific flag 'allocsize' not found")
}

func (suite *FilesystemServiceTestSuite) TestGetMountFlagsAndData() {
	testCases := []struct {
		name             string
		inputFlags       []dto.MountFlag
		expectedSyscall  uintptr
		expectedData     string
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name:            "Empty input",
			inputFlags:      []dto.MountFlag{},
			expectedSyscall: 0,
			expectedData:    "",
		},
		{
			name:            "Standard ro flag",
			inputFlags:      []dto.MountFlag{{Name: "ro"}},
			expectedSyscall: syscall.MS_RDONLY,
			expectedData:    "",
		},
		{
			name:            "Standard nosuid and noexec flags",
			inputFlags:      []dto.MountFlag{{Name: "nosuid"}, {Name: "noexec"}},
			expectedSyscall: syscall.MS_NOSUID | syscall.MS_NOEXEC,
			expectedData:    "",
		},
		{
			name:            "Flag with value (uid)",
			inputFlags:      []dto.MountFlag{{Name: "uid", FlagValue: "1000", NeedsValue: true}},
			expectedSyscall: 0,
			expectedData:    "uid=1000",
		},
		{
			name:            "Mixed flags (ro, uid)",
			inputFlags:      []dto.MountFlag{{Name: "ro"}, {Name: "uid", FlagValue: "1000", NeedsValue: true}},
			expectedSyscall: syscall.MS_RDONLY,
			expectedData:    "uid=1000",
		},
		{
			name:            "Multiple data flags",
			inputFlags:      []dto.MountFlag{{Name: "uid", FlagValue: "1000", NeedsValue: true}, {Name: "gid", FlagValue: "1001", NeedsValue: true}},
			expectedSyscall: 0,
			expectedData:    "uid=1000,gid=1001",
		},
		{
			name:            "Ignored flags (rw, defaults, async)",
			inputFlags:      []dto.MountFlag{{Name: "rw"}, {Name: "defaults"}, {Name: "async"}},
			expectedSyscall: 0,
			expectedData:    "",
		},
		{
			name:            "Unknown flag",
			inputFlags:      []dto.MountFlag{{Name: "unknownflag"}},
			expectedSyscall: 0,
			expectedData:    "", // Unknown flags for bitmask are ignored, those for data field would be passed if Value is set
		},
		{
			name:            "Flag with value for unknown flag",
			inputFlags:      []dto.MountFlag{{Name: "mycustomflag", FlagValue: "myvalue", NeedsValue: true}},
			expectedSyscall: 0,
			expectedData:    "mycustomflag=myvalue",
		},
		{
			name:             "Boolean flag with unexpected value",
			inputFlags:       []dto.MountFlag{{Name: "ro", FlagValue: "true", NeedsValue: false}},
			expectError:      true,
			expectedErrorMsg: "Boolean/switch flag was provided with a value",
		},
		{
			name:            "Flag with explicit NeedsValue false and no value",
			inputFlags:      []dto.MountFlag{{Name: "noatime", NeedsValue: false}},
			expectedSyscall: syscall.MS_NOATIME,
			expectedData:    "",
		},
		{
			name:            "Flag with explicit NeedsValue true and value",
			inputFlags:      []dto.MountFlag{{Name: "fmask", FlagValue: "0022", NeedsValue: true}},
			expectedSyscall: 0,
			expectedData:    "fmask=0022",
		},
		{
			name:            "Case insensitivity for syscall flags",
			inputFlags:      []dto.MountFlag{{Name: "Ro"}},
			expectedSyscall: syscall.MS_RDONLY,
			expectedData:    "",
		},
		{
			name:            "Flag with spaces (should be trimmed for syscall, preserved for data)",
			inputFlags:      []dto.MountFlag{{Name: " nosuid "}},
			expectedSyscall: syscall.MS_NOSUID,
			expectedData:    "",
		},
		{
			name:            "Bind and rec flags",
			inputFlags:      []dto.MountFlag{{Name: "bind"}, {Name: "rec"}},
			expectedSyscall: syscall.MS_BIND | syscall.MS_REC,
			expectedData:    "",
		},
		{
			name:            "ACL flag",
			inputFlags:      []dto.MountFlag{{Name: "acl"}},
			expectedSyscall: syscall.MS_POSIXACL,
			expectedData:    "",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			syscallVal, dataVal, err := suite.fsService.MountFlagsToSyscallFlagAndData(tc.inputFlags)

			if tc.expectError {
				require.Error(t, err)
				details := errors.Details(err)
				if details != nil { // Check if it's our custom error type
					assert.Contains(t, details["Message"], tc.expectedErrorMsg)
				} else { // Fallback for other error types if necessary, or assert on err.Error()
					assert.True(t, strings.Contains(err.Error(), tc.expectedErrorMsg) || (errors.Is(err, dto.ErrorInvalidParameter) && strings.Contains(errors.Details(err)["Message"].(string), tc.expectedErrorMsg)), "Error message mismatch")
				}
			} else {
				require.NoError(t, err, "Unexpected error", err)
				assert.Equal(t, tc.expectedSyscall, syscallVal, "Syscall flags mismatch")
				assert.Equal(t, tc.expectedData, dataVal, "Data string mismatch")
			}
		})
	}
}
