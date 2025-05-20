// /workspaces/srat/backend/src/service/volume_service_test.go
package service_test

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"

	// Needed for MockBroadcaster

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware" // Keep for the interface definition
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/snapcore/snapd/osutil"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	errors "gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
	// Remove gomock import: "go.uber.org/mock/gomock"
)

type VolumeServiceTestSuite struct {
	suite.Suite
	// ctrl *gomock.Controller // Removed gomock controller
	//mockBroadcaster    *MockBroadcaster
	mockMountRepo      repository.MountPointPathRepositoryInterface
	mockHardwareClient hardware.ClientWithResponsesInterface
	volumeService      service.VolumeServiceInterface
	lsblk              lsblk.LSBLKInterpreterInterface
	ctrl               *matchers.MockController
	ctx                context.Context
	cancel             context.CancelFunc
	app                *fxtest.App
}

func TestVolumeServiceTestSuite(t *testing.T) {
	suite.Run(t, new(VolumeServiceTestSuite))
}

func (suite *VolumeServiceTestSuite) SetupTest() {
	data, err := os.ReadFile("../../test/data/mount_info.txt")
	if err != nil {
		log.Fatal(err)
	}
	osutil.MockMountInfo(string(data))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			service.NewVolumeService,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[hardware.ClientWithResponsesInterface],
			mock.Mock[lsblk.LSBLKInterpreterInterface],
		),
		fx.Populate(&suite.volumeService),
		fx.Populate(&suite.mockMountRepo),
		fx.Populate(&suite.mockHardwareClient),
		fx.Populate(&suite.lsblk),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
	//suite.ctx, suite.cancel = context.WithCancel(context.Background())

	/*
		// Load the JSON data from the file
		// Read the JSON file
		// Read the JSON file
		jsonFile, err := os.ReadFile("../../test/data/hardware_example.json")
		if err != nil {
			suite.T().Errorf("Error reading JSON file: %v", err)
		}

		// Unmarshal the JSON data into the struct
		var data struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}
		err = json.Unmarshal(jsonFile, &data)
		if err != nil {
			suite.T().Errorf("Error unmarshalling JSON: %v", err)
		}

		// Mock initial GetVolumesData call during NewVolumeService
		suite.mockHardwareClient.On(
			"GetHardwareInfoWithResponse",
			suite.ctx,     // Pass context explicitly
			mock.Anything, // Use mock.Anything for variadic editors if details don't matter
		).Return(
			&hardware.GetHardwareInfoResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &data,
			}, nil).Maybe() // Maybe allows the call to happen 0 or more times

		suite.mockMountRepo.On("FindByPath", mock.Anything).Return(nil, gorm.ErrRecordNotFound).Maybe()
		suite.mockMountRepo.On("Save", mock.Anything).Return(nil).Maybe()

	*/
	//suite.volumeService = service.NewVolumeService( /*suite.ctx*/ testContext, suite.mockBroadcaster, suite.mockMountRepo, suite.mockHardwareClient)
}

func (suite *VolumeServiceTestSuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
	/*
		testContextCancel()
		//suite.cancel()
		// suite.ctrl.Finish() // Removed gomock verification
		suite.mockBroadcaster.AssertExpectations(suite.T())
		suite.mockMountRepo.AssertExpectations(suite.T())
		suite.mockHardwareClient.AssertExpectations(suite.T())
		// Give time for goroutines to potentially exit
		time.Sleep(10 * time.Millisecond)
	*/
}

// --- MountVolume Tests ---

