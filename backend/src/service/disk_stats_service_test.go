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

	// instantiate diskStatsService under test with mocks
	suite.ds = &diskStatsService{
		volumeService:  suite.volumeMock,
		blockdevice:    &blockdevice.FS{}, // zero value; tests avoid calling SysBlockDeviceStat
		ctx:            suite.ctx,
		lastUpdateTime: time.Now(),
		updateMutex:    &sync.Mutex{},
		lastStats:      make(map[string]*blockdevice.IOStats),
		smartService:   suite.smartMock,
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

	// Verify mock call
	mock.Verify(suite.volumeMock, matchers.Times(1)).GetVolumesData()
}

func (suite *DiskStatsServiceSuite) TestUpdateDiskStats_SkipsDiskWithNilDevice() {
	// Arrange: prepare a disk with nil Device but with partitions (which should be skipped)
	diskID := "disk-123"
	partitions := []dto.Partition{
		{
			Size: nil,
			MountPointData: &[]dto.MountPointData{
				{
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
	disks := []dto.Disk{d}

	mock.When(suite.volumeMock.GetVolumesData()).ThenReturn(&disks)

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
	mountData := []dto.MountPointData{
		{
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
	mountData := []dto.MountPointData{
		{
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
	mountData := []dto.MountPointData{
		{
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
