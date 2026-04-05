package filesystem_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

// BtrfsAdapterTestSuite tests the BtrfsAdapter
type BtrfsAdapterTestSuite struct {
	suite.Suite
	adapter    filesystem.FilesystemAdapter
	ctx        context.Context
	cleanExec  func() // Optional cleanup function for tests that set exec ops
	cleanGetFs func() // Optional cleanup function for tests that set GetFs ops
}

func TestBtrfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(BtrfsAdapterTestSuite))
}

func (suite *BtrfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewBtrfsAdapter()
	suite.Require().NotNil(suite.adapter)

	suite.cleanExec = suite.adapter.SetExecOpsForTesting(
		func(cmd string) (string, error) {
			if cmd != "" {
				return cmd, nil
			}
			return "", errors.New("command not found")
		},
	)
	suite.cleanGetFs = suite.adapter.SetGetFilesystemsForTesting(func() ([]string, error) {
		return []string{"btrfs", "ext4"}, nil
	})
}

func (suite *BtrfsAdapterTestSuite) TearDownTest() {
	if suite.cleanExec != nil {
		suite.cleanExec()
	}
	if suite.cleanGetFs != nil {
		suite.cleanGetFs()
	}
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
	suite.True(support.CanFormat)
	suite.True(support.CanMount)
	suite.True(support.CanCheck)
	suite.True(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("btrfs-progs", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}
