package service

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/procfs/blockdevice"
	"github.com/stretchr/testify/suite"
)

// DiskStatsServiceSuite contains unit tests for disk_stats_service.go
type DiskStatsServiceSuite struct {
	suite.Suite
	ctrl       *matchers.MockController
	volumeMock VolumeServiceInterface
	smartMock  SmartServiceInterface
	hdidleMock HDIdleServiceInterface
	fsMock     FilesystemServiceInterface
	ds         *diskStatsService
	ctx        context.Context
	cancel     context.CancelFunc
}

// Test runner
func TestDiskStatsServiceSuite(t *testing.T) {
	suite.Run(t, new(DiskStatsServiceSuite))
}

func (suite *DiskStatsServiceSuite) SetupTest() {
	suite.ctrl = mock.NewMockController(suite.T())
	// create context with waitgroup as used by service code
	var wg sync.WaitGroup
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", &wg))

	// create mocks
	suite.volumeMock = mock.Mock[VolumeServiceInterface](suite.ctrl)
	suite.smartMock = mock.Mock[SmartServiceInterface](suite.ctrl)
	suite.hdidleMock = mock.Mock[HDIdleServiceInterface](suite.ctrl)
	suite.fsMock = mock.Mock[FilesystemServiceInterface](suite.ctrl)

	// instantiate diskStatsService under test with mocks
	suite.ds = &diskStatsService{
		volumeService:     suite.volumeMock,
		blockdevice:       &blockdevice.FS{}, // zero value; tests avoid calling SysBlockDeviceStat
		ctx:               suite.ctx,
		lastUpdateTime:    time.Now(),
		updateMutex:       &sync.Mutex{},
		lastStats:         make(map[string]*blockdevice.IOStats),
		smartService:      suite.smartMock,
		hdidleService:     suite.hdidleMock,
		filesystemService: suite.fsMock,
		ioStatFetcher: func(string) (blockdevice.IOStats, error) {
			return blockdevice.IOStats{}, nil
		},
		readFile:          os.ReadFile,
		sysFsBasePath:     "/sys/fs",
		smartEnabledCache: cache.New(5*time.Minute, 10*time.Minute), // Short TTL for tests
	}
}

// Helper function to get cache value
func getCacheValue(c *cache.Cache, key string) (bool, bool) {
	if val, found := c.Get(key); found {
		if boolVal, ok := val.(bool); ok {
			return boolVal, true
		}
	}
	return false, false
}

func (suite *DiskStatsServiceSuite) TearDownTest() {
	// cancel context to clean up any potential goroutines
	suite.cancel()
}

func (suite *DiskStatsServiceSuite) TestGetDiskStatsNotInitialized() {
	// Ensure currentDiskHealth is nil
	suite.Nil(suite.ds.currentDiskHealth)

	// Call GetDiskStats and expect an error
	h, err := suite.ds.GetDiskStats()
	suite.Error(err)
	suite.Nil(h)
	suite.Contains(err.Error(), "disk stats not initialized")
}

func (suite *DiskStatsServiceSuite) TestUpdateDiskStats_NoVolumes() {
	// Arrange: VolumeService returns ErrorNotFound to simulate no volumes
	mock.When(suite.volumeMock.GetVolumesData()).ThenReturn(nil)
	mock.When(suite.hdidleMock.IsRunning()).ThenReturn(false)

	// Act
	err := suite.ds.updateDiskStats()

	// Assert
	suite.NoError(err)
	suite.NotNil(suite.ds.currentDiskHealth)
	suite.Empty(suite.ds.currentDiskHealth.PerDiskIO)
	suite.Equal(float64(0), suite.ds.currentDiskHealth.Global.TotalIOPS)
	suite.Equal(float64(0), suite.ds.currentDiskHealth.Global.TotalReadLatency)
	suite.Equal(float64(0), suite.ds.currentDiskHealth.Global.TotalWriteLatency)
	suite.NotNil(suite.ds.currentDiskHealth.PerPartitionInfo)
	suite.Empty(suite.ds.currentDiskHealth.PerPartitionInfo)
	suite.False(suite.ds.currentDiskHealth.HDIdleRunning)

	// Verify mock call
	mock.Verify(suite.volumeMock, matchers.Times(1)).GetVolumesData()
}

