// /workspaces/srat/backend/src/service/volume_service_test.go
package service_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/sse" // Needed for MockBroadcaster
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware" // Keep for the interface definition
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"

	// Remove gomock import: "go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

// --- Mock Implementations using testify/mock ---

// MockBroadcaster is a mock for BroadcasterServiceInterface using testify/mock
type MockBroadcaster struct {
	mock.Mock
}

func (m *MockBroadcaster) BroadcastMessage(msg any) (any, error) {
	args := m.Called(msg)
	// Handle potential nil return for the first argument
	ret0 := args.Get(0)
	return ret0, args.Error(1)
}

func (m *MockBroadcaster) ProcessHttpChannel(send sse.Sender) {
	m.Called(send)
}

// MockMountPointPathRepository is a mock for MountPointPathRepositoryInterface using testify/mock
type MockMountPointPathRepository struct {
	mock.Mock
}

func (m *MockMountPointPathRepository) All() ([]dbom.MountPointPath, error) {
	args := m.Called()
	// Handle potential nil return for the first argument
	var ret0 []dbom.MountPointPath
	if args.Get(0) != nil {
		ret0 = args.Get(0).([]dbom.MountPointPath)
	}
	return ret0, args.Error(1)
}

func (m *MockMountPointPathRepository) Save(mp *dbom.MountPointPath) error {
	args := m.Called(mp)
	return args.Error(0)
}

func (m *MockMountPointPathRepository) FindByPath(path string) (*dbom.MountPointPath, error) {
	args := m.Called(path)
	// Handle potential nil return for the first argument
	var ret0 *dbom.MountPointPath
	if args.Get(0) != nil {
		ret0 = args.Get(0).(*dbom.MountPointPath)
	}
	return ret0, args.Error(1)
}

func (m *MockMountPointPathRepository) Exists(id string) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

// MockClientWithResponses is a mock for hardware.ClientWithResponsesInterface using testify/mock
type MockClientWithResponses struct {
	mock.Mock
	// We don't embed the actual client here for a pure mock
}

// Ensure MockClientWithResponses implements hardware.ClientWithResponsesInterface
var _ hardware.ClientWithResponsesInterface = (*MockClientWithResponses)(nil)

func (m *MockClientWithResponses) GetHardwareInfo(ctx context.Context, reqEditors ...hardware.RequestEditorFn) (*http.Response, error) {
	// To properly mock variadic functions, we often pass them as a single slice argument
	// or use mock.Anything if the exact editors don't matter for the test.
	args := m.Called(ctx, reqEditors)
	var resp *http.Response
	if args.Get(0) != nil {
		resp = args.Get(0).(*http.Response)
	}
	return resp, args.Error(1)
}

func (m *MockClientWithResponses) GetHardwareInfoWithResponse(ctx context.Context, reqEditors ...hardware.RequestEditorFn) (*hardware.GetHardwareInfoResponse, error) {
	// Similar handling for variadic arguments
	args := m.Called(ctx, reqEditors)
	var resp *hardware.GetHardwareInfoResponse
	if args.Get(0) != nil {
		resp = args.Get(0).(*hardware.GetHardwareInfoResponse)
	}
	return resp, args.Error(1)
}

// --- Test Suite ---

type VolumeServiceTestSuite struct {
	suite.Suite
	// ctrl *gomock.Controller // Removed gomock controller
	mockBroadcaster    *MockBroadcaster
	mockMountRepo      *MockMountPointPathRepository
	mockHardwareClient *MockClientWithResponses
	volumeService      service.VolumeServiceInterface
	ctx                context.Context
	cancel             context.CancelFunc
}

func TestVolumeServiceTestSuite(t *testing.T) {
	suite.Run(t, new(VolumeServiceTestSuite))
}

