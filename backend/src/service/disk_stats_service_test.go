package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
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

	// instantiate diskStatsService under test with mocks
	suite.ds = &diskStatsService{
		volumeService:  suite.volumeMock,
		blockdevice:    &blockdevice.FS{}, // zero value; tests avoid calling SysBlockDeviceStat
		ctx:            suite.ctx,
		lastUpdateTime: time.Now(),
		updateMutex:    &sync.Mutex{},
		lastStats:      make(map[string]*blockdevice.IOStats),
		smartService:   suite.smartMock,
		hdidleService:  suite.hdidleMock,
		readFile:       os.ReadFile,
		sysFsBasePath:  "/sys/fs",
	}
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

func (suite *DiskStatsServiceSuite) TestDetermineFsckNeeded_UnmountedConfiguredPartition() {
	fsType := "ext4"
	deviceID := "sda1"
	mountPath := "/mnt/data"
	mountData := map[string]dto.MountPointData{
		mountPath: {
			Path:      mountPath,
			IsMounted: false,
		},
	}

	partition := dto.Partition{
		Id:               &deviceID,
		LegacyDeviceName: &deviceID,
		FsType:           &fsType,
		MountPointData:   &mountData,
	}

	needed := suite.ds.determineFsckNeeded(&partition, fsType, true)
	suite.True(needed)
}

func (suite *DiskStatsServiceSuite) TestDetermineFsckNeeded_ExtFilesystemDirtyState() {
	fsType := "ext4"
	deviceID := "sdb1"
	mountData := map[string]dto.MountPointData{
		"/data": {
			Path:      "/data",
			IsMounted: true,
		},
	}

	partition := dto.Partition{
		Id:               &deviceID,
		LegacyDeviceName: &deviceID,
		FsType:           &fsType,
		MountPointData:   &mountData,
	}

	tempDir := suite.T().TempDir()
	prevBase := suite.ds.sysFsBasePath
	suite.ds.sysFsBasePath = tempDir
	defer func() { suite.ds.sysFsBasePath = prevBase }()

	extDir := filepath.Join(tempDir, "ext4", deviceID)
	suite.Require().NoError(os.MkdirAll(extDir, 0o755))
	suite.Require().NoError(os.WriteFile(filepath.Join(extDir, "state"), []byte("needs_recovery"), 0o644))

	needed := suite.ds.determineFsckNeeded(&partition, fsType, true)
	suite.True(needed)
}

func (suite *DiskStatsServiceSuite) TestDetermineFsckNeeded_CleanMountedPartition() {
	fsType := "ext4"
	deviceID := "sdc1"
	mountData := map[string]dto.MountPointData{
		"/data": {
			Path:      "/data",
			IsMounted: true,
		},
	}

	partition := dto.Partition{
		Id:               &deviceID,
		LegacyDeviceName: &deviceID,
		FsType:           &fsType,
		MountPointData:   &mountData,
	}

	tempDir := suite.T().TempDir()
	prevBase := suite.ds.sysFsBasePath
	suite.ds.sysFsBasePath = tempDir
	defer func() { suite.ds.sysFsBasePath = prevBase }()

	extDir := filepath.Join(tempDir, "ext4", deviceID)
	suite.Require().NoError(os.MkdirAll(extDir, 0o755))
	suite.Require().NoError(os.WriteFile(filepath.Join(extDir, "state"), []byte("clean"), 0o644))

	needed := suite.ds.determineFsckNeeded(&partition, fsType, true)
	suite.False(needed)
}

func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_CacheHit() {
	// Test that cache hit returns cached value without querying SMART service
	diskID := "disk-with-smart"
	
	// Pre-populate cache
	suite.ds.smartEnabledCache = make(map[string]bool)
	suite.ds.smartEnabledCache[diskID] = true
	
	// Act - should use cache, not call SMART service
	enabled := suite.ds.isSmartEnabled(diskID)
	
	// Assert
	suite.True(enabled)
	// Verify no SMART service calls were made (use Times(0) instead of Never)
	mock.Verify(suite.smartMock, matchers.Times(0)).GetSmartInfo(mock.AnyContext(), mock.Any[string]())
	mock.Verify(suite.smartMock, matchers.Times(0)).GetSmartStatus(mock.AnyContext(), mock.Any[string]())
}