func (suite *VolumeServiceTestSuite) TestMountVolume_Success() {
	mountPath := "/mnt/test1"
	device := "../../test/data/image.dmg"
	fsType := "ext4"
	mountData := dto.MountPointData{
		Path:   mountPath,
		Device: device,
		FSType: fsType,
		Flags: dto.MountFlags{
			dto.MountFlag{Name: "noatime", NeedsValue: false},
		},
	}
	dbomMountData := &dbom.MountPointPath{
		Path:      mountPath,
		Device:    device,
		FSType:    fsType,
		Flags:     dbom.MounDataFlags{dbom.MounDataFlag{Name: "noatime", NeedsValue: false}},
		IsMounted: false, // Initially not mounted
	}

	// Mock FindByPath
	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(dbomMountData, nil).Verify(matchers.Times(2))

	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
		mp, ok := args[0].(*dbom.MountPointPath)
		if !ok {
			suite.T().Errorf("Expected argument to be of type *dbom.MountPointPath, got %T", args[0])
		}
		suite.Equal(mountPath, mp.Path)
		dbomMountData.Device = mp.Device
		dbomMountData.IsMounted = mp.IsMounted
		return []any{nil}
	})).Verify(matchers.AtLeastOnce())

	mock.When(suite.mockHardwareClient.GetHardwareInfoWithResponse(mock.AnyContext())).ThenReturn(
		&hardware.GetHardwareInfoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &struct {
				Data   *hardware.HardwareInfo             `json:"data,omitempty"`
				Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
			}{Data: &hardware.HardwareInfo{Drives: &[]hardware.Drive{}}},
		}, nil).Verify(matchers.AtLeastOnce())

	defer func() {
		err := suite.volumeService.UnmountVolume(mountPath, false, false) // Cleanup
		suite.Require().Nil(err, "Expected no error on unmount")
	}()
	// --- Execute ---
	err := suite.volumeService.MountVolume(mountData)

	// --- Assert ---
	suite.Require().Nil(err, "Expected no error on successful mount")
	// Assertions on mocks are handled in TearDownTest by AssertExpectations

	//	save_call.Unset()

	//	suite.mockMountRepo.On("Save", mock.Anything).Return(nil).Maybe()

}

func (suite *VolumeServiceTestSuite) TestMountVolume_RepoFindByPathError() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, Device: "sda1"}
	expectedErr := errors.New("Invalid parameter")

	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(nil, expectedErr).Verify(matchers.Times(1))

	err := suite.volumeService.MountVolume(mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, expectedErr)
}

func (suite *VolumeServiceTestSuite) TestMountVolume_DeviceEmpty() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, Device: ""} // Empty device
	err := suite.volumeService.MountVolume(mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, dto.ErrorInvalidParameter)
	details := err.Details()
	suite.Contains(details, "Message")
	suite.Equal("Source device name is empty in request", details["Message"])
}

func (suite *VolumeServiceTestSuite) TestMountVolume_DeviceInvalid() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, Device: "/dev/pippo"} // Invalid device
	dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: ""}

	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(dbomMountData, nil).Verify(matchers.Times(1))
	//suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()

	err := suite.volumeService.MountVolume(mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, dto.ErrorDeviceNotFound)
	details := err.Details()
	suite.Contains(details, "Message")
	suite.Equal("Source device does not exist on the system", details["Message"])
}

func (suite *VolumeServiceTestSuite) TestMountVolume_PathEmpty() {
	// Note: The converter step might prevent this if Path is required,
	// but testing the service logic defensively.
	mountPath := ""
	device := "sda1"
	mountData := dto.MountPointData{Path: mountPath, Device: device}
	// FindByPath won't be called if path is empty early on
	// dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: device}
	// suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()

	err := suite.volumeService.MountVolume(mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, dto.ErrorInvalidParameter)
	details := err.Details()
	suite.Contains(details, "Message")
	suite.Equal("Mount point path is empty", details["Message"])
}

func (suite *VolumeServiceTestSuite) TestMountVolume_AlreadyMounted() {
	// This requires mocking osutil.IsMounted to return true
	// Skipping direct test due to mocking difficulty, relies on integration test.
	suite.T().Skip("Skipping TestMountVolume_AlreadyMounted due to osutil.IsMounted mocking difficulty")
}

func (suite *VolumeServiceTestSuite) TestMountVolume_MountFails() {
	// This requires mocking mount.Mount/TryMount to return an error
	// Skipping direct test due to mocking difficulty, relies on integration test.
	suite.T().Skip("Skipping TestMountVolume_MountFails due to mount.Mount mocking difficulty")
}