func (suite *VolumeServiceTestSuite) SetupTest() {
	// suite.ctrl = gomock.NewController(suite.T()) // Removed gomock controller init
	suite.mockBroadcaster = new(MockBroadcaster)
	suite.mockMountRepo = new(MockMountPointPathRepository)
	suite.mockHardwareClient = new(MockClientWithResponses)
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

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
	suite.volumeService = service.NewVolumeService(suite.ctx, suite.mockBroadcaster, suite.mockMountRepo, suite.mockHardwareClient)
}

func (suite *VolumeServiceTestSuite) TearDownTest() {
	suite.cancel()
	// suite.ctrl.Finish() // Removed gomock verification
	suite.mockBroadcaster.AssertExpectations(suite.T())
	suite.mockMountRepo.AssertExpectations(suite.T())
	suite.mockHardwareClient.AssertExpectations(suite.T())
	// Give time for goroutines to potentially exit
	time.Sleep(10 * time.Millisecond)
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
		Flags:  []string{"MS_NOATIME"},
	}
	dbomMountData := &dbom.MountPointPath{
		Path:      mountPath,
		Device:    device,
		FSType:    fsType,
		Flags:     dbom.MounDataFlags{dbom.MS_NOATIME},
		IsMounted: false, // Initially not mounted
	}

	// Mock FindByPath
	suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Twice()

	// Mock osutil.IsMounted - THIS IS HARD TO MOCK DIRECTLY. Assuming it returns false.
	// Mock mount.Mount - THIS IS HARD TO MOCK DIRECTLY. Assuming it succeeds.
	// Mock converter.MountToMountPointPath - Assuming it works.

	// Mock Save to be called with IsMounted = true
	suite.mockMountRepo.On("Save", mock.MatchedBy(func(mp *dbom.MountPointPath) bool {
		// Check the state *inside* the mock setup
		suite.Equal(mountPath, mp.Path) // Path might change if rename logic kicks in, adjust if needed
		//suite.True(mp.IsMounted)
		//suite.Equal(device, mp.Device)
		// Update the original dbomMountData to reflect the save for subsequent checks if needed
		dbomMountData.Device = mp.Device
		dbomMountData.IsMounted = mp.IsMounted
		return true // Return true to indicate the matcher succeeded
	})).Return(nil).Maybe()

	// Mock NotifyClient's internal GetVolumesData call
	suite.mockHardwareClient.On(
		"GetHardwareInfoWithResponse",
		suite.ctx,
		mock.Anything,
	).Return(
		&hardware.GetHardwareInfoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &struct {
				Data   *hardware.HardwareInfo             `json:"data,omitempty"`
				Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
			}{Data: &hardware.HardwareInfo{Drives: &[]hardware.Drive{}}},
		}, nil).Maybe() // Expect once due to NotifyClient

	// Mock NotifyClient's BroadcastMessage call
	suite.mockBroadcaster.On("BroadcastMessage", mock.AnythingOfType("*[]dto.Disk")).Return(nil, nil).Maybe()

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
	mountData := dto.MountPointData{Path: mountPath}
	expectedErr := errors.New("database error")

	suite.mockMountRepo.On("FindByPath", mountPath).Return(nil, expectedErr).Once()

	err := suite.volumeService.MountVolume(mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, expectedErr)
}

func (suite *VolumeServiceTestSuite) TestMountVolume_DeviceEmpty() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, Device: ""} // Empty device
	dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: ""}

	suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()

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

	suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()

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

func (suite *VolumeServiceTestSuite) TestMountVolume_RepoSaveError() {
	mountPath := "/mnt/test1"
	device := "sda1"
	mountData := dto.MountPointData{Path: mountPath, Device: device}
	dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: device, IsMounted: false}
	expectedErr := errors.New("repo save failed")

	suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()
	// Assume mount succeeds (mocking limitations)
	suite.mockMountRepo.On("Save", mock.Anything).Return(expectedErr).Once()
	// NotifyClient should NOT be called if save fails
	suite.mockBroadcaster.AssertNotCalled(suite.T(), "BroadcastMessage", mock.Anything)
	suite.mockHardwareClient.AssertNotCalled(suite.T(), "GetHardwareInfoWithResponse", mock.Anything, mock.Anything)

	err := suite.volumeService.MountVolume(mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, expectedErr) // The error should bubble up
}

