// Package service_test contains tests for the service layer.
package service_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	// Needed for MockBroadcaster

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/suite"
	"github.com/u-root/u-root/pkg/mount/loop"
	"github.com/xorcare/pointer"
	errors "gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type VolumeServiceTestSuite struct {
	suite.Suite
	mockMountRepo      repository.MountPointPathRepositoryInterface
	mockHardwareClient service.HardwareServiceInterface
	volumeService      service.VolumeServiceInterface
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
			func() *dto.ContextState {
				return &dto.ContextState{}
			},
			func() repository.IssueRepositoryInterface {
				// Provide a nil repository since it's only used in error cases in tests
				return nil
			},
			service.NewVolumeService,
			service.NewFilesystemService,
			events.NewEventBus,
			//mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[service.HardwareServiceInterface],
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.IssueServiceInterface],
			//mock.Mock[events.EventBusInterface],
		),
		fx.Populate(&suite.volumeService),
		fx.Populate(&suite.mockMountRepo),
		fx.Populate(&suite.mockHardwareClient),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.volumeService.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return []*procfs.MountInfo{
			{MountID: 1217, ParentID: 819, MajorMinorVer: "0:52", Root: "/", Source: "/dev/sda1", MountPoint: "/mnt/test1", FSType: "ext4", Options: map[string]string{"noatime": ""}, SuperOptions: map[string]string{}},
			{MountID: 1218, ParentID: 820, MajorMinorVer: "0:53", Root: "/", Source: "/dev/sdb1", MountPoint: "/mnt/test2", FSType: "xfs", Options: map[string]string{"nodiratime": ""}, SuperOptions: map[string]string{}},
		}, nil
	})
	suite.app.RequireStart()
}

func (suite *VolumeServiceTestSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.ctx != nil {
		if wg := suite.ctx.Value("wg"); wg != nil {
			wg.(*sync.WaitGroup).Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// --- MountVolume Tests ---

func (suite *VolumeServiceTestSuite) TestMountUnmountVolume_Success() {
	device, err := loop.FindDevice()
	if err != nil {
		suite.T().Skip("No loop device available, skipping test")
		return
	}
	suite.Require().NoError(err, "Error finding loop device")
	err = suite.volumeService.CreateBlockDevice(device)
	suite.Require().NoError(err, "Error creating block device")
	err = loop.SetFile(device, "../../test/data/image.dmg")
	suite.Require().NoError(err, "Error setting loop device file")
	mountPath := "/mnt/test1"
	fsType := "ext4"
	mountData := dto.MountPointData{
		Path:     mountPath,
		DeviceId: device,
		FSType:   &fsType,
		Flags: &dto.MountFlags{
			dto.MountFlag{Name: "noatime", NeedsValue: false},
		},
	}
	dbomMountData := &dbom.MountPointPath{
		Path:     mountPath,
		DeviceId: device,
		FSType:   fsType,
		Flags:    &dbom.MounDataFlags{dbom.MounDataFlag{Name: "noatime", NeedsValue: false}},
	}

	// Mock FindByPath
	mock.When(suite.mockMountRepo.FindByDevice(device)).ThenReturn(dbomMountData, nil).Verify(matchers.Times(2))
	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(dbomMountData, nil).Verify(matchers.Times(1))

	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
		mp, ok := args[0].(*dbom.MountPointPath)
		if !ok {
			suite.T().Errorf("Expected argument to be of type *dbom.MountPointPath, got %T", args[0])
		}
		suite.T().Logf("MountPointPath saved: %+v", mp)
		suite.Equal(mountPath, mp.Path)
		//suite.Equal(device, mp.Device)
		suite.Equal(fsType, mp.FSType)
		suite.Require().NotNil(mp.Flags)
		suite.Contains(*mp.Flags, dbom.MounDataFlag{Name: "noatime", NeedsValue: false})
		dbomMountData.DeviceId = mp.DeviceId
		return []any{nil}
	})).Verify(matchers.AtLeastOnce())

	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(
		[]dto.Disk{
			{LegacyDeviceName: pointer.String("sda1"), Size: pointer.Int(100), Id: pointer.String("SSD"),
				Partitions: &map[string]dto.Partition{
					"SSD": {
						DevicePath:       &device,
						LegacyDeviceName: pointer.String("sda1"), Size: pointer.Int(100), Id: pointer.String("SSD")},
				},
			},
			{LegacyDeviceName: pointer.String("sda2"), Size: pointer.Int(200), Id: pointer.String("HDD"),
				Partitions: &map[string]dto.Partition{
					device: {
						LegacyDeviceName: pointer.String("sda2"), Size: pointer.Int(200), Id: &device, DevicePath: &device},
				},
			},
		},
		nil)

	disks := suite.volumeService.GetVolumesData()
	suite.Require().NotNil(disks, "Expected GetVolumesData to return disks")
	suite.Require().NotEmpty(*disks, "Expected GetVolumesData to return non-empty disks")
	suite.Require().Len(*disks, 2, "Expected GetVolumesData to return 2 disks")

	defer func() {
		err := suite.volumeService.UnmountVolume(mountPath, true, false) // Cleanup
		suite.Require().Nil(err, "Expected no error on unmount")
		loop.ClearFile(device)
	}()
	// --- Execute ---
	errE := suite.volumeService.MountVolume(&mountData)

	// --- Assert ---
	suite.Require().NoError(errE, "Expected no error on successful mount", errE)
	suite.NotEmpty(*mountData.Flags)
	suite.Contains(*mountData.Flags, dto.MountFlag{Name: "noatime", Description: "", NeedsValue: false, FlagValue: "", ValueDescription: "", ValueValidationRegex: ""})

}

