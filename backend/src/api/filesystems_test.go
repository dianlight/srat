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
	"github.com/dianlight/srat/internal/ctxkeys"
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
	handler       *api.FilesystemHandler
	mockFsService service.FilesystemServiceInterface
	diskMap       *dto.DiskMap
	testAPI       humatest.TestAPI
	ctx           context.Context
	cancel        context.CancelFunc
	app           *fxtest.App
}

func TestFilesystemHandlerSuite(t *testing.T) {
	suite.Run(t, new(FilesystemHandlerSuite))
}

func (suite *FilesystemHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			api.NewFilesystemHandler,
			mock.Mock[service.FilesystemServiceInterface],
			func() *dto.DiskMap { return &dto.DiskMap{} },
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockFsService),
		fx.Populate(&suite.diskMap),
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
			Name:        fsType,
			Type:        fsType,
			Description: fsType + " filesystem",
			//MountFlags:       []dto.MountFlag{{Name: "rw"}},
			CustomMountFlags: []dto.MountFlag{{Name: "discard"}},
			Support: &dto.FilesystemSupport{
				CanMount:               true,
				CanFormat:              true,
				CanCheck:               true,
				CanSetLabel:            true,
				IsFormatReportProgress: false,
				IsCheckReportProgress:  false,
				LabelRule:              `^[^\x00/]{1,16}$`,
				AlpinePackage:          fsType + "-progs",
			},
		}
		mock.When(suite.mockFsService.GetSupportAndInfo(mock.Any[context.Context](), mock.Any[string]())).
			ThenReturn(info, nil)
	}

	resp := suite.testAPI.Get("/filesystems")
	suite.Equal(http.StatusOK, resp.Code)

	result := dto.FilesystemsInfo{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.Len(result.Filesystems, 3)

	for _, fs := range result.Filesystems {
		suite.NotNil(fs.Support)
		suite.True(fs.Support.CanMount)
		suite.True(fs.Support.CanFormat)
	}
}

func (suite *FilesystemHandlerSuite) TestGetFilesystemSupport_Success() {
	fsType := "ext4"
	mock.When(suite.mockFsService.ListSupportedTypes()).ThenReturn([]string{fsType})
	expected := &dto.FilesystemInfo{
		Name:        fsType,
		Type:        fsType,
		Description: "ext4 filesystem",
		Support: &dto.FilesystemSupport{
			CanMount:               true,
			CanFormat:              true,
			CanCheck:               true,
			CanSetLabel:            true,
			CanGetState:            true,
			IsFormatReportProgress: false,
			IsCheckReportProgress:  false,
			LabelRule:              `^[^\x00/]{1,16}$`,
			AlpinePackage:          "e2fsprogs",
			MissingTools:           []string{},
		},
	}

	mock.When(suite.mockFsService.GetSupportAndInfo(mock.Any[context.Context](), mock.Exact(fsType))).
		ThenReturn(expected, nil)

	resp := suite.testAPI.Get("/filesystem/support?fstype=ext4")
	suite.Equal(http.StatusOK, resp.Code)

	var result dto.FilesystemSupport
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.True(result.CanCheck)
	suite.Equal("e2fsprogs", result.AlpinePackage)
	suite.False(result.IsFormatReportProgress)
	suite.False(result.IsCheckReportProgress)
	suite.Equal(`^[^\x00/]{1,16}$`, result.LabelRule)
}

func (suite *FilesystemHandlerSuite) TestGetFilesystemSupport_MissingFsType() {
	resp := suite.testAPI.Get("/filesystem/support")
	suite.Equal(http.StatusBadRequest, resp.Code)
}

func (suite *FilesystemHandlerSuite) TestGetFilesystemSupport_UnsupportedFsType() {
	mock.When(suite.mockFsService.ListSupportedTypes()).ThenReturn([]string{"ext4", "xfs"})

	resp := suite.testAPI.Get("/filesystem/support?fstype=zfsx")
	suite.Equal(http.StatusBadRequest, resp.Code)
}

