package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

// VfatAdapterTestSuite tests the VfatAdapter
type VfatAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestVfatAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(VfatAdapterTestSuite))
}

func (suite *VfatAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewVfatAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *VfatAdapterTestSuite) TestGetName() {
	suite.Equal("vfat", suite.adapter.GetName())
}

func (suite *VfatAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "FAT")
}

func (suite *VfatAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundUID := false
	foundIOCharset := false
	for _, flag := range flags {
		if flag.Name == "uid" {
			foundUID = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "iocharset" {
			foundIOCharset = true
			suite.True(flag.NeedsValue)
		}
	}

	suite.True(foundUID)
	suite.True(foundIOCharset)
}

func (suite *VfatAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.Equal("dosfstools", support.AlpinePackage)
}

// NtfsAdapterTestSuite tests the NtfsAdapter
type NtfsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestNtfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(NtfsAdapterTestSuite))
}

func (suite *NtfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewNtfsAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *NtfsAdapterTestSuite) TestGetName() {
	suite.Equal("ntfs", suite.adapter.GetName())
}

func (suite *NtfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "NTFS")
}

func (suite *NtfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundPermissions := false
	for _, flag := range flags {
		if flag.Name == "permissions" {
			foundPermissions = true
			suite.False(flag.NeedsValue)
		}
	}

	suite.True(foundPermissions)
}

func (suite *NtfsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.Equal("ntfs-3g-progs", support.AlpinePackage)
}

// BtrfsAdapterTestSuite tests the BtrfsAdapter
type BtrfsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestBtrfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(BtrfsAdapterTestSuite))
}

func (suite *BtrfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewBtrfsAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *BtrfsAdapterTestSuite) TestGetName() {
	suite.Equal("btrfs", suite.adapter.GetName())
}

func (suite *BtrfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "tree")
}

func (suite *BtrfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundCompress := false
	foundSubvol := false
	for _, flag := range flags {
		if flag.Name == "compress" {
			foundCompress = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "subvol" {
			foundSubvol = true
			suite.True(flag.NeedsValue)
		}
	}

	suite.True(foundCompress)
	suite.True(foundSubvol)
}

func (suite *BtrfsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.Equal("btrfs-progs", support.AlpinePackage)
}

// XfsAdapterTestSuite tests the XfsAdapter
type XfsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestXfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(XfsAdapterTestSuite))
}

func (suite *XfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewXfsAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *XfsAdapterTestSuite) TestGetName() {
	suite.Equal("xfs", suite.adapter.GetName())
}

func (suite *XfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "XFS")
}

func (suite *XfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundInode64 := false
	foundAllocsize := false
	for _, flag := range flags {
		if flag.Name == "inode64" {
			foundInode64 = true
			suite.False(flag.NeedsValue)
		}
		if flag.Name == "allocsize" {
			foundAllocsize = true
			suite.True(flag.NeedsValue)
		}
	}

	suite.True(foundInode64)
	suite.True(foundAllocsize)
}

func (suite *XfsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.Equal("xfsprogs", support.AlpinePackage)
}
