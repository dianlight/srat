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
	handler            *api.FilesystemHandler
	mockFsService      service.FilesystemServiceInterface
	mockVolumeService  service.VolumeServiceInterface
	mockAdapter        *mockFilesystemAdapter
	testAPI            humatest.TestAPI
	ctx                context.Context
	cancel             context.CancelFunc
	app                *fxtest.App
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

	suite.mockAdapter = &mockFilesystemAdapter{}

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
		adapter := suite.createMockAdapter(fsType)
		mock.When(suite.mockFsService.GetAdapter(fsType)).ThenReturn(adapter, nil)
		support := dto.FilesystemSupport{
			CanMount:      true,
			CanFormat:     true,
			CanCheck:      true,
			CanSetLabel:   true,
			AlpinePackage: fsType + "-progs",
		}
		mock.When(adapter.IsSupported(matchers.Any[context.Context]())).ThenReturn(support, nil)
	}

	standardFlags := []dto.MountFlag{{Name: "rw"}}
	customFlags := []dto.MountFlag{{Name: "discard"}}
	mock.When(suite.mockFsService.GetStandardMountFlags()).ThenReturn(standardFlags, nil).MaybeRepeat()
	mock.When(suite.mockFsService.GetFilesystemSpecificMountFlags(matchers.Any[string]())).
		ThenReturn(customFlags, nil).MaybeRepeat()

	resp := suite.testAPI.Get("/filesystems")
	suite.Equal(http.StatusOK, resp.Code)

	var result []api.FilesystemInfo
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

	// Mock partition data
	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	disk := &dto.Disk{
		Partitions: &map[string]dto.Partition{
			partitionID: partition,
		},
	}

	volumes := []*dto.Disk{disk}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(volumes)

	adapter := suite.createMockAdapter(fsType)
	mock.When(suite.mockFsService.GetAdapter(fsType)).ThenReturn(adapter, nil)

	support := dto.FilesystemSupport{
		CanFormat:     true,
		AlpinePackage: "e2fsprogs",
	}
	mock.When(adapter.IsSupported(matchers.Any[context.Context]())).ThenReturn(support, nil)
	mock.When(adapter.Format(
		matchers.Any[context.Context](),
		matchers.Eq(devicePath),
		matchers.Any[dto.FormatOptions](),
	)).ThenReturn(nil)

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
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn([]*dto.Disk{})

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

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	disk := &dto.Disk{
		Partitions: &map[string]dto.Partition{
			partitionID: partition,
		},
	}

	volumes := []*dto.Disk{disk}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(volumes)

	mock.When(suite.mockFsService.GetAdapter("unknown-fs")).
		ThenReturn(nil, errors.New("unsupported filesystem"))

	resp := suite.testAPI.Post("/filesystem/format", map[string]interface{}{
		"partitionId":    partitionID,
		"filesystemType": "unknown-fs",
	})

	suite.Equal(http.StatusBadRequest, resp.Code)
}