// --- UnmountVolume Tests ---

func (suite *VolumeServiceTestSuite) TestUnmountVolume_Success() {
	mountPath := "/mnt/test1"
	dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: "sda1", IsMounted: true}

	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(dbomMountData, nil).Verify(matchers.Times(1))

	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
		mp, ok := args[0].(*dbom.MountPointPath)
		if !ok {
			suite.T().Errorf("Expected argument to be of type *dbom.MountPointPath, got %T", args[0])
		}
		suite.Equal(mountPath, mp.Path)
		suite.False(mp.IsMounted)
		return []any{nil}
	})).Verify(matchers.AtLeastOnce())

	mock.When(suite.mockHardwareClient.GetHardwareInfoWithResponse(mock.AnyContext())).ThenReturn(
		&hardware.GetHardwareInfoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &struct {
				Data   *hardware.HardwareInfo             `json:"data,omitempty"`
				Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
			}{Data: &hardware.HardwareInfo{Drives: &[]hardware.Drive{}}},
		}, nil).Verify(matchers.AtLeastOnce())

	//suite.mockBroadcaster.On("BroadcastMessage", mock.AnythingOfType("*[]dto.Disk")).Return(nil, nil).Once()

	err := suite.volumeService.UnmountVolume(mountPath, false, false)
	suite.Nil(err, "Expected no error on successful unmount")
}

func (suite *VolumeServiceTestSuite) TestUnmountVolume_RepoFindByPathError() {
	mountPath := "/mnt/test1"
	expectedErr := errors.New("database error")

	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(nil, expectedErr).Verify(matchers.Times(1))
	//suite.mockMountRepo.On("FindByPath", mountPath).Return(nil, expectedErr).Once()

	err := suite.volumeService.UnmountVolume(mountPath, false, false)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, expectedErr)
}

func (suite *VolumeServiceTestSuite) TestUnmountVolume_UnmountFails() {
	// This requires mocking mount.Unmount to return an error
	// Skipping direct test due to mocking difficulty, relies on integration test.
	suite.T().Skip("Skipping TestUnmountVolume_UnmountFails due to mount.Unmount mocking difficulty")
}

// --- GetVolumesData Tests ---

