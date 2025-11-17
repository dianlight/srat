package converter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/internal/osutil"
	"github.com/stretchr/testify/suite"
)

// suite definition
type GoverterSuite struct {
	suite.Suite
	origReadDir  func(string) ([]os.DirEntry, error)
	origEvalLink func(string) (string, error)
	restoreMount func()
}

func TestGoverterSuite(t *testing.T) {
	suite.Run(t, new(GoverterSuite))
}

func (s *GoverterSuite) SetupTest() {
	s.origReadDir = osReadDir
	s.origEvalLink = evalSymlink
	s.restoreMount = nil
}

func (s *GoverterSuite) TearDownTest() {
	if s.origReadDir != nil {
		MockFuncOsReadDir(s.origReadDir)
	}
	if s.origEvalLink != nil {
		MockFuncEvalSymlink(s.origEvalLink)
	}
	if s.restoreMount != nil {
		s.restoreMount()
		s.restoreMount = nil
	}
}

// fakeDirEntry implements os.DirEntry for testing
type fakeDirEntry struct {
	name string
	mode os.FileMode
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.mode.IsDir() }
func (f fakeDirEntry) Type() os.FileMode          { return f.mode }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return fakeFileInfo{isDir: f.mode.IsDir()}, nil }

func (s *GoverterSuite) TestDeviceToDeviceId_MatchByResolved() {
	// Prepare mocks
	MockFuncOsReadDir(func(name string) ([]os.DirEntry, error) {
		s.Require().Equal("/dev/disk/by-id/", filepath.Clean(name)+"/")
		return []os.DirEntry{
			fakeDirEntry{name: "disk-ABC", mode: os.ModeSymlink},
			fakeDirEntry{name: "disk-DEF", mode: 0},
		}, nil
	})
	MockFuncEvalSymlink(func(path string) (string, error) {
		if filepath.Base(path) == "disk-ABC" {
			return "/dev/special0", nil
		}
		return "", errors.New("not a symlink")
	})

	id, err := deviceToDeviceId("/dev/special0")

	s.Require().NoError(err)
	s.Equal("by-id-disk-ABC", id)
}

func (s *GoverterSuite) TestDeviceToDeviceId_NoMatchReturnsSource() {
	MockFuncOsReadDir(func(name string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "disk-XYZ", mode: os.ModeSymlink}}, nil
	})
	MockFuncEvalSymlink(func(path string) (string, error) {
		return "/dev/another", nil
	})

	src := "/dev/sda1"
	id, err := deviceToDeviceId(src)
	s.Require().NoError(err)
	s.Equal(src, id)
}

func (s *GoverterSuite) TestMountPathToDeviceId_DirectMatch() {
	// Mock mountinfo to include our test mount
	s.restoreMount = osutil.MockMountInfo("1546 1508 8:8 / /mnt/test rw,relatime - ext4 /dev/special0 rw\n")

	// Mock by-id resolution to return stable id
	MockFuncOsReadDir(func(name string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{name: "disk-ABC", mode: os.ModeSymlink}}, nil
	})
	MockFuncEvalSymlink(func(path string) (string, error) {
		return "/dev/special0", nil
	})

	id, err := mountPathToDeviceId("/mnt/test")
	s.Require().NoError(err)
	s.Equal("by-id-disk-ABC", id)
}

func (s *GoverterSuite) TestFalseConsts() {
	s.False(falseConst())
	p := falsePConst()
	s.Require().NotNil(p)
	s.False(*p)
}

func (s *GoverterSuite) TestIsWriteSupported_Permutations() {
	dir := s.T().TempDir()

	// default temp dir should be writable
	p := isWriteSupported(dir)
	s.Require().NotNil(p)
	s.Require().True(*p)

	p2 := isWriteSupported(dir + "_test")
	s.Require().NotNil(p2)
	s.Require().False(*p2)

	// restore writable and verify true again
	s.Require().NoError(os.Chmod(dir, 0o755))
	p3 := isWriteSupported(dir)
	s.Require().NotNil(p3)
	s.True(*p3)
}