func (suite *DiskStatsServiceSuite) TestUpdateDiskStats_SkipsDiskWithNilDevice() {
	// Arrange: prepare a disk with nil Device but with partitions (which should be skipped)
	diskID := "disk-123"
	partID := "part-1"
	partitions := map[string]dto.Partition{
		partID: {
			Id:   &partID,
			Size: nil,
			MountPointData: &map[string]dto.MountPointData{
				"/nonexistent": {
					Path: "/nonexistent",
				},
			},
		},
	}
	d := dto.Disk{
		Id:               &diskID,
		LegacyDeviceName: nil, // important: should be skipped
		Partitions:       &partitions,
	}
	disks := []*dto.Disk{&d}

	mock.When(suite.volumeMock.GetVolumesData()).ThenReturn(disks)
	mock.When(suite.hdidleMock.IsRunning()).ThenReturn(false)

	// Act
	err := suite.ds.updateDiskStats()

	// Assert
	suite.NoError(err)
	// Device was nil so no PerDiskIO entries should be added
	suite.NotNil(suite.ds.currentDiskHealth)
	suite.Empty(suite.ds.currentDiskHealth.PerDiskIO)
	// PerPartitionInfo map should not have an entry for the skipped device
	if d.Id != nil {
		_, exists := suite.ds.currentDiskHealth.PerPartitionInfo[*d.Id]
		suite.False(exists)
	}

	// Verify mock call
	mock.Verify(suite.volumeMock, matchers.Times(1)).GetVolumesData()
}

func (suite *DiskStatsServiceSuite) TestUpdateDiskStats_FsckStateFromFilesystemService() {
	fsType := "ext4"
	diskID := "disk-1"
	deviceName := "sda"
	devicePath := "/dev/sda"
	partitionID := "sda1"
	partitionPath := "/dev/sda1"

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDeviceName: &partitionID,
		LegacyDevicePath: &partitionPath,
		FsType:           &fsType,
	}
	partitions := map[string]dto.Partition{partitionID: partition}

	disk := &dto.Disk{
		Id:               &diskID,
		LegacyDeviceName: &deviceName,
		DevicePath:       &devicePath,
		Partitions:       &partitions,
	}

	mock.When(suite.volumeMock.GetVolumesData()).ThenReturn([]*dto.Disk{disk})
	mock.When(suite.hdidleMock.IsRunning()).ThenReturn(false)

	support := &dto.FilesystemInfo{
		Support: &dto.FilesystemSupport{
			CanCheck:    true,
			CanGetState: true,
		},
	}
	mock.When(suite.fsMock.GetSupportAndInfo(mock.Any[context.Context](), mock.Any[string]())).
		ThenReturn(support, nil)

	state := &dto.FilesystemState{
		IsClean:   false,
		HasErrors: true,
	}
	mock.When(suite.fsMock.GetPartitionState(mock.Any[context.Context](), mock.Any[string](), mock.Any[string]())).
		ThenReturn(state, nil)

	err := suite.ds.updateDiskStats()
	suite.NoError(err)
	suite.NotNil(suite.ds.currentDiskHealth)

	info := suite.ds.currentDiskHealth.PerPartitionInfo[diskID]
	suite.Len(info, 1)
	suite.NotNil(info[0].FilesystemState)
	suite.True(info[0].FilesystemState.HasErrors)
	suite.False(info[0].FilesystemState.IsClean)
}

func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_CacheHit() {
	// Test that cache hit returns cached value without querying SMART service
	diskID := "disk-with-smart"

	// Pre-populate cache
	// Cache is already initialized in SetupTest
	suite.ds.smartEnabledCache.Set(diskID, true, cache.DefaultExpiration)

	// Act - should use cache, not call SMART service
	enabled := suite.ds.isSmartEnabled(diskID)

	// Assert
	suite.True(enabled)
	// Verify no SMART service calls were made (use Times(0) instead of Never)
	mock.Verify(suite.smartMock, matchers.Times(0)).GetSmartInfo(mock.AnyContext(), mock.Any[string]())
}