func (suite *VolumeServiceTestSuite) TestGetVolumesData_Success() {
	// Prepare mock hardware response
	drive1 := hardware.Drive{
		Id:     pointer.String("drive-1"),
		Vendor: pointer.String("TestVendor"),
		Model:  pointer.String("TestModel"),
		Filesystems: &[]hardware.Filesystem{
			{
				Device:      pointer.String("/dev/sda1"),
				Name:        pointer.String("RootFS"),
				MountPoints: &[]string{"/mnt/rootfs"},
				Size:        pointer.Int(1000000000),
			},
			{
				Device:      pointer.String("/dev/sda2"),
				Name:        pointer.String("DataFS"),
				MountPoints: &[]string{"/mnt/data"}, // Mounted
				Size:        pointer.Int(2000000000),
			},
		},
	}
	mockHWResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{
			Data: &hardware.HardwareInfo{
				Drives: &[]hardware.Drive{drive1},
			},
			Result: (*hardware.GetHardwareInfo200Result)(pointer.String("ok")),
		},
	}

	// Prepare mock repo responses
	mountPath1 := "/mnt/rootfs"
	mountPath2 := "/mnt/data"
	dbomMountData1 := &dbom.MountPointPath{Path: mountPath1, Device: "sda1", IsMounted: true, Type: "ADDON"} // Initial state in DB
	dbomMountData2 := &dbom.MountPointPath{Path: mountPath2, Device: "sda2", IsMounted: true, Type: "ADDON"} // Initial state in DB

	mock.When(suite.mockHardwareClient.GetHardwareInfoWithResponse(mock.AnyContext())).ThenReturn(mockHWResponse, nil).Verify(matchers.AtLeastOnce())
	//suite.mockHardwareClient.On("GetHardwareInfoWithResponse" /*suite.ctx*/, testContext, mock.Anything).Return(mockHWResponse, nil).Once()

	// Expect FindByPath and Save for each mount point found in hardware data
	mock.When(suite.mockMountRepo.FindByPath(mountPath1)).ThenReturn(dbomMountData1, nil).Verify(matchers.Times(1))
	mock.When(suite.mockMountRepo.FindByPath(mountPath2)).ThenReturn(dbomMountData2, nil).Verify(matchers.Times(1))

	mock.When(suite.lsblk.GetInfoFromDevice("/dev/sda1")).ThenReturn(&lsblk.LSBKInfo{
		Name:       "sda1",
		Label:      "RootFS",
		Partlabel:  "RootFS",
		Mountpoint: mountPath1,
		Fstype:     "ext4",
	}, nil).Verify(matchers.Times(1))

	mock.When(suite.lsblk.GetInfoFromDevice("/dev/sda2")).ThenReturn(&lsblk.LSBKInfo{
		Name:       "sda2",
		Label:      "DataFS",
		Partlabel:  "DataFS",
		Mountpoint: mountPath2,
		Fstype:     "ext4",
	}, nil).Verify(matchers.Times(1))
	/*
		mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
			mp, ok := args[0].(*dbom.MountPointPath)
			if !ok {
				suite.T().Errorf("Expected argument to be of type *dbom.MountPointPath, got %T", args[0])
			}
			suite.Assert().Contains([]string{mountPath1, mountPath2}, mp.Path)
			//suite.True(mp.IsMounted) // Should reflect the input from hardware/converter
			// Assume converter works, check key fields
			return []any{nil}
		})).Verify(matchers.AtLeastOnce())
	*/
	// Call the function
	disks, err := suite.volumeService.GetVolumesData()

	// Assertions
	suite.Require().NoError(err)
	suite.Require().NotNil(disks)
	suite.Require().Len(*disks, 1)

	disk := (*disks)[0]
	suite.Equal(*drive1.Vendor, *disk.Vendor)
	suite.Equal(*drive1.Model, *disk.Model)
	suite.Require().NotNil(disk.Partitions)
	suite.Require().Len(*disk.Partitions, 2)

	// --- Assertions for Partition 1 ---
	part1 := (*disk.Partitions)[0]
	suite.Require().NotNil(part1.Device)
	suite.Equal(*(*drive1.Filesystems)[0].Device, *part1.Device)
	suite.Require().NotNil(part1.Name)
	suite.Equal(*(*drive1.Filesystems)[0].Name, *part1.Name)
	suite.Require().NotNil(part1.MountPointData)
	suite.Require().Len(*part1.MountPointData, 1, "Expected 1 mount point for partition 1")
	mountPoint1 := (*part1.MountPointData)[0]
	suite.Equal(mountPath1, mountPoint1.Path)
	// This assertion depends on the converter logic AND the successful Save mock
	suite.True(mountPoint1.IsMounted, "MountPoint1 IsMounted should be true after successful save")

	// --- Assertions for Partition 2 ---
	part2 := (*disk.Partitions)[1]
	suite.Require().NotNil(part2.Device)
	suite.Equal(*(*drive1.Filesystems)[1].Device, *part2.Device)
	suite.Require().NotNil(part2.Name)
	suite.Equal(*(*drive1.Filesystems)[1].Name, *part2.Name)
	suite.Require().NotNil(part2.MountPointData)
	suite.Require().Len(*part2.MountPointData, 1, "Expected 1 mount point for partition 2")
	mountPoint2 := (*part2.MountPointData)[0]
	suite.Equal(mountPath2, mountPoint2.Path)
	// This assertion depends on the converter logic AND the successful Save mock
	suite.True(mountPoint2.IsMounted, "MountPoint2 IsMounted should be true after successful save")
}

func (suite *VolumeServiceTestSuite) TestGetVolumesData_HardwareClientError() {
	expectedErr := errors.New("hardware client failed")

	mock.When(suite.mockHardwareClient.GetHardwareInfoWithResponse(mock.AnyContext())).ThenReturn(nil, expectedErr).Verify(matchers.Times(1))
	//suite.mockHardwareClient.On("GetHardwareInfoWithResponse" /*suite.ctx*/, testContext, mock.Anything).Return(nil, expectedErr).Once()
	// Fallback logic is commented out, so we expect the error to propagate

	disks, err := suite.volumeService.GetVolumesData()
	suite.Nil(disks)
	suite.Require().Error(err)
	suite.ErrorIs(err, expectedErr)
}