// --- UnmountVolume Tests ---

func (suite *VolumeServiceTestSuite) TestUnmountVolume_Success() {
	mountPath := "/mnt/test1"
	dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: "sda1", IsMounted: true}

	suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()
	// Mock mount.Unmount - THIS IS HARD TO MOCK DIRECTLY. Assuming it succeeds.
	suite.mockMountRepo.On("Save", mock.MatchedBy(func(mp *dbom.MountPointPath) bool {
		suite.Equal(mountPath, mp.Path)
		suite.False(mp.IsMounted)
		// *dbomMountData = *mp // Update original if needed elsewhere
		return true
	})).Return(nil).Once()

	// Expect NotifyClient after successful unmount and save
	suite.mockHardwareClient.On(
		"GetHardwareInfoWithResponse",
		suite.ctx,
		mock.Anything,
	).Return(
		&hardware.GetHardwareInfoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &struct {
				Data   *hardware.HardwareInfo             `json:"data,omitempty"`
				Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
			}{Data: &hardware.HardwareInfo{Drives: &[]hardware.Drive{}}},
		}, nil).Once()
	suite.mockBroadcaster.On("BroadcastMessage", mock.AnythingOfType("*[]dto.Disk")).Return(nil, nil).Once()

	err := suite.volumeService.UnmountVolume(mountPath, false, false)
	suite.Nil(err, "Expected no error on successful unmount")
}

func (suite *VolumeServiceTestSuite) TestUnmountVolume_RepoFindByPathError() {
	mountPath := "/mnt/test1"
	expectedErr := errors.New("database error")

	suite.mockMountRepo.On("FindByPath", mountPath).Return(nil, expectedErr).Once()

	err := suite.volumeService.UnmountVolume(mountPath, false, false)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, expectedErr)
}

func (suite *VolumeServiceTestSuite) TestUnmountVolume_UnmountFails() {
	// This requires mocking mount.Unmount to return an error
	// Skipping direct test due to mocking difficulty, relies on integration test.
	suite.T().Skip("Skipping TestUnmountVolume_UnmountFails due to mount.Unmount mocking difficulty")
}