func (suite *FilesystemHandlerSuite) TestCheckPartition_Success() {
	partitionID := "test-partition-id"
	devicePath := "/dev/sdb1"
	fsType := "ext4"

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	disk := &dto.Disk{
		Partitions: &map[string]dto.Partition{
			partitionID: partition,
		},
	}

	volumes := []*dto.Disk{disk}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(volumes)

	adapter := suite.createMockAdapter(fsType)
	mock.When(suite.mockFsService.GetAdapter(fsType)).ThenReturn(adapter, nil)

	support := dto.FilesystemSupport{
		CanCheck:      true,
		AlpinePackage: "e2fsprogs",
	}
	mock.When(adapter.IsSupported(matchers.Any[context.Context]())).ThenReturn(support, nil)

	checkResult := dto.CheckResult{
		Success:      true,
		ErrorsFound:  false,
		ErrorsFixed:  false,
		Message:      "Filesystem is clean",
		ExitCode:     0,
	}
	mock.When(adapter.Check(
		matchers.Any[context.Context](),
		matchers.Eq(devicePath),
		matchers.Any[dto.CheckOptions](),
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

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	disk := &dto.Disk{
		Partitions: &map[string]dto.Partition{
			partitionID: partition,
		},
	}

	volumes := []*dto.Disk{disk}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(volumes)

	adapter := suite.createMockAdapter(fsType)
	mock.When(suite.mockFsService.GetAdapter(fsType)).ThenReturn(adapter, nil)

	support := dto.FilesystemSupport{
		CanGetState:   true,
		AlpinePackage: "e2fsprogs",
	}
	mock.When(adapter.IsSupported(matchers.Any[context.Context]())).ThenReturn(support, nil)

	state := dto.FilesystemState{
		IsClean:          true,
		IsMounted:        false,
		HasErrors:        false,
		StateDescription: "clean",
	}
	mock.When(adapter.GetState(
		matchers.Any[context.Context](),
		matchers.Eq(devicePath),
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

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	disk := &dto.Disk{
		Partitions: &map[string]dto.Partition{
			partitionID: partition,
		},
	}

	volumes := []*dto.Disk{disk}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(volumes)

	adapter := suite.createMockAdapter(fsType)
	mock.When(suite.mockFsService.GetAdapter(fsType)).ThenReturn(adapter, nil)
	mock.When(adapter.GetLabel(
		matchers.Any[context.Context](),
		matchers.Eq(devicePath),
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

	partition := dto.Partition{
		Id:               &partitionID,
		LegacyDevicePath: &devicePath,
		FsType:           &fsType,
	}

	disk := &dto.Disk{
		Partitions: &map[string]dto.Partition{
			partitionID: partition,
		},
	}

	volumes := []*dto.Disk{disk}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(volumes)

	adapter := suite.createMockAdapter(fsType)
	mock.When(suite.mockFsService.GetAdapter(fsType)).ThenReturn(adapter, nil)

	support := dto.FilesystemSupport{
		CanSetLabel:   true,
		AlpinePackage: "e2fsprogs",
	}
	mock.When(adapter.IsSupported(matchers.Any[context.Context]())).ThenReturn(support, nil)
	mock.When(adapter.SetLabel(
		matchers.Any[context.Context](),
		matchers.Eq(devicePath),
		matchers.Eq(newLabel),
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

// Helper method to create a mock adapter
func (suite *FilesystemHandlerSuite) createMockAdapter(fsType string) *mockFilesystemAdapter {
	adapter := &mockFilesystemAdapter{
		name:        fsType,
		description: fmt.Sprintf("%s filesystem", fsType),
	}
	return adapter
}

// Mock filesystem adapter for testing
type mockFilesystemAdapter struct {
	name        string
	description string
}

func (m *mockFilesystemAdapter) GetName() string {
	return m.name
}

func (m *mockFilesystemAdapter) GetDescription() string {
	return m.description
}

func (m *mockFilesystemAdapter) GetMountFlags() []dto.MountFlag {
	return []dto.MountFlag{}
}

func (m *mockFilesystemAdapter) IsSupported(ctx context.Context) (dto.FilesystemSupport, errors.E) {
	return dto.FilesystemSupport{}, nil
}

func (m *mockFilesystemAdapter) Format(ctx context.Context, device string, options dto.FormatOptions) errors.E {
	return nil
}

func (m *mockFilesystemAdapter) Check(ctx context.Context, device string, options dto.CheckOptions) (dto.CheckResult, errors.E) {
	return dto.CheckResult{}, nil
}

func (m *mockFilesystemAdapter) GetLabel(ctx context.Context, device string) (string, errors.E) {
	return "", nil
}

func (m *mockFilesystemAdapter) SetLabel(ctx context.Context, device string, label string) errors.E {
	return nil
}

func (m *mockFilesystemAdapter) GetState(ctx context.Context, device string) (dto.FilesystemState, errors.E) {
	return dto.FilesystemState{}, nil
}

func (m *mockFilesystemAdapter) IsDeviceSupported(ctx context.Context, devicePath string) (bool, errors.E) {
	return true, nil
}

func (m *mockFilesystemAdapter) GetFsSignatureMagic() []dto.FsMagicSignature {
	return []dto.FsMagicSignature{}
}