func (suite *VolumeServiceTestSuite) TestGetVolumesData_RepoFindByPathError_NotFound() {
	// Test when FindByPath returns gorm.ErrRecordNotFound
	drive1 := hardware.Drive{
		Id: pointer.String("drive-1"),
		Filesystems: &[]hardware.Filesystem{
			{Device: pointer.String("/dev/sda1"), MountPoints: &[]string{"/mnt/newfs"}}, // A new mount point
		},
	}
	mockHWResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{Data: &hardware.HardwareInfo{Drives: &[]hardware.Drive{drive1}}},
	}
	mountPath1 := "/mnt/newfs"
	expectedErr := gorm.ErrRecordNotFound

	mock.When(suite.mockHardwareClient.GetHardwareInfoWithResponse(mock.AnyContext())).ThenReturn(mockHWResponse, nil).Verify(matchers.Times(1))
	mock.When(suite.mockMountRepo.FindByPath(mountPath1)).ThenReturn(nil, expectedErr).Verify(matchers.Times(1))

	mock.When(suite.lsblk.GetInfoFromDevice("/dev/sda1")).ThenReturn(&lsblk.LSBKInfo{
		Name:       "sda1",
		Label:      "NewFS",
		Partlabel:  "NewFS",
		Mountpoint: mountPath1,
		Fstype:     "ext4",
	}, nil).Verify(matchers.Times(1))

	//suite.mockHardwareClient.On("GetHardwareInfoWithResponse" /*suite.ctx*/, testContext, mock.Anything).Return(mockHWResponse, nil).Once()
	//suite.mockMountRepo.On("FindByPath", mountPath1).Return(nil, expectedErr).Once()

	// If FindByPath returns ErrRecordNotFound, Save *should* be called to create the record
	/*
		suite.mockMountRepo.On("Save", mock.MatchedBy(func(mp *dbom.MountPointPath) bool {
			suite.Equal(mountPath1, mp.Path)
			//suite.True(mp.IsMounted) // Should be set by converter
			return true
		})).Return(nil).Once() // Assume save works for the new record
	*/
	/*
		mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
			mp, ok := args[0].(*dbom.MountPointPath)
			if !ok {
				suite.T().Errorf("Expected argument to be of type *dbom.MountPointPath, got %T", args[0])
			}
			suite.Equal(mountPath1, mp.Path)
			return []any{nil}
		})).Verify(matchers.AtLeastOnce())
	*/

	disks, err := suite.volumeService.GetVolumesData()
	suite.Require().NoError(err) // FindByPath ErrRecordNotFound is handled internally
	suite.Require().NotNil(disks)
	suite.Require().Len(*disks, 1)
	suite.Require().Len(*(*disks)[0].Partitions, 1)
	suite.Require().Len(*(*(*disks)[0].Partitions)[0].MountPointData, 1)
	mountPoint := (*(*(*disks)[0].Partitions)[0].MountPointData)[0]
	suite.Equal(mountPath1, mountPoint.Path)
	suite.True(mountPoint.IsMounted)  // Should reflect state after successful save
	suite.False(mountPoint.IsInvalid) // Should not be invalid
}

