// Package volume holds white-box tests for unexported helpers.
package volume

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/require"
)

// TestSameDevicePath verifies that a procfs source such as /dev/sdb2 matches a
// partition's by-id symlink, which is how HA-created mounts are detected.
func TestSameDevicePath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sdb2")
	require.NoError(t, os.WriteFile(target, []byte(""), 0o600))
	link := filepath.Join(dir, "usb-by-id")
	require.NoError(t, os.Symlink(target, link))

	if !sameDevicePath(link, target) {
		t.Error("symlink and target must resolve to the same device")
	}
	if !sameDevicePath(target, link) {
		t.Error("comparison must be symmetric")
	}
	if sameDevicePath(target, filepath.Join(dir, "sdc1")) {
		t.Error("different paths must not match")
	}
	if sameDevicePath("", target) {
		t.Error("empty path must not match")
	}
}

func TestNormalizeMountPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/mnt/ata-WD-part2", "/mnt/ata_WD_part2"},
		{"/mnt/ata_WD_part2", "/mnt/ata_WD_part2"},
		{"/data/ata-WD", "/data/ata-WD"},
		{"/mnt/nodash", "/mnt/nodash"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := normalizeMountPath(tc.in); got != tc.want {
			t.Errorf("normalizeMountPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if got := NormalizeMountPath(tc.in); got != tc.want {
			t.Errorf("NormalizeMountPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMountRowIsStaleOrphan(t *testing.T) {
	startup := true
	mounted := dto.MountPointData{IsMounted: true, IsInvalid: true}
	if mountRowIsStaleOrphan(&mounted) {
		t.Error("mounted row must never be an orphan")
	}
	if mountRowIsStaleOrphan(nil) {
		t.Error("nil row must never be an orphan")
	}
	withShare := dto.MountPointData{IsInvalid: true, Share: &dto.SharedResource{}}
	if mountRowIsStaleOrphan(&withShare) {
		t.Error("share-bearing row must never be an orphan")
	}
	host := dto.MountPointData{Type: "HOST", IsInvalid: true}
	if mountRowIsStaleOrphan(&host) {
		t.Error("HOST row must never be an orphan (may back HA share configs)")
	}
	hostMounted := dto.MountPointData{Type: "HOST", IsMounted: true, IsInvalid: true}
	if mountRowIsStaleOrphan(&hostMounted) {
		t.Error("mounted HOST row must never be an orphan")
	}
	wantStartup := dto.MountPointData{IsToMountAtStartup: &startup, IsInvalid: true}
	if mountRowIsStaleOrphan(&wantStartup) {
		t.Error("startup-marked row must never be an orphan")
	}
	healthy := dto.MountPointData{Type: "ADDON"}
	if mountRowIsStaleOrphan(&healthy) {
		t.Error("present row must never be an orphan")
	}
	orphan := dto.MountPointData{Type: "ADDON", IsInvalid: true}
	if !mountRowIsStaleOrphan(&orphan) {
		t.Error("unmounted share-less missing ADDON row must be an orphan")
	}
}

func TestLogUdevMonitorError_Levels(t *testing.T) {
	h := &UdevHandler{}
	h.logUdevMonitorError(nil)
	h.logUdevMonitorError(errors.New("unable to parse uevent: invalid env data"))
	h.logUdevMonitorError(errors.New("unable to parse uevent: truncated"))
	h.logUdevMonitorError(errors.New("netlink socket failed"))
}

func TestApplyFormatOverrides_ExpiredEntryIgnored(t *testing.T) {
	h := &UdevHandler{}
	dev, partID, diskID := "/dev/sdz99", "part-x", "disk-x"
	devCopy, partCopy, diskCopy := dev, partID, diskID
	seed := dto.Partition{Id: &partCopy, DiskId: &diskCopy, DevicePath: &devCopy, Name: new("old")}
	h.recordFormatOverride(&seed, "new-label", "ext4")

	h.formatMu.Lock()
	for _, e := range h.formatOverrides {
		e.at = time.Now().Add(-formatOverrideTTL - time.Minute)
	}
	h.formatMu.Unlock()

	devCopy2, partCopy2, diskCopy2, oldCopy := dev, partID, diskID, "old"
	stale := dto.Partition{Id: &partCopy2, DiskId: &diskCopy2, DevicePath: &devCopy2, Name: &oldCopy}
	if h.ApplyFormatOverrides(&stale) {
		t.Error("expired override must not rewrite the partition")
	}
	if *stale.Name != "old" {
		t.Errorf("expired override changed the name to %q", *stale.Name)
	}
	h.formatMu.Lock()
	defer h.formatMu.Unlock()
	if len(h.formatOverrides) != 0 {
		t.Errorf("expired override must be evicted, %d entries left", len(h.formatOverrides))
	}
}

func TestRecordFormatOverride_CapsMapSize(t *testing.T) {
	h := &UdevHandler{}
	for i := 0; i < maxFormatOverrides+10; i++ {
		dev := "/dev/sdz-cap-" + strconv.Itoa(i)
		partID, diskID := "part-cap-"+strconv.Itoa(i), "disk-cap"
		devCopy, partCopy, diskCopy := dev, partID, diskID
		seed := dto.Partition{Id: &partCopy, DiskId: &diskCopy, DevicePath: &devCopy}
		h.recordFormatOverride(&seed, "label", "ext4")
	}
	h.formatMu.Lock()
	defer h.formatMu.Unlock()
	if len(h.formatOverrides) > maxFormatOverrides {
		t.Errorf("override map grew to %d entries, cap is %d", len(h.formatOverrides), maxFormatOverrides)
	}
}