func (suite *FilesystemHandlerSuite) TestGetFilesystemSupport_MultiFilesystemCapabilityProfiles() {
	mock.When(suite.mockFsService.ListSupportedTypes()).ThenReturn([]string{"f2fs", "zfs"})

	mock.When(suite.mockFsService.GetSupportAndInfo(mock.Any[context.Context](), mock.Exact("f2fs"))).
		ThenReturn(&dto.FilesystemInfo{
			Type: "f2fs",
			Support: &dto.FilesystemSupport{
				CanMount:               true,
				CanFormat:              true,
				CanCheck:               true,
				CanSetLabel:            false,
				CanGetState:            true,
				IsFormatReportProgress: false,
				IsCheckReportProgress:  false,
				LabelRule:              `^[^\x00/]{1,512}$`,
				AlpinePackage:          "f2fs-tools",
				MissingTools:           []string{},
			},
		}, nil)

	mock.When(suite.mockFsService.GetSupportAndInfo(mock.Any[context.Context](), mock.Exact("zfs"))).
		ThenReturn(&dto.FilesystemInfo{
			Type: "zfs",
			Support: &dto.FilesystemSupport{
				CanMount:               true,
				CanFormat:              false,
				CanCheck:               false,
				CanSetLabel:            false,
				CanGetState:            true,
				IsFormatReportProgress: false,
				IsCheckReportProgress:  false,
				LabelRule:              "",
				AlpinePackage:          "zfs",
				MissingTools:           []string{"zpool"},
			},
		}, nil)

	respF2fs := suite.testAPI.Get("/filesystem/support?fstype=f2fs")
	suite.Equal(http.StatusOK, respF2fs.Code)
	var supportF2fs dto.FilesystemSupport
	err := json.Unmarshal(respF2fs.Body.Bytes(), &supportF2fs)
	suite.Require().NoError(err)
	suite.True(supportF2fs.CanCheck)
	suite.True(supportF2fs.CanFormat)
	suite.False(supportF2fs.CanSetLabel)
	suite.Equal("f2fs-tools", supportF2fs.AlpinePackage)
	suite.False(supportF2fs.IsFormatReportProgress)
	suite.False(supportF2fs.IsCheckReportProgress)
	suite.Equal(`^[^\x00/]{1,512}$`, supportF2fs.LabelRule)

	respZfs := suite.testAPI.Get("/filesystem/support?fstype=zfs")
	suite.Equal(http.StatusOK, respZfs.Code)
	var supportZfs dto.FilesystemSupport
	err = json.Unmarshal(respZfs.Body.Bytes(), &supportZfs)
	suite.Require().NoError(err)
	suite.False(supportZfs.CanCheck)
	suite.False(supportZfs.CanFormat)
	suite.False(supportZfs.CanSetLabel)
	suite.Equal("zfs", supportZfs.AlpinePackage)
	suite.Contains(supportZfs.MissingTools, "zpool")
	suite.False(supportZfs.IsFormatReportProgress)
	suite.False(supportZfs.IsCheckReportProgress)
	suite.Empty(supportZfs.LabelRule)
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

	// Populate disk map with test disk
	(*suite.diskMap)[diskID] = disk

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
	// diskMap is empty by default (no partitions found)

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

	mock.When(suite.mockFsService.FormatPartition(
		mock.Any[context.Context](),
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[dto.FormatOptions](),
	)).ThenReturn(nil, errors.New("unsupported filesystem"))

	// Populate disk map with test disk
	(*suite.diskMap)[diskID] = disk

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

	(*suite.diskMap)[diskID] = disk

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

func (suite *FilesystemHandlerSuite) TestAbortCheckPartition_Success() {
	partitionID := "test-partition-id"
	devicePath := "/dev/sdb1"
	fsType := "ext4"
	diskID := "test-disk-id"

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	partitions := make(map[string]dto.Partition)
	partitions[partitionID] = partition
	disk := &dto.Disk{
		Id:         &diskID,
		Partitions: &partitions,
	}

	(*suite.diskMap)[diskID] = disk

	mock.When(suite.mockFsService.AbortCheckPartition(
		mock.Any[context.Context](),
		mock.Any[string](),
	)).ThenReturn(nil)

	resp := suite.testAPI.Post("/filesystem/check/abort", map[string]interface{}{
		"partitionId": partitionID,
	})

	suite.Equal(http.StatusOK, resp.Code)

	var result struct {
		Success bool `json:"success"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)
	suite.True(result.Success)
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

	(*suite.diskMap)[diskID] = disk

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

	(*suite.diskMap)[diskID] = disk

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

	(*suite.diskMap)[diskID] = disk

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
