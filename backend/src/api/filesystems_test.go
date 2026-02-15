package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type FilesystemHandlerSuite struct {
	suite.Suite
	handler           *api.FilesystemHandler
	mockFsService     service.FilesystemServiceInterface
	mockVolumeService service.VolumeServiceInterface
	testAPI           humatest.TestAPI
	ctx               context.Context
	cancel            context.CancelFunc
	app               *fxtest.App
}

func TestFilesystemHandlerSuite(t *testing.T) {
	suite.Run(t, new(FilesystemHandlerSuite))
}

func (suite *FilesystemHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			api.NewFilesystemHandler,
			mock.Mock[service.FilesystemServiceInterface],
			mock.Mock[service.VolumeServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockFsService),
		fx.Populate(&suite.mockVolumeService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()

	_, testAPI := humatest.New(suite.T())
	suite.handler.RegisterFilesystemHandler(testAPI)
	suite.testAPI = testAPI
}

func (suite *FilesystemHandlerSuite) TearDownTest() {
	suite.cancel()
	suite.app.RequireStop()
}

func (suite *FilesystemHandlerSuite) TestListFilesystems_Success() {
	fsTypes := []string{"ext4", "vfat", "ntfs"}
	mock.When(suite.mockFsService.ListSupportedTypes()).ThenReturn(fsTypes)

	for _, fsType := range fsTypes {
		info := &dto.FilesystemInfo{
			Name:             fsType,
			Type:             fsType,
			Description:      fsType + " filesystem",
			MountFlags:       []dto.MountFlag{{Name: "rw"}},
			CustomMountFlags: []dto.MountFlag{{Name: "discard"}},
			Support: &dto.FilesystemSupport{
				CanMount:      true,
				CanFormat:     true,
				CanCheck:      true,
				CanSetLabel:   true,
				AlpinePackage: fsType + "-progs",
			},
		}
		mock.When(suite.mockFsService.GetSupportAndInfo(mock.Any[context.Context](), mock.Any[string]())).
			ThenReturn(info, nil)
	}

	resp := suite.testAPI.Get("/filesystems")
	suite.Equal(http.StatusOK, resp.Code)

	var result []dto.FilesystemInfo
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.Len(result, 3)

	for _, fs := range result {
		suite.NotNil(fs.Support)
		suite.True(fs.Support.CanMount)
		suite.True(fs.Support.CanFormat)
	}
}

func (suite *FilesystemHandlerSuite) TestFormatPartition_Success() {
	partitionID := "test-partition-id"
	devicePath := "/dev/sdb1"
	fsType := "ext4"
	label := "TestLabel"
	diskID := "test-disk-id"

	// Mock partition data
	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	// Create disk with partition
	partitions := make(map[string]dto.Partition)
	partitions[partitionID] = partition
	disk := &dto.Disk{
		Id:         &diskID,
		Partitions: &partitions,
	}

	// Mock volume service to return disk data
	mock.When(suite.mockVolumeService.GetVolumesData()).
		ThenReturn([]*dto.Disk{disk})

	// Mock filesystem service to format
	formatResult := &dto.CheckResult{
		Success:  true,
		Message:  fmt.Sprintf("Successfully formatted %s as %s", devicePath, fsType),
		ExitCode: 0,
	}
	mock.When(suite.mockFsService.FormatPartition(
		mock.Any[context.Context](),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[dto.FormatOptions](),
	)).ThenReturn(formatResult, nil)

	resp := suite.testAPI.Post("/filesystem/format", map[string]interface{}{
		"partitionId":    partitionID,
		"filesystemType": fsType,
		"label":          label,
		"force":          true,
	})

	suite.Equal(http.StatusOK, resp.Code)

	var result dto.CheckResult
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.True(result.Success)
}

func (suite *FilesystemHandlerSuite) TestFormatPartition_PartitionNotFound() {
	// Mock empty disks array (no partitions found)
	mock.When(suite.mockVolumeService.GetVolumesData()).
		ThenReturn([]*dto.Disk{})

	resp := suite.testAPI.Post("/filesystem/format", map[string]interface{}{
		"partitionId":    "non-existent",
		"filesystemType": "ext4",
	})

	suite.Equal(http.StatusNotFound, resp.Code)
}

func (suite *FilesystemHandlerSuite) TestFormatPartition_UnsupportedFilesystem() {
	partitionID := "test-partition-id"
	devicePath := "/dev/sdb1"
	fsType := "ext4"
	diskID := "test-disk-id"

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	// Create disk with partition
	partitions := make(map[string]dto.Partition)
	partitions[partitionID] = partition
	disk := &dto.Disk{
		Id:         &diskID,
		Partitions: &partitions,
	}

	mock.When(suite.mockVolumeService.GetVolumesData()).
		ThenReturn([]*dto.Disk{disk})

	mock.When(suite.mockFsService.FormatPartition(
		mock.Any[context.Context](),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[dto.FormatOptions](),
	)).ThenReturn(nil, errors.New("unsupported filesystem"))

	resp := suite.testAPI.Post("/filesystem/format", map[string]interface{}{
		"partitionId":    partitionID,
		"filesystemType": "unknown-fs",
	})

	suite.Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *FilesystemHandlerSuite) TestCheckPartition_Success() {
	partitionID := "test-partition-id"
	devicePath := "/dev/sdb1"
	fsType := "ext4"
	diskID := "test-disk-id"

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	// Create disk with partition
	partitions := make(map[string]dto.Partition)
	partitions[partitionID] = partition
	disk := &dto.Disk{
		Id:         &diskID,
		Partitions: &partitions,
	}

	mock.When(suite.mockVolumeService.GetVolumesData()).
		ThenReturn([]*dto.Disk{disk})

	checkResult := &dto.CheckResult{
		Success:     true,
		ErrorsFound: false,
		ErrorsFixed: false,
		Message:     "Filesystem is clean",
		ExitCode:    0,
	}
	mock.When(suite.mockFsService.CheckPartition(
		mock.Any[context.Context](),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[dto.CheckOptions](),
	)).ThenReturn(checkResult, nil)

	resp := suite.testAPI.Post("/filesystem/check", map[string]interface{}{
		"partitionId": partitionID,
		"autoFix":     true,
	})

	suite.Equal(http.StatusOK, resp.Code)

	var result dto.CheckResult
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.True(result.Success)
	suite.False(result.ErrorsFound)
}

func (suite *FilesystemHandlerSuite) TestGetPartitionState_Success() {
	partitionID := "test-partition-id"
	devicePath := "/dev/sdb1"
	fsType := "ext4"
	diskID := "test-disk-id"

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	// Create disk with partition
	partitions := make(map[string]dto.Partition)
	partitions[partitionID] = partition
	disk := &dto.Disk{
		Id:         &diskID,
		Partitions: &partitions,
	}

	mock.When(suite.mockVolumeService.GetVolumesData()).
		ThenReturn([]*dto.Disk{disk})

	state := &dto.FilesystemState{
		IsClean:          true,
		IsMounted:        false,
		HasErrors:        false,
		StateDescription: "clean",
	}
	mock.When(suite.mockFsService.GetPartitionState(
		mock.Any[context.Context](),
		mock.Any[string](),
		mock.Any[string](),
	)).ThenReturn(state, nil)

	resp := suite.testAPI.Get(fmt.Sprintf("/filesystem/state?partition_id=%s", partitionID))

	suite.Equal(http.StatusOK, resp.Code)

	var result dto.FilesystemState
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.True(result.IsClean)
	suite.False(result.IsMounted)
}

func (suite *FilesystemHandlerSuite) TestGetPartitionLabel_Success() {
	partitionID := "test-partition-id"
	devicePath := "/dev/sdb1"
	fsType := "ext4"
	expectedLabel := "MyLabel"
	diskID := "test-disk-id"

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	// Create disk with partition
	partitions := make(map[string]dto.Partition)
	partitions[partitionID] = partition
	disk := &dto.Disk{
		Id:         &diskID,
		Partitions: &partitions,
	}

	mock.When(suite.mockVolumeService.GetVolumesData()).
		ThenReturn([]*dto.Disk{disk})

	mock.When(suite.mockFsService.GetPartitionLabel(
		mock.Any[context.Context](),
		mock.Any[string](),
		mock.Any[string](),
	)).ThenReturn(expectedLabel, nil)

	resp := suite.testAPI.Get(fmt.Sprintf("/filesystem/label?partition_id=%s", partitionID))

	suite.Equal(http.StatusOK, resp.Code)

	var result struct {
		Label string `json:"label"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.Equal(expectedLabel, result.Label)
}

func (suite *FilesystemHandlerSuite) TestSetPartitionLabel_Success() {
	partitionID := "test-partition-id"
	devicePath := "/dev/sdb1"
	fsType := "ext4"
	newLabel := "NewLabel"
	diskID := "test-disk-id"

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	// Create disk with partition
	partitions := make(map[string]dto.Partition)
	partitions[partitionID] = partition
	disk := &dto.Disk{
		Id:         &diskID,
		Partitions: &partitions,
	}

	mock.When(suite.mockVolumeService.GetVolumesData()).
		ThenReturn([]*dto.Disk{disk})

	mock.When(suite.mockFsService.SetPartitionLabel(
		mock.Any[context.Context](),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[string](),
	)).ThenReturn(nil)

	resp := suite.testAPI.Put("/filesystem/label", map[string]interface{}{
		"partitionId": partitionID,
		"label":       newLabel,
	})

	suite.Equal(http.StatusOK, resp.Code)

	var result struct {
		Success bool `json:"success"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.True(result.Success)
}
