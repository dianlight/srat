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