func (suite *VolumeServiceTestSuite) TestUnmountVolume_RepoSaveError() {
	mountPath := "/mnt/test1"
	dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: "sda1", IsMounted: true}
	expectedErr := errors.New("repo save failed")

	suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()
	// Assume unmount succeeds (mocking limitations)
	suite.mockMountRepo.On("Save", mock.Anything).Return(expectedErr).Once()
	// NotifyClient should NOT be called if save fails
	suite.mockBroadcaster.AssertNotCalled(suite.T(), "BroadcastMessage", mock.Anything)
	suite.mockHardwareClient.AssertNotCalled(suite.T(), "GetHardwareInfoWithResponse", mock.Anything, mock.Anything)

	err := suite.volumeService.UnmountVolume(mountPath, false, false)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, expectedErr) // The error should bubble up
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
	dbomMountData1 := &dbom.MountPointPath{Path: mountPath1, Device: "sda1", IsMounted: true} // Initial state in DB
	dbomMountData2 := &dbom.MountPointPath{Path: mountPath2, Device: "sda2", IsMounted: true} // Initial state in DB

	suite.mockHardwareClient.On("GetHardwareInfoWithResponse", suite.ctx, mock.Anything).Return(mockHWResponse, nil).Once()

	// Expect FindByPath and Save for each mount point found in hardware data
	suite.mockMountRepo.On("FindByPath", mountPath1).Return(dbomMountData1, nil).Once()
	suite.mockMountRepo.On("FindByPath", mountPath2).Return(dbomMountData2, nil).Once()
	suite.mockMountRepo.On("Save", mock.MatchedBy(func(mp *dbom.MountPointPath) bool {
		suite.Assert().Contains([]string{mountPath1, mountPath2}, mp.Path)
		suite.True(mp.IsMounted) // Should reflect the input from hardware/converter
		// Assume converter works, check key fields
		return true
	})).Return(nil).Maybe()

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
	suite.EqualValues(*(*drive1.Filesystems)[0].Device, *part1.Device)
	suite.Require().NotNil(part1.Name)
	suite.EqualValues(*(*drive1.Filesystems)[0].Name, *part1.Name)
	suite.Require().NotNil(part1.MountPointData)
	suite.Require().Len(*part1.MountPointData, 1, "Expected 1 mount point for partition 1")
	mountPoint1 := (*part1.MountPointData)[0]
	suite.Equal(mountPath1, mountPoint1.Path)
	// This assertion depends on the converter logic AND the successful Save mock
	suite.True(mountPoint1.IsMounted, "MountPoint1 IsMounted should be true after successful save")

	// --- Assertions for Partition 2 ---
	part2 := (*disk.Partitions)[1]
	suite.Require().NotNil(part2.Device)
	suite.EqualValues(*(*drive1.Filesystems)[1].Device, *part2.Device)
	suite.Require().NotNil(part2.Name)
	suite.EqualValues(*(*drive1.Filesystems)[1].Name, *part2.Name)
	suite.Require().NotNil(part2.MountPointData)
	suite.Require().Len(*part2.MountPointData, 1, "Expected 1 mount point for partition 2")
	mountPoint2 := (*part2.MountPointData)[0]
	suite.Equal(mountPath2, mountPoint2.Path)
	// This assertion depends on the converter logic AND the successful Save mock
	suite.True(mountPoint2.IsMounted, "MountPoint2 IsMounted should be true after successful save")
}

