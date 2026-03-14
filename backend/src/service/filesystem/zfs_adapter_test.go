package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

// ZfsAdapterTestSuite tests the ZfsAdapter.
type ZfsAdapterTestSuite struct {
	suite.Suite
	adapter    filesystem.FilesystemAdapter
	ctx        context.Context
	cleanGetFs func()
}

func TestZfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(ZfsAdapterTestSuite))
}

func (suite *ZfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewZfsAdapter()
	suite.Require().NotNil(suite.adapter)

	suite.cleanGetFs = suite.adapter.SetGetFilesystemsForTesting(func() ([]string, error) {
		return []string{"zfs", "ext4"}, nil
	})
}

func (suite *ZfsAdapterTestSuite) TearDownTest() {
	if suite.cleanGetFs != nil {
		suite.cleanGetFs()
	}
}

func (suite *ZfsAdapterTestSuite) TestGetName() {
	suite.Equal("zfs", suite.adapter.GetName())
}

func (suite *ZfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "ZFS")
}

func (suite *ZfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundZfsutil := false
	for _, flag := range flags {
		if flag.Name == "zfsutil" {
			foundZfsutil = true
			suite.False(flag.NeedsValue)
		}
	}

	suite.True(foundZfsutil)
}

func (suite *ZfsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.True(support.CanMount)
	suite.False(support.CanFormat)
	suite.False(support.CanCheck)
	suite.False(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("zfs", support.AlpinePackage)
}

func (suite *ZfsAdapterTestSuite) TestUnsupportedOperations() {
	formatErr := suite.adapter.Format(suite.ctx, "/dev/sdz1", dto.FormatOptions{}, nil)
	suite.Error(formatErr)
	suite.Contains(formatErr.Error(), "not supported")

	_, checkErr := suite.adapter.Check(suite.ctx, "/dev/sdz1", dto.CheckOptions{}, nil)
	suite.Error(checkErr)
	suite.Contains(checkErr.Error(), "not supported")

	_, getLabelErr := suite.adapter.GetLabel(suite.ctx, "/dev/sdz1")
	suite.Error(getLabelErr)

	setLabelErr := suite.adapter.SetLabel(suite.ctx, "/dev/sdz1", "pool")
	suite.Error(setLabelErr)
}
