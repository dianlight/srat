package filesystem_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
)

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

	controller := mock.NewMockController(suite.T())
	execMock := mock.Mock[filesystem.ExecCmd](controller)
	mock.When(execMock.StdoutPipe()).ThenReturn(io.NopCloser(strings.NewReader("")), nil)
	mock.When(execMock.StderrPipe()).ThenReturn(io.NopCloser(strings.NewReader("")), nil)
	mock.When(execMock.Start()).ThenReturn(nil)
	mock.When(execMock.Wait()).ThenReturn(nil)
	filesystem.SetExecOpsForTesting(
		func(cmd string) (string, error) {
			if cmd != "" {
				return cmd, nil
			}
			return "", errors.New("command not found")
		},
		func(ctx context.Context, cmd string, args ...string) filesystem.ExecCmd {
			return execMock
		},
	)
}

func (suite *BtrfsAdapterTestSuite) TearDownTest() {
	filesystem.ResetExecOpsForTesting()
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