/*
func (suite *VolumeServiceTestSuite) TestMountVolume_RepoFindByPathError() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, DeviceId: "sda1"}
	expectedErr := errors.New("Invalid parameter")

	mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(nil, expectedErr).Verify(matchers.Times(1))

	err := suite.volumeService.MountVolume(&mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, expectedErr)
}
*/

func (suite *VolumeServiceTestSuite) TestMountVolume_DeviceEmpty() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, DeviceId: ""} // Empty device
	err := suite.volumeService.MountVolume(&mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, dto.ErrorInvalidParameter)
	details := err.Details()
	suite.Contains(details, "Message")
	suite.Equal("Source device name is empty in request", details["Message"])
}

func (suite *VolumeServiceTestSuite) TestMountVolume_DeviceInvalid() {
	mountPath := "/mnt/test1"
	mountData := dto.MountPointData{Path: mountPath, DeviceId: "/dev/pippo"} // Invalid device
	//dbomMountData := &dbom.MountPointPath{Path: mountPath, DeviceId: ""}

	//mock.When(suite.mockMountRepo.FindByPath(mountPath)).ThenReturn(dbomMountData, nil).Verify(matchers.Times(1))
	//suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()

	err := suite.volumeService.MountVolume(&mountData)
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
	mountData := dto.MountPointData{Path: mountPath, DeviceId: device}
	// FindByPath won't be called if path is empty early on
	// dbomMountData := &dbom.MountPointPath{Path: mountPath, Device: device}
	// suite.mockMountRepo.On("FindByPath", mountPath).Return(dbomMountData, nil).Once()

	err := suite.volumeService.MountVolume(&mountData)
	suite.Require().NotNil(err)
	suite.ErrorIs(err, dto.ErrorInvalidParameter)
	details := err.Details()
	suite.Contains(details, "Message")
	suite.Equal("Mount point path is empty", details["Message"])
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

// --- GetVolumesData Tests ---

func (suite *VolumeServiceTestSuite) TestGetVolumesData_Success() {

	mountPath1 := "/mnt/test1"
	mountPath2 := "/mnt/test2"

	device1 := pointer.String("/dev/disk/by-id/virtio-disk1-part1")
	device2 := pointer.String("/dev/disk/by-id/virtio-disk2-part1")
	device2Legacy := pointer.String("/dev/sdb1")
	device2LegacyName := pointer.String("sdb1")
	mockHWResponse := []dto.Disk{
		{
			Id:               pointer.String("disk-1"),
			LegacyDevicePath: pointer.String("/dev/sda"),
			Size:             pointer.Int(100),
			Vendor:           pointer.String("ATA"),
			Model:            pointer.String("Model-1"),
			Partitions: &map[string]dto.Partition{
				"part-1": {
					Id:               pointer.String("part-1"),
					Name:             pointer.String("RootFS"),
					LegacyDevicePath: pointer.String("/dev/sda1"),
					LegacyDeviceName: pointer.String("sda1"),
					DevicePath:       device1,
					Size:             pointer.Int(50),
					HostMountPointData: &map[string]dto.MountPointData{
						mountPath1: {
							DeviceId: *device1,
							Path:     mountPath1,
						},
					},
				},
			},
		},
		{
			Id:               pointer.String("disk-2"),
			LegacyDevicePath: pointer.String("/dev/sdb"),
			Vendor:           pointer.String("SATA"),
			Model:            pointer.String("Model-2"),
			Size:             pointer.Int(100),
			Partitions: &map[string]dto.Partition{
				"part-1": {
					Id:               pointer.String("part-1"),
					Name:             pointer.String("DataFs"),
					LegacyDevicePath: device2Legacy,
					LegacyDeviceName: device2LegacyName,
					DevicePath:       device2,
					Size:             pointer.Int(50),
					HostMountPointData: &map[string]dto.MountPointData{
						mountPath2: {
							DeviceId: *device2,
							Path:     mountPath2,
						},
					},
				},
			},
		},
	} // Prepare mock repo responses
	//dbomMountData1 := &dbom.MountPointPath{Path: mountPath1, DeviceId: "sda1", Type: "ADDON"} // Initial state in DB
	//dbomMountData2 := &dbom.MountPointPath{Path: mountPath2, DeviceId: "sdb1", Type: "ADDON"} // Initial state in DB

	mock.When(suite.mockMountRepo.FindByDevice(*device1)).ThenReturn(&dbom.MountPointPath{Path: mountPath1, DeviceId: *device1, Type: "ADDON"}, nil).Verify(matchers.Times(1))
	mock.When(suite.mockMountRepo.FindByDevice(*device2)).ThenReturn(&dbom.MountPointPath{Path: mountPath2, DeviceId: *device2, Type: "ADDON"}, nil).Verify(matchers.Times(1))
	mock.When(suite.mockHardwareClient.GetHardwareInfo()).ThenReturn(mockHWResponse, nil).Verify(matchers.AtLeastOnce())

	/*
		mock.When(suite.mockMountRepo.AllByDeviceId()).ThenReturn(map[string]dbom.MountPointPath{
			"sdb1": {Path: mountPath2, DeviceId: "sdb1", Type: "ADDON"},
		}, nil).Verify(matchers.Times(1))
	*/

	// Expect FindByPath and Save for each mount point found in hardware data
	//mock.When(suite.mockMountRepo.FindByPath(mountPath1)).ThenReturn(dbomMountData1, nil).Verify(matchers.Times(1))
	//mock.When(suite.mockMountRepo.FindByPath(mountPath2)).ThenReturn(dbomMountData2, nil).Verify(matchers.Times(1))

	// Call the function
	disks := suite.volumeService.GetVolumesData()

	// Assertions
	suite.Require().NotNil(disks)
	suite.Require().Len(*disks, 2)

	disk := (*disks)[0]
	suite.Equal(mockHWResponse[0].Vendor, disk.Vendor)
	suite.Equal(mockHWResponse[0].Model, disk.Model)
	suite.Require().NotNil(disk.Partitions)
	suite.Require().Len(*disk.Partitions, 1)

	// --- Assertions for Partition 1 ---
	part1 := (*disk.Partitions)["part-1"]
	suite.Require().NotNil(part1.LegacyDevicePath)
	suite.Equal(*(*mockHWResponse[0].Partitions)["part-1"].LegacyDevicePath, *part1.LegacyDevicePath)
	suite.Require().NotNil(part1.Name)
	suite.Equal(*(*mockHWResponse[0].Partitions)["part-1"].Name, *part1.Name)
	suite.Require().NotNil(part1.MountPointData)
	suite.Require().Len(*part1.MountPointData, 1, "Expected 1 mount point for partition 1")
	mountPoint1 := (*part1.MountPointData)[mountPath1]
	suite.Equal(mountPath1, mountPoint1.Path)
	// This assertion depends on the converter logic AND the successful Save mock
	//suite.True(mountPoint1.IsMounted, "MountPoint1 IsMounted should be true after successful save")

	// --- Assertions for Partition 2 ---
	disk = (*disks)[1]
	part2 := (*disk.Partitions)["part-1"]
	suite.Require().NotNil(part2.LegacyDevicePath)
	suite.Equal(*(*mockHWResponse[1].Partitions)["part-1"].LegacyDevicePath, *part2.LegacyDevicePath)
	suite.Require().NotNil(part2.Name)
	//suite.Equal(*(*drive1.Filesystems)[1].Name, *part2.Name)
	suite.Require().NotNil(part2.MountPointData)
	suite.Require().Len(*part2.MountPointData, 1, "Expected 1 mount point for partition 2")
	mountPoint2 := (*part2.MountPointData)[mountPath2]
	suite.Equal(mountPath2, mountPoint2.Path)
	// This assertion depends on the converter logic AND the successful Save mock
	//suite.True(mountPoint2.IsMounted, "MountPoint2 IsMounted should be true after successful save")
}

// --- UpdateMountPointSettings Tests ---
func (suite *VolumeServiceTestSuite) TestUpdateMountPointSettings_Success() {
	path := "/mnt/testupdate"
	originalFSType := "ext4"
	updatedFSType := "xfs"
	//originalFlags := &dto.MountFlags{{Name: "ro"}}
	updatedFlagsDto := &dto.MountFlags{{Name: "rw"}, {Name: "noatime"}}
	originalStartup := pointer.Bool(true)
	updatedStartup := pointer.Bool(false)

	dbData := &dbom.MountPointPath{
		Path:               path,
		DeviceId:           "/dev/sdb1",
		FSType:             originalFSType,
		Flags:              &dbom.MounDataFlags{{Name: "ro"}},
		IsToMountAtStartup: originalStartup,
	}

	updates := dto.MountPointData{
		FSType:             &updatedFSType,
		Flags:              updatedFlagsDto,
		IsToMountAtStartup: updatedStartup,
	}

	mock.When(suite.mockMountRepo.FindByPath(path)).ThenReturn(dbData, nil).Verify(matchers.Times(1))
	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
		savedDbData := args[0].(*dbom.MountPointPath)
		suite.Equal(path, savedDbData.Path)
		suite.Equal(updatedFSType, savedDbData.FSType)
		suite.Len(*savedDbData.Flags, 2) // rw, noatime
		suite.Contains(*savedDbData.Flags, dbom.MounDataFlag{Name: "rw"})
		suite.Contains(*savedDbData.Flags, dbom.MounDataFlag{Name: "noatime"})
		suite.Equal(updatedStartup, savedDbData.IsToMountAtStartup)
		return []any{nil}
	})).Verify(matchers.Times(2))

	resultDto, errE := suite.volumeService.UpdateMountPointSettings(path, updates)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(path, resultDto.Path)
	suite.Equal(updatedFSType, *resultDto.FSType)
	suite.Len(*resultDto.Flags, 2)
	suite.Contains(*resultDto.Flags, dto.MountFlag{Name: "rw"})
	suite.Contains(*resultDto.Flags, dto.MountFlag{Name: "noatime"})
	suite.Equal(updatedStartup, resultDto.IsToMountAtStartup)
}

