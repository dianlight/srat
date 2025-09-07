package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/prometheus/procfs/blockdevice"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
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
	mock.When(suite.volumeMock.GetVolumesData()).ThenReturn(nil, errors.WithStack(dto.ErrorNotFound))

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

	mock.When(suite.volumeMock.GetVolumesData()).ThenReturn(&disks, nil)

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