func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_CacheMiss_SmartEnabled() {
	// Test that cache miss queries SMART and caches the result
	diskID := "disk-enabled"
	smartInfo := &dto.SmartInfo{
		DiskId:    diskID,
		Supported: true,
		Enabled:   true, // Now using Enabled field from SmartInfo
	}

	// Arrange - cache is already empty from SetupTest
	// No need to reinitialize

	// Mock SMART service calls - now only GetSmartInfo is needed
	mock.When(suite.smartMock.GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))).ThenReturn(smartInfo, nil)

	// Act
	enabled := suite.ds.isSmartEnabled(diskID)

	// Assert
	suite.True(enabled)
	// Check cache was populated
	cachedValue, found := getCacheValue(suite.ds.smartEnabledCache, diskID)
	suite.True(found, "Cache should be populated")
	suite.True(cachedValue, "Cache should contain enabled=true")
	mock.Verify(suite.smartMock, matchers.Times(1)).GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))
}

func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_CacheMiss_SmartDisabled() {
	// Test that disabled SMART is cached
	diskID := "disk-disabled"
	smartInfo := &dto.SmartInfo{
		DiskId:    diskID,
		Supported: true,
		Enabled:   false, // SMART is supported but disabled
	}

	// Arrange - cache is already empty from SetupTest

	// Mock SMART service call
	mock.When(suite.smartMock.GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))).ThenReturn(smartInfo, nil)

	// Act
	enabled := suite.ds.isSmartEnabled(diskID)

	// Assert
	suite.False(enabled)
	// Check cache was populated
	cachedValue, found := getCacheValue(suite.ds.smartEnabledCache, diskID)
	suite.True(found, "Cache should be populated")
	suite.False(cachedValue, "Disabled state should be cached")

	// Second call should use cache
	enabled2 := suite.ds.isSmartEnabled(diskID)
	suite.False(enabled2)

	// Verify SMART service called only once (on first call)
	mock.Verify(suite.smartMock, matchers.Times(1)).GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))
}

func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_SmartNotSupported() {
	// Test that unsupported SMART is cached as disabled
	diskID := "disk-no-smart"
	smartInfo := &dto.SmartInfo{
		DiskId:    diskID,
		Supported: false,
		Enabled:   false,
	}

	// Arrange - cache is already empty from SetupTest

	// Mock SMART service call
	mock.When(suite.smartMock.GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))).ThenReturn(smartInfo, nil)

	// Act
	enabled := suite.ds.isSmartEnabled(diskID)

	// Assert
	suite.False(enabled)
	cachedValue, found := getCacheValue(suite.ds.smartEnabledCache, diskID)
	suite.True(found, "Cache should be populated")
	suite.False(cachedValue, "Unsupported SMART should be cached as disabled")

	// Verify only GetSmartInfo was called
	mock.Verify(suite.smartMock, matchers.Times(1)).GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))
}

func (suite *DiskStatsServiceSuite) TestInvalidateSmartCache_SpecificDisk() {
	// Test that invalidating a specific disk clears only that disk's cache entry
	disk1 := "disk-1"
	disk2 := "disk-2"

	// Arrange - populate cache
	// Cache is already initialized in SetupTest
	suite.ds.smartEnabledCache.Set(disk1, true, cache.DefaultExpiration)
	suite.ds.smartEnabledCache.Set(disk2, false, cache.DefaultExpiration)

	// Act - invalidate disk1
	suite.ds.InvalidateSmartCache(disk1)

	// Assert - disk1 should be removed, disk2 should remain
	_, disk1Exists := getCacheValue(suite.ds.smartEnabledCache, disk1)
	suite.False(disk1Exists, "disk1 should be removed from cache")

	disk2Value, disk2Exists := getCacheValue(suite.ds.smartEnabledCache, disk2)
	suite.True(disk2Exists, "disk2 should remain in cache")
	suite.False(disk2Value, "disk2 value should be unchanged")
}

func (suite *DiskStatsServiceSuite) TestInvalidateSmartCache_AllDisks() {
	// Test that invalidating with empty string clears entire cache
	disk1 := "disk-1"
	disk2 := "disk-2"

	// Arrange - populate cache
	// Cache is already initialized in SetupTest
	suite.ds.smartEnabledCache.Set(disk1, true, cache.DefaultExpiration)
	suite.ds.smartEnabledCache.Set(disk2, false, cache.DefaultExpiration)

	// Act - invalidate all
	suite.ds.InvalidateSmartCache("")

	// Assert - cache should be empty (both entries should be gone)
	_, disk1Exists := getCacheValue(suite.ds.smartEnabledCache, disk1)
	suite.False(disk1Exists, "disk1 should be removed from cache")

	_, disk2Exists := getCacheValue(suite.ds.smartEnabledCache, disk2)
	suite.False(disk2Exists, "disk2 should be removed from cache")
}