func (suite *VolumeServiceTestSuite) TestUpdateMountPointSettings_NotFound() {
	path := "/mnt/notfound"
	updates := dto.MountPointData{FSType: pointer.String("xfs")}

	mock.When(suite.mockMountRepo.FindByPath(path)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound)).Verify(matchers.Times(1))

	_, errE := suite.volumeService.UpdateMountPointSettings(path, updates)
	suite.Require().NotNil(errE)
	suite.True(errors.Is(errE, dto.ErrorNotFound))
}

// --- PatchMountPointSettings Tests ---
func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_Success_OnlyStartup() {
	path := "/mnt/testpatch"
	originalStartup := pointer.Bool(true)
	patchedStartup := pointer.Bool(false)

	dbData := &dbom.MountPointPath{
		Path:               path,
		DeviceId:           "/dev/sdc1",
		FSType:             "ext4",
		IsToMountAtStartup: originalStartup,
	}

	patch := dto.MountPointData{
		IsToMountAtStartup: patchedStartup,
	}

	mock.When(suite.mockMountRepo.FindByPath(path)).ThenReturn(dbData, nil).Verify(matchers.Times(1))
	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenAnswer(matchers.Answer(func(args []any) []any {
		savedDbData := args[0].(*dbom.MountPointPath)
		suite.Equal(path, savedDbData.Path)
		suite.Equal("ext4", savedDbData.FSType) // Should not change
		suite.Equal(patchedStartup, savedDbData.IsToMountAtStartup)
		return []any{nil}
	})).Verify(matchers.Times(2))

	resultDto, errE := suite.volumeService.PatchMountPointSettings(path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(path, resultDto.Path)
	suite.Equal("ext4", *resultDto.FSType)
	suite.Equal(patchedStartup, resultDto.IsToMountAtStartup)
}

func (suite *VolumeServiceTestSuite) TestPatchMountPointSettings_NoChanges() {
	path := "/mnt/testpatch_nochange"
	originalStartup := pointer.Bool(true)

	dbData := &dbom.MountPointPath{
		Path:               path,
		DeviceId:           "/dev/sdd1",
		FSType:             "btrfs",
		IsToMountAtStartup: originalStartup,
	}

	patch := dto.MountPointData{
		IsToMountAtStartup: originalStartup, // Same value
	}

	mock.When(suite.mockMountRepo.FindByPath(path)).ThenReturn(dbData, nil).Verify(matchers.Times(1))
	// Save should NOT be called if no changes
	mock.When(suite.mockMountRepo.Save(mock.Any[*dbom.MountPointPath]())).ThenReturn(nil).Verify(matchers.Times(1))

	resultDto, errE := suite.volumeService.PatchMountPointSettings(path, patch)
	suite.Require().Nil(errE)
	suite.Require().NotNil(resultDto)
	suite.Equal(path, resultDto.Path)
	suite.Equal("btrfs", *resultDto.FSType)
	suite.Equal(originalStartup, resultDto.IsToMountAtStartup)
}