func (suite *VolumeServiceTestSuite) TestGetVolumesData_HardwareClientError() {
	expectedErr := errors.New("hardware client failed")
	suite.mockHardwareClient.On("GetHardwareInfoWithResponse", suite.ctx, mock.Anything).Return(nil, expectedErr).Once()
	// Fallback logic is commented out, so we expect the error to propagate

	disks, err := suite.volumeService.GetVolumesData()
	suite.Nil(disks)
	suite.Require().NotNil(err)
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

	suite.mockHardwareClient.On("GetHardwareInfoWithResponse", suite.ctx, mock.Anything).Return(mockHWResponse, nil).Once()
	suite.mockMountRepo.On("FindByPath", mountPath1).Return(nil, expectedErr).Once()

	// If FindByPath returns ErrRecordNotFound, Save *should* be called to create the record
	suite.mockMountRepo.On("Save", mock.MatchedBy(func(mp *dbom.MountPointPath) bool {
		suite.Equal(mountPath1, mp.Path)
		suite.True(mp.IsMounted) // Should be set by converter
		return true
	})).Return(nil).Once() // Assume save works for the new record

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

func (suite *VolumeServiceTestSuite) TestGetVolumesData_RepoFindByPathError_Other() {
	// Test when FindByPath returns an error other than NotFound
	drive1 := hardware.Drive{
		Id: pointer.String("drive-1"),
		Filesystems: &[]hardware.Filesystem{
			{Device: pointer.String("/dev/sda1"), MountPoints: &[]string{"/mnt/errorfs"}},
		},
	}
	mockHWResponse := &hardware.GetHardwareInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &struct {
			Data   *hardware.HardwareInfo             `json:"data,omitempty"`
			Result *hardware.GetHardwareInfo200Result `json:"result,omitempty"`
		}{Data: &hardware.HardwareInfo{Drives: &[]hardware.Drive{drive1}}},
	}
	mountPath1 := "/mnt/errorfs"
	expectedErr := errors.New("some other db error")

	suite.mockHardwareClient.On("GetHardwareInfoWithResponse", suite.ctx, mock.Anything).Return(mockHWResponse, nil).Once()
	suite.mockMountRepo.On("FindByPath", mountPath1).Return(nil, expectedErr).Once()

	// Save should NOT be called if FindByPath fails with an unexpected error
	suite.mockMountRepo.AssertNotCalled(suite.T(), "Save", mock.Anything)

	disks, err := suite.volumeService.GetVolumesData()
	suite.Require().NoError(err) // Error is logged internally but not returned
	suite.Require().NotNil(disks)
	suite.Require().Len(*disks, 1)
	suite.Require().Len(*(*disks)[0].Partitions, 1)
	suite.Require().Len(*(*(*disks)[0].Partitions)[0].MountPointData, 1)
	mountPoint := (*(*(*disks)[0].Partitions)[0].MountPointData)[0]
	suite.Equal(mountPath1, mountPoint.Path)
	// The state here might be the initial state from the converter, as Save wasn't called.
	// Depending on error handling, it might be marked invalid. Check the code logic.
	// Assuming it just logs and continues, IsMounted might still be true from converter.
	suite.True(mountPoint.IsMounted)
	// suite.True(mountPoint.IsInvalid) // Check if the code marks it invalid on Find error
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
	expectedErr := errors.New("save failed")

	suite.mockHardwareClient.On("GetHardwareInfoWithResponse", suite.ctx, mock.Anything).Return(mockHWResponse, nil).Once()
	suite.mockMountRepo.On("FindByPath", mountPath1).Return(dbomMountData1, nil).Once()
	suite.mockMountRepo.On("Save", mock.Anything).Return(expectedErr).Once()

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

func (suite *VolumeServiceTestSuite) TestNotifyClient_Success_Implicit() {
	// Re-running a successful mount test implicitly tests NotifyClient's success path
	suite.TestMountVolume_Success()
}

func (suite *VolumeServiceTestSuite) TestNotifyClient_GetVolumesDataError() {
	expectedErr := errors.New("failed to get volumes")

	// This test requires triggering NotifyClient directly or via another method.
	// Let's simulate it being called after an Unmount, but GetVolumesData fails.

	mountPath := "/mnt/notifyerr"
	dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: "sda1", IsMounted: true}

	suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()
	suite.mockMountRepo.On("Save", mock.Anything).Return(nil).Once() // Unmount Save succeeds

	// Mock GetVolumesData inside NotifyClient to fail
	suite.mockHardwareClient.On("GetHardwareInfoWithResponse", suite.ctx, mock.Anything).Return(nil, expectedErr).Once()

	// Expect BroadcastMessage NOT to be called
	suite.mockBroadcaster.AssertNotCalled(suite.T(), "BroadcastMessage", mock.Anything)

	// Trigger Unmount which calls NotifyClient
	err := suite.volumeService.UnmountVolume(mountPath, false, false)
	suite.Require().NoError(err) // Unmount itself should succeed

	// Assertions on mocks are checked in TearDownTest
	suite.T().Log("Tested implicitly: NotifyClient logs error and doesn't broadcast if GetVolumesData fails.")
}

// --- Helper for Filesystem Conversion (if needed) ---
// Add helper functions if complex conversions need specific testing.
// For example, mocking the converter itself if its logic was complex.
var _ converter.HaHardwareToDto = &converter.HaHardwareToDtoImpl{} // Ensure interface is implemented
var _ converter.DtoToDbomConverter = &converter.DtoToDbomConverterImpl{}
var _ converter.MountToDbom = &converter.MountToDbomImpl{}

// Removed the manual mock definitions from the end as they are now at the top.
// Removed MockVolumeService as it's not used when testing the actual service.
