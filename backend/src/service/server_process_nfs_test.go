package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

// newNFSTestServer returns a ServerService writing its exports to a temp file.
func newNFSTestServer(t *testing.T) (*ServerService, HostServiceInterface, ShareServiceInterface, string) {
	t.Helper()
	ctrl := mock.NewMockController(t)
	host := mock.Mock[HostServiceInterface](ctrl)
	shares := mock.Mock[ShareServiceInterface](ctrl)
	exports := filepath.Join(t.TempDir(), "exports")
	previous := nfsExportsFile
	nfsExportsFile = exports
	t.Cleanup(func() { nfsExportsFile = previous })
	return &ServerService{host_service: host, share_service: shares, disks: dto.NewDiskMap()}, host, shares, exports
}

func nfsShare(name, path string, subfolder *string) dto.SharedResource {
	return dto.SharedResource{
		Name:      name,
		Usage:     dto.UsageAsShare,
		Subfolder: subfolder,
		MountPointData: &dto.MountPointData{
			Path:      path,
			Partition: &dto.Partition{FsType: new("ext4")},
		},
	}
}

func TestWriteNFSConfigExportsShareFolders(t *testing.T) {
	server, host, shares, exports := newNFSTestServer(t)
	mock.When(host.GetHostName()).ThenReturn("homeassistant", nil)

	disabled := nfsShare("OFF", "/mnt/OFF", nil)
	disabled.Disabled = new(true)
	invalid := nfsShare("INVALID", "/mnt/INVALID", nil)
	invalid.Status = &dto.SharedResourceStatus{IsValid: false}
	internal := nfsShare("CONFIG", "/config", nil)
	internal.Usage = dto.UsageAsInternal
	invalidMount := nfsShare("GONE", "/mnt/GONE", nil)
	invalidMount.MountPointData.IsInvalid = true
	noPath := nfsShare("NOPATH", "", nil)
	noMount := dto.SharedResource{Name: "NOMOUNT", Usage: dto.UsageAsShare}
	apfs := nfsShare("APFS", "/mnt/APFS", nil)
	apfs.MountPointData.Partition.FsType = new("apfs")
	mock.When(shares.ListShares()).ThenReturn([]dto.SharedResource{
		nfsShare("DATA1", "/mnt/DATA1", nil),
		nfsShare("EMPEROR", "/mnt/DATA1", new("emperor")),
		disabled, invalid, internal, invalidMount, noPath, noMount, apfs,
	}, nil)

	require.NoError(t, server.writeNFSConfig(context.Background()))

	content, err := os.ReadFile(exports)
	require.NoError(t, err)
	assert.Contains(t, string(content), "/mnt homeassistant(ro,")
	assert.Contains(t, string(content), "\n/mnt/DATA1 homeassistant(rw,")
	assert.Contains(t, string(content), "\n/mnt/DATA1/emperor homeassistant(rw,", "a subfolder share exports only its folder")
	for _, skipped := range []string{"/mnt/OFF", "/mnt/INVALID", "/config", "/mnt/GONE", "/mnt/APFS"} {
		assert.NotContains(t, string(content), skipped)
	}
}

func TestWriteNFSConfigHostNameError(t *testing.T) {
	server, host, _, exports := newNFSTestServer(t)
	mock.When(host.GetHostName()).ThenReturn("", errors.New("no hostname"))

	require.Error(t, server.writeNFSConfig(context.Background()))
	assert.NoFileExists(t, exports)
}

func TestWriteNFSConfigListSharesError(t *testing.T) {
	server, host, shares, exports := newNFSTestServer(t)
	mock.When(host.GetHostName()).ThenReturn("homeassistant", nil)
	mock.When(shares.ListShares()).ThenReturn(nil, errors.New("database locked"))

	require.Error(t, server.writeNFSConfig(context.Background()))
	assert.NoFileExists(t, exports)
}

func TestWriteNFSConfigWriteError(t *testing.T) {
	server, host, shares, _ := newNFSTestServer(t)
	nfsExportsFile = filepath.Join(t.TempDir(), "missing-dir", "exports")
	mock.When(host.GetHostName()).ThenReturn("homeassistant", nil)
	mock.When(shares.ListShares()).ThenReturn([]dto.SharedResource{}, nil)

	require.Error(t, server.writeNFSConfig(context.Background()))
}