func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_CacheMiss_SmartEnabled() {
	// Test that cache miss queries SMART and caches the result
	diskID := "disk-enabled"
	smartInfo := &dto.SmartInfo{
		DiskId:    diskID,
		Supported: true,
	}
	smartStatus := &dto.SmartStatus{
		Enabled: true,
	}
	
	// Arrange - fresh cache
	suite.ds.smartEnabledCache = make(map[string]bool)
	
	// Mock SMART service calls
	mock.When(suite.smartMock.GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))).ThenReturn(smartInfo, nil)
	mock.When(suite.smartMock.GetSmartStatus(mock.AnyContext(), mock.Exact(diskID))).ThenReturn(smartStatus, nil)
	
	// Act
	enabled := suite.ds.isSmartEnabled(diskID)
	
	// Assert
	suite.True(enabled)
	suite.True(suite.ds.smartEnabledCache[diskID], "Cache should be populated")
	mock.Verify(suite.smartMock, matchers.Times(1)).GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))
	mock.Verify(suite.smartMock, matchers.Times(1)).GetSmartStatus(mock.AnyContext(), mock.Exact(diskID))
}

func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_CacheMiss_SmartDisabled() {
	// Test that disabled SMART is cached
	diskID := "disk-disabled"
	smartInfo := &dto.SmartInfo{
		DiskId:    diskID,
		Supported: true,
	}
	smartStatus := &dto.SmartStatus{
		Enabled: false,
	}
	
	// Arrange - fresh cache
	suite.ds.smartEnabledCache = make(map[string]bool)
	
	// Mock SMART service calls
	mock.When(suite.smartMock.GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))).ThenReturn(smartInfo, nil)
	mock.When(suite.smartMock.GetSmartStatus(mock.AnyContext(), mock.Exact(diskID))).ThenReturn(smartStatus, nil)
	
	// Act
	enabled := suite.ds.isSmartEnabled(diskID)
	
	// Assert
	suite.False(enabled)
	suite.False(suite.ds.smartEnabledCache[diskID], "Disabled state should be cached")
	
	// Second call should use cache
	enabled2 := suite.ds.isSmartEnabled(diskID)
	suite.False(enabled2)
	
	// Verify SMART service called only once (on first call)
	mock.Verify(suite.smartMock, matchers.Times(1)).GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))
	mock.Verify(suite.smartMock, matchers.Times(1)).GetSmartStatus(mock.AnyContext(), mock.Exact(diskID))
}

func (suite *DiskStatsServiceSuite) TestIsSmartEnabled_SmartNotSupported() {
	// Test that unsupported SMART is cached as disabled
	diskID := "disk-no-smart"
	smartInfo := &dto.SmartInfo{
		DiskId:    diskID,
		Supported: false,
	}
	
	// Arrange - fresh cache
	suite.ds.smartEnabledCache = make(map[string]bool)
	
	// Mock SMART service call
	mock.When(suite.smartMock.GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))).ThenReturn(smartInfo, nil)
	
	// Act
	enabled := suite.ds.isSmartEnabled(diskID)
	
	// Assert
	suite.False(enabled)
	suite.False(suite.ds.smartEnabledCache[diskID], "Unsupported SMART should be cached as disabled")
	
	// Verify GetSmartStatus was NOT called since SMART is not supported
	mock.Verify(suite.smartMock, matchers.Times(1)).GetSmartInfo(mock.AnyContext(), mock.Exact(diskID))
	mock.Verify(suite.smartMock, matchers.Times(0)).GetSmartStatus(mock.AnyContext(), mock.Any[string]())
}

func (suite *DiskStatsServiceSuite) TestInvalidateSmartCache_SpecificDisk() {
	// Test that invalidating a specific disk clears only that disk's cache entry
	disk1 := "disk-1"
	disk2 := "disk-2"
	
	// Arrange - populate cache
	suite.ds.smartEnabledCache = make(map[string]bool)
	suite.ds.smartEnabledCache[disk1] = true
	suite.ds.smartEnabledCache[disk2] = false
	
	// Act - invalidate disk1
	suite.ds.InvalidateSmartCache(disk1)
	
	// Assert - disk1 should be removed, disk2 should remain
	_, disk1Exists := suite.ds.smartEnabledCache[disk1]
	suite.False(disk1Exists, "disk1 should be removed from cache")
	
	disk2Value, disk2Exists := suite.ds.smartEnabledCache[disk2]
	suite.True(disk2Exists, "disk2 should remain in cache")
	suite.False(disk2Value, "disk2 value should be unchanged")
}

func (suite *DiskStatsServiceSuite) TestInvalidateSmartCache_AllDisks() {
	// Test that invalidating with empty string clears entire cache
	disk1 := "disk-1"
	disk2 := "disk-2"
	
	// Arrange - populate cache
	suite.ds.smartEnabledCache = make(map[string]bool)
	suite.ds.smartEnabledCache[disk1] = true
	suite.ds.smartEnabledCache[disk2] = false
	
	// Act - invalidate all
	suite.ds.InvalidateSmartCache("")
	
	// Assert - cache should be empty
	suite.Empty(suite.ds.smartEnabledCache, "Cache should be completely cleared")
}
