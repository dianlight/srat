package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
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

func (suite *VfatAdapterTestSuite) TestGetLinuxFsModule() {
	module := suite.adapter.GetLinuxFsModule()
	suite.Equal("vfat", module)
}

func (suite *VfatAdapterTestSuite) TestFormat_NonExistentDevice() {
	err := suite.adapter.Format(suite.ctx, "/dev/nonexistent-vfat-12345", dto.FormatOptions{}, nil)
	suite.Error(err)
}

func (suite *VfatAdapterTestSuite) TestCheck_NonExistentDevice() {
	result, err := suite.adapter.Check(suite.ctx, "/dev/nonexistent-vfat-12345", dto.CheckOptions{}, nil)
	suite.Error(err)
	_ = result
}

func (suite *VfatAdapterTestSuite) TestGetLabel_NonExistentDevice() {
	label, err := suite.adapter.GetLabel(suite.ctx, "/dev/nonexistent-vfat-12345")
	suite.Error(err)
	suite.Empty(label)
}

func (suite *VfatAdapterTestSuite) TestSetLabel_NonExistentDevice() {
	err := suite.adapter.SetLabel(suite.ctx, "/dev/nonexistent-vfat-12345", "testlabel")
	suite.Error(err)
}

func (suite *VfatAdapterTestSuite) TestGetState_NonExistentDevice() {
	state, err := suite.adapter.GetState(suite.ctx, "/dev/nonexistent-vfat-12345")
	suite.Error(err)
	_ = state
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

func (suite *NtfsAdapterTestSuite) TestGetLinuxFsModule() {
	module := suite.adapter.GetLinuxFsModule()
	suite.Equal("ntfs3", module)
}

func (suite *NtfsAdapterTestSuite) TestFormat_NonExistentDevice() {
	err := suite.adapter.Format(suite.ctx, "/dev/nonexistent-ntfs-12345", dto.FormatOptions{}, nil)
	suite.Error(err)
}

func (suite *NtfsAdapterTestSuite) TestCheck_NonExistentDevice() {
	result, err := suite.adapter.Check(suite.ctx, "/dev/nonexistent-ntfs-12345", dto.CheckOptions{}, nil)
	suite.Error(err)
	_ = result
}

func (suite *NtfsAdapterTestSuite) TestGetLabel_NonExistentDevice() {
	label, err := suite.adapter.GetLabel(suite.ctx, "/dev/nonexistent-ntfs-12345")
	suite.Error(err)
	suite.Empty(label)
}

func (suite *NtfsAdapterTestSuite) TestSetLabel_NonExistentDevice() {
	err := suite.adapter.SetLabel(suite.ctx, "/dev/nonexistent-ntfs-12345", "testlabel")
	suite.Error(err)
}

func (suite *NtfsAdapterTestSuite) TestGetState_NonExistentDevice() {
	state, err := suite.adapter.GetState(suite.ctx, "/dev/nonexistent-ntfs-12345")
	suite.Error(err)
	_ = state
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
	suite.Contains(desc, "BTRFS")
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

func (suite *BtrfsAdapterTestSuite) TestGetLinuxFsModule() {
	module := suite.adapter.GetLinuxFsModule()
	suite.Equal("btrfs", module)
}

func (suite *BtrfsAdapterTestSuite) TestFormat_NonExistentDevice() {
	err := suite.adapter.Format(suite.ctx, "/dev/nonexistent-btrfs-12345", dto.FormatOptions{}, nil)
	suite.Error(err)
}

func (suite *BtrfsAdapterTestSuite) TestCheck_NonExistentDevice() {
	result, err := suite.adapter.Check(suite.ctx, "/dev/nonexistent-btrfs-12345", dto.CheckOptions{}, nil)
	suite.Error(err)
	_ = result
}

func (suite *BtrfsAdapterTestSuite) TestGetLabel_NonExistentDevice() {
	label, err := suite.adapter.GetLabel(suite.ctx, "/dev/nonexistent-btrfs-12345")
	suite.Error(err)
	suite.Empty(label)
}

func (suite *BtrfsAdapterTestSuite) TestSetLabel_NonExistentDevice() {
	err := suite.adapter.SetLabel(suite.ctx, "/dev/nonexistent-btrfs-12345", "testlabel")
	suite.Error(err)
}

func (suite *BtrfsAdapterTestSuite) TestGetState_NonExistentDevice() {
	state, err := suite.adapter.GetState(suite.ctx, "/dev/nonexistent-btrfs-12345")
	suite.Error(err)
	_ = state
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

func (suite *XfsAdapterTestSuite) TestGetLinuxFsModule() {
	module := suite.adapter.GetLinuxFsModule()
	suite.Equal("xfs", module)
}

func (suite *XfsAdapterTestSuite) TestFormat_NonExistentDevice() {
	err := suite.adapter.Format(suite.ctx, "/dev/nonexistent-xfs-12345", dto.FormatOptions{}, nil)
	suite.Error(err)
}

func (suite *XfsAdapterTestSuite) TestCheck_NonExistentDevice() {
	result, err := suite.adapter.Check(suite.ctx, "/dev/nonexistent-xfs-12345", dto.CheckOptions{}, nil)
	suite.Error(err)
	_ = result
}

func (suite *XfsAdapterTestSuite) TestGetLabel_NonExistentDevice() {
	label, err := suite.adapter.GetLabel(suite.ctx, "/dev/nonexistent-xfs-12345")
	suite.Error(err)
	suite.Empty(label)
}

func (suite *XfsAdapterTestSuite) TestSetLabel_NonExistentDevice() {
	err := suite.adapter.SetLabel(suite.ctx, "/dev/nonexistent-xfs-12345", "testlabel")
	suite.Error(err)
}

func (suite *XfsAdapterTestSuite) TestGetState_NonExistentDevice() {
	state, err := suite.adapter.GetState(suite.ctx, "/dev/nonexistent-xfs-12345")
	suite.Error(err)
	_ = state
}