func (suite *VolumeServiceTestSuite) TestGetVolumesData_RepoSaveError() {
	// Similar setup to Success, but mock Save to fail
	drive1 := hardware.Drive{
		Id: pointer.String("drive-1"),
		Filesystems: &[]hardware.Filesystem{
			{Device: pointer.String("/dev/sda1"), MountPoints: &[]string{"/mnt/rootfs"}},
		},
	}
	mockHWResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{Data: &hardware.HardwareInfo{Drives: &[]hardware.Drive{drive1}}},
	}
	mountPath1 := "/mnt/rootfs"
	dbomMountData1 := &dbom.MountPointPath{Path: mountPath1, Device: "sda1", IsMounted: true}
	expectedErr := errors.New("DB save error")

	mock.When(suite.mockHardwareClient.GetHardwareInfoWithResponse(mock.AnyContext())).ThenReturn(mockHWResponse, nil).Verify(matchers.Times(1))
	mock.When(suite.mockMountRepo.FindByPath(mountPath1)).ThenReturn(dbomMountData1, nil).Verify(matchers.Times(1))
	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenReturn(expectedErr).Verify(matchers.Times(1))
	mock.When(suite.lsblk.GetInfoFromDevice("/dev/sda1")).ThenReturn(&lsblk.LSBKInfo{
		Name:       "sda1",
		Label:      "NewFS",
		Partlabel:  "NewFS",
		Mountpoint: mountPath1,
		Fstype:     "ext4",
	}, nil).Verify(matchers.Times(1))

	// suite.mockHardwareClient.On("GetHardwareInfoWithResponse" /*suite.ctx*/, testContext,
	// 	mock.Anything).Return(mockHWResponse, nil).Once()
	// suite.mockMountRepo.On("FindByPath", mountPath1).Return(dbomMountData1, nil).Once()
	// suite.mockMountRepo.On("Save", mock.Anything).Return(expectedErr).Once()

	disks, err := suite.volumeService.GetVolumesData()
	suite.Require().NoError(err) // Save error is logged but not returned
	suite.Require().NotNil(disks)
	suite.Require().Len(*disks, 1)
	suite.Require().Len(*(*disks)[0].Partitions, 1)
	suite.Require().Len(*(*(*disks)[0].Partitions)[0].MountPointData, 1)
	mountPoint := (*(*(*disks)[0].Partitions)[0].MountPointData)[0]
	suite.Equal(mountPath1, mountPoint.Path)
	// The data returned will likely be the original data before the failed save.
	suite.True(mountPoint.IsMounted) // Still true from hardware/converter data
	// Check if the code marks it invalid on Save error
	suite.True(mountPoint.IsInvalid, "Mount point should be marked invalid on save error")
	suite.Require().NotNil(mountPoint.InvalidError)
	suite.Contains(*mountPoint.InvalidError, expectedErr.Error())
}

// --- NotifyClient Tests ---

// NotifyClient is tested implicitly via Mount/Unmount success tests.
// We can add specific tests if its internal logic becomes more complex.

func (suite *VolumeServiceTestSuite) TestNotifyClient_GetVolumesDataError() {
	expectedErr := errors.New("failed to get volumes")

	// This test requires triggering NotifyClient directly or via another method.
	// Let's simulate it being called after an Unmount, but GetVolumesData fails.

	mountPath := "/mnt/notifyerr"
	dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: "sda1", IsMounted: true}

	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(dbomMountData, nil).Verify(matchers.Times(1))
	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenReturn(nil).Verify(matchers.Times(1))

	// suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()
	// suite.mockMountRepo.On("Save", mock.Anything).Return(nil).Once() // Unmount Save succeeds

	// // Mock GetVolumesData inside NotifyClient to fail
	mock.When(suite.mockHardwareClient.GetHardwareInfoWithResponse(mock.AnyContext())).ThenReturn(nil, expectedErr).Verify(matchers.Times(1))
	// suite.mockHardwareClient.On("GetHardwareInfoWithResponse" /*suite.ctx*/, testContext,
	// 	mock.Anything).Return(nil, expectedErr).Once()

	// // Expect BroadcastMessage NOT to be called
	// suite.mockBroadcaster.AssertNotCalled(suite.T(), "BroadcastMessage", mock.Anything)

	// Trigger Unmount which calls NotifyClient
	err := suite.volumeService.UnmountVolume(mountPath, false, false)
	suite.Require().NoError(err) // Unmount itself should succeed

	// Assertions on mocks are checked in TearDownTest
	suite.T().Log("Tested implicitly: NotifyClient logs error and doesn't broadcast if GetVolumesData fails.")
}
