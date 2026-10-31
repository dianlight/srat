//go:build linux

package mount

import (
	"testing"

	"github.com/stretchr/testify/suite"
	uremount "github.com/u-root/u-root/pkg/mount"
)

type FSFromBlockTestSuite struct {
	suite.Suite
	dir string
}

func TestFSFromBlockTestSuite(t *testing.T) {
	suite.Run(t, new(FSFromBlockTestSuite))
}

func (suite *FSFromBlockTestSuite) SetupTest() {
	suite.dir = suite.T().TempDir()
}

// u-root's FSFromBlock reads 128 KiB and only knows a fixed set of magics, so
// an NTFS or exFAT volume must be recovered by the boot sector fallback
// (srat#1513).
func (suite *FSFromBlockTestSuite) TestDetectsNTFSAndExFAT() {
	for oemID, want := range map[string]string{"NTFS    ": "ntfs", "EXFAT   ": "exfat"} {
		suite.Run(want, func() {
			img, err := writeImage(suite.T().TempDir(), bootSector(oemID, true), 256*1024)
			suite.Require().NoError(err)
			fs, flags, err := FSFromBlock(img)
			suite.Require().NoError(err)
			suite.Equal(want, fs)
			suite.Zero(flags)
		})
	}
}

// A file system u-root already recognises must keep u-root's answer: the boot
// sector fallback only runs when u-root fails.
func (suite *FSFromBlockTestSuite) TestKeepsUrootResultForExt4() {
	if err := uremount.FindFileSystem("ext4"); err != nil {
		suite.T().Skip("ext4 not listed in /proc/filesystems on this host")
	}
	data := make([]byte, 0x43A)
	data[0x438], data[0x439] = 0x53, 0xEF // ext2/3/4 superblock magic
	img, err := writeImage(suite.dir, data, 256*1024)
	suite.Require().NoError(err)
	fs, _, err := FSFromBlock(img)
	suite.Require().NoError(err)
	suite.Equal("ext4", fs)
}

func (suite *FSFromBlockTestSuite) TestUnknownStillFails() {
	img, err := writeImage(suite.dir, nil, 256*1024)
	suite.Require().NoError(err)
	_, _, err = FSFromBlock(img)
	suite.Error(err)
}
