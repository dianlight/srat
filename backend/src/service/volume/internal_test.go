// Package volume holds white-box tests for unexported helpers.
package volume

import (
	"errors"
	"testing"

	"github.com/dianlight/srat/dto"
)

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
		t.Error("HOST row must never be an orphan")
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
