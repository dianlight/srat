package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

// TestCheckDeviceSupport_CachesProbeResults verifies that repeated
// CheckDeviceSupport calls for the same device do not re-issue the
// side-effecting SCSI probes (SG open + ATA PASS-THROUGH).
//
// The SG_IO probes can cause USB-attached bridges to trigger kernel
// partition-table rescans which emit fresh udev events; without caching,
// every hardware re-enumeration re-probes every disk and feeds a udev
// event storm that pegs a CPU core.
func TestCheckDeviceSupport_CachesProbeResults(t *testing.T) {
	origSG := sgOpenFn
	origATA := ataProbeFn
	defer func() {
		sgOpenFn = origSG
		ataProbeFn = origATA
	}()

	sgCalls := 0
	ataCalls := 0
	sgOpenFn = func(string) (*os.File, error) {
		sgCalls++
		return os.Open(os.DevNull)
	}
	ataProbeFn = func(string, bool) error {
		ataCalls++
		return nil
	}

	svc := &HDIdleService{
		ctx:          context.Background(),
		supportCache: cache.New(10*time.Minute, 20*time.Minute),
	}

	first, err := svc.CheckDeviceSupport(os.DevNull)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.True(t, first.SupportsSCSI, "stubbed SG open must succeed")
	require.Equal(t, 1, sgCalls, "first call must issue exactly one SG probe")
	// ataCalls depends on sysfs layout (/sys/block may not exist on the dev
	// machine); only require at most one probe on the first call.
	require.LessOrEqual(t, ataCalls, 1)

	sgBefore, ataBefore := sgCalls, ataCalls
	second, err := svc.CheckDeviceSupport(os.DevNull)
	require.NoError(t, err)
	require.Equal(t, sgBefore, sgCalls, "second call must be served from cache (no new SG probe)")
	require.Equal(t, ataBefore, ataCalls, "second call must be served from cache (no new ATA probe)")
	require.Equal(t, first.SupportsSCSI, second.SupportsSCSI)
	require.Equal(t, first.SupportsATA, second.SupportsATA)
	require.Equal(t, first.Supported, second.Supported)
	require.Equal(t, os.DevNull, second.DevicePath)
}

// TestCheckDeviceSupport_CacheHitOverridesDevicePath verifies that a cache
// hit returns a copy whose DevicePath reflects the path actually requested,
// even when the cached entry was stored under a different alias resolving
// to the same device name (the cache key is the resolved base name).
func TestCheckDeviceSupport_CacheHitOverridesDevicePath(t *testing.T) {
	origSG := sgOpenFn
	defer func() { sgOpenFn = origSG }()

	sgCalls := 0
	sgOpenFn = func(string) (*os.File, error) {
		sgCalls++
		return os.Open(os.DevNull)
	}

	dirA := t.TempDir()
	dirB := t.TempDir()
	linkA := filepath.Join(dirA, "sdb")
	linkB := filepath.Join(dirB, "sdb")
	require.NoError(t, os.Symlink(os.DevNull, linkA))
	require.NoError(t, os.Symlink(os.DevNull, linkB))

	svc := &HDIdleService{
		ctx:          context.Background(),
		supportCache: cache.New(10*time.Minute, 20*time.Minute),
	}

	viaA, err := svc.CheckDeviceSupport(linkA)
	require.NoError(t, err)
	require.Equal(t, linkA, viaA.DevicePath)
	require.Equal(t, 1, sgCalls)

	viaB, err := svc.CheckDeviceSupport(linkB)
	require.NoError(t, err)
	require.Equal(t, 1, sgCalls, "alias with same resolved name must hit cache")
	require.Equal(t, linkB, viaB.DevicePath, "cached copy must report requested DevicePath")
	require.Equal(t, viaA.Supported, viaB.Supported)
}

// TestCheckDeviceSupport_NilCacheStillWorks guards the defensive nil-cache
// path so a misconstructed service cannot panic.
func TestCheckDeviceSupport_NilCacheStillWorks(t *testing.T) {
	origSG := sgOpenFn
	defer func() { sgOpenFn = origSG }()
	sgOpenFn = func(string) (*os.File, error) {
		return os.Open(os.DevNull)
	}

	svc := &HDIdleService{ctx: context.Background(), supportCache: nil}
	support, err := svc.CheckDeviceSupport(os.DevNull)
	require.NoError(t, err)
	require.NotNil(t, support)
}

// TestCheckDeviceSupport_EmptyPathNotCached verifies cheap validation
// failures are not poisoned into the probe cache.
func TestCheckDeviceSupport_EmptyPathNotCached(t *testing.T) {
	origSG := sgOpenFn
	defer func() { sgOpenFn = origSG }()
	sgCalls := 0
	sgOpenFn = func(string) (*os.File, error) {
		sgCalls++
		return os.Open(os.DevNull)
	}

	svc := &HDIdleService{
		ctx:          context.Background(),
		supportCache: cache.New(10*time.Minute, 20*time.Minute),
	}
	support, err := svc.CheckDeviceSupport("")
	require.NoError(t, err)
	require.False(t, support.Supported)
	require.Zero(t, sgCalls)

	// A subsequent valid call must still probe (nothing bogus got cached).
	_, err = svc.CheckDeviceSupport(os.DevNull)
	require.NoError(t, err)
	require.Equal(t, 1, sgCalls)
}
