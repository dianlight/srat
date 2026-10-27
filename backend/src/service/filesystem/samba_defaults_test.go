package filesystem_test

import (
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

// SambaDefaultsTestSuite verifies the per-filesystem Samba/mount defaults
// introduced for hassio-addons#769 / srat#1264: filesystems without native
// Unix ownership declare mount defaults and opt out of the legacy
// "force user/group = root" smb.conf lines, while native-Unix filesystems
// keep the legacy behavior.
type SambaDefaultsTestSuite struct {
	suite.Suite
	registry *filesystem.Registry
}

func TestSambaDefaultsTestSuite(t *testing.T) {
	suite.Run(t, new(SambaDefaultsTestSuite))
}

func (suite *SambaDefaultsTestSuite) SetupTest() {
	suite.registry = filesystem.NewRegistry()
	suite.Require().NotNil(suite.registry)
}

func (suite *SambaDefaultsTestSuite) adapter(fsType string) filesystem.FilesystemAdapter {
	adapter, err := suite.registry.Get(fsType)
	suite.Require().NoError(err, "adapter for %s", fsType)
	suite.Require().NotNil(adapter)
	return adapter
}

func (suite *SambaDefaultsTestSuite) TestNtfsDefaults() {
	adapter := suite.adapter("ntfs")

	defaults := adapter.GetDefaultMountFlags()
	byName := make(map[string]string, len(defaults))
	for _, flag := range defaults {
		byName[flag.Name] = flag.FlagValue
	}
	suite.Equal("000", byName["fmask"])
	suite.Equal("000", byName["dmask"])
	suite.Equal("0", byName["uid"])
	suite.Equal("0", byName["gid"])

	suite.Equal("", adapter.GetSambaForceUser())
	suite.Equal("", adapter.GetSambaForceGroup())
}

func (suite *SambaDefaultsTestSuite) TestNtfsAliasDefaults() {
	for _, alias := range []string{"ntfs3", "ntfs-3g", "fuseblk"} {
		adapter := suite.adapter(alias)
		suite.NotEmpty(adapter.GetDefaultMountFlags(), "alias %s", alias)
		suite.Equal("", adapter.GetSambaForceUser(), "alias %s", alias)
		suite.Equal("", adapter.GetSambaForceGroup(), "alias %s", alias)
	}
}

func (suite *SambaDefaultsTestSuite) TestFatFamilyDefaults() {
	for _, fsType := range []string{"exfat", "vfat"} {
		adapter := suite.adapter(fsType)

		byName := make(map[string]string)
		for _, flag := range adapter.GetDefaultMountFlags() {
			byName[flag.Name] = flag.FlagValue
		}
		suite.Equal("0", byName["uid"], fsType)
		suite.Equal("0", byName["gid"], fsType)
		suite.Equal("000", byName["umask"], fsType)

		suite.Equal("root", adapter.GetSambaForceUser(), fsType)
		suite.Equal("root", adapter.GetSambaForceGroup(), fsType)
	}
}

func (suite *SambaDefaultsTestSuite) TestHfsPlusDefaults() {
	adapter := suite.adapter("hfsplus")

	byName := make(map[string]string)
	for _, flag := range adapter.GetDefaultMountFlags() {
		byName[flag.Name] = flag.FlagValue
	}
	suite.Equal("0", byName["uid"])
	suite.Equal("0", byName["gid"])
	suite.Equal("000", byName["umask"])
}

func (suite *SambaDefaultsTestSuite) TestApfsDefaults() {
	adapter := suite.adapter("apfs")

	byName := make(map[string]string)
	for _, flag := range adapter.GetDefaultMountFlags() {
		byName[flag.Name] = flag.FlagValue
	}
	suite.Equal("0", byName["uid"])
	suite.Equal("0", byName["gid"])
}

func (suite *SambaDefaultsTestSuite) TestNativeUnixFilesystemsKeepLegacyBehavior() {
	for _, fsType := range []string{"ext4", "btrfs", "xfs"} {
		adapter := suite.adapter(fsType)
		suite.Empty(adapter.GetDefaultMountFlags(), fsType)
		suite.Equal("root", adapter.GetSambaForceUser(), fsType)
		suite.Equal("root", adapter.GetSambaForceGroup(), fsType)
	}
}
