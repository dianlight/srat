package mount

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type BootSectorTestSuite struct {
	suite.Suite
	dir string
}

func TestBootSectorTestSuite(t *testing.T) {
	suite.Run(t, new(BootSectorTestSuite))
}

func (suite *BootSectorTestSuite) SetupTest() {
	suite.dir = suite.T().TempDir()
}

// bootSector builds a 512-byte boot sector with the given OEM ID at offset 3
// and, when sign is true, the 0x55AA signature at offset 510.
func bootSector(oemID string, sign bool) []byte {
	s := make([]byte, bootSectorSize)
	copy(s[0:3], []byte{0xEB, 0x52, 0x90})
	copy(s[3:11], []byte(oemID))
	if sign {
		s[510], s[511] = 0x55, 0xAA
	}
	return s
}

// writeImage writes data padded with zeros to size bytes into a file in dir.
func writeImage(dir string, data []byte, size int) (string, error) {
	img := make([]byte, size)
	copy(img, data)
	p := filepath.Join(dir, "disk.img")
	return p, os.WriteFile(p, img, 0o600)
}

func (suite *BootSectorTestSuite) image(data []byte, size int) string {
	p, err := writeImage(suite.dir, data, size)
	suite.Require().NoError(err)
	return p
}

func (suite *BootSectorTestSuite) TestFsFromBootSector() {
	cases := []struct {
		name   string
		sector []byte
		want   string
	}{
		{"ntfs", bootSector("NTFS    ", true), "ntfs"},
		{"exfat", bootSector("EXFAT   ", true), "exfat"},
		{"ntfs without 0x55AA signature", bootSector("NTFS    ", false), ""},
		{"fat32 OEM ID", bootSector("MSDOS5.0", true), ""},
		{"ntfs prefix only", bootSector("NTFS", true), ""},
		{"all zeros", make([]byte, bootSectorSize), ""},
		{"short buffer", bootSector("NTFS    ", true)[:400], ""},
	}
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.Equal(tc.want, fsFromBootSector(tc.sector))
		})
	}
}

func (suite *BootSectorTestSuite) TestProbeBootSectorFSNTFS() {
	fs, err := probeBootSectorFS(suite.image(bootSector("NTFS    ", true), 4096))
	suite.Require().NoError(err)
	suite.Equal("ntfs", fs)
}

func (suite *BootSectorTestSuite) TestProbeBootSectorFSExFAT() {
	fs, err := probeBootSectorFS(suite.image(bootSector("EXFAT   ", true), 4096))
	suite.Require().NoError(err)
	suite.Equal("exfat", fs)
}

func (suite *BootSectorTestSuite) TestProbeBootSectorFSUnknownImage() {
	_, err := probeBootSectorFS(suite.image(nil, 4096))
	suite.Error(err)
}

func (suite *BootSectorTestSuite) TestProbeBootSectorFSShortImage() {
	_, err := probeBootSectorFS(suite.image(nil, 100))
	suite.Error(err)
}

func (suite *BootSectorTestSuite) TestProbeBootSectorFSMissingDevice() {
	_, err := probeBootSectorFS(filepath.Join(suite.dir, "missing"))
	suite.Error(err)
}
